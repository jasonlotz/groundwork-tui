// Package categories provides the categories list TUI screen.
package categories

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

type categoriesLoadedMsg struct{ data []model.Category }

// OpenCategoryMsg is sent when the user presses enter on a category.
type OpenCategoryMsg struct{ CategoryID string }

// Model is the Bubble Tea model for the categories list screen.
type Model struct {
	client     *api.Client
	categories []model.Category
	cursor     int
	loading    bool
	err        error
	width      int
	height     int
	spinner    spinner.Model
}

func New(client *api.Client) Model {
	return Model{client: client, loading: true, spinner: common.NewSpinner()}
}

func load(c *api.Client) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetAllCategories()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return categoriesLoadedMsg{data}
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

	case categoriesLoadedMsg:
		m.categories = msg.data
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
			if m.cursor < len(m.categories)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if len(m.categories) > 0 {
				id := m.categories[m.cursor].ID
				return m, func() tea.Msg { return OpenCategoryMsg{CategoryID: id} }
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
		return common.SpinnerView(m.spinner)
	}
	if m.err != nil {
		return common.ErrorView(m.err)
	}

	var b strings.Builder
	b.WriteString(common.RenderTitle("Categories", m.width))
	b.WriteString("\n")

	if len(m.categories) == 0 {
		b.WriteString(common.MutedStyle.Render("  No categories found.\n"))
	} else {
		visibleHeight := m.height - 6
		if visibleHeight < 5 {
			visibleHeight = 5
		}
		start, end := common.VisibleWindow(m.cursor, len(m.categories), visibleHeight)

		for i := start; i < end; i++ {
			b.WriteString(m.renderRow(i))
			b.WriteString("\n")
		}

		if len(m.categories) > visibleHeight {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d\n", start+1, end, len(m.categories),
			)))
		}
	}

	b.WriteString("\n")
	keys := []string{
		common.KeyHelp("j/k", "navigate"),
		common.KeyHelp("enter", "open"),
		common.KeyHelp("r", "refresh"),
		common.KeyHelp("esc", "back"),
	}
	b.WriteString(common.HelpStyle.Render(strings.Join(keys, "   ")))
	return b.String()
}

func (m Model) renderRow(i int) string {
	cat := m.categories[i]
	cursorStr := "  "
	nameStyle := common.MutedStyle
	switch {
	case i == m.cursor:
		cursorStr = common.SelectedStyle.Render("▶ ")
		nameStyle = common.SelectedStyle
	case cat.IsArchived:
		nameStyle = common.ArchivedNameStyle
	}

	archived := ""
	if cat.IsArchived {
		archived = "  " + common.MutedStyle.Render("[archived]")
	}

	skillCount := common.MutedStyle.Render(fmt.Sprintf("(%d skills)", cat.SkillCount()))
	name := nameStyle.Render(common.Truncate(cat.Name, 30))

	return cursorStr + name + "  " + skillCount + archived
}
