// Package progress provides the progress log history TUI screen.
package progress

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

type logsLoadedMsg struct{ data []model.ProgressLog }

// Model is the Bubble Tea model for the progress history screen.
type Model struct {
	client  *api.Client
	logs    []model.ProgressLog
	cursor  int
	loading bool
	err     error
	width   int
	height  int
	spinner spinner.Model
	help    help.Model
	keys    common.SimpleKeyMap
}

func New(client *api.Client) Model {
	return Model{
		client:  client,
		loading: true,
		spinner: common.NewSpinner(),
		help:    common.NewHelp(),
		keys: common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("r", "refresh"),
			common.KB("esc", "back"),
		}},
	}
}

func load(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetAllProgress(nil)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return logsLoadedMsg{data}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(load(m.client), m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

	case logsLoadedMsg:
		m.logs = msg.data
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
		return common.SpinnerView(m.spinner)
	}
	if m.err != nil {
		return common.ErrorView(m.err)
	}

	var b strings.Builder
	b.WriteString(common.RenderTitle("Progress Log", m.width))
	b.WriteString("\n")

	if len(m.logs) == 0 {
		b.WriteString(common.MutedStyle.Render("  No progress entries yet.\n"))
	} else {
		// title(1) + marginBottom(1) + header(1) + blank(1) + help(1) + marginTop(1) = 6
		visibleHeight := m.height - 6
		if visibleHeight < 5 {
			visibleHeight = 5
		}
		start, end := common.VisibleWindow(m.cursor, len(m.logs), visibleHeight)

		// Header
		dateHdr := common.MutedStyle.Copy().Width(12).Render("Date")
		nameHdr := common.MutedStyle.Copy().Width(30).Render("Material")
		unitsHdr := common.MutedStyle.Render("Units")
		b.WriteString("  " + dateHdr + " " + nameHdr + " " + unitsHdr + "\n")

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
	b.WriteString(common.HelpStyle.Render(m.help.View(m.keys)))
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

	units := fmt.Sprintf("%.4g %s", log.Units, log.Material.UnitType.Label())
	name := common.Truncate(log.MaterialName(), 28)

	// Use lipgloss Width() for columns so ANSI codes don't break alignment.
	dateCol := common.MutedStyle.Copy().Width(12).Render(log.Date.Value)
	nameCol := nameStyle.Copy().Width(30).Render(name)
	unitsCol := common.StatValueStyle.Render(units)

	return cursorStr + dateCol + " " + nameCol + " " + unitsCol
}
