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
type errMsg struct{ err error }

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
			return errMsg{err}
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

	case errMsg:
		m.err = msg.err
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
		return common.MutedStyle.Render("\n  Loading…")
	}
	if m.err != nil {
		return common.DangerStyle.Render("\n  Error: " + m.err.Error() + "\n\n  Press r to retry, esc to go back.")
	}

	var b strings.Builder

	// Header
	title := "Materials"
	if m.activeOnly {
		title += common.MutedStyle.Render("  [active only]")
	}
	b.WriteString(common.TitleStyle.Render(title))
	b.WriteString("\n")

	// Column headers
	b.WriteString(renderHeader())
	b.WriteString("\n")

	if len(m.filtered) == 0 {
		b.WriteString(common.MutedStyle.Render("  No materials found.\n"))
	} else {
		// Visible window
		visibleHeight := m.height - 8
		if visibleHeight < 5 {
			visibleHeight = 5
		}
		start, end := visibleWindow(m.cursor, len(m.filtered), visibleHeight)
		for i := start; i < end; i++ {
			b.WriteString(m.renderRow(i))
			b.WriteString("\n")
		}
		if len(m.filtered) > visibleHeight {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d\n", start+1, end, len(m.filtered),
			)))
		}
	}

	b.WriteString("\n")
	b.WriteString(m.renderHelp())
	return b.String()
}

func renderHeader() string {
	hdr := lipgloss.NewStyle().Foreground(common.ColorSubtle).Bold(true)
	return hdr.Render(fmt.Sprintf("  %-32s %-12s %-10s %s", "Name", "Skill", "Progress", "Status"))
}

func (m Model) renderRow(i int) string {
	mat := m.filtered[i]

	cursorStr := "  "
	nameStyle := lipgloss.NewStyle()
	if i == m.cursor {
		cursorStr = common.SelectedStyle.Render("▶ ")
		nameStyle = common.SelectedStyle
	}

	pct := 0.0
	if mat.TotalUnits > 0 {
		pct = mat.CompletedUnits / mat.TotalUnits
	}

	statusStyle := common.MutedStyle
	switch mat.Status {
	case model.StatusActive:
		statusStyle = common.SuccessStyle
	case model.StatusComplete:
		statusStyle = lipgloss.NewStyle().Foreground(common.ColorPrimary)
	}

	name := nameStyle.Render(truncate(mat.Name, 30))
	skill := common.MutedStyle.Render(truncate(mat.SkillName, 10))
	progress := common.MutedStyle.Render(fmt.Sprintf("%.0f%%", pct*100))
	status := statusStyle.Render(strings.ToLower(string(mat.Status)))

	return fmt.Sprintf("%s%-32s %-12s %-10s %s",
		cursorStr, name, skill, progress, status)
}

func (m Model) renderHelp() string {
	filter := "a  active only"
	if m.activeOnly {
		filter = "a  all materials"
	}
	keys := []string{
		common.KeyHelp("j/k", "navigate"),
		common.KeyHelp("a", strings.TrimPrefix(filter, "a  ")),
		common.KeyHelp("r", "refresh"),
		common.KeyHelp("esc", "back"),
		common.KeyHelp("q", "back"),
	}
	return common.HelpStyle.Render(strings.Join(keys, "   "))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
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
