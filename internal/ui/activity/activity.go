// Package activity provides the unified activity log history TUI screen,
// showing both learning log entries and workout sessions merged and sorted by date.
package activity

import (
	"fmt"
	"sort"
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

// entryKind distinguishes the two kinds of activity row.
type entryKind int

const (
	kindLearning entryKind = iota
	kindLifting
	kindRunning
)

// activityEntry is a unified row for the merged activity list.
type activityEntry struct {
	kind   entryKind
	id     string
	date   string // "YYYY-MM-DD"
	detail string
	notes  string
	// for delete routing
	isWorkout bool
}

// typeFilter controls which rows are shown.
type typeFilter int

const (
	filterAll      typeFilter = iota // 0
	filterLearning                   // 1
	filterLifting                    // 2
	filterRunning                    // 3
)

var filterLabels = []string{"All", "Learning", "Lifting", "Running"}

// --- internal messages ---

type logsLoadedMsg struct{ data []model.ProgressLog }
type sessionsLoadedMsg struct{ data []model.WorkoutSession }

// Model is the Bubble Tea model for the unified activity log screen.
type Model struct {
	client   *api.Client
	logs     []model.ProgressLog
	sessions []model.WorkoutSession
	entries  []activityEntry // merged + filtered
	cursor   int
	filter   typeFilter
	loading  bool
	logsOK   bool
	sessOK   bool
	err      error
	width    int
	height   int
	spinner  spinner.Model
	keys     common.SimpleKeyMap
	overlay  tea.Model
}

func New(client *api.Client) Model {
	return Model{
		client:  client,
		loading: true,
		spinner: common.NewSpinner(),
		keys:    buildKeys(),
	}
}

func buildKeys() common.SimpleKeyMap {
	return common.SimpleKeyMap{Bindings: []common.Binding{
		common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
		common.KBKeys("1-4", "filter", "1", "2", "3", "4"),
		common.KB("D", "delete"),
		common.KB("r", "refresh"),
		common.KB("esc", "back"),
	}}
}

// HasOverlay reports whether a confirm dialog is open.
func (m Model) HasOverlay() bool { return m.overlay != nil }

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadLogs(m.client), loadSessions(m.client), m.spinner.Tick)
}

func loadLogs(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetAllProgress(nil)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return logsLoadedMsg{data}
	}
}

func loadSessions(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetWorkoutSessions(nil, nil)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return sessionsLoadedMsg{data}
	}
}

func deleteLog(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.DeleteProgressEntry(id); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.ToastMsg{Text: "Entry deleted"}
	}
}

func deleteSession(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.DeleteWorkoutSession(id); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.ToastMsg{Text: "Session deleted"}
	}
}

