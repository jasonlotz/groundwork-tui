// Package progress provides the progress log history TUI screen.
package progress

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

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
	keys    common.SimpleKeyMap
}

func New(client *api.Client) Model {
	return Model{
		client:  client,
		loading: true,
		spinner: common.NewSpinner(),
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
		return common.ErrorView(m.err, m.width)
	}

	var b strings.Builder
	b.WriteString(common.RenderTitle("Progress Log", m.width))
	b.WriteString("\n")

	if len(m.logs) == 0 {
		b.WriteString(common.MutedStyle.Render("  No progress entries yet.\n"))
	} else {
		// RenderTitle=3 + blank=1 + table-header=1 + table-sep=1 + blank=1 + help=2 = 9 overhead
		visibleHeight := m.height - 9
		if visibleHeight < 5 {
			visibleHeight = 5
		}
		start, end := common.VisibleWindow(m.cursor, len(m.logs), visibleHeight)

		// Build the visible slice of rows for the table.
		// Columns: Date | Material | Units | Notes
		rows := make([][]string, end-start)
		for i := start; i < end; i++ {
			log := m.logs[i]
			units := fmt.Sprintf("%.4g %s", log.Units, log.Material.UnitType.Label())
			notes := ""
			if log.Notes != nil {
				notes = common.Truncate(*log.Notes, 30)
			}
			rows[i-start] = []string{
				log.Date.Value,
				common.Truncate(log.MaterialName(), 28),
				units,
				notes,
			}
		}

		// StyleFunc: header row gets muted; selected data row gets highlight+bold;
		// others get default foreground.
		selectedIdx := m.cursor - start // index within visible slice
		t := table.New().
			Headers("Date", "Material", "Units", "Notes").
			Rows(rows...).
			Border(lipgloss.HiddenBorder()).
			BorderHeader(true).
			BorderStyle(common.TableBorderStyle).
			StyleFunc(func(row, col int) lipgloss.Style {
				switch {
				case row == table.HeaderRow:
					return common.TableHeaderStyle
				case row == selectedIdx:
					return common.TableSelectedStyle
				default:
					return common.TableCellStyle
				}
			})

		b.WriteString(t.Render())
		b.WriteString("\n")

		if len(m.logs) > visibleHeight {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d entries\n", start+1, end, len(m.logs),
			)))
		}
	}

	b.WriteString("\n")
	b.WriteString(common.RenderHelp(m.keys, m.width))
	return b.String()
}
