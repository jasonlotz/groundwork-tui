// Detail screen for a single skill — KPI cards and materials table.
package skills

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

type skillDetailLoadedMsg struct{ data *model.SkillDetail }

// OpenMaterialMsg is sent when the user presses enter on a material.
type OpenMaterialMsg struct{ MaterialID string }

// DetailModel is the Bubble Tea model for the skill detail screen.
type DetailModel struct {
	client  *api.Client
	skillID string
	data    *model.SkillDetail
	cursor  int
	loading bool
	err     error
	width   int
	height  int
	spinner spinner.Model
	keys    common.SimpleKeyMap
	overlay *forms.LogForm
}

func NewDetail(client *api.Client, skillID string) DetailModel {
	return DetailModel{
		client:  client,
		skillID: skillID,
		loading: true,
		spinner: common.NewSpinner(),
		keys: common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("enter", "detail"),
			common.KB("l", "log progress"),
			common.KB("r", "refresh"),
			common.KB("esc", "back"),
		}},
	}
}

func loadSkillDetail(c *api.Client, skillID string) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetSkillData(skillID)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return skillDetailLoadedMsg{data}
	}
}

func (m DetailModel) Init() tea.Cmd {
	return tea.Batch(loadSkillDetail(m.client, m.skillID), m.spinner.Tick)
}

func (m DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Route to overlay when active.
	if m.overlay != nil {
		if k, ok := msg.(tea.KeyMsg); ok && (k.String() == "ctrl+c" || k.String() == "q") {
			return m, tea.Quit
		}
		updated, cmd := m.overlay.Update(msg)
		if lf, ok := updated.(forms.LogForm); ok {
			m.overlay = &lf
		}
		if done, ok := msg.(forms.LogDoneMsg); ok {
			m.overlay = nil
			if !done.Cancelled {
				return m, tea.Batch(
					loadSkillDetail(m.client, m.skillID),
					func() tea.Msg { return common.LearningLoggedMsg{} },
					func() tea.Msg { return common.ToastMsg{Text: "Progress logged!"} },
				)
			}
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case skillDetailLoadedMsg:
		m.data = msg.data
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
			if m.data != nil && m.cursor < len(m.data.AllMaterials)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.data != nil && len(m.data.AllMaterials) > 0 {
				id := m.data.AllMaterials[m.cursor].ID
				return m, func() tea.Msg { return OpenMaterialMsg{MaterialID: id} }
			}
		case "l":
			if m.data != nil && len(m.data.AllMaterials) > 0 {
				mat := m.data.AllMaterials[m.cursor]
				if mat.Status == model.StatusActive {
					lf := forms.NewLogForm(m.client, mat.ID, mat.Name)
					m.overlay = &lf
					return m, m.overlay.Init()
				}
				return m, func() tea.Msg {
					return common.ToastMsg{Text: "Only active materials can be logged.", IsError: true}
				}
			}
		case "r":
			m.loading = true
			m.err = nil
			return m, loadSkillDetail(m.client, m.skillID)
		}
	}
	return m, nil
}

// HasOverlay reports whether a log form is currently open.
func (m DetailModel) HasOverlay() bool { return m.overlay != nil }

func (m DetailModel) View() string {
	if m.loading {
		return common.SpinnerView(m.spinner)
	}
	if m.err != nil {
		return common.ErrorView(m.err, m.width)
	}
	if m.data == nil {
		return ""
	}

	d := m.data
	var b strings.Builder

	// Title + breadcrumb
	crumb := common.DimStyle.Render(d.Skill.Category.Name + " › ")
	b.WriteString(common.RenderTitle(crumb+d.Skill.Name, m.width))
	b.WriteString("\n")

	// KPI row
	cards := []string{
		common.StatCard("Materials", fmt.Sprintf("%d active / %d done", d.ActiveMaterialCount, d.CompletedMaterialCount)),
		common.StatCard("Total", fmt.Sprintf("%d", d.TotalMaterials)),
		common.StatCard("Progress", fmt.Sprintf("%.1f%%", d.PctCompleted)),
		common.StatCard("This week", fmt.Sprintf("%.1f%%", d.PctThisWeek)),
	}
	b.WriteString(common.RenderKPICards(cards))
	b.WriteString("\n")

	// Materials list
	b.WriteString(common.SectionStyle.Render("Materials"))
	b.WriteString("\n")

	if len(d.AllMaterials) == 0 {
		b.WriteString(common.DimStyle.Render("  No materials.\n"))
	} else {
		// RenderTitle=3 + blank=1 + KPIs=3 + blank=1 + Section=2 + table-header=1 + table-sep=1 + blank=1 + help=2 = 15 overhead; tab bar=3 → 18
		visibleItems := m.height - 18
		if visibleItems < 3 {
			visibleItems = 3
		}
		start, end := common.VisibleWindow(m.cursor, len(d.AllMaterials), visibleItems)

		rows := make([][]string, end-start)
		for i := start; i < end; i++ {
			rows[i-start] = m.buildSkillDetailMaterialRow(i)
		}

		selectedIdx := m.cursor - start
		t := table.New().
			Headers("", "Material", "Status", "Progress", "Skill").
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

		if len(d.AllMaterials) > visibleItems {
			b.WriteString(common.DimStyle.Render(fmt.Sprintf(
				"  %d–%d of %d\n", start+1, end, len(d.AllMaterials),
			)))
		}
	}

	b.WriteString("\n")
	b.WriteString(common.RenderHelp(m.keys, m.width))

	if m.overlay != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.overlay.View())
	}
	return b.String()
}

func (m DetailModel) buildSkillDetailMaterialRow(i int) []string {
	mat := m.data.AllMaterials[i]
	selected := i == m.cursor

	cursor := " "
	if selected {
		cursor = common.SelectedStyle.Render("▶")
	}

	skillColor := ""
	if m.data.Skill.Color != nil {
		skillColor = *m.data.Skill.Color
	}
	dot := common.ColorDot(skillColor)

	nameStyle := common.TableCellStyle
	switch {
	case selected:
		nameStyle = common.TableSelectedStyle
	case mat.Status == model.StatusComplete:
		nameStyle = common.CompletedNameStyle
	case mat.Status == model.StatusInactive:
		nameStyle = common.InactiveNameStyle
	}
	name := dot + " " + nameStyle.Render(common.Truncate(mat.Name, 32))

	statusStyle := common.TableCellStyle
	statusStr := "inactive"
	switch mat.Status {
	case model.StatusActive:
		statusStyle = common.SuccessStyle
		statusStr = "active"
	case model.StatusComplete:
		statusStyle = common.CompletedStatusStyle
		statusStr = "done"
	}
	status := statusStyle.Render(statusStr)

	pct := 0.0
	if mat.TotalUnits > 0 {
		pct = mat.CompletedUnits / mat.TotalUnits
	}
	bar := common.RenderOverallBar(16, pct)

	skillCol := common.ColoredName(skillColor, common.Truncate(m.data.Skill.Name, 14), common.TableCellStyle)

	return []string{cursor, name, status, bar, skillCol}
}
