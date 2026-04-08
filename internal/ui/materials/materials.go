// Package materials provides the materials list TUI screen.
package materials

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

type materialsLoadedMsg struct{ data []model.Material }

// LogFromMaterialMsg is sent when the user presses l to log progress on the selected material.
type LogFromMaterialMsg struct{ MaterialID string }

// OpenMaterialMsg is sent when the user presses enter on the selected material.
type OpenMaterialMsg struct{ MaterialID string }

// Model is the Bubble Tea model for the materials screen.
type Model struct {
	client     *api.Client
	materials  []model.Material
	filtered   []model.Material
	cursor     int
	activeOnly bool
	loading    bool
	err        error
	width      int
	height     int
}

func New(client *api.Client) Model {
	return Model{
		client:     client,
		activeOnly: false,
		loading:    true,
	}
}

func load(c *api.Client, activeOnly bool) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetAllMaterials(activeOnly)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return materialsLoadedMsg{data}
	}
}

func (m Model) Init() tea.Cmd {
	return load(m.client, m.activeOnly)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case materialsLoadedMsg:
		m.materials = msg.data
		m.applyFilter()
		m.loading = false

	case common.ErrMsg:
		m.err = msg.Err
		m.loading = false

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return common.GoBackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "a":
			m.activeOnly = !m.activeOnly
			m.loading = true
			m.cursor = 0
			return m, load(m.client, m.activeOnly)
		case "r":
			m.loading = true
			m.err = nil
			return m, load(m.client, m.activeOnly)
		case "l":
			if len(m.filtered) > 0 {
				mat := m.filtered[m.cursor]
				if mat.Status == model.StatusActive {
					id := mat.ID
					return m, func() tea.Msg { return LogFromMaterialMsg{MaterialID: id} }
				}
				return m, func() tea.Msg {
					return common.ToastMsg{Text: "Only active materials can be logged.", IsError: true}
				}
			}
		case "enter":
			if len(m.filtered) > 0 {
				id := m.filtered[m.cursor].ID
				return m, func() tea.Msg { return OpenMaterialMsg{MaterialID: id} }
			}
		}
	}
	return m, nil
}

func (m *Model) applyFilter() {
	m.filtered = m.materials
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m Model) View() string {
	if m.loading {
		return common.LoadingView()
	}
	if m.err != nil {
		return common.ErrorView(m.err)
	}

	var b strings.Builder

	// Header
	filterTag := ""
	if m.activeOnly {
		filterTag = "  " + common.MutedStyle.Render("[active only]")
	}
	b.WriteString(common.TitleStyle.Render("Materials") + filterTag)
	b.WriteString("\n")

	if len(m.filtered) == 0 {
		b.WriteString(common.MutedStyle.Render("  No materials found.\n"))
	} else {
		// title(1) + marginBottom(1) + blank(1) + help(1) + marginTop(1) = 5; each item is 2 lines
		visibleItems := (m.height - 5) / 2
		if visibleItems < 3 {
			visibleItems = 3
		}
		start, end := common.VisibleWindow(m.cursor, len(m.filtered), visibleItems)
		for i := start; i < end; i++ {
			b.WriteString(m.renderRow(i))
			b.WriteString("\n")
		}
		if len(m.filtered) > visibleItems {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d\n", start+1, end, len(m.filtered),
			)))
		}
	}

	b.WriteString("\n")
	b.WriteString(m.renderHelp())
	return b.String()
}

func (m Model) renderRow(i int) string {
	mat := m.filtered[i]

	selected := i == m.cursor
	cursorStr := "  "
	nameStyle := lipgloss.NewStyle()
	if selected {
		cursorStr = common.SelectedStyle.Render("▶ ")
		nameStyle = common.SelectedStyle
	}

	// Progress
	pct := 0.0
	if mat.TotalUnits > 0 {
		pct = mat.CompletedUnits / mat.TotalUnits
	}
	bar := common.ProgressBar(pct, 18)
	progressText := common.MutedStyle.Render(fmt.Sprintf(
		"%.4g / %.4g %s", mat.CompletedUnits, mat.TotalUnits, mat.UnitType.Label(),
	))

	// Status badge
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
	status := statusStyle.Render(statusStr)

	// Type + skill
	meta := common.MutedStyle.Render(common.Truncate(mat.TypeName(), 14)) +
		"  " + common.MutedStyle.Render(common.Truncate(mat.SkillName(), 18))

	// Weekly goal indicator (only for active)
	weeklyInfo := ""
	if mat.Status == model.StatusActive && mat.WeeklyUnitGoal != nil && *mat.WeeklyUnitGoal > 0 {
		// We don't have unitsThisWeek on Material; just show the goal
		weeklyInfo = "  " + common.MutedStyle.Render(fmt.Sprintf("goal: %d/%s", *mat.WeeklyUnitGoal, mat.UnitType.Label()))
	}

	name := nameStyle.Render(common.Truncate(mat.Name, 36))

	line1 := cursorStr + name + "  " + status
	line2 := "    " + bar + "  " + progressText + "  " + meta + weeklyInfo

	return line1 + "\n" + line2
}

func (m Model) renderHelp() string {
	filterLabel := "active only"
	if m.activeOnly {
		filterLabel = "all materials"
	}
	keys := []string{
		common.KeyHelp("j/k", "navigate"),
		common.KeyHelp("enter", "detail"),
		common.KeyHelp("l", "log progress"),
		common.KeyHelp("a", filterLabel),
		common.KeyHelp("r", "refresh"),
		common.KeyHelp("esc", "back"),
	}
	return common.HelpStyle.Render(strings.Join(keys, "   "))
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n-1]) + "…"
}

func visibleWindow(cursor, total, height int) (start, end int) {
	if total <= height {
		return 0, total
	}
	start = cursor - height/2
	if start < 0 {
		start = 0
	}
	end = start + height
	if end > total {
		end = total
		start = end - height
	}
	return start, end
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
