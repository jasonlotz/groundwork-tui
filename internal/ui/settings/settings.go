// Package settings provides the settings screen: theme picker and exercise management.
package settings

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
	"github.com/jasonlotz/groundwork-tui/internal/ui/forms"
	"github.com/jasonlotz/groundwork-tui/internal/ui/theme"
)

// settingsSection identifies which section is active.
type settingsSection int

const (
	sectionTheme     settingsSection = 0
	sectionExercises settingsSection = 1
)

// exerciseAction identifies which action overlay is open.
type exerciseAction int

const (
	actionNone    exerciseAction = 0
	actionAdd     exerciseAction = 1
	actionRename  exerciseAction = 2
	actionConfirm exerciseAction = 3
)

// --- internal messages ---

type exercisesLoadedMsg struct{ data []model.Exercise }
type exerciseNameForm struct {
	form  *huh.Form
	value *string
}

// Model is the Bubble Tea model for the settings screen.
type Model struct {
	client      *api.Client
	section     settingsSection
	themeCursor int
	// exercises section
	exercises       []model.Exercise
	exerciseCursor  int
	exerciseLoading bool
	showArchived    bool
	// overlay
	action      exerciseAction
	confirmTag  string
	nameForm    *exerciseNameForm
	confirmForm *forms.ConfirmForm
	pendingID   string
	width       int
	height      int
	keys        common.SimpleKeyMap
}

func New(client *api.Client) Model {
	// Start cursor on the currently active theme.
	cursor := 0
	for i, t := range theme.All {
		if t.Name == theme.Active.Name {
			cursor = i
			break
		}
	}
	return Model{
		client:      client,
		section:     sectionTheme,
		themeCursor: cursor,
		keys:        buildKeys(sectionTheme),
	}
}

func buildKeys(s settingsSection) common.SimpleKeyMap {
	switch s {
	case sectionTheme:
		return common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("enter", "select theme"),
			common.KB("tab", "exercises"),
			common.KB("esc", "back"),
		}}
	default:
		return common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("a", "add"),
			common.KB("e", "rename"),
			common.KB("D", "archive/delete"),
			common.KB("tab", "theme"),
			common.KB("esc", "back"),
		}}
	}
}

// HasOverlay reports whether a form overlay is currently active.
func (m Model) HasOverlay() bool { return m.action != actionNone }

func loadExercises(c *api.Client, includeArchived bool) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetAllExercises(includeArchived)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return exercisesLoadedMsg{data}
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Overlay routing.
	if m.action == actionAdd || m.action == actionRename {
		if m.nameForm != nil {
			if k, ok := msg.(tea.KeyMsg); ok && (k.String() == "ctrl+c") {
				return m, tea.Quit
			}
			if k, ok := msg.(tea.KeyMsg); ok && k.String() == "esc" {
				m.action = actionNone
				m.nameForm = nil
				return m, nil
			}
			// huh forms don't bubble esc cleanly — handle via NameFormDoneMsg pattern
			f, cmd, done := updateHuhForm(m.nameForm.form, msg)
			m.nameForm.form = f
			if done {
				name := strings.TrimSpace(*m.nameForm.value)
				act := m.action
				id := m.pendingID
				m.action = actionNone
				m.nameForm = nil
				m.pendingID = ""
				if name != "" {
					if act == actionAdd {
						return m, createExercise(m.client, name)
					}
					return m, renameExercise(m.client, id, name)
				}
			}
			return m, cmd
		}
		m.action = actionNone
		return m, nil
	}

	if m.action == actionConfirm {
		if m.confirmForm != nil {
			if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
				return m, tea.Quit
			}
			updated, cmd := m.confirmForm.Update(msg)
			if cf, ok := updated.(forms.ConfirmForm); ok {
				*m.confirmForm = cf
			}
			if done, ok := msg.(forms.ConfirmDoneMsg); ok {
				m.action = actionNone
				id := m.pendingID
				tag := m.confirmTag
				m.pendingID = ""
				m.confirmTag = ""
				m.confirmForm = nil
				if done.Confirmed {
					if tag == "archive" {
						return m, archiveExercise(m.client, id)
					}
					if tag == "delete" {
						return m, deleteExercise(m.client, id)
					}
					if tag == "unarchive" {
						return m, unarchiveExercise(m.client, id)
					}
				}
			}
			return m, cmd
		}
		m.action = actionNone
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case exercisesLoadedMsg:
		m.exercises = msg.data
		m.exerciseLoading = false
		if m.exerciseCursor >= len(m.exercises) && m.exerciseCursor > 0 {
			m.exerciseCursor = len(m.exercises) - 1
		}

	case common.ErrMsg:
		m.exerciseLoading = false

	case common.ExerciseChangedMsg:
		m.exerciseLoading = true
		return m, loadExercises(m.client, m.showArchived)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return common.GoBackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.section == sectionTheme {
				m.section = sectionExercises
				m.keys = buildKeys(sectionExercises)
				if len(m.exercises) == 0 && !m.exerciseLoading {
					m.exerciseLoading = true
					return m, loadExercises(m.client, m.showArchived)
				}
			} else {
				m.section = sectionTheme
				m.keys = buildKeys(sectionTheme)
			}

		default:
			if m.section == sectionTheme {
				return m.handleThemeKey(msg.String())
			}
			return m.handleExerciseKey(msg.String())
		}
	}
	return m, nil
}

