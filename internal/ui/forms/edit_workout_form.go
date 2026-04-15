package forms

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

// EditWorkoutForm is a Bubble Tea model for editing an existing workout session.
// It skips the type-select step (type is fixed) and follows the same step order
// as LogWorkoutForm: subtype first, then details, then the row editor.
type EditWorkoutForm struct {
	client       *api.Client
	session      model.WorkoutSession
	exercises    []exerciseOption  // used only for LIFTING
	subtypes     []subtypeOption
	subtypeID    string
	step         int // stepSubtype, stepDetails, or stepRows
	subtypeForm  *huh.Form
	liftEditor   liftEditor
	cardioEditor cardioEditor
	details      detailsState
	detailForm   *huh.Form
}

// NewEditWorkoutForm creates a pre-populated edit form for an existing session.
// exercises must be pre-loaded by the caller (pass nil for cardio sessions).
// subtypes must be pre-loaded by the caller.
func NewEditWorkoutForm(client *api.Client, session model.WorkoutSession, exercises []exerciseOption, subtypes []subtypeOption) *EditWorkoutForm {
	f := &EditWorkoutForm{
		client:    client,
		session:   session,
		exercises: exercises,
		subtypes:  subtypes,
		subtypeID: session.SubtypeID,
		step:      stepSubtype,
	}

	// Pre-populate details from existing session.
	f.details = detailsState{
		date: session.Date.Value,
	}
	if session.Notes != nil {
		f.details.notes = *session.Notes
	}
	if session.Type == model.WorkoutTypeLifting && session.DurationMinutes != nil {
		f.details.durationStr = strconv.Itoa(*session.DurationMinutes)
	}

	// Pre-build the editors so they're ready when we advance to stepRows.
	if session.Type == model.WorkoutTypeLifting {
		f.liftEditor = f.buildLiftEditor()
	} else {
		f.cardioEditor = f.buildCardioEditor()
	}

	f.subtypeForm = f.buildSubtypeForm()
	return f
}

// ExerciseOptions returns the unexported exerciseOption slice so fitness.go
// can pass exercises fetched separately into NewEditWorkoutForm.
func ExerciseOptions(exercises []model.Exercise) []exerciseOption {
	opts := make([]exerciseOption, len(exercises))
	for i, e := range exercises {
		opts[i] = exerciseOption{id: e.ID, name: e.Name}
	}
	return opts
}

func (f *EditWorkoutForm) buildLiftEditor() liftEditor {
	e := newLiftEditor(f.exercises)
	e.rows = nil
	for _, le := range f.session.LiftEntries {
		idx := -1
		for i, ex := range f.exercises {
			if ex.id == le.Exercise.ID {
				idx = i
				break
			}
		}
		row := liftRow{
			exerciseIdx: idx,
			weightStr:   formatWeight(le.WeightLbs),
		}
		e.rows = append(e.rows, row)
	}
	// Always have at least one row.
	if len(e.rows) == 0 {
		e.rows = []liftRow{{exerciseIdx: -1}}
	}
	e.cursor = 0
	return e
}

func (f *EditWorkoutForm) buildCardioEditor() cardioEditor {
	e := newCardioEditor()
	if f.session.CardioEntry == nil || len(f.session.CardioEntry.Segments) == 0 {
		return e
	}
	e.rows = nil
	for _, seg := range f.session.CardioEntry.Segments {
		zoneIdx := 0
		for i, z := range cardioZones {
			if z == seg.Zone {
				zoneIdx = i
				break
			}
		}
		row := cardioSegRow{
			zoneIdx:     zoneIdx,
			durationStr: secondsToMMSS(seg.DurationSeconds),
		}
		if seg.DistanceMiles != nil {
			row.distanceStr = fmt.Sprintf("%.2f", *seg.DistanceMiles)
		}
		if seg.ElevationGainFt != nil {
			row.elevationStr = fmt.Sprintf("%.0f", *seg.ElevationGainFt)
		}
		if seg.Steps != nil {
			row.stepsStr = strconv.Itoa(*seg.Steps)
		}
		e.rows = append(e.rows, row)
	}
	e.cursor = 0
	return e
}

func (f *EditWorkoutForm) buildSubtypeForm() *huh.Form {
	opts := make([]huh.Option[string], len(f.subtypes))
	for i, st := range f.subtypes {
		opts[i] = huh.NewOption(st.name, st.id)
	}
	if len(opts) == 0 {
		opts = []huh.Option[string]{huh.NewOption("(no subtypes — create in Settings)", "")}
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Edit "+common.TitleCase(string(f.session.Type))).
				Description("Choose subtype").
				Options(opts...).
				Value(&f.subtypeID),
		),
	).WithTheme(ActiveTheme)
}

func (f *EditWorkoutForm) buildDetailForm() *huh.Form {
	if f.session.Type == model.WorkoutTypeCardio {
		return huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Date (YYYY-MM-DD)").
					Validate(common.ValidateDate).
					Value(&f.details.date),
				huh.NewText().
					Title("Notes").
					Description("Optional").
					Value(&f.details.notes),
			),
		).WithTheme(ActiveTheme)
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Date (YYYY-MM-DD)").
				Validate(common.ValidateDate).
				Value(&f.details.date),
			huh.NewInput().
				Title("Duration (minutes)").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("duration is required")
					}
					v, err := strconv.Atoi(strings.TrimSpace(s))
					if err != nil {
						return fmt.Errorf("must be a whole number")
					}
					if v <= 0 {
						return fmt.Errorf("must be greater than 0")
					}
					return nil
				}).
				Value(&f.details.durationStr),
			huh.NewText().
				Title("Notes").
				Description("Optional").
				Value(&f.details.notes),
		),
	).WithTheme(ActiveTheme)
}

