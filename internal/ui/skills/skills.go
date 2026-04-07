// Package skills provides the skills list TUI screen.
package skills

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

type skillsLoadedMsg struct{ data []model.Skill }
type errMsg struct{ err error }

// Model is the Bubble Tea model for the skills screen.
type Model struct {
	client  *api.Client
	skills  []model.Skill
	cursor  int
	loading bool
	err     error
	width   int
	height  int
}

func New(client *api.Client) Model {
	return Model{client: client, loading: true}
}

func load(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetAllSkills()
		if err != nil {
			return errMsg{err}
		}
		return skillsLoadedMsg{data}
	}
}

func (m Model) Init() tea.Cmd {
	return load(m.client)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case skillsLoadedMsg:
		m.skills = msg.data
		m.loading = false

	case errMsg:
		m.err = msg.err
		m.loading = false

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
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return common.MutedStyle.Render("\n  Loading…")
	}
	if m.err != nil {
		return common.DangerStyle.Render("\n  Error: " + m.err.Error() + "\n\n  Press r to retry, esc to go back.")
	}

	var b strings.Builder
	b.WriteString(common.TitleStyle.Render("Skills"))
	b.WriteString("\n")

	if len(m.skills) == 0 {
		b.WriteString(common.MutedStyle.Render("  No skills found.\n"))
	} else {
		// Group by category
		type group struct {
			category string
			skills   []model.Skill
		}
		var groups []group
		catIdx := map[string]int{}

		for _, s := range m.skills {
			if idx, ok := catIdx[s.CategoryID]; ok {
				groups[idx].skills = append(groups[idx].skills, s)
			} else {
				catIdx[s.CategoryID] = len(groups)
				groups = append(groups, group{category: s.CategoryName, skills: []model.Skill{s}})
			}
		}

		// Flatten into display rows to track cursor position
		type row struct {
			isHeader bool
			label    string
			skillIdx int // index into m.skills
		}
		var rows []row
		skillRowIdx := map[int]int{} // m.skills index → rows index
		for _, g := range groups {
			rows = append(rows, row{isHeader: true, label: g.category})
			for _, s := range g.skills {
				// find index in m.skills
				for si, ms := range m.skills {
					if ms.ID == s.ID {
						skillRowIdx[si] = len(rows)
						break
					}
				}
				rows = append(rows, row{isHeader: false, label: s.Name, skillIdx: func() int {
					for si, ms := range m.skills {
						if ms.ID == s.ID {
							return si
						}
					}
					return 0
				}()})
			}
		}

		visibleHeight := m.height - 8
		if visibleHeight < 5 {
			visibleHeight = 5
		}
		// Determine which rows row to show as selected
		selectedRow := skillRowIdx[m.cursor]
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
				b.WriteString(cursorStr + nameStyle.Render(r.label))
			}
			b.WriteString("\n")
		}

		b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
			"\n  %d skills across %d categories\n", len(m.skills), len(groups),
		)))
	}

	b.WriteString("\n")
	keys := []string{
		common.KeyHelp("j/k", "navigate"),
		common.KeyHelp("r", "refresh"),
		common.KeyHelp("esc", "back"),
	}
	b.WriteString(common.HelpStyle.Render(strings.Join(keys, "   ")))
	return b.String()
}
