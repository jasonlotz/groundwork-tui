// Package habits provides the habit tracking TUI screen.
// Shows a list of habits with compact 30-day ASCII heatmaps and streak counts.
package habits

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
	"github.com/jasonlotz/groundwork-tui/internal/ui/forms"
)

// internal messages
type habitsLoadedMsg struct {
	habits  []model.Habit
	today   []model.ActiveHabitStatus
	heatmap map[string][]model.HeatmapEntry
}
type habitStatsLoadedMsg struct{ stats *model.HabitStats }

// Model is the root Bubble Tea model for the habits screen.
type Model struct {
	client          *api.Client
	habits          []model.Habit
	todayStatus     map[string]bool            // habitID -> loggedToday
	heatmapData     map[string][]model.HeatmapEntry // habitID -> entries
	stats           *model.HabitStats
	activeOnly      bool
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
		client:      client,
		loading:     true,
		todayStatus: make(map[string]bool),
		heatmapData: make(map[string][]model.HeatmapEntry),
		spinner:     common.NewSpinner(),
		keys: common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("enter", "toggle today"),
			common.KB("n", "new habit"),
			common.KB("e", "edit"),
			common.KB("D", "delete"),
			common.KB("a", "toggle active"),
			common.KB("r", "refresh"),
			common.KB("esc", "back"),
		}},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadHabits(m.client, m.activeOnly), loadStats(m.client), m.spinner.Tick)
}

// HasOverlay reports whether a form overlay is currently open.
func (m Model) HasOverlay() bool { return m.overlay != nil }

func loadHabits(c *api.Client, activeOnly bool) tea.Cmd {
	return func() tea.Msg {
		var status *string
		if activeOnly {
			s := "ACTIVE"
			status = &s
		}
		habits, err := c.GetAllHabits(status)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		today, err := c.GetActiveHabitsWithTodayStatus()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		heatmap := make(map[string][]model.HeatmapEntry, len(habits))
		for _, h := range habits {
			entries, err := c.GetHabitHeatmapData(h.ID)
			if err != nil {
				continue
			}
			heatmap[h.ID] = entries
		}
		return habitsLoadedMsg{habits: habits, today: today, heatmap: heatmap}
	}
}

func loadStats(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		stats, err := c.GetHabitStats()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return habitStatsLoadedMsg{stats}
	}
}

func (m Model) selectedHabit() (model.Habit, bool) {
	if len(m.habits) == 0 || m.cursor >= len(m.habits) {
		return model.Habit{}, false
	}
	return m.habits[m.cursor], true
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Overlay routing.
	if m.overlay != nil {
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
			return m, tea.Quit
		}
		updated, cmd := m.overlay.Update(msg)
		m.overlay = updated
		if done, ok := msg.(forms.HabitFormDoneMsg); ok {
			m.overlay = nil
			if !done.Cancelled {
				return m, submitHabitForm(m.client, updated)
			}
		}
		if done, ok := msg.(forms.ConfirmDoneMsg); ok {
			m.overlay = nil
			if done.Confirmed && done.Tag == "delete-habit" && m.pendingDeleteID != "" {
				id := m.pendingDeleteID
				m.pendingDeleteID = ""
				return m, deleteHabit(m.client, id)
			}
			m.pendingDeleteID = ""
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case habitsLoadedMsg:
		m.habits = msg.habits
		m.loading = false
		m.todayStatus = make(map[string]bool, len(msg.today))
		for _, s := range msg.today {
			m.todayStatus[s.ID] = s.LoggedToday
		}
		m.heatmapData = msg.heatmap
		if m.cursor >= len(m.habits) && m.cursor > 0 {
			m.cursor = len(m.habits) - 1
		}

	case habitStatsLoadedMsg:
		m.stats = msg.stats

	case common.HabitChangedMsg:
		return m, tea.Batch(loadHabits(m.client, m.activeOnly), loadStats(m.client))

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
			if m.cursor < len(m.habits)-1 {
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
			return m, tea.Batch(loadHabits(m.client, m.activeOnly), m.spinner.Tick)
		case "r":
			m.loading = true
			return m, tea.Batch(loadHabits(m.client, m.activeOnly), loadStats(m.client), m.spinner.Tick)
		case "enter":
			if h, ok := m.selectedHabit(); ok {
				return m, toggleToday(m.client, h.ID)
			}
		case "n":
			f := forms.NewHabitCreateForm()
			m.overlay = f
			return m, f.Init()
		case "e":
			if h, ok := m.selectedHabit(); ok {
				var endDate *string
				if h.EndDate != nil {
					s := h.EndDate.Value
					endDate = &s
				}
				f := forms.NewHabitEditForm(h.ID, h.Name, h.StartDate.Value, endDate)
				m.overlay = f
				return m, f.Init()
			}
		case "D":
			if h, ok := m.selectedHabit(); ok {
				m.pendingDeleteID = h.ID
				cf := forms.NewConfirmForm("Delete habit?", fmt.Sprintf("Permanently delete \"%s\" and all its logs?", h.Name), "delete-habit")
				m.overlay = cf
				return m, cf.Init()
			}
		}
	}

	return m, nil
}

