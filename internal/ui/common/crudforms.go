package common

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/jasonlotz/groundwork-tui/internal/model"
)

// colorOptions are the 10 bg-*-300 Tailwind classes used in the color picker.
var colorOptions = []huh.Option[string]{
	huh.NewOption("● Slate", "bg-slate-300"),
	huh.NewOption("● Red", "bg-red-300"),
	huh.NewOption("● Orange", "bg-orange-300"),
	huh.NewOption("● Amber", "bg-amber-300"),
	huh.NewOption("● Green", "bg-green-300"),
	huh.NewOption("● Teal", "bg-teal-300"),
	huh.NewOption("● Blue", "bg-blue-300"),
	huh.NewOption("● Indigo", "bg-indigo-300"),
	huh.NewOption("● Purple", "bg-purple-300"),
	huh.NewOption("● Pink", "bg-pink-300"),
}

// ────────────────────────────────────────────────────────────────
// Done messages
// ────────────────────────────────────────────────────────────────

// CategoryFormDoneMsg is sent when the category create/edit form completes.
type CategoryFormDoneMsg struct{ Cancelled bool }

// SkillFormDoneMsg is sent when the skill create/edit form completes.
type SkillFormDoneMsg struct{ Cancelled bool }

// ConfirmDoneMsg is sent when a confirm dialog completes.
type ConfirmDoneMsg struct {
	Confirmed bool
	// Tag identifies which action triggered the confirm (e.g. "archive", "delete").
	Tag string
}

// ────────────────────────────────────────────────────────────────
// CategoryForm
// ────────────────────────────────────────────────────────────────

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
		form:   buildCategoryForm(st, fmt.Sprintf("Edit — %s", Truncate(name, 30))),
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
	return PopupStyle.Render(cf.form.View())
}

// ────────────────────────────────────────────────────────────────
// SkillForm
// ────────────────────────────────────────────────────────────────

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
		form:       buildSkillForm(st, fmt.Sprintf("Edit — %s", Truncate(name, 30))),
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
	return PopupStyle.Render(sf.form.View())
}

// ────────────────────────────────────────────────────────────────
// ConfirmForm
// ────────────────────────────────────────────────────────────────

// ConfirmForm is a small yes/no popup for destructive actions.
type ConfirmForm struct {
	form      *huh.Form
	confirmed *bool
	tag       string
}

// NewConfirmForm returns a confirm dialog with the given prompt and tag.
func NewConfirmForm(title, description, tag string) ConfirmForm {
	confirmed := false
	cf := ConfirmForm{tag: tag, confirmed: &confirmed}
	cf.form = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Affirmative("Yes").
				Negative("No").
				Value(cf.confirmed),
		),
	).WithTheme(huh.ThemeDracula())
	return cf
}

func (cf ConfirmForm) Init() tea.Cmd { return cf.form.Init() }

func (cf ConfirmForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return cf, func() tea.Msg { return ConfirmDoneMsg{Confirmed: false, Tag: cf.tag} }
		}
	}

	form, cmd := cf.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		cf.form = f
	}
	if cf.form.State == huh.StateCompleted {
		tag := cf.tag
		confirmed := *cf.confirmed
		return cf, func() tea.Msg { return ConfirmDoneMsg{Confirmed: confirmed, Tag: tag} }
	}
	return cf, cmd
}

func (cf ConfirmForm) View() string {
	return PopupStyle.Render(cf.form.View())
}

// ────────────────────────────────────────────────────────────────
// MaterialForm
// ────────────────────────────────────────────────────────────────

// MaterialFormDoneMsg is sent when the material create/edit form completes.
type MaterialFormDoneMsg struct{ Cancelled bool }