func (m Model) handleThemeKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.themeCursor < len(theme.All)-1 {
			m.themeCursor++
		}
	case "k", "up":
		if m.themeCursor > 0 {
			m.themeCursor--
		}
	case "enter", " ":
		selected := theme.All[m.themeCursor]
		name := selected.Name
		return m, func() tea.Msg { return common.ThemeChangedMsg{ThemeName: name} }
	}
	return m, nil
}

func (m Model) handleExerciseKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if m.exerciseCursor < len(m.exercises)-1 {
			m.exerciseCursor++
		}
	case "k", "up":
		if m.exerciseCursor > 0 {
			m.exerciseCursor--
		}
	case "H":
		m.showArchived = !m.showArchived
		m.exerciseLoading = true
		return m, loadExercises(m.client, m.showArchived)
	case "a":
		nf := buildNameForm("New Exercise", "Name", "")
		m.nameForm = nf
		m.action = actionAdd
		return m, nf.form.Init()
	case "e":
		if ex := m.selectedExercise(); ex != nil {
			nf := buildNameForm("Rename Exercise", "Name", ex.Name)
			m.nameForm = nf
			m.pendingID = ex.ID
			m.action = actionRename
			return m, nf.form.Init()
		}
	case "D":
		if ex := m.selectedExercise(); ex != nil {
			m.pendingID = ex.ID
			if ex.IsArchived {
				// Archived: offer delete
				cf := forms.NewConfirmForm("Delete Exercise?",
					fmt.Sprintf("Permanently delete %q?", ex.Name), "delete")
				m.confirmForm = &cf
				m.confirmTag = "delete"
				m.action = actionConfirm
				return m, cf.Init()
			}
			// Not archived: offer archive (or unarchive if logic changes)
			cf := forms.NewConfirmForm("Archive Exercise?",
				fmt.Sprintf("Archive %q? It will be hidden from workouts.", ex.Name), "archive")
			m.confirmForm = &cf
			m.confirmTag = "archive"
			m.action = actionConfirm
			return m, cf.Init()
		}
	case "u":
		if ex := m.selectedExercise(); ex != nil && ex.IsArchived {
			m.pendingID = ex.ID
			cf := forms.NewConfirmForm("Unarchive Exercise?",
				fmt.Sprintf("Unarchive %q?", ex.Name), "unarchive")
			m.confirmForm = &cf
			m.confirmTag = "unarchive"
			m.action = actionConfirm
			return m, cf.Init()
		}
	}
	return m, nil
}

func (m *Model) selectedExercise() *model.Exercise {
	if len(m.exercises) == 0 || m.exerciseCursor >= len(m.exercises) {
		return nil
	}
	ex := m.exercises[m.exerciseCursor]
	return &ex
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(common.RenderTitle("Settings", m.width))
	b.WriteString("\n")

	// Section selector
	themeLabel := "[ Theme ]"
	exLabel := "[ Exercises ]"
	if m.section == sectionTheme {
		b.WriteString(common.SelectedStyle.Render(themeLabel) + "  " + common.DimStyle.Render(exLabel) + "\n")
	} else {
		b.WriteString(common.DimStyle.Render(themeLabel) + "  " + common.SelectedStyle.Render(exLabel) + "\n")
	}
	b.WriteString("\n")

	if m.section == sectionTheme {
		b.WriteString(m.renderThemeSection())
	} else {
		b.WriteString(m.renderExercisesSection())
	}

	b.WriteString("\n")
	b.WriteString(common.RenderHelp(m.keys, m.width))

	view := b.String()
	if m.action == actionAdd || m.action == actionRename {
		if m.nameForm != nil {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.nameForm.form.View())
		}
	}
	if m.action == actionConfirm && m.confirmForm != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.confirmForm.View())
	}
	return view
}

