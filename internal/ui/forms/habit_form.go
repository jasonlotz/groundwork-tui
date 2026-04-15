package forms

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

// HabitFormDoneMsg is sent when the habit create/edit form completes.
type HabitFormDoneMsg struct{ Cancelled bool }

// habitFormState holds bound form values behind stable pointers.
type habitFormState struct {
	name      string
	startDate string
	endDate   string
}

// HabitForm is a popup model for creating or editing a habit.
type HabitForm struct {
	form   *huh.Form
	state  *habitFormState
	isEdit bool
	editID string
}

// NewHabitCreateForm returns a blank habit creation form.
func NewHabitCreateForm() HabitForm {
	st := &habitFormState{
		startDate: time.Now().Format("2006-01-02"),
	}
	return HabitForm{
		state: st,
		form:  buildHabitForm(st, "New Habit"),
	}
}

// NewHabitEditForm returns a pre-populated habit edit form.
func NewHabitEditForm(id, name, startDate string, endDate *string) HabitForm {
	st := &habitFormState{
		name:      name,
		startDate: startDate,
	}
	if endDate != nil {
		st.endDate = *endDate
	}
	return HabitForm{
		state:  st,
		isEdit: true,
		editID: id,
		form:   buildHabitForm(st, fmt.Sprintf("Edit — %s", common.Truncate(name, 30))),
	}
}

func buildHabitForm(st *habitFormState, title string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Description("Name").
				Placeholder("e.g. Drink Water").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}).
				Value(&st.name),

			huh.NewInput().
				Title("Start Date").
				Description("YYYY-MM-DD").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("start date is required")
					}
					if _, err := time.Parse("2006-01-02", s); err != nil {
						return fmt.Errorf("invalid date format")
					}
					return nil
				}).
				Value(&st.startDate),

			huh.NewInput().
				Title("End Date (optional)").
				Description("YYYY-MM-DD — leave blank for ongoing").
				Value(&st.endDate),
		),
	).WithTheme(ActiveTheme)
}

// IsEdit reports whether this form is editing an existing habit.
func (hf HabitForm) IsEdit() bool { return hf.isEdit }

// EditID returns the ID of the habit being edited (empty for create).
func (hf HabitForm) EditID() string { return hf.editID }

// Name returns the submitted name.
func (hf HabitForm) Name() string { return hf.state.name }

// StartDate returns the submitted start date.
func (hf HabitForm) StartDate() string { return hf.state.startDate }

// EndDate returns the submitted end date (nil if empty).
func (hf HabitForm) EndDate() *string {
	if hf.state.endDate == "" {
		return nil
	}
	s := hf.state.endDate
	return &s
}

func (hf HabitForm) Init() tea.Cmd { return hf.form.Init() }

func (hf HabitForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return hf, func() tea.Msg { return HabitFormDoneMsg{Cancelled: true} }
		}
	}

	var cmd tea.Cmd
	var done bool
	hf.form, cmd, done = updateHuhForm(hf.form, msg)
	if done {
		return hf, func() tea.Msg { return HabitFormDoneMsg{Cancelled: false} }
	}
	return hf, cmd
}

func (hf HabitForm) View() string {
	return common.PopupStyle.Render(hf.form.View())
}