// rebuild merges logs + sessions into a sorted, filtered slice.
func (m *Model) rebuild() {
	all := make([]activityEntry, 0, len(m.logs)+len(m.sessions))

	for _, l := range m.logs {
		units := fmt.Sprintf("%.4g %s", l.Units, l.Material.UnitType.Label())
		notes := ""
		if l.Notes != nil {
			notes = *l.Notes
		}
		all = append(all, activityEntry{
			kind:      kindLearning,
			id:        l.ID,
			date:      l.Date.Value,
			detail:    common.Truncate(l.MaterialName(), 30) + " — " + units,
			notes:     notes,
			isWorkout: false,
		})
	}

	for _, s := range m.sessions {
		var kind entryKind
		var detail string
		if s.Type == model.WorkoutTypeRunning {
			kind = kindRunning
			if s.DurationMinutes != nil {
				detail = fmt.Sprintf("Run — %d min", *s.DurationMinutes)
			} else {
				detail = "Run"
			}
		} else {
			kind = kindLifting
			if s.DurationMinutes != nil {
				detail = fmt.Sprintf("Lift — %d min", *s.DurationMinutes)
			} else {
				detail = "Lift"
			}
		}
		notes := ""
		if s.Notes != nil {
			notes = *s.Notes
		}
		all = append(all, activityEntry{
			kind:      kind,
			id:        s.ID,
			date:      s.Date.Value,
			detail:    detail,
			notes:     notes,
			isWorkout: true,
		})
	}

	// Sort by date descending, then by id descending for stable ordering same-day.
	sort.Slice(all, func(i, j int) bool {
		if all[i].date != all[j].date {
			return all[i].date > all[j].date
		}
		return all[i].id > all[j].id
	})

	// Apply filter.
	switch m.filter {
	case filterLearning:
		filtered := all[:0]
		for _, e := range all {
			if e.kind == kindLearning {
				filtered = append(filtered, e)
			}
		}
		m.entries = filtered
	case filterLifting:
		filtered := all[:0]
		for _, e := range all {
			if e.kind == kindLifting {
				filtered = append(filtered, e)
			}
		}
		m.entries = filtered
	case filterRunning:
		filtered := all[:0]
		for _, e := range all {
			if e.kind == kindRunning {
				filtered = append(filtered, e)
			}
		}
		m.entries = filtered
	default:
		m.entries = all
	}

	// Clamp cursor.
	if m.cursor >= len(m.entries) && m.cursor > 0 {
		m.cursor = len(m.entries) - 1
	}
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
			if done.Confirmed && done.Tag == "delete" && m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
				if entry.isWorkout {
					return m, tea.Batch(
						deleteSession(m.client, entry.id),
						loadSessions(m.client),
					)
				}
				return m, tea.Batch(
					deleteLog(m.client, entry.id),
					loadLogs(m.client),
				)
			}
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
		m.logsOK = true
		m.loading = !(m.logsOK && m.sessOK)
		m.rebuild()

	case sessionsLoadedMsg:
		m.sessions = msg.data
		m.sessOK = true
		m.loading = !(m.logsOK && m.sessOK)
		m.rebuild()

	case common.LearningLoggedMsg:
		m.logsOK = false
		return m, loadLogs(m.client)

	case common.WorkoutLoggedMsg:
		m.sessOK = false
		return m, loadSessions(m.client)

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
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "1":
			m.filter = filterAll
			m.cursor = 0
			m.rebuild()
		case "2":
			m.filter = filterLearning
			m.cursor = 0
			m.rebuild()
		case "3":
			m.filter = filterLifting
			m.cursor = 0
			m.rebuild()
		case "4":
			m.filter = filterRunning
			m.cursor = 0
			m.rebuild()
		case "r":
			m.loading = true
			m.logsOK = false
			m.sessOK = false
			m.err = nil
			return m, tea.Batch(loadLogs(m.client), loadSessions(m.client))
		case "D":
			if len(m.entries) > 0 && m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
				var desc string
				if entry.isWorkout {
					desc = fmt.Sprintf("Delete %s on %s?", entry.detail, entry.date)
				} else {
					desc = fmt.Sprintf("Delete %s on %s?",
						common.Truncate(entry.detail, 40), entry.date)
				}
				f := forms.NewConfirmForm("Delete entry?", desc, "delete")
				m.overlay = f
				return m, f.Init()
			}
		}
	}
	return m, nil
}

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
	b.WriteString(common.RenderTitle("Activity Log", m.width))
	b.WriteString("\n")

	// Filter bar: [1 All] [2 Learning] [3 Lifting] [4 Running]
	b.WriteString(renderFilterBar(m.filter, m.width))
	b.WriteString("\n")

	if len(m.entries) == 0 {
		msg := "  No activity entries yet."
		if m.filter != filterAll {
			msg = "  No " + strings.ToLower(filterLabels[m.filter]) + " entries."
		}
		b.WriteString(common.MutedStyle.Render(msg + "\n"))
	} else {
		// RenderTitle=3 + blank=1 + filterBar=1 + blank=1 + table-header=1 + table-sep=1 + blank=1 + help=2 = 11 overhead; tab bar=3 → 14
		visibleHeight := m.height - 14
		if visibleHeight < 5 {
			visibleHeight = 5
		}
		start, end := common.VisibleWindow(m.cursor, len(m.entries), visibleHeight)
		selectedIdx := m.cursor - start

		rows := make([][]string, end-start)
		for i := start; i < end; i++ {
			e := m.entries[i]
			rows[i-start] = []string{
				e.date,
				kindLabel(e.kind),
				common.Truncate(e.detail, 40),
				common.Truncate(e.notes, 25),
			}
		}

		t := table.New().
			Headers("Date", "Type", "Detail", "Notes").
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
					// Dim workout rows slightly vs learning rows using muted for type col.
					return common.DefaultNameStyle
				}
			})

		b.WriteString(t.Render())
		b.WriteString("\n")

		if len(m.entries) > visibleHeight {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d entries\n", start+1, end, len(m.entries),
			)))
		}
	}

	b.WriteString("\n")
	b.WriteString(common.RenderHelp(m.keys, m.width))
	return b.String()
}

// kindLabel returns a short type label for a row.
func kindLabel(k entryKind) string {
	switch k {
	case kindLearning:
		return "Learning"
	case kindLifting:
		return "Lifting"
	case kindRunning:
		return "Running"
	}
	return ""
}

// renderFilterBar draws the filter strip.
func renderFilterBar(active typeFilter, width int) string {
	labels := []string{"1:All", "2:Learning", "3:Lifting", "4:Running"}
	var parts []string
	for i, label := range labels {
		if typeFilter(i) == active {
			parts = append(parts, common.SelectedStyle.Render("["+label+"]"))
		} else {
			parts = append(parts, common.MutedStyle.Render("["+label+"]"))
		}
	}
	_ = width
	return "  " + strings.Join(parts, "  ")
}
