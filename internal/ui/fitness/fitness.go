// Package fitness provides the fitness tracker TUI screen.
// A single sessions list replaces the old sub-tab system.
// Press t to cycle the type filter: All → Lifting → Running → All.
package fitness

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

// filterMode controls which session types are shown.
type filterMode int

const (
	filterAll     filterMode = 0
	filterLifting filterMode = 1
	filterRunning filterMode = 2
)

func (f filterMode) Label() string {
	switch f {
	case filterLifting:
		return "Lifting"
	case filterRunning:
		return "Running"
	default:
		return "All"
	}
}

func (f filterMode) next() filterMode {
	switch f {
	case filterAll:
		return filterLifting
	case filterLifting:
		return filterRunning
	default:
		return filterAll
	}
}

// internal load messages
type fitnessSessionsLoadedMsg struct{ data []model.WorkoutSession }
type fitnessStatsLoadedMsg struct {
	stats *model.WorkoutStats
	goals []model.WorkoutGoal
}

// Model is the root Bubble Tea model for the fitness screen.
type Model struct {
	client          *api.Client
	sessions        []model.WorkoutSession // all sessions (unfiltered)
	filtered        []model.WorkoutSession // after applying filter
	filter          filterMode
	stats           *model.WorkoutStats
	goals           []model.WorkoutGoal
	cursor          int
	loading         bool
	err             error
	overlay         tea.Model
	pendingDeleteID string
	width           int
	height          int
	spinner         spinner.Model
	keys            common.SimpleKeyMap
}

func New(client *api.Client) Model {
	return Model{
		client:  client,
		loading: true,
		spinner: common.NewSpinner(),
		keys: common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("t", "filter type"),
			common.KB("w", "log workout"),
			common.KB("e", "edit"),
			common.KB("D", "delete"),
			common.KB("r", "refresh"),
			common.KB("esc", "back"),
		}},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadSessions(m.client), loadStats(m.client), m.spinner.Tick)
}

// HasOverlay reports whether a form overlay is currently open.
func (m Model) HasOverlay() bool { return m.overlay != nil }

func loadSessions(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		limit := 100
		data, err := c.GetWorkoutSessions(nil, &limit)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return fitnessSessionsLoadedMsg{data}
	}
}

func loadStats(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		stats, err := c.GetWorkoutStats()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		goals, err := c.GetWorkoutGoals()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return fitnessStatsLoadedMsg{stats, goals}
	}
}

// applyFilter rebuilds m.filtered from m.sessions based on m.filter.
// Resets cursor to 0 (caller must adjust if needed).
func (m *Model) applyFilter() {
	if m.filter == filterAll {
		m.filtered = m.sessions
		return
	}
	wt := model.WorkoutTypeLifting
	if m.filter == filterRunning {
		wt = model.WorkoutTypeRunning
	}
	out := make([]model.WorkoutSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		if s.Type == wt {
			out = append(out, s)
		}
	}
	m.filtered = out
}