func (m Model) renderThemeSection() string {
	var b strings.Builder
	for i, t := range theme.All {
		active := t.Name == theme.Active.Name
		selected := i == m.themeCursor

		prefix := "  "
		if selected {
			prefix = common.SelectedStyle.Render("▶ ")
		}

		var nameStr string
		switch {
		case active && selected:
			nameStr = common.SuccessStyle.Render("✓ " + t.Name)
		case active:
			nameStr = common.SuccessStyle.Render("✓ " + t.Name)
		case selected:
			nameStr = common.SelectedStyle.Render(t.Name)
		default:
			nameStr = common.DimStyle.Render(t.Name)
		}

		swatch := renderSwatch(t)
		b.WriteString(fmt.Sprintf("%s%-14s  %s\n", prefix, nameStr, swatch))
	}
	return b.String()
}

func (m Model) renderExercisesSection() string {
	var b strings.Builder

	if m.exerciseLoading {
		b.WriteString(common.DimStyle.Render("  Loading exercises…\n"))
		return b.String()
	}

	archived := ""
	if m.showArchived {
		archived = common.DimStyle.Render(" (showing archived)")
	}
	b.WriteString(common.SectionStyle.Render("Exercises") + archived + "\n")

	if len(m.exercises) == 0 {
		b.WriteString(common.DimStyle.Render("  No exercises. Press 'a' to add one.\n"))
		return b.String()
	}

	// overhead: RenderTitle=3 + blank=1 + section selector=1 + blank=1 + Section header=2 + blank=1 + help=2 = 11
	// tab bar = 3  → total 14
	visibleItems := (m.height - 14)
	if visibleItems < 3 {
		visibleItems = 3
	}
	start, end := common.VisibleWindow(m.exerciseCursor, len(m.exercises), visibleItems)

	for i := start; i < end; i++ {
		ex := m.exercises[i]
		selected := i == m.exerciseCursor
		cursor := "  "
		if selected {
			cursor = common.SelectedStyle.Render("▶ ")
		}
		var nameStr string
		if ex.IsArchived {
			nameStr = common.ArchivedNameStyle.Render(ex.Name + " [archived]")
		} else if selected {
			nameStr = common.SelectedStyle.Render(ex.Name)
		} else {
			nameStr = ex.Name
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, nameStr))
	}

	if len(m.exercises) > visibleItems {
		b.WriteString(common.DimStyle.Render(fmt.Sprintf(
			"  %d–%d of %d\n", start+1, end, len(m.exercises),
		)))
	}
	return b.String()
}

// renderSwatch renders a row of colored blocks showing the theme palette.
func renderSwatch(t theme.AppTheme) string {
	swatchColors := []lipgloss.Color{
		t.Colors.Primary,
		t.Colors.Highlight,
		t.Colors.Success,
		t.Colors.Warning,
		t.Colors.Danger,
		t.Colors.Dim,
	}
	var parts []string
	for _, c := range swatchColors {
		parts = append(parts, lipgloss.NewStyle().Foreground(c).Render("■"))
	}
	return strings.Join(parts, "")
}

// buildNameForm creates a single-field huh form for a name input.
func buildNameForm(title, fieldTitle, placeholder string) *exerciseNameForm {
	value := placeholder
	nf := &exerciseNameForm{value: &value}
	nf.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Description(fieldTitle).
				Placeholder(placeholder).
				Value(nf.value),
		),
	).WithTheme(forms.ActiveTheme)
	return nf
}

// updateHuhForm is a wrapper that advances a huh.Form and reports completion.
func updateHuhForm(f *huh.Form, msg tea.Msg) (*huh.Form, tea.Cmd, bool) {
	updated, cmd := f.Update(msg)
	if uf, ok := updated.(*huh.Form); ok {
		f = uf
	}
	return f, cmd, f.State == huh.StateCompleted
}

// --- API commands ---

func createExercise(c *api.Client, name string) tea.Cmd {
	return func() tea.Msg {
		if err := c.CreateExercise(name); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.ExerciseChangedMsg{}
	}
}

func renameExercise(c *api.Client, id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := c.UpdateExercise(id, name); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.ExerciseChangedMsg{}
	}
}

func archiveExercise(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.ArchiveExercise(id); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.ExerciseChangedMsg{}
	}
}

func unarchiveExercise(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.UnarchiveExercise(id); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.ExerciseChangedMsg{}
	}
}

func deleteExercise(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.DeleteExercise(id); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return common.ExerciseChangedMsg{}
	}
}
