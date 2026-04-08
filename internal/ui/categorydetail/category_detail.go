// Package categorydetail provides the category detail TUI screen.
package categorydetail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

type dataLoadedMsg struct{ data *model.CategoryDetail }

// OpenSkillMsg is sent when the user presses enter on a skill.
type OpenSkillMsg struct{ SkillID string }

// Model is the Bubble Tea model for the category detail screen.
type Model struct {
	client     *api.Client
	categoryID string
	data       *model.CategoryDetail
	cursor     int
	loading    bool
	err        error
	width      int
	height     int
	spinner    spinner.Model
}

func New(client *api.Client, categoryID string) Model {
	return Model{client: client, categoryID: categoryID, loading: true, spinner: common.NewSpinner()}
}

func load(c *api.Client, categoryID string) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetCategoryData(categoryID)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return dataLoadedMsg{data}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(load(m.client, m.categoryID), m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case dataLoadedMsg:
		m.data = msg.data
		m.loading = false

	case common.ErrMsg:
		m.err = msg.Err
		m.loading = false

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return common.GoBackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.data != nil && m.cursor < len(m.data.SkillsSummary)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.data != nil && len(m.data.SkillsSummary) > 0 {
				id := m.data.SkillsSummary[m.cursor].ID
				return m, func() tea.Msg { return OpenSkillMsg{SkillID: id} }
			}
		case "r":
			m.loading = true
			m.err = nil
			return m, load(m.client, m.categoryID)
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return common.SpinnerView(m.spinner)
	}
	if m.err != nil {
		return common.ErrorView(m.err)
	}
	if m.data == nil {
		return ""
	}

	d := m.data
	var b strings.Builder

	// Title
	b.WriteString(common.TitleStyle.Render(d.Category.Name))
	b.WriteString("\n")

	// KPI row
	cards := []string{
		common.StatCard("Materials", fmt.Sprintf("%d active / %d done", d.ActiveMaterialCount, d.CompletedMaterialCount)),
		common.StatCard("Total", fmt.Sprintf("%d", d.TotalMaterials)),
		common.StatCard("Progress", fmt.Sprintf("%.1f%%", d.PctCompleted)),
		common.StatCard("This week", fmt.Sprintf("%.1f%%", d.PctThisWeek)),
	}
	b.WriteString(common.RenderKPICards(cards))
	b.WriteString("\n")

	// Active materials (brief list)
	if len(d.ActiveMaterials) > 0 {
		b.WriteString(common.SectionStyle.Render("Active Materials"))
		b.WriteString("\n")
		limit := 5
		if len(d.ActiveMaterials) < limit {
			limit = len(d.ActiveMaterials)
		}
		for _, mat := range d.ActiveMaterials[:limit] {
			pct := 0.0
			if mat.TotalUnits > 0 {
				pct = mat.CompletedUnits / mat.TotalUnits
			}
			bar := common.ProgressBar(pct, 16)
			skillLabel := common.MutedStyle.Render(common.Truncate(mat.SkillName, 16))
			name := common.Truncate(mat.Name, 28)
			b.WriteString(fmt.Sprintf("  %s  %s  %s\n", bar, common.MutedStyle.Render(name), skillLabel))
		}
		if len(d.ActiveMaterials) > limit {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf("  … and %d more\n", len(d.ActiveMaterials)-limit)))
		}
	}

	// Skills list
	b.WriteString(common.SectionStyle.Render("Skills"))
	b.WriteString("\n")

	if len(d.SkillsSummary) == 0 {
		b.WriteString(common.MutedStyle.Render("  No skills.\n"))
	} else {
		// Reserve: title(2) + kpis(3) + active header+rows(~7) + skills header(2) + help(2)
		usedLines := 16
		if len(d.ActiveMaterials) == 0 {
			usedLines = 9
		}
		visibleHeight := m.height - usedLines
		if visibleHeight < 3 {
			visibleHeight = 3
		}
		start, end := common.VisibleWindow(m.cursor, len(d.SkillsSummary), visibleHeight)

		for i := start; i < end; i++ {
			b.WriteString(m.renderSkillRow(i))
			b.WriteString("\n")
		}
		if len(d.SkillsSummary) > visibleHeight {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d skills\n", start+1, end, len(d.SkillsSummary),
			)))
		}
	}

	b.WriteString("\n")
	keys := []string{
		common.KeyHelp("j/k", "navigate skills"),
		common.KeyHelp("enter", "open skill"),
		common.KeyHelp("r", "refresh"),
		common.KeyHelp("esc", "back"),
	}
	b.WriteString(common.HelpStyle.Render(strings.Join(keys, "   ")))
	return b.String()
}

func (m Model) renderSkillRow(i int) string {
	s := m.data.SkillsSummary[i]
	cursorStr := "  "
	nameStyle := common.MutedStyle
	if i == m.cursor {
		cursorStr = common.SelectedStyle.Render("▶ ")
		nameStyle = common.SelectedStyle
	}

	archived := ""
	if s.IsArchived {
		archived = common.MutedStyle.Render(" [archived]")
	}

	pct := 0.0
	if s.TotalUnits > 0 {
		pct = s.CompletedUnits / s.TotalUnits
	}
	bar := common.ProgressBar(pct, 12)
	meta := common.MutedStyle.Render(fmt.Sprintf("%d active / %d total", s.ActiveMaterialCount, s.MaterialCount))

	return cursorStr + nameStyle.Render(common.Truncate(s.Name, 24)) + archived + "  " + bar + "  " + meta
}
