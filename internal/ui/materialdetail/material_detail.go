// Package materialdetail provides the material detail TUI screen.
package materialdetail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	bbprogress "github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/muesli/termenv"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
	"github.com/jasonlotz/groundwork-tui/internal/ui/progress"
)

type dataLoadedMsg struct{ data *model.MaterialDetail }

// preloadMsg carries skills + types fetched before opening the material form.
type preloadMsg struct {
	skills []model.Skill
	types  []model.MaterialType
}

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
	spinner    spinner.Model
	bar        bbprogress.Model
	help       help.Model
	keys       common.SimpleKeyMap
	overlay    tea.Model
}

func New(client *api.Client, materialID string) Model {
	return Model{
		client:     client,
		materialID: materialID,
		loading:    true,
		spinner:    common.NewSpinner(),
		bar:        common.NewProgressBar(40),
		help:       common.NewHelp(),
		keys: common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "scroll log", "j", "k", "down", "up"),
			common.KB("l", "log progress"),
			common.KB("e", "edit"),
			common.KB("D", "delete"),
			common.KB("r", "refresh"),
			common.KB("esc", "back"),
		}},
	}
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

// preload fetches skills and types needed to populate the material edit form.
func preload(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		skills, err := c.GetAllSkills()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		types, err := c.GetAllMaterialTypes()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return preloadMsg{skills: skills, types: types}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(load(m.client, m.materialID), m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ── overlay routing ──────────────────────────────────────────────────────
	if m.overlay != nil {
		if k, ok := msg.(tea.KeyMsg); ok && (k.String() == "ctrl+c" || k.String() == "q") {
			return m, tea.Quit
		}

		updated, cmd := m.overlay.Update(msg)
		m.overlay = updated

		switch msg := msg.(type) {
		case progress.LogDoneMsg:
			m.overlay = nil
			if !msg.Cancelled {
				return m, tea.Batch(
					load(m.client, m.materialID),
					func() tea.Msg { return common.ToastMsg{Text: "Progress logged!"} },
				)
			}
			return m, nil

		case common.MaterialFormDoneMsg:
			m.overlay = nil
			if !msg.Cancelled {
				if mf, ok := updated.(common.MaterialForm); ok {
					return m, submitMaterialForm(m.client, m.materialID, mf)
				}
			}
			return m, nil

		case common.ConfirmDoneMsg:
			m.overlay = nil
			if msg.Confirmed && msg.Tag == "delete" {
				return m, deleteMaterial(m.client, m.materialID)
			}
			return m, nil

		case common.ToastMsg:
			return m, func() tea.Msg { return msg }
		}

		return m, cmd
	}

	// ── normal update ────────────────────────────────────────────────────────
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

	case dataLoadedMsg:
		m.data = msg.data
		m.loading = false

	case preloadMsg:
		// Preload completed — open the edit form overlay pre-populated from current data.
		if m.data == nil {
			return m, nil
		}
		info := m.data.Material
		// Convert MaterialDetailInfo → model.Material for the form constructor.
		mat := model.Material{
			ID:             info.ID,
			Name:           info.Name,
			UnitType:       info.UnitType,
			TotalUnits:     info.TotalUnits,
			CompletedUnits: info.CompletedUnits,
			Status:         info.Status,
			SkillID:        info.Skill.ID,
			WeeklyUnitGoal: info.WeeklyUnitGoal,
			URL:            info.URL,
			StartDate:      info.StartDate,
			CompletedDate:  info.CompletedDate,
			Skill:          info.Skill,
			MaterialType:   info.MaterialType,
		}
		f := common.NewMaterialEditForm(mat.ID, mat, msg.skills, msg.types)
		m.overlay = f
		return m, f.Init()

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
			if m.data != nil && m.logCursor < len(m.data.ProgressLogs)-1 {
				m.logCursor++
			}
		case "k", "up":
			if m.logCursor > 0 {
				m.logCursor--
			}
		case "l":
			if m.data == nil {
				break
			}
			if m.data.Material.Status == model.StatusActive {
				lf := progress.NewLogForm(m.client, m.materialID, m.data.Material.Name)
				m.overlay = lf
				return m, m.overlay.Init()
			}
			return m, func() tea.Msg {
				return common.ToastMsg{Text: "Only active materials can be logged.", IsError: true}
			}
		case "e":
			if m.data != nil {
				return m, preload(m.client)
			}
		case "D":
			if m.data != nil {
				mat := m.data.Material
				f := common.NewConfirmForm(
					"Delete material?",
					fmt.Sprintf("Permanently delete \"%s\" and all its progress logs?", common.Truncate(mat.Name, 40)),
					"delete",
				)
				m.overlay = f
				return m, f.Init()
			}
		case "r":
			m.loading = true
			m.err = nil
			return m, load(m.client, m.materialID)
		}
	}
	return m, nil
}

