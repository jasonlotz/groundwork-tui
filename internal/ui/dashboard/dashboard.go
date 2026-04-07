// Package dashboard provides the main dashboard TUI screen.
package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

// NavigateMsg is sent when the user navigates to another screen.
type NavigateMsg string

const (
	ScreenMaterials   NavigateMsg = "materials"
	ScreenSkills      NavigateMsg = "skills"
	ScreenProgress    NavigateMsg = "progress"
	ScreenLogProgress NavigateMsg = "log"
)

// --- messages ---

type overviewLoadedMsg struct{ data *model.Overview }
type activeMaterialsLoadedMsg struct{ data []model.ActiveMaterial }
type errMsg struct{ err error }

// --- model ---

type Model struct {
	client          *api.Client
	overview        *model.Overview
	activeMaterials []model.ActiveMaterial
	cursor          int
	loading         bool
	err             error
	width           int
	height          int
}

func New(client *api.Client) Model {
	return Model{
		client:  client,
		loading: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadOverview(m.client),
		loadActiveMaterials(m.client),
	)
}

func loadOverview(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetOverview()
		if err != nil {
			return errMsg{err}
		}
		return overviewLoadedMsg{data}
	}
}

func loadActiveMaterials(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetActiveMaterials()
		if err != nil {
			return errMsg{err}
		}
		return activeMaterialsLoadedMsg{data}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case overviewLoadedMsg:
		m.overview = msg.data

	case activeMaterialsLoadedMsg:
		m.activeMaterials = msg.data
		m.loading = false

	case errMsg:
		m.err = msg.err
		m.loading = false

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			m.err = nil
			return m, tea.Batch(loadOverview(m.client), loadActiveMaterials(m.client))
		case "j", "down":
			if m.cursor < len(m.activeMaterials)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "m":
			return m, func() tea.Msg { return ScreenMaterials }
		case "s":
			return m, func() tea.Msg { return ScreenSkills }
		case "p":
			return m, func() tea.Msg { return ScreenProgress }
		case "l":
			return m, func() tea.Msg { return ScreenLogProgress }
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return common.MutedStyle.Render("\n  Loading…")
	}
	if m.err != nil {
		return common.DangerStyle.Render("\n  Error: " + m.err.Error() + "\n\n  Press r to retry, q to quit.")
	}

	var b strings.Builder

	// Title
	b.WriteString(common.TitleStyle.Render("Groundwork"))
	b.WriteString("\n")

	// KPI row
	if m.overview != nil {
		b.WriteString(m.renderKPIs())
		b.WriteString("\n")
	}

	// Active materials list
	b.WriteString(common.SectionStyle.Render("Active Materials"))
	b.WriteString("\n")

	if len(m.activeMaterials) == 0 {
		b.WriteString(common.MutedStyle.Render("  No active materials.\n"))
	} else {
		for i, mat := range m.activeMaterials {
			b.WriteString(m.renderMaterialRow(i, mat))
			b.WriteString("\n")
		}
	}

	// Help bar
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m Model) renderKPIs() string {
	o := m.overview
	streak := fmt.Sprintf("%d day streak", o.CurrentStreak)
	if o.CurrentStreak == 1 {
		streak = "1 day streak"
	}

	cards := []string{
		common.StatCard("Active", fmt.Sprintf("%d", o.ActiveMaterials)),
		common.StatCard("Completed", fmt.Sprintf("%d", o.CompletedCount)),
		common.StatCard("Progress", fmt.Sprintf("%.0f%%", o.CompletionPct)),
		common.StatCard("Streak", streak),
	}

	rendered := make([]string, len(cards))
	for i, c := range cards {
		rendered[i] = common.BorderStyle.Render(c)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m Model) renderMaterialRow(i int, mat model.ActiveMaterial) string {
	cursor := "  "
	style := lipgloss.NewStyle()
	if i == m.cursor {
		cursor = common.SelectedStyle.Render("▶ ")
		style = common.SelectedStyle
	}

	// Progress bar (20 chars wide)
	pct := 0.0
	if mat.TotalUnits > 0 {
		pct = mat.CompletedUnits / mat.TotalUnits
	}
	bar := common.ProgressBar(pct, 20)

	// Units info
	units := fmt.Sprintf("%.0f / %.0f %s", mat.CompletedUnits, mat.TotalUnits, mat.UnitType.Label())

	// Weekly goal
	weeklyInfo := ""
	if mat.WeeklyUnitGoal != nil && *mat.WeeklyUnitGoal > 0 {
		weeklyPct := mat.UnitsThisWeek / float64(*mat.WeeklyUnitGoal)
		weekColor := common.SuccessStyle
		if weeklyPct < 0.5 {
			weekColor = common.DangerStyle
		} else if weeklyPct < 1.0 {
			weekColor = common.WarningStyle
		}
		weeklyInfo = "  " + weekColor.Render(fmt.Sprintf("%.0f/%d this week", mat.UnitsThisWeek, *mat.WeeklyUnitGoal))
	}

	// Projected end
	projInfo := ""
	if mat.ProjectedEndDate != nil {
		projInfo = common.MutedStyle.Render("  est. " + *mat.ProjectedEndDate)
	}

	name := style.Render(mat.Name)
	skill := common.MutedStyle.Render(mat.SkillName)

	line1 := cursor + name + "  " + common.MutedStyle.Render(skill)
	line2 := "    " + bar + "  " + common.MutedStyle.Render(units) + weeklyInfo + projInfo

	return line1 + "\n" + line2
}

func (m Model) renderHelp() string {
	keys := []string{
		common.KeyHelp("j/k", "navigate"),
		common.KeyHelp("l", "log progress"),
		common.KeyHelp("m", "materials"),
		common.KeyHelp("s", "skills"),
		common.KeyHelp("p", "progress log"),
		common.KeyHelp("r", "refresh"),
		common.KeyHelp("q", "quit"),
	}
	return common.HelpStyle.Render(strings.Join(keys, "   "))
}