// selectedSession returns the session at the current cursor in the filtered list.
func (m Model) selectedSession() (model.WorkoutSession, bool) {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return model.WorkoutSession{}, false
	}
	return m.filtered[m.cursor], true
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Overlay routing.
	if m.overlay != nil {
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
			return m, tea.Quit
		}
		updated, cmd := m.overlay.Update(msg)
		m.overlay = updated
		if done, ok := msg.(forms.WorkoutLogDoneMsg); ok {
			m.overlay = nil
			if !done.Cancelled {
				return m, tea.Batch(loadSessions(m.client), loadStats(m.client))
			}
		}
		if done, ok := msg.(forms.ConfirmDoneMsg); ok {
			m.overlay = nil
			if done.Confirmed && done.Tag == "delete" && m.pendingDeleteID != "" {
				id := m.pendingDeleteID
				m.pendingDeleteID = ""
				return m, deleteSession(m.client, id)
			}
			m.pendingDeleteID = ""
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case fitnessSessionsLoadedMsg:
		m.sessions = msg.data
		m.loading = false
		m.applyFilter()
		if m.cursor >= len(m.filtered) && m.cursor > 0 {
			m.cursor = len(m.filtered) - 1
		}

	case fitnessStatsLoadedMsg:
		m.stats = msg.stats
		m.goals = msg.goals

	case common.WorkoutLoggedMsg:
		return m, tea.Batch(loadSessions(m.client), loadStats(m.client))

	case common.ErrMsg:
		m.err = msg.Err
		m.loading = false

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, func() tea.Msg { return common.GoBackMsg{} }
		case "j", "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "t":
			m.filter = m.filter.next()
			m.applyFilter()
			m.cursor = 0
		case "r":
			m.loading = true
			return m, tea.Batch(loadSessions(m.client), loadStats(m.client), m.spinner.Tick)
		case "w":
			lf := forms.NewLogWorkoutForm(m.client)
			m.overlay = lf
			return m, lf.Init()
		case "e":
			if sess, ok := m.selectedSession(); ok {
				exes, _ := m.client.GetAllExercises(false)
				opts := forms.ExerciseOptions(exes)
				ef := forms.NewEditWorkoutForm(m.client, sess, opts)
				m.overlay = ef
				return m, ef.Init()
			}
		case "D":
			if sess, ok := m.selectedSession(); ok {
				m.pendingDeleteID = sess.ID
				cf := forms.NewConfirmForm("Delete session?", "Permanently delete this workout session?", "delete")
				m.overlay = cf
				return m, cf.Init()
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

	b.WriteString(common.RenderTitleWithTag("Fitness", m.filter.Label(), m.width))
	b.WriteString("\n")

	// KPI cards
	if m.stats != nil {
		b.WriteString(m.renderKPIs())
		b.WriteString("\n")
	}

	if len(m.filtered) == 0 {
		label := "No workout sessions yet."
		if m.filter != filterAll {
			label = "No " + strings.ToLower(m.filter.Label()) + " sessions yet."
		}
		b.WriteString(common.MutedStyle.Render("  " + label + "\n"))
		b.WriteString("\n")
		b.WriteString(common.RenderHelp(m.keys, m.width))
		view := b.String()
		if m.overlay != nil {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.overlay.View())
		}
		return view
	}

	// Table
	// Overhead: RenderTitleWithTag=2 + blank=1 + KPIs=3 + blank=1 = 7 lines above table
	// (if no KPIs: 2 + 1 = 3)
	overhead := 3
	if m.stats != nil {
		overhead = 7
	}
	visibleRows := m.height - overhead - 2 // -2 for table header+sep
	if visibleRows < 3 {
		visibleRows = 3
	}
	start, end := common.VisibleWindow(m.cursor, len(m.filtered), visibleRows)
	selectedIdx := m.cursor - start

	// Columns: Date | Type (when showing all) | Duration | Details | Notes
	showType := m.filter == filterAll

	rows := make([][]string, end-start)
	for i := start; i < end; i++ {
		s := m.filtered[i]
		dur := "—"
		if s.DurationMinutes != nil {
			dur = fmt.Sprintf("%d min", *s.DurationMinutes)
		}
		notes := ""
		if s.Notes != nil {
			notes = common.Truncate(*s.Notes, 25)
		}
		details := common.Truncate(s.Details, 40)
		if showType {
			rows[i-start] = []string{s.Date.Value, titleCase(string(s.Type)), dur, details, notes}
		} else {
			rows[i-start] = []string{s.Date.Value, dur, details, notes}
		}
	}

	var t *table.Table
	if showType {
		t = table.New().Headers("Date", "Type", "Duration", "Details", "Notes")
	} else {
		t = table.New().Headers("Date", "Duration", "Details", "Notes")
	}
	t = t.Rows(rows...).
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

	if len(m.filtered) > visibleRows {
		b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
			"  %d–%d of %d sessions\n", start+1, end, len(m.filtered),
		)))
	}

	b.WriteString("\n")
	b.WriteString(common.RenderHelp(m.keys, m.width))

	view := b.String()
	if m.overlay != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.overlay.View())
	}
	return view
}

func (m Model) renderKPIs() string {
	liftGoal, runGoal := 0, 0
	for _, g := range m.goals {
		if g.Type == model.WorkoutTypeLifting {
			liftGoal = g.SessionsPerWeek
		} else if g.Type == model.WorkoutTypeRunning {
			runGoal = g.SessionsPerWeek
		}
	}

	liftLabel := fmt.Sprintf("%d", m.stats.LiftingThisWeek)
	if liftGoal > 0 {
		liftLabel = fmt.Sprintf("%d / %d", m.stats.LiftingThisWeek, liftGoal)
	}
	runLabel := fmt.Sprintf("%d", m.stats.RunningThisWeek)
	if runGoal > 0 {
		runLabel = fmt.Sprintf("%d / %d", m.stats.RunningThisWeek, runGoal)
	}
	total := m.stats.LiftingThisWeek + m.stats.RunningThisWeek

	cards := []string{
		common.StatCard("Lifting/wk", liftLabel),
		common.StatCard("Running/wk", runLabel),
		common.StatCard("Total/wk", fmt.Sprintf("%d", total)),
	}
	return common.RenderKPICards(cards)
}

// deleteSession sends a delete mutation and reloads.
func deleteSession(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.DeleteWorkoutSession(id); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.WorkoutLoggedMsg{}
	}
}

// titleCase converts "LIFTING" → "Lifting", "RUNNING" → "Running".
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