// submitMaterialForm runs the update API call after the edit form completes.
func submitMaterialForm(c *api.Client, materialID string, mf common.MaterialForm) tea.Cmd {
	return func() tea.Msg {
		r := mf.Result()
		err := c.UpdateMaterial(api.MaterialUpdateResult{
			ID:            materialID,
			Name:          r.Name,
			SkillID:       r.SkillID,
			TypeID:        r.TypeID,
			UnitType:      r.UnitType,
			TotalUnits:    r.TotalUnits,
			URL:           r.URL,
			Notes:         r.Notes,
			StartDate:     r.StartDate,
			CompletedDate: r.CompletedDate,
			WeeklyGoal:    r.WeeklyGoal,
		})
		if err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		data, loadErr := c.GetMaterialDetail(materialID)
		if loadErr != nil {
			return common.ToastMsg{Text: "Material updated (refresh to see changes)"}
		}
		return dataLoadedMsg{data: data}
	}
}

// deleteMaterial deletes the material and navigates back.
func deleteMaterial(c *api.Client, materialID string) tea.Cmd {
	return func() tea.Msg {
		if err := c.DeleteMaterial(materialID); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		// Navigate back since the material no longer exists.
		return common.GoBackMsg{}
	}
}

func (m Model) View() string {
	if m.overlay != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.overlay.View())
	}

	if m.loading {
		return common.SpinnerView(m.spinner)
	}
	if m.err != nil {
		return common.ErrorView(m.err)
	}
	if m.data == nil {
		return ""
	}

	mat := m.data.Material
	var b strings.Builder

	// Title + breadcrumb with colored dots
	catDot := common.ColorDot(func() string {
		if mat.Skill.Category.Color != nil {
			return *mat.Skill.Category.Color
		}
		return ""
	}())
	skillDot := common.ColorDot(func() string {
		if mat.Skill.Color != nil {
			return *mat.Skill.Color
		}
		return ""
	}())
	crumb := catDot + " " + common.MutedStyle.Render(mat.Skill.Category.Name+" › ") +
		skillDot + " " + common.MutedStyle.Render(mat.Skill.Name+" › ")
	b.WriteString(common.RenderTitle(crumb+mat.Name, m.width))
	b.WriteString("\n")

	// KPI row
	statusStr := strings.ToLower(string(mat.Status))
	statusStyle := common.MutedStyle
	switch mat.Status {
	case model.StatusActive:
		statusStyle = common.SuccessStyle
	case model.StatusComplete:
		statusStyle = common.CompletedStatusStyle
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

	// Bar width: terminal width minus indent and labels.
	barWidth := common.ClampBarWidth(m.width)

	// Weekly goal bar (only when a goal is set).
	if mat.WeeklyUnitGoal != nil && *mat.WeeklyUnitGoal > 0 {
		pacePct := common.PaceFraction()
		weeklyPct := mat.UnitsThisWeek / float64(*mat.WeeklyUnitGoal)
		weeklyBar := common.RenderWeeklyBar(barWidth, weeklyPct, pacePct)
		weeklyLabel := fmt.Sprintf("%.4g / %d %s this week",
			mat.UnitsThisWeek, *mat.WeeklyUnitGoal, mat.UnitType.Label())
		if mat.ProjectedEndDate != nil {
			weeklyLabel += common.MutedStyle.Render("  · est. " + common.FormatProjectedDate(*mat.ProjectedEndDate))
		}
		b.WriteString("  " + weeklyBar + "  " + common.MutedStyle.Render(weeklyLabel) + "\n")
	}

	// Overall progress bar.
	pct := 0.0
	if mat.TotalUnits > 0 {
		pct = mat.CompletedUnits / mat.TotalUnits
	}
	overallLabel := fmt.Sprintf("%.4g / %.4g %s overall",
		mat.CompletedUnits, mat.TotalUnits, mat.UnitType.Label())
	b.WriteString("  " + common.RenderBar(m.bar, pct, barWidth) + "  " + common.MutedStyle.Render(overallLabel) + "\n")

	// Meta info
	metaLines := []string{}
	metaLines = append(metaLines, fmt.Sprintf("  Type: %s", common.MutedStyle.Render(mat.MaterialType.Name)))
	if mat.WeeklyUnitGoal != nil && *mat.WeeklyUnitGoal > 0 {
		metaLines = append(metaLines, fmt.Sprintf("  Goal: %s", common.MutedStyle.Render(fmt.Sprintf("%d %s/week", *mat.WeeklyUnitGoal, mat.UnitType.Label()))))
	}
	if mat.ProjectedEndDate != nil {
		metaLines = append(metaLines, fmt.Sprintf("  Est. completion: %s", common.MutedStyle.Render(common.FormatProjectedDate(*mat.ProjectedEndDate))))
	}
	if mat.StartDate != nil && mat.StartDate.Value != "" {
		metaLines = append(metaLines, fmt.Sprintf("  Started: %s", common.MutedStyle.Render(mat.StartDate.Value)))
	}
	if mat.CompletedDate != nil && mat.CompletedDate.Value != "" {
		metaLines = append(metaLines, fmt.Sprintf("  Completed: %s", common.MutedStyle.Render(mat.CompletedDate.Value)))
	}
	if mat.URL != nil && *mat.URL != "" {
		label := common.Truncate(*mat.URL, 50)
		link := termenv.Hyperlink(*mat.URL, label)
		metaLines = append(metaLines, fmt.Sprintf("  URL: %s", common.MutedStyle.Render(link)))
	}
	for _, line := range metaLines {
		b.WriteString(line + "\n")
	}

	// Progress log
	if len(m.data.ProgressLogs) > 0 {
		b.WriteString(common.SectionStyle.Render("Progress Log"))
		b.WriteString("\n")

		// title(2)+kpis(3)+bars(~2)+meta(~4)+section(2)+table header(1)+separator(1)+help(2) = ~16
		usedLines := 16 + len(metaLines)
		visibleHeight := m.height - usedLines
		if visibleHeight < 3 {
			visibleHeight = 3
		}
		start, end := common.VisibleWindow(m.logCursor, len(m.data.ProgressLogs), visibleHeight)

		rows := make([][]string, end-start)
		for i := start; i < end; i++ {
			log := m.data.ProgressLogs[i]
			cursor := " "
			if i == m.logCursor {
				cursor = common.SelectedStyle.Render("▶")
			}
			units := fmt.Sprintf("%.4g %s", log.Units, mat.UnitType.Label())
			notes := ""
			if log.Notes != nil && *log.Notes != "" {
				notes = common.Truncate(*log.Notes, 30)
			}
			rows[i-start] = []string{cursor, log.Date.Value, units, notes}
		}

		selectedIdx := m.logCursor - start
		t := table.New().
			Headers("", "Date", "Units", "Notes").
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

		if len(m.data.ProgressLogs) > visibleHeight {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d entries\n", start+1, end, len(m.data.ProgressLogs),
			)))
		}
	}

	b.WriteString("\n")
	b.WriteString(common.HelpStyle.Render(m.help.View(m.keys)))
	return b.String()
}