// unitTypeOptions lists all UnitType values as huh select options.
var unitTypeOptions = []huh.Option[string]{
	huh.NewOption("Hours", "HOURS"),
	huh.NewOption("Pages", "PAGES"),
	huh.NewOption("Chapters", "CHAPTERS"),
	huh.NewOption("Sections", "SECTIONS"),
	huh.NewOption("Modules", "MODULES"),
	huh.NewOption("Lessons", "LESSONS"),
	huh.NewOption("Videos", "VIDEOS"),
	huh.NewOption("Episodes", "EPISODES"),
}

// materialFormState holds all bound form values behind stable pointers.
type materialFormState struct {
	name          string
	skillID       string
	typeID        string
	unitType      string
	totalUnitsStr string
	url           string
	startDate     string
	completedDate string
	weeklyGoalStr string
	notes         string
}

// MaterialFormResult carries the validated values after submission.
type MaterialFormResult struct {
	Name          string
	SkillID       string
	TypeID        string
	UnitType      string
	TotalUnits    float64
	URL           *string
	StartDate     *string
	CompletedDate *string
	WeeklyGoal    *int
	Notes         *string
}

// MaterialForm is a popup model for creating or editing a material.
type MaterialForm struct {
	form   *huh.Form
	state  *materialFormState
	isEdit bool
	editID string
}

// NewMaterialCreateForm returns a blank material creation form.
// skills and types are loaded by the caller before opening the overlay.
func NewMaterialCreateForm(skills []model.Skill, types []model.MaterialType) MaterialForm {
	st := &materialFormState{unitType: "HOURS"}
	return MaterialForm{
		state: st,
		form:  buildMaterialForm(st, skills, types, "New Material"),
	}
}

// NewMaterialEditForm returns a pre-populated material edit form.
func NewMaterialEditForm(
	id string,
	mat model.Material,
	skills []model.Skill,
	types []model.MaterialType,
) MaterialForm {
	st := &materialFormState{
		name:          mat.Name,
		skillID:       mat.SkillID,
		typeID:        mat.MaterialType.ID,
		unitType:      string(mat.UnitType),
		totalUnitsStr: strconv.FormatFloat(mat.TotalUnits, 'f', -1, 64),
	}
	if mat.URL != nil {
		st.url = *mat.URL
	}
	if mat.StartDate != nil {
		st.startDate = mat.StartDate.Value
	}
	if mat.CompletedDate != nil {
		st.completedDate = mat.CompletedDate.Value
	}
	if mat.WeeklyUnitGoal != nil {
		st.weeklyGoalStr = strconv.Itoa(*mat.WeeklyUnitGoal)
	}
	return MaterialForm{
		state:  st,
		isEdit: true,
		editID: id,
		form:   buildMaterialForm(st, skills, types, fmt.Sprintf("Edit — %s", Truncate(mat.Name, 28))),
	}
}

