// Package skills provides the skills list TUI screen.
package skills

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

type skillsLoadedMsg struct{ data []model.Skill }

// OpenSkillMsg is sent when the user presses enter on the selected skill.
type OpenSkillMsg struct{ SkillID string }

// Model is the Bubble Tea model for the skills screen.
type Model struct {
	client  *api.Client
	skills  []model.Skill
	cursor  int
	loading bool
	err     error
	width   int
	height  int
	spinner spinner.Model
	help    help.Model
	keys    common.SimpleKeyMap
}

func New(client *api.Client) Model {
	return Model{
		client:  client,
		loading: true,
		spinner: common.NewSpinner(),
		help:    common.NewHelp(),
		keys: common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("enter", "open skill"),
			common.KB("r", "refresh"),
			common.KB("esc", "back"),
		}},
	}
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width

	case skillsLoadedMsg:
		m.skills = msg.data
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
			if m.cursor < len(m.skills)-1 {
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
		case "enter":
			if len(m.skills) > 0 {
				id := m.skills[m.cursor].ID
				return m, func() tea.Msg { return OpenSkillMsg{SkillID: id} }
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
		return common.ErrorView(m.err)
	}

	var b strings.Builder
	b.WriteString(common.RenderTitle("Skills", m.width))
	b.WriteString("\n")

	if len(m.skills) == 0 {
		b.WriteString(common.MutedStyle.Render("  No skills found.\n"))
	} else {
		// Group by category, preserving encounter order.
		type group struct {
			category string
			indices  []int // indices into m.skills
		}
		var groups []group
		catIdx := map[string]int{} // categoryID → index in groups

		for si, s := range m.skills {
			if gi, ok := catIdx[s.CategoryID]; ok {
				groups[gi].indices = append(groups[gi].indices, si)
			} else {
				catIdx[s.CategoryID] = len(groups)
				groups = append(groups, group{category: s.CategoryName(), indices: []int{si}})
			}
		}

		// Flatten into display rows; track which row each skill sits at.
		type row struct {
			isHeader bool
			label    string
			skillIdx int // index into m.skills (only valid when !isHeader)
		}
		var rows []row
		// skillToRow[si] = row index for m.skills[si]
		skillToRow := make([]int, len(m.skills))
		for _, g := range groups {
			rows = append(rows, row{isHeader: true, label: g.category})
			for _, si := range g.indices {
				skillToRow[si] = len(rows)
				rows = append(rows, row{isHeader: false, label: m.skills[si].Name, skillIdx: si})
			}
		}

		// title(1) + marginBottom(1) + blank(1) + count(1) + help(1) + marginTop(1) = 6
		visibleHeight := m.height - 6
		if visibleHeight < 5 {
			visibleHeight = 5
		}
		selectedRow := skillToRow[m.cursor]
		start := 0
		if selectedRow > visibleHeight/2 {
			start = selectedRow - visibleHeight/2
		}
		end := start + visibleHeight
		if end > len(rows) {
			end = len(rows)
		}

		for ri := start; ri < end; ri++ {
			r := rows[ri]
			if r.isHeader {
				b.WriteString(common.SectionStyle.Render("  " + r.label))
			} else {
				cursorStr := "    "
				nameStyle := common.MutedStyle
				if r.skillIdx == m.cursor {
					cursorStr = "  " + common.SelectedStyle.Render("▶ ")
					nameStyle = common.SelectedStyle
				}
				s := m.skills[r.skillIdx]
				matCount := common.MutedStyle.Render(fmt.Sprintf("(%d)", s.MaterialCount()))
				dot := common.ColorDot(func() string {
					if s.Color != nil {
						return *s.Color
					}
					return ""
				}())
				b.WriteString(cursorStr + dot + " " + nameStyle.Render(r.label) + "  " + matCount)
			}
			b.WriteString("\n")
		}

		b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
			"\n  %d skills across %d categories\n", len(m.skills), len(groups),
		)))
	}

	b.WriteString("\n")
	b.WriteString(common.HelpStyle.Render(m.help.View(m.keys)))
	return b.String()
}
