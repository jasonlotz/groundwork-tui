// Package categorydetail provides the category detail TUI screen.
package categorydetail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

type dataLoadedMsg struct{ data *model.CategoryDetail }

// skillMutatedMsg is an internal result from submitSkillForm / submitSkillConfirm.
type skillMutatedMsg struct{ toast string }

// OpenSkillMsg is sent when the user presses enter on a skill.
type OpenSkillMsg struct{ SkillID string }

// Model is the Bubble Tea model for the category detail screen.
type Model struct {
	client     *api.Client
	categoryID string
	data       *model.CategoryDetail
	cursor     int
	loading    bool
	err        error
	width      int
	height     int
	spinner    spinner.Model
	barWide    progress.Model // width 16 — active materials list
	barNarrow  progress.Model // width 12 — skill rows
	keys       common.SimpleKeyMap
	overlay    tea.Model
}

func New(client *api.Client, categoryID string) Model {
	return Model{
		client:     client,
		categoryID: categoryID,
		loading:    true,
		spinner:    common.NewSpinner(),
		barWide:    common.NewProgressBar(16),
		barNarrow:  common.NewProgressBar(12),
		keys:       buildKeys(false),
	}
}

func buildKeys(selectedIsArchived bool) common.SimpleKeyMap {
	bindings := []common.Binding{
		common.KBKeys("j/k", "navigate skills", "j", "k", "down", "up"),
		common.KB("enter", "open skill"),
		common.KB("n", "new skill"),
		common.KB("e", "edit skill"),
	}
	if selectedIsArchived {
		bindings = append(bindings, common.KB("A", "unarchive skill"))
	} else {
		bindings = append(bindings, common.KB("A", "archive skill"))
	}
	bindings = append(bindings,
		common.KB("D", "delete (archived)"),
		common.KB("r", "refresh"),
		common.KB("esc", "back"),
	)
	return common.SimpleKeyMap{Bindings: bindings}
}

