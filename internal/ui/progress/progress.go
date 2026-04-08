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
	"github.com/jasonlotz/groundwork-tui/internal/ui/forms"
)

type logsLoadedMsg struct{ data []model.ProgressLog }

type deleteResultMsg struct{ toast string }

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
	overlay tea.Model
}

func New(client *api.Client) Model {
	return Model{
		client:  client,
		loading: true,
		spinner: common.NewSpinner(),
		keys: common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("D", "delete entry"),
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

func deleteEntry(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.DeleteProgressEntry(id); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return deleteResultMsg{toast: "Entry deleted"}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(load(m.client), m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ── overlay routing ──────────────────────────────────────────────────────
	if m.overlay != nil {
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
			return m, tea.Quit
		}

		updated, cmd := m.overlay.Update(msg)
		m.overlay = updated

		if done, ok := msg.(forms.ConfirmDoneMsg); ok {
			m.overlay = nil
			if done.Confirmed && done.Tag == "delete" && m.cursor < len(m.logs) {
				id := m.logs[m.cursor].ID
				return m, deleteEntry(m.client, id)
			}
			return m, cmd
		}

		return m, cmd
	}

	// ── normal update ────────────────────────────────────────────────────────
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case logsLoadedMsg:
		m.logs = msg.data
		m.loading = false
		// clamp cursor after reload
		if m.cursor >= len(m.logs) && m.cursor > 0 {
			m.cursor = len(m.logs) - 1
		}

	case deleteResultMsg:
		t := msg.toast
		return m, tea.Batch(
			func() tea.Msg { return common.ToastMsg{Text: t} },
			func() tea.Msg { return common.ProgressLoggedMsg{} },
			load(m.client),
		)

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
		case "D":
			if len(m.logs) > 0 {
				entry := m.logs[m.cursor]
				desc := fmt.Sprintf("Delete %.4g %s logged on %s for \"%s\"?",
					entry.Units,
					entry.Material.UnitType.Label(),
					entry.Date.Value,
					common.Truncate(entry.MaterialName(), 30),
				)
				f := forms.NewConfirmForm("Delete entry?", desc, "delete")
				m.overlay = f
				return m, f.Init()
			}
		}
	}
	return m, nil
}

// HasOverlay reports whether a form or confirm dialog is currently open.
func (m Model) HasOverlay() bool { return m.overlay != nil }

func (m Model) View() string {
	if m.overlay != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.overlay.View())
	}

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
		// RenderTitle=3 + blank=1 + table-header=1 + table-sep=1 + blank=1 + help=2 = 9 overhead; tab bar=3 → 12
		visibleHeight := m.height - 12
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
