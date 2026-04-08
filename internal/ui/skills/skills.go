// Package skills provides the skills list TUI screen.
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
)

type skillsLoadedMsg struct{ data []model.Skill }

// OpenSkillMsg is sent when the user presses enter on the selected skill.
type OpenSkillMsg struct{ SkillID string }

// Model is the Bubble Tea model for the skills screen.
type Model struct {
	client       *api.Client
	skills       []model.Skill // all skills from API
	filtered     []model.Skill // skills after archive filter
	showArchived bool
	cursor       int
	loading      bool
	err          error
	width        int
	height       int
	spinner      spinner.Model
	keys         common.SimpleKeyMap
	overlay      tea.Model
}

func New(client *api.Client) Model {
	return Model{
		client:  client,
		loading: true,
		spinner: common.NewSpinner(),
		keys:    buildKeys(false, false),
	}
}

func buildKeys(isArchived bool, showArchived bool) common.SimpleKeyMap {
	bindings := []common.Binding{
		common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
		common.KB("enter", "open skill"),
		common.KB("e", "edit"),
	}
	if isArchived {
		bindings = append(bindings, common.KB("A", "unarchive"))
	} else {
		bindings = append(bindings, common.KB("A", "archive"))
	}
	archivedLabel := "show archived"
	if showArchived {
		archivedLabel = "hide archived"
	}
	bindings = append(bindings,
		common.KB("D", "delete (archived)"),
		common.KB("a", archivedLabel),
		common.KB("r", "refresh"),
		common.KB("esc", "back"),
	)
	return common.SimpleKeyMap{Bindings: bindings}
}

func load(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetAllSkills()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return skillsLoadedMsg{data}
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

		switch msg := msg.(type) {
		case common.SkillFormDoneMsg:
			m.overlay = nil
			if !msg.Cancelled {
				if sf, ok := updated.(common.SkillForm); ok {
					return m, submitSkillForm(m.client, sf)
				}
			}
			return m, cmd

		case common.ConfirmDoneMsg:
			m.overlay = nil
			if msg.Confirmed {
				return m, submitConfirm(m.client, m.filtered, m.cursor, msg.Tag)
			}
			return m, cmd

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

	case skillsLoadedMsg:
		m.skills = msg.data
		m.loading = false
		m.applyFilter()
		if m.cursor >= len(m.filtered) && m.cursor > 0 {
			m.cursor = len(m.filtered) - 1
		}
		m.keys = buildKeys(m.selectedIsArchived(), m.showArchived)

	case common.SkillChangedMsg:
		return m, load(m.client)

	case skillMutatedMsg:
		return m, tea.Batch(
			func() tea.Msg { return common.ToastMsg{Text: msg.toast} },
			func() tea.Msg { return common.SkillChangedMsg{} },
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
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.keys = buildKeys(m.selectedIsArchived(), m.showArchived)
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.keys = buildKeys(m.selectedIsArchived(), m.showArchived)
			}
		case "enter":
			if len(m.filtered) > 0 {
				id := m.filtered[m.cursor].ID
				return m, func() tea.Msg { return OpenSkillMsg{SkillID: id} }
			}
		case "e":
			if len(m.filtered) > 0 {
				s := m.filtered[m.cursor]
				f := common.NewSkillEditForm(s.ID, s.Name, s.CategoryID, s.Color)
				m.overlay = f
				return m, f.Init()
			}
		case "A":
			if len(m.filtered) > 0 {
				s := m.filtered[m.cursor]
				var title, desc, tag string
				if s.IsArchived {
					title = "Unarchive skill?"
					desc = fmt.Sprintf("Unarchive \"%s\"?", common.Truncate(s.Name, 40))
					tag = "unarchive"
				} else {
					title = "Archive skill?"
					desc = fmt.Sprintf("Archive \"%s\"?", common.Truncate(s.Name, 40))
					tag = "archive"
				}
				f := common.NewConfirmForm(title, desc, tag)
				m.overlay = f
				return m, f.Init()
			}
		case "D":
			if len(m.filtered) > 0 {
				s := m.filtered[m.cursor]
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
		case "a":
			m.showArchived = !m.showArchived
			m.applyFilter()
			if m.cursor >= len(m.filtered) && len(m.filtered) > 0 {
				m.cursor = len(m.filtered) - 1
			}
			m.keys = buildKeys(m.selectedIsArchived(), m.showArchived)
		case "r":
			m.loading = true
			m.err = nil
			return m, load(m.client)
		}
	}
	return m, nil
}

