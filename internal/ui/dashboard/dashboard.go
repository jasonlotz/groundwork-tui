// Package dashboard provides the main dashboard TUI screen.
package dashboard

import (
	"fmt"
	"strings"

	bbprogress "github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
	"github.com/jasonlotz/groundwork-tui/internal/ui/progress"
)

// NavigateMsg is sent when the user navigates to another screen.
type NavigateMsg string

const (
	ScreenMaterials  NavigateMsg = "materials"
	ScreenSkills     NavigateMsg = "skills"
	ScreenProgress   NavigateMsg = "progress"
	ScreenCategories NavigateMsg = "categories"
)

// OpenMaterialMsg is sent when the user presses enter on an active material.
type OpenMaterialMsg struct{ MaterialID string }

// --- messages ---

type overviewLoadedMsg struct{ data *model.Overview }
type activeMaterialsLoadedMsg struct{ data []model.ActiveMaterial }

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
	spinner         spinner.Model
	bar             bbprogress.Model
	keys            common.SimpleKeyMap
	overlay         *progress.LogForm
}

func New(client *api.Client) Model {
	return Model{
		client:  client,
		loading: true,
		spinner: common.NewSpinner(),
		bar:     common.NewProgressBar(20),
		keys: common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("enter", "detail"),
			common.KB("l", "log progress"),
			common.KB("r", "refresh"),
			common.KB("q", "quit"),
		}},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadOverview(m.client),
		loadActiveMaterials(m.client),
		m.spinner.Tick,
	)
}

func loadOverview(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetOverview()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return overviewLoadedMsg{data}
	}
}

func loadActiveMaterials(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetActiveMaterials()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return activeMaterialsLoadedMsg{data}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Route to overlay when active.
	if m.overlay != nil {
		if k, ok := msg.(tea.KeyMsg); ok && (k.String() == "ctrl+c" || k.String() == "q") {
			return m, tea.Quit
		}
		updated, cmd := m.overlay.Update(msg)
		if lf, ok := updated.(progress.LogForm); ok {
			m.overlay = &lf
		}
		if done, ok := msg.(progress.LogDoneMsg); ok {
			m.overlay = nil
			if !done.Cancelled {
				return m, func() tea.Msg { return common.ProgressLoggedMsg{} }
			}
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case overviewLoadedMsg:
		m.overview = msg.data

	case activeMaterialsLoadedMsg:
		m.activeMaterials = msg.data
		m.loading = false

	case common.MaterialChangedMsg, common.ProgressLoggedMsg:
		return m, tea.Batch(loadOverview(m.client), loadActiveMaterials(m.client))

	case common.ErrMsg:
		m.err = msg.Err
		m.loading = false

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

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
		case "enter":
			if len(m.activeMaterials) > 0 {
				id := m.activeMaterials[m.cursor].ID
				return m, func() tea.Msg { return OpenMaterialMsg{MaterialID: id} }
			}
		case "l":
			if len(m.activeMaterials) > 0 {
				mat := m.activeMaterials[m.cursor]
				lf := progress.NewLogForm(m.client, mat.ID, mat.Name)
				m.overlay = &lf
				return m, m.overlay.Init()
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return common.SpinnerView(m.spinner)
	}
	if m.err != nil {
		return common.ErrorView(m.err, m.width)
	}

	var b strings.Builder

	// Title
	b.WriteString(common.RenderTitle("Groundwork", m.width))
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
		// RenderTitle=3 + blank=1 + KPIs=3 + blank=1 + Section=2 + blank=1 + help=2 = 13 overhead; tab bar=3 → 16; each item is 4 lines
		visibleItems := (m.height - 16) / 4
		if visibleItems < 2 {
			visibleItems = 2
		}
		start, end := common.VisibleWindow(m.cursor, len(m.activeMaterials), visibleItems)
		for i := start; i < end; i++ {
			b.WriteString(m.renderMaterialRow(i, m.activeMaterials[i]))
			b.WriteString("\n")
		}
		if len(m.activeMaterials) > visibleItems {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d\n", start+1, end, len(m.activeMaterials),
			)))
		}
	}

	// Help bar
	b.WriteString("\n")
	b.WriteString(common.RenderHelp(m.keys, m.width))

	if m.overlay != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.overlay.View())
	}
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
	return common.RenderKPICards(cards)
}

func (m Model) renderMaterialRow(i int, mat model.ActiveMaterial) string {
	selected := i == m.cursor
	cursor := "  "
	if selected {
		cursor = common.SelectedStyle.Render("▶ ")
	}

	skillColor := ""
	if mat.Skill.Color != nil {
		skillColor = *mat.Skill.Color
	}
	dot := common.ColorDot(skillColor)

	nameStyle := common.DefaultNameStyle
	if selected {
		nameStyle = common.SelectedStyle
	}
	skill := common.MutedStyle.Render(common.Truncate(mat.SkillName(), 18))
	name := nameStyle.Render(common.Truncate(mat.Name, 36))
	pctLabel := common.MutedStyle.Render(fmt.Sprintf("%.0f%%", mat.PctComplete))
	line1 := cursor + dot + " " + name + "  " + skill + "  " + pctLabel

	// Bar width: terminal width minus indent and labels.
	barWidth := common.ClampBarWidth(m.width)

	// Day-of-week pace: Mon=1/7 … Sun=7/7 (matches web app logic).
	pacePct := common.PaceFraction()

	// Line 2: weekly goal bar (or blank spacer if no goal).
	var line2 string
	if mat.WeeklyUnitGoal != nil && *mat.WeeklyUnitGoal > 0 {
		weeklyPct := mat.UnitsThisWeek / float64(*mat.WeeklyUnitGoal)
		bar := common.RenderWeeklyBar(barWidth, weeklyPct, pacePct)
		label := fmt.Sprintf("%.4g / %d %s this week",
			mat.UnitsThisWeek, *mat.WeeklyUnitGoal, mat.UnitType.Label())
		if mat.ProjectedEndDate != nil {
			label += common.MutedStyle.Render("  · est. " + common.FormatProjectedDate(*mat.ProjectedEndDate))
		}
		line2 = "    " + bar + "  " + common.MutedStyle.Render(label)
	} else {
		// No goal — show a dim placeholder so row height stays consistent.
		line2 = "    " + common.MutedStyle.Render("no weekly goal set")
	}

	// Line 3: overall progress bar.
	overallPct := 0.0
	if mat.TotalUnits > 0 {
		overallPct = mat.CompletedUnits / mat.TotalUnits
	}
	overallBar := common.RenderBar(m.bar, overallPct, barWidth)
	overallLabel := fmt.Sprintf("%.4g / %.4g %s overall",
		mat.CompletedUnits, mat.TotalUnits, mat.UnitType.Label())
	line3 := "    " + overallBar + "  " + common.MutedStyle.Render(overallLabel)

	return line1 + "\n" + line2 + "\n" + line3 + "\n"
}