func load(c *api.Client, categoryID string) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetCategoryData(categoryID)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return dataLoadedMsg{data}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(load(m.client, m.categoryID), m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ── overlay routing ──────────────────────────────────────────────────────
	if m.overlay != nil {
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
			return m, tea.Quit
		}

		updated, cmd := m.overlay.Update(msg)
		m.overlay = updated

		switch msg := msg.(type) {
		case common.SkillFormDoneMsg:
			m.overlay = nil
			if !msg.Cancelled {
				if sf, ok := updated.(common.SkillForm); ok {
					return m, submitSkillForm(m.client, m.categoryID, sf)
				}
			}
			return m, cmd

		case common.ConfirmDoneMsg:
			m.overlay = nil
			if msg.Confirmed {
				return m, submitSkillConfirm(m.client, m.data, m.cursor, msg.Tag)
			}
			return m, cmd

		case skillMutatedMsg:
			var cmds []tea.Cmd
			if msg.toast != "" {
				t := msg.toast
				cmds = append(cmds, func() tea.Msg { return common.ToastMsg{Text: t} })
			}
			cmds = append(cmds, func() tea.Msg { return common.SkillChangedMsg{} })
			cmds = append(cmds, load(m.client, m.categoryID))
			return m, tea.Batch(cmds...)

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

	case dataLoadedMsg:
		m.data = msg.data
		m.loading = false
		if m.cursor >= skillCount(m.data) && m.cursor > 0 {
			m.cursor = skillCount(m.data) - 1
		}
		m.keys = buildKeys(m.selectedIsArchived())

	case skillMutatedMsg:
		var cmds []tea.Cmd
		if msg.toast != "" {
			t := msg.toast
			cmds = append(cmds, func() tea.Msg { return common.ToastMsg{Text: t} })
		}
		cmds = append(cmds, func() tea.Msg { return common.SkillChangedMsg{} })
		cmds = append(cmds, load(m.client, m.categoryID))
		return m, tea.Batch(cmds...)

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
			if m.data != nil && m.cursor < len(m.data.SkillsSummary)-1 {
				m.cursor++
				m.keys = buildKeys(m.selectedIsArchived())
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.keys = buildKeys(m.selectedIsArchived())
			}
		case "enter":
			if m.data != nil && len(m.data.SkillsSummary) > 0 {
				id := m.data.SkillsSummary[m.cursor].ID
				return m, func() tea.Msg { return OpenSkillMsg{SkillID: id} }
			}
		case "r":
			m.loading = true
			m.err = nil
			return m, load(m.client, m.categoryID)
		case "n":
			f := common.NewSkillCreateForm(m.categoryID)
			m.overlay = f
			return m, f.Init()
		case "e":
			if m.data != nil && len(m.data.SkillsSummary) > 0 {
				s := m.data.SkillsSummary[m.cursor]
				f := common.NewSkillEditForm(s.ID, s.Name, m.categoryID, s.Color)
				m.overlay = f
				return m, f.Init()
			}
		case "A":
			if m.data != nil && len(m.data.SkillsSummary) > 0 {
				s := m.data.SkillsSummary[m.cursor]
				var title, desc, tag string
				if s.IsArchived {
					title = "Unarchive skill?"
					desc = fmt.Sprintf("Unarchive \"%s\"?", common.Truncate(s.Name, 40))
					tag = "unarchive"
				} else {
					title = "Archive skill?"
					desc = fmt.Sprintf("Archive \"%s\"? Its materials will become inactive.", common.Truncate(s.Name, 40))
					tag = "archive"
				}
				f := common.NewConfirmForm(title, desc, tag)
				m.overlay = f
				return m, f.Init()
			}
		case "D":
			if m.data != nil && len(m.data.SkillsSummary) > 0 {
				s := m.data.SkillsSummary[m.cursor]
				if !s.IsArchived {
					return m, func() tea.Msg {
						return common.ToastMsg{Text: "Archive the skill first before deleting.", IsError: true}
					}
				}
				f := common.NewConfirmForm(
					"Delete skill?",
					fmt.Sprintf("Permanently delete \"%s\" and all its materials?", common.Truncate(s.Name, 40)),
					"delete",
				)
				m.overlay = f
				return m, f.Init()
			}
		}
	}
	return m, nil
}

// skillCount returns the number of skills in the data (safe on nil).
func skillCount(d *model.CategoryDetail) int {
	if d == nil {
		return 0
	}
	return len(d.SkillsSummary)
}

// selectedIsArchived reports whether the highlighted skill is archived.
func (m Model) selectedIsArchived() bool {
	if m.data == nil || len(m.data.SkillsSummary) == 0 || m.cursor >= len(m.data.SkillsSummary) {
		return false
	}
	return m.data.SkillsSummary[m.cursor].IsArchived
}

// HasOverlay reports whether a form or confirm dialog is currently open.
func (m Model) HasOverlay() bool { return m.overlay != nil }

// submitSkillForm runs the create/update API call after form completion.
func submitSkillForm(c *api.Client, categoryID string, sf common.SkillForm) tea.Cmd {
	return func() tea.Msg {
		var err error
		var action string
		if sf.IsEdit() {
			err = c.UpdateSkill(sf.EditID(), sf.Name(), categoryID, sf.Color())
			action = "updated"
		} else {
			err = c.CreateSkill(sf.Name(), categoryID, sf.Color())
			action = "created"
		}
		if err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return skillMutatedMsg{toast: "Skill " + action}
	}
}

