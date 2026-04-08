package forms

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

// SkillFormDoneMsg is sent when the skill create/edit form completes.
type SkillFormDoneMsg struct{ Cancelled bool }

// skillFormState holds bound form values behind stable pointers.
type skillFormState struct {
	name       string
	color      string
	categoryID string // only used when picking category in create-from-flat-list flow
}

// SkillForm is a popup model for creating or editing a skill.
type SkillForm struct {
	form         *huh.Form
	state        *skillFormState
	categoryID   string // fixed when creating from category detail; empty when using picker
	hasCatPicker bool   // true when the form includes a category picker
	isEdit       bool
	editID       string
}

// NewSkillCreateForm returns a blank skill creation form for the given category.
func NewSkillCreateForm(categoryID string) SkillForm {
	st := &skillFormState{}
	return SkillForm{
		state:      st,
		categoryID: categoryID,
		form:       buildSkillForm(st, "New Skill"),
	}
}

// NewSkillCreateFormWithCategories returns a skill creation form that includes
// a category picker. Used from the flat skills list where there is no fixed category.
func NewSkillCreateFormWithCategories(categories []model.Category) SkillForm {
	st := &skillFormState{}
	return SkillForm{
		state:        st,
		hasCatPicker: true,
		form:         buildSkillFormWithCatPicker(st, categories, "New Skill"),
	}
}

// NewSkillEditForm returns a pre-populated skill edit form.
func NewSkillEditForm(id, name, categoryID string, color *string) SkillForm {
	st := &skillFormState{name: name}
	if color != nil {
		st.color = *color
	}
	return SkillForm{
		state:      st,
		isEdit:     true,
		editID:     id,
		categoryID: categoryID,
		form:       buildSkillForm(st, fmt.Sprintf("Edit — %s", common.Truncate(name, 30))),
	}
}

func buildSkillForm(st *skillFormState, title string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Description("Name").
				Placeholder("e.g. Go").
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

func buildSkillFormWithCatPicker(st *skillFormState, categories []model.Category, title string) *huh.Form {
	catOptions := make([]huh.Option[string], len(categories))
	for i, c := range categories {
		catOptions[i] = huh.NewOption(c.Name, c.ID)
	}
	if len(catOptions) > 0 {
		st.categoryID = catOptions[0].Value
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Description("Name").
				Placeholder("e.g. Go").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}).
				Value(&st.name),

			huh.NewSelect[string]().
				Title("Category").
				Options(catOptions...).
				Value(&st.categoryID),

			huh.NewSelect[string]().
				Title("Color (optional)").
				Options(append([]huh.Option[string]{huh.NewOption("None", "")}, colorOptions...)...).
				Value(&st.color),
		),
	).WithTheme(huh.ThemeDracula())
}

// IsEdit reports whether this form is editing an existing skill.
func (sf SkillForm) IsEdit() bool { return sf.isEdit }

// EditID returns the ID of the skill being edited (empty for create).
func (sf SkillForm) EditID() string { return sf.editID }

// CategoryID returns the category ID for this skill.
// When the form has a category picker, this returns the user's selection.
func (sf SkillForm) CategoryID() string {
	if sf.hasCatPicker {
		return sf.state.categoryID
	}
	return sf.categoryID
}

// Name returns the submitted name.
func (sf SkillForm) Name() string { return sf.state.name }

// Color returns the submitted color (nil if empty/none).
func (sf SkillForm) Color() *string {
	if sf.state.color == "" {
		return nil
	}
	c := sf.state.color
	return &c
}

func (sf SkillForm) Init() tea.Cmd { return sf.form.Init() }

func (sf SkillForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return sf, func() tea.Msg { return SkillFormDoneMsg{Cancelled: true} }
		}
	}

	form, cmd := sf.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		sf.form = f
	}
	if sf.form.State == huh.StateCompleted {
		return sf, func() tea.Msg { return SkillFormDoneMsg{Cancelled: false} }
	}
	return sf, cmd
}

func (sf SkillForm) View() string {
	return common.PopupStyle.Render(sf.form.View())
}
