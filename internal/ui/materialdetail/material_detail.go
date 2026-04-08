// Package materialdetail provides the material detail TUI screen.
package materialdetail

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

type dataLoadedMsg struct{ data *model.MaterialDetail }

// LogFromDetailMsg is sent when the user presses l to log progress.
type LogFromDetailMsg struct{ MaterialID string }

// Model is the Bubble Tea model for the material detail screen.
type Model struct {
	client     *api.Client
	materialID string
	data       *model.MaterialDetail
	logCursor  int
	loading    bool
	err        error
	width      int
	height     int
}

func New(client *api.Client, materialID string) Model {
	return Model{client: client, materialID: materialID, loading: true}
}

func load(c *api.Client, materialID string) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetMaterialDetail(materialID)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return dataLoadedMsg{data}
	}
}

func (m Model) Init() tea.Cmd {
	return load(m.client, m.materialID)
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

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return common.GoBackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.data != nil && m.logCursor < len(m.data.ProgressLogs)-1 {
				m.logCursor++
			}
		case "k", "up":
			if m.logCursor > 0 {
				m.logCursor--
			}
		case "l":
			if m.data != nil && m.data.Material.Status == model.StatusActive {
				id := m.materialID
				return m, func() tea.Msg { return LogFromDetailMsg{MaterialID: id} }
			}
			if m.data != nil && m.data.Material.Status != model.StatusActive {
				return m, func() tea.Msg {
					return common.ToastMsg{Text: "Only active materials can be logged.", IsError: true}
				}
			}
		case "r":
			m.loading = true
			m.err = nil
			return m, load(m.client, m.materialID)
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return common.LoadingView()
	}
	if m.err != nil {
		return common.ErrorView(m.err)
	}
	if m.data == nil {
		return ""
	}

	mat := m.data.Material
	var b strings.Builder

	// Title + breadcrumb
	crumb := common.MutedStyle.Render(mat.Skill.Category.Name + " › " + mat.Skill.Name + " › ")
	b.WriteString(common.TitleStyle.Render(crumb + mat.Name))
	b.WriteString("\n")

	// KPI row
	statusStr := strings.ToLower(string(mat.Status))
	statusStyle := common.MutedStyle
	switch mat.Status {
	case model.StatusActive:
		statusStyle = common.SuccessStyle
	case model.StatusComplete:
		statusStyle = lipgloss.NewStyle().Foreground(common.ColorPrimary)
	}

	// KPI row
	cards := []string{
		common.StatCard("Status", statusStyle.Render(statusStr)),
		common.StatCard("Progress", fmt.Sprintf("%.4g / %.4g %s (%.0f%%)", mat.CompletedUnits, mat.TotalUnits, mat.UnitType.Label(), mat.PctComplete)),
		common.StatCard("This week", fmt.Sprintf("%.4g %s", mat.UnitsThisWeek, mat.UnitType.Label())),
		common.StatCard("Streak", fmt.Sprintf("%d days", mat.MaterialStreak)),
	}
	b.WriteString(common.RenderKPICards(cards))
	b.WriteString("\n")

	// Overall progress bar
	pct := 0.0
	if mat.TotalUnits > 0 {
		pct = mat.CompletedUnits / mat.TotalUnits
	}
	b.WriteString("  " + common.ProgressBar(pct, 40) + "\n")

	// Meta info
	metaLines := []string{}
	metaLines = append(metaLines, fmt.Sprintf("  Type: %s", common.MutedStyle.Render(mat.MaterialType.Name)))
	if mat.WeeklyUnitGoal != nil && *mat.WeeklyUnitGoal > 0 {
		metaLines = append(metaLines, fmt.Sprintf("  Goal: %s", common.MutedStyle.Render(fmt.Sprintf("%d %s/week", *mat.WeeklyUnitGoal, mat.UnitType.Label()))))
	}
	if mat.ProjectedEndDate != nil {
		metaLines = append(metaLines, fmt.Sprintf("  Est. completion: %s", common.MutedStyle.Render(*mat.ProjectedEndDate)))
	}
	if mat.StartDate != nil && mat.StartDate.Value != "" {
		metaLines = append(metaLines, fmt.Sprintf("  Started: %s", common.MutedStyle.Render(mat.StartDate.Value)))
	}
	if mat.CompletedDate != nil && mat.CompletedDate.Value != "" {
		metaLines = append(metaLines, fmt.Sprintf("  Completed: %s", common.MutedStyle.Render(mat.CompletedDate.Value)))
	}
	if mat.URL != nil && *mat.URL != "" {
		metaLines = append(metaLines, fmt.Sprintf("  URL: %s", common.MutedStyle.Render(common.Truncate(*mat.URL, 50))))
	}
	for _, line := range metaLines {
		b.WriteString(line + "\n")
	}

	// Progress log
	if len(m.data.ProgressLogs) > 0 {
		b.WriteString(common.SectionStyle.Render("Progress Log"))
		b.WriteString("\n")

		// title(2)+kpis(3)+bar(1)+meta(~4)+section(2)+help(2) = ~14
		usedLines := 14 + len(metaLines)
		visibleHeight := m.height - usedLines
		if visibleHeight < 3 {
			visibleHeight = 3
		}
		start, end := common.VisibleWindow(m.logCursor, len(m.data.ProgressLogs), visibleHeight)

		for i := start; i < end; i++ {
			b.WriteString(m.renderLogRow(i))
			b.WriteString("\n")
		}
		if len(m.data.ProgressLogs) > visibleHeight {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d entries\n", start+1, end, len(m.data.ProgressLogs),
			)))
		}
	}

	b.WriteString("\n")
	keys := []string{
		common.KeyHelp("j/k", "scroll log"),
	}
	if mat.Status == model.StatusActive {
		keys = append(keys, common.KeyHelp("l", "log progress"))
	}
	keys = append(keys,
		common.KeyHelp("r", "refresh"),
		common.KeyHelp("esc", "back"),
	)
	b.WriteString(common.HelpStyle.Render(strings.Join(keys, "   ")))
	return b.String()
}

func (m Model) renderLogRow(i int) string {
	log := m.data.ProgressLogs[i]
	cursorStr := "  "
	if i == m.logCursor {
		cursorStr = common.SelectedStyle.Render("▶ ")
	}

	mat := m.data.Material
	units := fmt.Sprintf("%.4g %s", log.Units, mat.UnitType.Label())
	notesStr := ""
	if log.Notes != nil && *log.Notes != "" {
		notesStr = "  " + common.MutedStyle.Render(common.Truncate(*log.Notes, 30))
	}

	dateCol := common.MutedStyle.Copy().Width(12).Render(log.Date.Value)
	unitsCol := common.StatValueStyle.Render(fmt.Sprintf("%-12s", units))
	return cursorStr + dateCol + " " + unitsCol + notesStr
}