func submitHabitForm(c *api.Client, overlay tea.Model) tea.Cmd {
	return func() tea.Msg {
		hf, ok := overlay.(forms.HabitForm)
		if !ok {
			return nil
		}
		if hf.IsEdit() {
			name := hf.Name()
			startDate := hf.StartDate()
			err := c.UpdateHabit(hf.EditID(), &name, &startDate, hf.EndDate())
			if err != nil {
				return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
			}
		} else {
			err := c.CreateHabit(hf.Name(), hf.StartDate(), hf.EndDate())
			if err != nil {
				return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
			}
		}
		return common.HabitChangedMsg{}
	}
}

func deleteHabit(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.DeleteHabit(id); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.HabitChangedMsg{}
	}
}

func toggleToday(c *api.Client, habitID string) tea.Cmd {
	return func() tea.Msg {
		today := time.Now().Format("2006-01-02")
		_, err := c.ToggleHabitLog(habitID, today)
		if err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.HabitChangedMsg{}
	}
}

func (m Model) View() string {
	if m.loading {
		return common.SpinnerView(m.spinner)
	}
	if m.err != nil {
		return common.ErrorView(m.err, m.width)
	}

	var b strings.Builder

	tag := ""
	if m.activeOnly {
		tag = common.DimStyle.Render("[active only]")
	}
	b.WriteString(common.RenderTitleWithTag("Habits", tag, m.width))
	b.WriteString("\n")

	// KPI cards
	if m.stats != nil {
		b.WriteString(m.renderKPIs())
		b.WriteString("\n")
	}

	if len(m.habits) == 0 {
		label := "No habits found."
		if !m.activeOnly {
			label = "No habits yet. Press n to create one."
		}
		b.WriteString(common.DimStyle.Render("  " + label + "\n"))
		b.WriteString("\n")
		b.WriteString(common.RenderHelp(m.keys, m.width))
		view := b.String()
		if m.overlay != nil {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.overlay.View())
		}
		return view
	}

	// Habit list with heatmaps
	overhead := 3
	if m.stats != nil {
		overhead = 7
	}
	visibleRows := m.height - overhead - 2
	if visibleRows < 3 {
		visibleRows = 3
	}
	start, end := common.VisibleWindow(m.cursor, len(m.habits), visibleRows)

	for i := start; i < end; i++ {
		h := m.habits[i]
		selected := i == m.cursor

		// Status indicator
		todayDone := m.todayStatus[h.ID]
		check := "○"
		if todayDone {
			check = "●"
		}

		// Name line
		nameStyle := common.DefaultNameStyle
		if selected {
			nameStyle = common.TableSelectedStyle
		}

		statusStyle := common.SuccessStyle
		statusStr := "active"
		if h.Status == "INACTIVE" {
			statusStyle = common.DimStyle
			statusStr = "inactive"
		}
		status := statusStyle.Render(statusStr)

		b.WriteString(nameStyle.Render(fmt.Sprintf("  %s %s", check, h.Name)) + "  " + status + "\n")

		// Heatmap line
		heatmap := renderHeatmap(m.heatmapData[h.ID])
		b.WriteString(common.DimStyle.Render("    ") + heatmap + "\n")
	}

	if len(m.habits) > visibleRows {
		b.WriteString(common.DimStyle.Render(fmt.Sprintf(
			"  %d–%d of %d habits\n", start+1, end, len(m.habits),
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
	cards := []string{
		common.StatCard("Active", fmt.Sprintf("%d", m.stats.ActiveCount)),
		common.StatCard("Done Today", fmt.Sprintf("%d / %d", m.stats.DoneToday, m.stats.ActiveCount)),
		common.StatCard("Streak", fmt.Sprintf("%d days", m.stats.CurrentStreak)),
	}
	return common.RenderKPICards(cards)
}

// renderHeatmap builds a compact 30-day ASCII heatmap from heatmap entries.
// Format: ··█·███████·██████████·████████  27/30
func renderHeatmap(entries []model.HeatmapEntry) string {
	// Build a set of logged dates.
	logged := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.Count > 0 {
			logged[e.Date] = true
		}
	}

	// Render last 30 days.
	now := time.Now()
	var buf strings.Builder
	doneCount := 0
	for i := 29; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		if logged[d] {
			buf.WriteString("█")
			doneCount++
		} else {
			buf.WriteString("·")
		}
	}

	heatStyle := lipgloss.NewStyle().Foreground(common.ColorHighlight)
	dimStyle := lipgloss.NewStyle().Foreground(common.ColorDim)

	// Color the filled blocks with highlight, dots with dim.
	var colored strings.Builder
	raw := buf.String()
	for _, r := range raw {
		if r == '█' {
			colored.WriteString(heatStyle.Render("█"))
		} else {
			colored.WriteString(dimStyle.Render("·"))
		}
	}

	ratio := dimStyle.Render(fmt.Sprintf("  %d/30", doneCount))
	return colored.String() + ratio
}