// selectedIsArchived returns true if the currently highlighted skill is archived.
func (m Model) selectedIsArchived() bool {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return false
	}
	return m.filtered[m.cursor].IsArchived
}

// applyFilter rebuilds m.filtered from m.skills based on showArchived.
func (m *Model) applyFilter() {
	if m.showArchived {
		m.filtered = m.skills
		return
	}
	filtered := m.skills[:0:0]
	for _, s := range m.skills {
		if !s.IsArchived {
			filtered = append(filtered, s)
		}
	}
	m.filtered = filtered
}

// skillMutatedMsg is an internal message carrying the result of a mutation.
type skillMutatedMsg struct{ toast string }

// submitSkillForm runs the update API call after form completion.
func submitSkillForm(c *api.Client, sf common.SkillForm) tea.Cmd {
	return func() tea.Msg {
		err := c.UpdateSkill(sf.EditID(), sf.Name(), sf.CategoryID(), sf.Color())
		if err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.SkillChangedMsg{}
	}
}

// submitConfirm runs the archive/unarchive/delete API call after confirmation.
func submitConfirm(c *api.Client, skills []model.Skill, cursor int, tag string) tea.Cmd {
	return func() tea.Msg {
		if cursor >= len(skills) {
			return nil
		}
		id := skills[cursor].ID
		name := skills[cursor].Name

		var err error
		var successText string
		switch tag {
		case "archive":
			err = c.ArchiveSkill(id)
			successText = fmt.Sprintf("Archived \"%s\"", common.Truncate(name, 30))
		case "unarchive":
			err = c.UnarchiveSkill(id)
			successText = fmt.Sprintf("Unarchived \"%s\"", common.Truncate(name, 30))
		case "delete":
			err = c.DeleteSkill(id)
			successText = fmt.Sprintf("Deleted \"%s\"", common.Truncate(name, 30))
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

	var b strings.Builder
	filterTag := ""
	if m.showArchived {
		filterTag = "  " + common.MutedStyle.Render("[showing archived]")
	}
	b.WriteString(common.RenderTitle("Skills", m.width) + filterTag)
	b.WriteString("\n")

	if len(m.filtered) == 0 {
		b.WriteString(common.MutedStyle.Render("  No skills found.\n"))
	} else {
		// RenderTitle=3 + blank=1 + table-header=1 + table-sep=1 + blank=1 + help=2 = 9 overhead; tab bar=3 → 12 (data rows only)
		visibleHeight := m.height - 12
		if visibleHeight < 3 {
			visibleHeight = 3
		}
		start, end := common.VisibleWindow(m.cursor, len(m.filtered), visibleHeight)

		rows := make([][]string, end-start)
		for i := start; i < end; i++ {
			rows[i-start] = m.buildRow(i)
		}

		selectedIdx := m.cursor - start
		t := table.New().
			Headers("", "Skill", "Category", "Materials", "").
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

		if len(m.filtered) > visibleHeight {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d\n", start+1, end, len(m.filtered),
			)))
		}
	}

	b.WriteString("\n")
	b.WriteString(common.RenderHelp(m.keys, m.width))
	return b.String()
}

func (m Model) buildRow(i int) []string {
	s := m.filtered[i]

	cursor := " "
	if i == m.cursor {
		cursor = common.SelectedStyle.Render("▶")
	}

	skillColor := ""
	if s.Color != nil {
		skillColor = *s.Color
	}
	dot := common.ColorDot(skillColor)

	nameStyle := common.TableCellStyle
	switch {
	case i == m.cursor:
		nameStyle = common.TableSelectedStyle
	case s.IsArchived:
		nameStyle = common.ArchivedNameStyle
	}
	name := common.ColoredName(skillColor, common.Truncate(s.Name, 30), nameStyle)

	catColor := ""
	if s.Category.Color != nil {
		catColor = *s.Category.Color
	}
	catDot := common.ColorDot(catColor)
	catName := common.ColoredName(catColor, common.Truncate(s.CategoryName(), 25), common.TableCellStyle)

	matCount := fmt.Sprintf("%d", s.MaterialCount())

	archived := ""
	if s.IsArchived {
		archived = "[archived]"
	}

	return []string{cursor, dot + " " + name, catDot + " " + catName, matCount, archived}
}
