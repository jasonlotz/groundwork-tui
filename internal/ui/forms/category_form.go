package forms

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

// CategoryFormDoneMsg is sent when the category create/edit form completes.
type CategoryFormDoneMsg struct{ Cancelled bool }

// categoryFormState holds the bound form values behind stable pointers so they
// survive struct copies (huh holds pointers to these fields).
type categoryFormState struct {
	name  string
	color string
}

// CategoryForm is a popup model for creating or editing a category.
type CategoryForm struct {
	form   *huh.Form
	state  *categoryFormState
	isEdit bool
	editID string
}

// NewCategoryCreateForm returns a blank category creation form.
func NewCategoryCreateForm() CategoryForm {
	st := &categoryFormState{}
	return CategoryForm{
		state: st,
		form:  buildCategoryForm(st, "New Category"),
	}
}

// NewCategoryEditForm returns a pre-populated category edit form.
func NewCategoryEditForm(id, name string, color *string) CategoryForm {
	st := &categoryFormState{name: name}
	if color != nil {
		st.color = *color
	}
	return CategoryForm{
		state:  st,
		isEdit: true,
		editID: id,
		form:   buildCategoryForm(st, fmt.Sprintf("Edit — %s", common.Truncate(name, 30))),
	}
}

func buildCategoryForm(st *categoryFormState, title string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Description("Name").
				Placeholder("e.g. Programming").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}).
				Value(&st.name),

			huh.NewSelect[string]().
				Title("Color (optional)").
				Options(append([]huh.Option[string]{huh.NewOption("None", "")}, colorOptions...)...).
				Value(&st.color),
		),
	).WithTheme(huh.ThemeDracula())
}

// IsEdit reports whether this form is editing an existing category.
func (cf CategoryForm) IsEdit() bool { return cf.isEdit }

// EditID returns the ID of the category being edited (empty for create).
func (cf CategoryForm) EditID() string { return cf.editID }

// Name returns the submitted name.
func (cf CategoryForm) Name() string { return cf.state.name }

// Color returns the submitted color (nil if empty/none).
func (cf CategoryForm) Color() *string {
	if cf.state.color == "" {
		return nil
	}
	c := cf.state.color
	return &c
}

func (cf CategoryForm) Init() tea.Cmd { return cf.form.Init() }

func (cf CategoryForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return cf, func() tea.Msg { return CategoryFormDoneMsg{Cancelled: true} }
		}
	}

	form, cmd := cf.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		cf.form = f
	}
	if cf.form.State == huh.StateCompleted {
		return cf, func() tea.Msg { return CategoryFormDoneMsg{Cancelled: false} }
	}
	return cf, cmd
}

func (cf CategoryForm) View() string {
	return common.PopupStyle.Render(cf.form.View())
}