func buildMaterialForm(st *materialFormState, skills []model.Skill, types []model.MaterialType, title string) *huh.Form {
	// Build skill options grouped by category name for readability.
	skillOpts := make([]huh.Option[string], 0, len(skills))
	for _, s := range skills {
		label := s.Name
		if s.Category.Name != "" {
			label = s.Category.Name + " › " + s.Name
		}
		skillOpts = append(skillOpts, huh.NewOption(label, s.ID))
	}

	typeOpts := make([]huh.Option[string], 0, len(types))
	for _, t := range types {
		typeOpts = append(typeOpts, huh.NewOption(t.Name, t.ID))
	}

	return huh.NewForm(
		// Group 1: core required fields
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Description("Name").
				Placeholder("e.g. The Go Programming Language").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}).
				Value(&st.name),

			huh.NewSelect[string]().
				Title("Skill").
				Options(skillOpts...).
				Value(&st.skillID),

			huh.NewSelect[string]().
				Title("Type").
				Options(typeOpts...).
				Value(&st.typeID),
		),
		// Group 2: units
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Unit type").
				Options(unitTypeOptions...).
				Value(&st.unitType),

			huh.NewInput().
				Title("Total units").
				Placeholder("e.g. 12").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("total units is required")
					}
					v, err := strconv.ParseFloat(s, 64)
					if err != nil {
						return fmt.Errorf("must be a number")
					}
					if v <= 0 {
						return fmt.Errorf("must be greater than 0")
					}
					return nil
				}).
				Value(&st.totalUnitsStr),
		),
		// Group 3: optional fields
		huh.NewGroup(
			huh.NewInput().
				Title("URL (optional)").
				Placeholder("https://...").
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					if len(s) > 2048 {
						return fmt.Errorf("URL is too long")
					}
					return nil
				}).
				Value(&st.url),

			huh.NewInput().
				Title("Start date (optional)").
				Description("YYYYMMDD or YYYY-MM-DD").
				Placeholder("e.g. 20250101").
				Validate(validateOptionalDate).
				Value(&st.startDate),

			huh.NewInput().
				Title("Completed date (optional)").
				Description("YYYYMMDD or YYYY-MM-DD").
				Placeholder("e.g. 20250630").
				Validate(validateOptionalDate).
				Value(&st.completedDate),

			huh.NewInput().
				Title("Weekly goal (optional)").
				Description("Units per week").
				Placeholder("e.g. 3").
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					v, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("must be a whole number")
					}
					if v < 1 {
						return fmt.Errorf("must be at least 1")
					}
					return nil
				}).
				Value(&st.weeklyGoalStr),

			huh.NewText().
				Title("Notes (optional)").
				Value(&st.notes),
		),
	).WithTheme(huh.ThemeDracula())
}

// validateOptionalDate accepts empty string or a valid YYYY-MM-DD.
func validateOptionalDate(s string) error {
	if s == "" {
		return nil
	}
	return ValidateDate(normalizeDateInput(s))
}

// normalizeDateInput converts a bare 8-digit string (YYYYMMDD) to YYYY-MM-DD.
// Any other input is returned unchanged.
func normalizeDateInput(s string) string {
	if len(s) == 8 {
		if _, err := strconv.Atoi(s); err == nil {
			return s[:4] + "-" + s[4:6] + "-" + s[6:]
		}
	}
	return s
}

// IsEdit reports whether this form is editing an existing material.
func (mf MaterialForm) IsEdit() bool { return mf.isEdit }

// EditID returns the ID of the material being edited (empty for create).
func (mf MaterialForm) EditID() string { return mf.editID }

// Result parses and returns the form values. Should only be called after the form completes.
func (mf MaterialForm) Result() MaterialFormResult {
	st := mf.state
	r := MaterialFormResult{
		Name:     st.name,
		SkillID:  st.skillID,
		TypeID:   st.typeID,
		UnitType: st.unitType,
	}
	r.TotalUnits, _ = strconv.ParseFloat(st.totalUnitsStr, 64)

	if st.url != "" {
		u := st.url
		r.URL = &u
	}
	if st.startDate != "" {
		d := normalizeDateInput(st.startDate)
		r.StartDate = &d
	}
	if st.completedDate != "" {
		d := normalizeDateInput(st.completedDate)
		r.CompletedDate = &d
	}
	if st.weeklyGoalStr != "" {
		if v, err := strconv.Atoi(st.weeklyGoalStr); err == nil {
			r.WeeklyGoal = &v
		}
	}
	if st.notes != "" {
		n := st.notes
		r.Notes = &n
	}
	return r
}

func (mf MaterialForm) Init() tea.Cmd { return mf.form.Init() }

func (mf MaterialForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return mf, func() tea.Msg { return MaterialFormDoneMsg{Cancelled: true} }
		}
	}

	form, cmd := mf.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		mf.form = f
	}
	if mf.form.State == huh.StateCompleted {
		return mf, func() tea.Msg { return MaterialFormDoneMsg{Cancelled: false} }
	}
	return mf, cmd
}

func (mf MaterialForm) View() string {
	return PopupStyle.Render(mf.form.View())
}