// submitSkillConfirm runs archive/unarchive/delete after confirmation.
func submitSkillConfirm(c *api.Client, d *model.CategoryDetail, cursor int, tag string) tea.Cmd {
	return func() tea.Msg {
		if d == nil || cursor >= len(d.SkillsSummary) {
			return nil
		}
		s := d.SkillsSummary[cursor]

		var err error
		var successText string
		switch tag {
		case "archive":
			err = c.ArchiveSkill(s.ID)
			successText = fmt.Sprintf("Archived \"%s\"", common.Truncate(s.Name, 30))
		case "unarchive":
			err = c.UnarchiveSkill(s.ID)
			successText = fmt.Sprintf("Unarchived \"%s\"", common.Truncate(s.Name, 30))
		case "delete":
			err = c.DeleteSkill(s.ID)
			successText = fmt.Sprintf("Deleted \"%s\"", common.Truncate(s.Name, 30))
		default:
			return nil
		}

		if err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return skillMutatedMsg{toast: successText}
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
		return common.ErrorView(m.err, m.width)
	}
	if m.data == nil {
		return ""
	}

	d := m.data
	var b strings.Builder

	// Title
	b.WriteString(common.RenderTitle(d.Category.Name, m.width))
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

	// Active materials (brief list)
	if len(d.ActiveMaterials) > 0 {
		b.WriteString(common.SectionStyle.Render("Active Materials"))
		b.WriteString("\n")
		limit := 5
		if len(d.ActiveMaterials) < limit {
			limit = len(d.ActiveMaterials)
		}
		for _, mat := range d.ActiveMaterials[:limit] {
			pct := 0.0
			if mat.TotalUnits > 0 {
				pct = mat.CompletedUnits / mat.TotalUnits
			}
			bar := common.RenderBar(m.barWide, pct, 0)
			skillLabel := common.MutedStyle.Render(common.Truncate(mat.SkillName, 16))
			name := common.Truncate(mat.Name, 28)
			b.WriteString(fmt.Sprintf("  %s  %s  %s\n", bar, common.MutedStyle.Render(name), skillLabel))
		}
		if len(d.ActiveMaterials) > limit {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf("  … and %d more\n", len(d.ActiveMaterials)-limit)))
		}
	}

	// Skills list
	b.WriteString(common.SectionStyle.Render("Skills"))
	b.WriteString("\n")

	if len(d.SkillsSummary) == 0 {
		b.WriteString(common.MutedStyle.Render("  No skills.\n"))
	} else {
		// Reserve: title(2) + kpis(3) + active header+rows(~7) + skills header(2) + table header(1) + separator(1) + help(2); tab bar=3
		usedLines := 21
		if len(d.ActiveMaterials) == 0 {
			usedLines = 14
		}
		visibleHeight := m.height - usedLines
		if visibleHeight < 3 {
			visibleHeight = 3
		}
		start, end := common.VisibleWindow(m.cursor, len(d.SkillsSummary), visibleHeight)

		rows := make([][]string, end-start)
		for i := start; i < end; i++ {
			rows[i-start] = m.buildSkillRow(i)
		}

		selectedIdx := m.cursor - start
		t := table.New().
			Headers("", "Skill", "Progress", "Materials", "").
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

		if len(d.SkillsSummary) > visibleHeight {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d skills\n", start+1, end, len(d.SkillsSummary),
			)))
		}
	}

	b.WriteString("\n")
	b.WriteString(common.RenderHelp(m.keys, m.width))
	return b.String()
}

func (m Model) buildSkillRow(i int) []string {
	s := m.data.SkillsSummary[i]

	cursor := " "
	if i == m.cursor {
		cursor = common.SelectedStyle.Render("▶")
	}

	colorClass := ""
	if s.Color != nil {
		colorClass = *s.Color
	}
	dot := common.ColorDot(colorClass)

	nameStyle := common.TableCellStyle
	switch {
	case i == m.cursor:
		nameStyle = common.TableSelectedStyle
	case s.IsArchived:
		nameStyle = common.ArchivedNameStyle
	}
	name := dot + " " + common.ColoredName(colorClass, common.Truncate(s.Name, 24), nameStyle)

	pct := 0.0
	if s.TotalUnits > 0 {
		pct = s.CompletedUnits / s.TotalUnits
	}
	bar := common.RenderBar(m.barNarrow, pct, 0)

	meta := fmt.Sprintf("%d active / %d total", s.ActiveMaterialCount, s.MaterialCount)

	archived := ""
	if s.IsArchived {
		archived = "[archived]"
	}

	return []string{cursor, name, bar, meta, archived}
}
