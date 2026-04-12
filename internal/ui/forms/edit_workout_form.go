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
// as LogWorkoutForm: details first, then the row editor.
type EditWorkoutForm struct {
	client     *api.Client
	session    model.WorkoutSession
	exercises  []exerciseOption // used only for LIFTING
	step       int              // stepDetails or stepRows
	liftEditor liftEditor
	runEditor  runEditor
	details    detailsState
	detailForm *huh.Form
}

// NewEditWorkoutForm creates a pre-populated edit form for an existing session.
// exercises must be pre-loaded by the caller (pass nil for running sessions).
func NewEditWorkoutForm(client *api.Client, session model.WorkoutSession, exercises []exerciseOption) *EditWorkoutForm {
	f := &EditWorkoutForm{
		client:    client,
		session:   session,
		exercises: exercises,
		step:      stepDetails,
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
		f.runEditor = f.buildRunEditor()
	}

	f.detailForm = f.buildDetailForm()
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

func (f *EditWorkoutForm) buildRunEditor() runEditor {
	e := newRunEditor()
	if f.session.RunEntry == nil || len(f.session.RunEntry.Segments) == 0 {
		return e
	}
	e.rows = nil
	for _, seg := range f.session.RunEntry.Segments {
		zoneIdx := 0
		for i, z := range runZones {
			if z == seg.Zone {
				zoneIdx = i
				break
			}
		}
		e.rows = append(e.rows, runSegRow{
			zoneIdx:     zoneIdx,
			distanceStr: fmt.Sprintf("%.2f", seg.DistanceMiles),
			durationStr: secondsToMMSS(seg.DurationSeconds),
		})
	}
	e.cursor = 0
	return e
}

func (f *EditWorkoutForm) buildDetailForm() *huh.Form {
	if f.session.Type == model.WorkoutTypeRunning {
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
				Description("Optional").
				Value(&f.details.durationStr),
			huh.NewText().
				Title("Notes").
				Description("Optional").
				Value(&f.details.notes),
		),
	).WithTheme(ActiveTheme)
}

func (f *EditWorkoutForm) Init() tea.Cmd {
	return f.detailForm.Init()
}

func (f *EditWorkoutForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "ctrl+c" {
			return f, func() tea.Msg { return WorkoutLogDoneMsg{Cancelled: true} }
		}
		if key.String() == "esc" {
			if f.step == stepRows {
				// rows → back to details (mirrors log form: esc on rows → details)
				f.step = stepDetails
				f.detailForm = f.buildDetailForm()
				return f, f.detailForm.Init()
			}
			// details → cancel (no type step in edit form)
			return f, func() tea.Msg { return WorkoutLogDoneMsg{Cancelled: true} }
		}
	}

	switch f.step {
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
					typing = f.runEditor.isTyping()
				}
				if !typing {
					return f, f.submit()
				}
			}
			if f.session.Type == model.WorkoutTypeLifting {
				f.liftEditor.update(key)
			} else {
				f.runEditor.update(key)
			}
		}
		return f, nil
	}

	return f, nil
}

func (f *EditWorkoutForm) submit() tea.Cmd {
	return func() tea.Msg {
		var err error
		if f.session.Type == model.WorkoutTypeLifting {
			err = f.submitLift()
		} else {
			err = f.submitRun()
		}
		if err != nil {
			return common.ToastMsg{Text: "Failed to update workout: " + err.Error(), IsError: true}
		}
		return WorkoutLogDoneMsg{Cancelled: false}
	}
}

func (f *EditWorkoutForm) submitLift() error {
	var durPtr *int
	if f.details.durationStr != "" {
		d, err := strconv.Atoi(strings.TrimSpace(f.details.durationStr))
		if err == nil && d > 0 {
			durPtr = &d
		}
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
		DurationMinutes: durPtr,
		Notes:           notes,
		Lifts:           f.liftEditor.toLiftEntries(),
	})
}

func (f *EditWorkoutForm) submitRun() error {
	segments := f.runEditor.toSegments()
	if len(segments) == 0 {
		return fmt.Errorf("each segment needs a distance (mi) and duration (mm:ss)")
	}
	var notes *string
	if f.details.notes != "" {
		n := f.details.notes
		notes = &n
	}
	date := f.details.date
	return f.client.UpdateRunSession(api.UpdateRunSessionInput{
		SessionID: f.session.ID,
		Date:      &date,
		Notes:     notes,
		Segments:  segments,
	})
}

func (f *EditWorkoutForm) View() string {
	var b strings.Builder

	switch f.step {
	case stepDetails:
		return common.PopupStyle.Render(f.detailForm.View())

	case stepRows:
		popupW := 62
		title := "Edit " + capitalize(string(f.session.Type))
		b.WriteString(common.SelectedStyle.Render(title) + "\n\n")
		if f.session.Type == model.WorkoutTypeLifting {
			b.WriteString(common.DimStyle.Render("  Exercises (optional)") + "\n\n")
			b.WriteString(f.liftEditor.view(popupW))
		} else {
			b.WriteString(common.DimStyle.Render("  Segments") + "\n\n")
			b.WriteString(f.runEditor.view(popupW))
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
