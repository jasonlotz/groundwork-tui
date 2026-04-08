// Package skilldetail provides the skill detail TUI screen.
package skilldetail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

type dataLoadedMsg struct{ data *model.SkillDetail }

// OpenMaterialMsg is sent when the user presses enter on a material.
type OpenMaterialMsg struct{ MaterialID string }

// LogFromSkillMsg is sent when the user presses l on a material.
type LogFromSkillMsg struct{ MaterialID string }

// Model is the Bubble Tea model for the skill detail screen.
type Model struct {
	client  *api.Client
	skillID string
	data    *model.SkillDetail
	cursor  int
	loading bool
	err     error
	width   int
	height  int
	spinner spinner.Model
	bar     progress.Model
}

func New(client *api.Client, skillID string) Model {
	return Model{client: client, skillID: skillID, loading: true, spinner: common.NewSpinner(), bar: common.NewProgressBar(16)}
}

func load(c *api.Client, skillID string) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetSkillData(skillID)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return dataLoadedMsg{data}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(load(m.client, m.skillID), m.spinner.Tick)
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
			if m.data != nil && m.cursor < len(m.data.AllMaterials)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.data != nil && len(m.data.AllMaterials) > 0 {
				id := m.data.AllMaterials[m.cursor].ID
				return m, func() tea.Msg { return OpenMaterialMsg{MaterialID: id} }
			}
		case "l":
			if m.data != nil && len(m.data.AllMaterials) > 0 {
				mat := m.data.AllMaterials[m.cursor]
				if mat.Status == model.StatusActive {
					id := mat.ID
					return m, func() tea.Msg { return LogFromSkillMsg{MaterialID: id} }
				}
				return m, func() tea.Msg {
					return common.ToastMsg{Text: "Only active materials can be logged.", IsError: true}
				}
			}
		case "r":
			m.loading = true
			m.err = nil
			return m, load(m.client, m.skillID)
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

	// Title + breadcrumb
	crumb := common.MutedStyle.Render(d.Skill.Category.Name + " › ")
	b.WriteString(common.TitleStyle.Render(crumb + d.Skill.Name))
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

	// Materials list
	b.WriteString(common.SectionStyle.Render("Materials"))
	b.WriteString("\n")

	if len(d.AllMaterials) == 0 {
		b.WriteString(common.MutedStyle.Render("  No materials.\n"))
	} else {
		// title(2) + kpis(3) + section(2) + help(2) = 9
		visibleItems := (m.height - 9) / 2
		if visibleItems < 3 {
			visibleItems = 3
		}
		start, end := common.VisibleWindow(m.cursor, len(d.AllMaterials), visibleItems)

		for i := start; i < end; i++ {
			b.WriteString(m.renderMaterialRow(i))
			b.WriteString("\n")
		}
		if len(d.AllMaterials) > visibleItems {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d\n", start+1, end, len(d.AllMaterials),
			)))
		}
	}

	b.WriteString("\n")
	keys := []string{
		common.KeyHelp("j/k", "navigate"),
		common.KeyHelp("enter", "detail"),
		common.KeyHelp("l", "log progress"),
		common.KeyHelp("r", "refresh"),
		common.KeyHelp("esc", "back"),
	}
	b.WriteString(common.HelpStyle.Render(strings.Join(keys, "   ")))
	return b.String()
}

func (m Model) renderMaterialRow(i int) string {
	mat := m.data.AllMaterials[i]
	selected := i == m.cursor

	cursorStr := "  "
	nameStyle := lipgloss.NewStyle()
	if selected {
		cursorStr = common.SelectedStyle.Render("▶ ")
		nameStyle = common.SelectedStyle
	}

	pct := 0.0
	if mat.TotalUnits > 0 {
		pct = mat.CompletedUnits / mat.TotalUnits
	}
	bar := common.RenderBar(m.bar, pct)
	progress := common.MutedStyle.Render(fmt.Sprintf("%.4g/%.4g %s", mat.CompletedUnits, mat.TotalUnits, mat.UnitType.Label()))

	statusStyle := common.MutedStyle
	statusStr := "inactive"
	switch mat.Status {
	case model.StatusActive:
		statusStyle = common.SuccessStyle
		statusStr = "active"
	case model.StatusComplete:
		statusStyle = lipgloss.NewStyle().Foreground(common.ColorPrimary)
		statusStr = "done"
	}

	typeName := common.MutedStyle.Render(common.Truncate(mat.MaterialType.Name, 14))
	name := nameStyle.Render(common.Truncate(mat.Name, 32))
	line1 := cursorStr + name + "  " + statusStyle.Render(statusStr)
	line2 := "    " + bar + "  " + progress + "  " + typeName
	return line1 + "\n" + line2
}
