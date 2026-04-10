// Package categories provides the categories list TUI screen.
package categories

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

type categoriesLoadedMsg struct{ data []model.Category }

// OpenCategoryMsg is sent when the user presses enter on a category.
type OpenCategoryMsg struct{ CategoryID string }

// Model is the Bubble Tea model for the categories list screen.
type Model struct {
	client       *api.Client
	categories   []model.Category // all categories from API
	filtered     []model.Category // categories after archive filter
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

func buildKeys(hasArchived bool, showArchived bool) common.SimpleKeyMap {
	bindings := []common.Binding{
		common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
		common.KB("enter", "open"),
		common.KB("n", "new"),
		common.KB("e", "edit"),
	}
	if hasArchived {
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

func load(c *api.Client, includeArchived bool) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetAllCategories(includeArchived)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return categoriesLoadedMsg{data}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(load(m.client, m.showArchived), m.spinner.Tick)
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
		case forms.CategoryFormDoneMsg:
			m.overlay = nil
			if !msg.Cancelled {
				if cf, ok := updated.(forms.CategoryForm); ok {
					return m, submitCategoryForm(m.client, cf)
				}
			}
			return m, cmd

		case forms.ConfirmDoneMsg:
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

	case categoriesLoadedMsg:
		m.categories = msg.data
		m.loading = false
		m.applyFilter()
		if m.cursor >= len(m.filtered) && m.cursor > 0 {
			m.cursor = len(m.filtered) - 1
		}
		m.keys = buildKeys(m.selectedIsArchived(), m.showArchived)

	case common.CategoryChangedMsg:
		return m, load(m.client, m.showArchived)

	case categoryMutatedMsg:
		return m, tea.Batch(
			func() tea.Msg { return common.ToastMsg{Text: msg.toast} },
			func() tea.Msg { return common.CategoryChangedMsg{} },
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
				return m, func() tea.Msg { return OpenCategoryMsg{CategoryID: id} }
			}
		case "r":
			m.loading = true
			m.err = nil
			return m, load(m.client, m.showArchived)
		case "a":
			m.showArchived = !m.showArchived
			m.loading = true
			m.err = nil
			return m, load(m.client, m.showArchived)
		case "n":
			f := forms.NewCategoryCreateForm()
			m.overlay = f
			return m, f.Init()
		case "e":
			if len(m.filtered) > 0 {
				cat := m.filtered[m.cursor]
				f := forms.NewCategoryEditForm(cat.ID, cat.Name, cat.Color)
				m.overlay = f
				return m, f.Init()
			}
		case "A":
			if len(m.filtered) > 0 {
				cat := m.filtered[m.cursor]
				var title, desc, tag string
				if cat.IsArchived {
					title = "Unarchive category?"
					desc = fmt.Sprintf("Unarchive \"%s\"?", common.Truncate(cat.Name, 40))
					tag = "unarchive"
				} else {
					title = "Archive category?"
					desc = fmt.Sprintf("Archive \"%s\"? All its skills will also be archived.", common.Truncate(cat.Name, 40))
					tag = "archive"
				}
				f := forms.NewConfirmForm(title, desc, tag)
				m.overlay = f
				return m, f.Init()
			}
		case "D":
			if len(m.filtered) > 0 {
				cat := m.filtered[m.cursor]
				if !cat.IsArchived {
					return m, func() tea.Msg {
						return common.ToastMsg{Text: "Archive the category first before deleting.", IsError: true}
					}
				}
				f := forms.NewConfirmForm(
					"Delete category?",
					fmt.Sprintf("Permanently delete \"%s\" and all its skills?", common.Truncate(cat.Name, 40)),
					"delete",
				)
				m.overlay = f
				return m, f.Init()
			}
		}
	}
	return m, nil
}

// selectedIsArchived returns true if the currently highlighted category is archived.
func (m Model) selectedIsArchived() bool {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return false
	}
	return m.filtered[m.cursor].IsArchived
}

// HasOverlay reports whether a form or confirm dialog is currently open.
func (m Model) HasOverlay() bool { return m.overlay != nil }

// applyFilter rebuilds m.filtered from m.categories.
// Archive filtering is handled server-side; this exists for consistency with other screens.
func (m *Model) applyFilter() {
	m.filtered = m.categories
}

// categoryMutatedMsg is an internal message carrying the result of a mutation.
type categoryMutatedMsg struct{ toast string }

// submitCategoryForm runs the create or update API call after form completion.
func submitCategoryForm(c *api.Client, cf forms.CategoryForm) tea.Cmd {
	return func() tea.Msg {
		var err error
		if cf.IsEdit() {
			err = c.UpdateCategory(cf.EditID(), cf.Name(), cf.Color())
		} else {
			err = c.CreateCategory(cf.Name(), cf.Color())
		}
		if err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.CategoryChangedMsg{}
	}
}

// submitConfirm runs the archive/unarchive/delete API call after confirmation.
func submitConfirm(c *api.Client, cats []model.Category, cursor int, tag string) tea.Cmd {
	return func() tea.Msg {
		if cursor >= len(cats) {
			return nil
		}
		id := cats[cursor].ID
		name := cats[cursor].Name

		var err error
		var successText string
		switch tag {
		case "archive":
			err = c.ArchiveCategory(id)
			successText = fmt.Sprintf("Archived \"%s\"", common.Truncate(name, 30))
		case "unarchive":
			err = c.UnarchiveCategory(id)
			successText = fmt.Sprintf("Unarchived \"%s\"", common.Truncate(name, 30))
		case "delete":
			err = c.DeleteCategory(id)
			successText = fmt.Sprintf("Deleted \"%s\"", common.Truncate(name, 30))
		default:
			return nil
		}

		if err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return categoryMutatedMsg{toast: successText}
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
	tag := ""
	if m.showArchived {
		tag = common.MutedStyle.Render("[showing archived]")
	}
	b.WriteString(common.RenderTitleWithTag("Categories", tag, m.width))
	b.WriteString("\n")

	if len(m.filtered) == 0 {
		if m.showArchived {
			b.WriteString(common.MutedStyle.Render("  No categories found.\n"))
		} else {
			b.WriteString(common.MutedStyle.Render("  No categories found.\n"))
		}
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
			Headers("", "Name", "Skills", "").
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
					return common.DefaultNameStyle
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
	cat := m.filtered[i]

	cursor := " "
	if i == m.cursor {
		cursor = common.SelectedStyle.Render("▶")
	}

	colorClass := ""
	if cat.Color != nil {
		colorClass = *cat.Color
	}
	dot := common.ColorDot(colorClass)

	nameStyle := common.TableCellStyle
	switch {
	case i == m.cursor:
		nameStyle = common.TableSelectedStyle
	case cat.IsArchived:
		nameStyle = common.ArchivedNameStyle
	}
	name := common.ColoredName(colorClass, common.Truncate(cat.Name, 30), nameStyle)

	skillCount := fmt.Sprintf("%d", cat.SkillCount())

	archived := ""
	if cat.IsArchived {
		archived = "[archived]"
	}

	return []string{cursor, dot + " " + name, skillCount, archived}
}
