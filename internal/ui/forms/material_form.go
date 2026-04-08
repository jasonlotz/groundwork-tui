package forms

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

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
		form:   buildMaterialForm(st, skills, types, fmt.Sprintf("Edit — %s", common.Truncate(mat.Name, 28))),
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
	return common.ValidateDate(normalizeDateInput(s))
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
	return common.PopupStyle.Render(mf.form.View())
}
