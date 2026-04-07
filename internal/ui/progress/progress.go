// Package progress provides the progress log history TUI screen.
package progress

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

type logsLoadedMsg struct{ data []model.ProgressLog }
type errMsg struct{ err error }

// Model is the Bubble Tea model for the progress history screen.
type Model struct {
	client  *api.Client
	logs    []model.ProgressLog
	cursor  int
	loading bool
	err     error
	width   int
	height  int
}

func New(client *api.Client) Model {
	return Model{client: client, loading: true}
}

func load(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetAllProgress(nil)
		if err != nil {
			return errMsg{err}
		}
		return logsLoadedMsg{data}
	}
}

func (m Model) Init() tea.Cmd {
	return load(m.client)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case logsLoadedMsg:
		m.logs = msg.data
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
			if m.cursor < len(m.logs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "r":
			m.loading = true
			m.err = nil
			return m, load(m.client)
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return common.MutedStyle.Render("\n  Loading…")
	}
	if m.err != nil {
		return common.DangerStyle.Render("\n  Error: " + m.err.Error() + "\n\n  Press r to retry, esc to go back.")
	}

	var b strings.Builder
	b.WriteString(common.TitleStyle.Render("Progress Log"))
	b.WriteString("\n")

	if len(m.logs) == 0 {
		b.WriteString(common.MutedStyle.Render("  No progress entries yet.\n"))
	} else {
		visibleHeight := m.height - 8
		if visibleHeight < 5 {
			visibleHeight = 5
		}
		start, end := visibleWindow(m.cursor, len(m.logs), visibleHeight)

		// Header
		b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
			"  %-12s %-30s %s\n", "Date", "Material", "Units",
		)))

		for i := start; i < end; i++ {
			b.WriteString(m.renderRow(i))
			b.WriteString("\n")
		}

		if len(m.logs) > visibleHeight {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"\n  %d–%d of %d entries\n", start+1, end, len(m.logs),
			)))
		}
	}

	b.WriteString("\n")
	keys := []string{
		common.KeyHelp("j/k", "navigate"),
		common.KeyHelp("r", "refresh"),
		common.KeyHelp("esc", "back"),
	}
	b.WriteString(common.HelpStyle.Render(strings.Join(keys, "   ")))
	return b.String()
}

func (m Model) renderRow(i int) string {
	log := m.logs[i]

	cursorStr := "  "
	nameStyle := common.MutedStyle
	if i == m.cursor {
		cursorStr = common.SelectedStyle.Render("▶ ")
		nameStyle = common.SelectedStyle
	}

	units := fmt.Sprintf("%.2g", log.Units)
	name := truncate(log.MaterialName, 28)

	return fmt.Sprintf("%s%-12s %-30s %s",
		cursorStr,
		common.MutedStyle.Render(log.Date),
		nameStyle.Render(name),
		common.StatValueStyle.Render(units),
	)
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