func (f *EditWorkoutForm) Init() tea.Cmd {
	return f.subtypeForm.Init()
}

func (f *EditWorkoutForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "ctrl+c" {
			return f, func() tea.Msg { return WorkoutLogDoneMsg{Cancelled: true} }
		}
		if key.String() == "esc" {
			if f.step == stepRows {
				// rows → back to details
				f.step = stepDetails
				f.detailForm = f.buildDetailForm()
				return f, f.detailForm.Init()
			}
			if f.step == stepDetails {
				// details → back to subtype
				f.step = stepSubtype
				f.subtypeForm = f.buildSubtypeForm()
				return f, f.subtypeForm.Init()
			}
			// subtype → cancel
			return f, func() tea.Msg { return WorkoutLogDoneMsg{Cancelled: true} }
		}
	}

	switch f.step {
	case stepSubtype:
		var cmd tea.Cmd
		var done bool
		f.subtypeForm, cmd, done = updateHuhForm(f.subtypeForm, msg)
		if done {
			// Advance to details.
			f.step = stepDetails
			f.detailForm = f.buildDetailForm()
			return f, f.detailForm.Init()
		}
		return f, cmd

	case stepDetails:
		var cmd tea.Cmd
		var done bool
		f.detailForm, cmd, done = updateHuhForm(f.detailForm, msg)
		if done {
			// Advance to row editor.
			f.step = stepRows
			return f, nil
		}
		return f, cmd

	case stepRows:
		if key, ok := msg.(tea.KeyMsg); ok {
			// enter submits when not in a text-typing field — same as log form.
			if key.String() == "enter" {
				typing := false
				if f.session.Type == model.WorkoutTypeLifting {
					typing = f.liftEditor.isTyping()
				} else {
					typing = f.cardioEditor.isTyping()
				}
				if !typing {
					return f, f.submit()
				}
			}
			if f.session.Type == model.WorkoutTypeLifting {
				f.liftEditor.update(key)
			} else {
				f.cardioEditor.update(key)
			}
		}
		return f, nil
	}

	return f, nil
}

func (f *EditWorkoutForm) submit() tea.Cmd {
	if f.subtypeID == "" {
		return func() tea.Msg {
			return common.ToastMsg{Text: "No subtype selected — create one in Settings first", IsError: true}
		}
	}
	return func() tea.Msg {
		var err error
		if f.session.Type == model.WorkoutTypeLifting {
			err = f.submitLift()
		} else {
			err = f.submitCardio()
		}
		if err != nil {
			return common.ToastMsg{Text: "Failed to update workout: " + err.Error(), IsError: true}
		}
		return WorkoutLogDoneMsg{Cancelled: false}
	}
}

func (f *EditWorkoutForm) submitLift() error {
	dur, err := strconv.Atoi(strings.TrimSpace(f.details.durationStr))
	if err != nil || dur <= 0 {
		return fmt.Errorf("duration is required and must be > 0")
	}
	var notes *string
	if f.details.notes != "" {
		n := f.details.notes
		notes = &n
	}
	date := f.details.date
	return f.client.UpdateLiftSession(api.UpdateLiftSessionInput{
		SessionID:       f.session.ID,
		Date:            &date,
		DurationMinutes: dur,
		Notes:           notes,
		SubtypeID:       f.subtypeID,
		Lifts:           f.liftEditor.toLiftEntries(),
	})
}

func (f *EditWorkoutForm) submitCardio() error {
	segments := f.cardioEditor.toSegments()
	if len(segments) == 0 {
		return fmt.Errorf("each segment needs a duration (mm:ss)")
	}
	var notes *string
	if f.details.notes != "" {
		n := f.details.notes
		notes = &n
	}
	date := f.details.date
	return f.client.UpdateCardioSession(api.UpdateCardioSessionInput{
		SessionID: f.session.ID,
		Date:      &date,
		Notes:     notes,
		SubtypeID: f.subtypeID,
		Segments:  segments,
	})
}

func (f *EditWorkoutForm) View() string {
	var b strings.Builder

	switch f.step {
	case stepSubtype:
		return common.PopupStyle.Render(f.subtypeForm.View())

	case stepDetails:
		return common.PopupStyle.Render(f.detailForm.View())

	case stepRows:
		popupW := 76
		title := "Edit " + common.TitleCase(string(f.session.Type))
		b.WriteString(common.SelectedStyle.Render(title) + "\n\n")
		if f.session.Type == model.WorkoutTypeLifting {
			b.WriteString(common.DimStyle.Render("  Exercises (optional)") + "\n\n")
			b.WriteString(f.liftEditor.view(popupW))
		} else {
			b.WriteString(common.DimStyle.Render("  Segments") + "\n\n")
			b.WriteString(f.cardioEditor.view(popupW))
		}
		b.WriteString("\n")
		b.WriteString(common.DimStyle.Render(
			"  space: cycle  tab: next col  j/k: rows  a: add  D: delete  enter: submit  esc: back",
		))
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(common.ColorPrimary).
			Padding(1, 2).
			Width(popupW).
			Render(b.String())
	}

	return ""
}

// formatWeight formats a float weight as a clean string (no trailing ".0").
func formatWeight(w float64) string {
	if w == float64(int(w)) {
		return strconv.Itoa(int(w))
	}
	return fmt.Sprintf("%.1f", w)
}

// secondsToMMSS converts total seconds to "MM:SS" string.
func secondsToMMSS(secs int) string {
	if secs <= 0 {
		return ""
	}
	m := secs / 60
	s := secs % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
