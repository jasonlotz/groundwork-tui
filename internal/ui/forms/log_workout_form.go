package forms

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

// WorkoutLogDoneMsg is sent when the log-workout form completes or is cancelled.
type WorkoutLogDoneMsg struct{ Cancelled bool }

// ── exercise option ───────────────────────────────────────────────────────────

type exerciseOption struct {
	id   string
	name string
}

// ── lift row editor ───────────────────────────────────────────────────────────

// liftRow is one exercise+weight entry in the lift row editor.
type liftRow struct {
	exerciseIdx int    // index into exercises slice; -1 = none selected
	weightStr   string // raw text input
}

// liftEditor is the custom multi-row exercise editor used in step 1 for lifting.
type liftEditor struct {
	exercises []exerciseOption
	rows      []liftRow
	cursor    int // which row is active
	col       int // 0 = exercise, 1 = weight
	typing    bool
	typingBuf string
}

func newLiftEditor(exercises []exerciseOption) liftEditor {
	e := liftEditor{exercises: exercises}
	e.rows = []liftRow{{exerciseIdx: -1}}
	return e
}

func (e *liftEditor) addRow() {
	if len(e.rows) < 12 {
		e.rows = append(e.rows, liftRow{exerciseIdx: -1})
		e.cursor = len(e.rows) - 1
		e.col = 0
		e.typing = false
	}
}

func (e *liftEditor) deleteRow() {
	if len(e.rows) <= 1 {
		return
	}
	e.rows = append(e.rows[:e.cursor], e.rows[e.cursor+1:]...)
	if e.cursor >= len(e.rows) {
		e.cursor = len(e.rows) - 1
	}
	e.typing = false
}

func (e *liftEditor) update(msg tea.KeyMsg) {
	if e.col == 1 && e.typing {
		// Text entry mode for weight field.
		switch msg.String() {
		case "enter", "tab", "down":
			e.typing = false
			if msg.String() != "tab" {
				e.col = 0
				if msg.String() == "down" || msg.String() == "enter" {
					e.cursor++
					if e.cursor >= len(e.rows) {
						e.addRow()
					}
				}
			} else {
				// tab moves back to exercise col
				e.col = 0
			}
		case "esc":
			e.typing = false
		case "backspace":
			if len(e.typingBuf) > 0 {
				e.typingBuf = e.typingBuf[:len(e.typingBuf)-1]
				e.rows[e.cursor].weightStr = e.typingBuf
			}
		default:
			if len(msg.String()) == 1 {
				e.typingBuf += msg.String()
				e.rows[e.cursor].weightStr = e.typingBuf
			}
		}
		return
	}

	switch msg.String() {
	case "j", "down":
		e.col = 0
		e.cursor++
		if e.cursor >= len(e.rows) {
			e.cursor = len(e.rows) - 1
		}
	case "k", "up":
		e.col = 0
		e.cursor--
		if e.cursor < 0 {
			e.cursor = 0
		}
	case "left", "h":
		e.col = 0
		e.typing = false
	case "right", "l":
		e.col = 1
		e.typing = true
		e.typingBuf = e.rows[e.cursor].weightStr
	case "tab":
		if e.col == 0 {
			e.col = 1
			e.typing = true
			e.typingBuf = e.rows[e.cursor].weightStr
		} else {
			e.col = 0
			e.typing = false
		}
	case "enter":
		if e.col == 0 {
			// Cycle to next exercise.
			e.rows[e.cursor].exerciseIdx++
			if e.rows[e.cursor].exerciseIdx >= len(e.exercises) {
				e.rows[e.cursor].exerciseIdx = -1
			}
		}
	case " ":
		// Space also cycles exercise when on exercise col.
		if e.col == 0 {
			e.rows[e.cursor].exerciseIdx++
			if e.rows[e.cursor].exerciseIdx >= len(e.exercises) {
				e.rows[e.cursor].exerciseIdx = -1
			}
		}
	case "+", "a":
		e.addRow()
	case "D":
		e.deleteRow()
	}
}

func (e *liftEditor) view(width int) string {
	var b strings.Builder
	header := fmt.Sprintf("  %-30s  %s", "Exercise", "Weight (lbs)")
	b.WriteString(common.TableHeaderStyle.Render(header) + "\n")
	b.WriteString(common.MutedStyle.Render("  "+strings.Repeat("─", width-4)) + "\n")

	for i, row := range e.rows {
		exName := "— none —"
		if row.exerciseIdx >= 0 && row.exerciseIdx < len(e.exercises) {
			exName = e.exercises[row.exerciseIdx].name
		}
		weight := row.weightStr
		if weight == "" {
			weight = "—"
		}

		exCell := fmt.Sprintf("%-30s", common.Truncate(exName, 30))
		wtCell := fmt.Sprintf("%-12s", weight)

		var line string
		if i == e.cursor {
			cursor := common.SelectedStyle.Render(">")
			if e.col == 0 {
				line = cursor + " " + common.SelectedStyle.Render(exCell) + "  " + common.TableCellStyle.Render(wtCell)
			} else {
				// Weight column active — show cursor in weight
				wtDisplay := wtCell
				if e.typing {
					wtDisplay = fmt.Sprintf("%-12s", e.typingBuf+"_")
				}
				line = cursor + " " + common.TableCellStyle.Render(exCell) + "  " + common.SelectedStyle.Render(wtDisplay)
			}
		} else {
			line = "  " + common.TableCellStyle.Render(exCell) + "  " + common.TableCellStyle.Render(wtCell)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (e *liftEditor) toLiftEntries() []api.LiftEntry {
	var out []api.LiftEntry
	for _, row := range e.rows {
		if row.exerciseIdx < 0 || row.exerciseIdx >= len(e.exercises) {
			continue
		}
		w, err := strconv.ParseFloat(strings.TrimSpace(row.weightStr), 64)
		if err != nil || w <= 0 {
			w = 0
		}
		out = append(out, api.LiftEntry{
			ExerciseID: e.exercises[row.exerciseIdx].id,
			WeightLbs:  w,
		})
	}
	return out
}

// ── run segment editor ────────────────────────────────────────────────────────

var runZones = []string{"FREE", "Z1", "Z2", "Z3", "Z4", "Z5"}
var runZoneLabels = []string{"Free Run", "Zone 1", "Zone 2", "Zone 3", "Zone 4", "Zone 5"}

type runSegRow struct {
	zoneIdx     int // index into runZones
	distanceStr string
	durationStr string // mm:ss or plain minutes
}

// runCol constants
const (
	runColZone     = 0
	runColDistance = 1
	runColDuration = 2
)

type runEditor struct {
	rows      []runSegRow
	cursor    int
	col       int
	typing    bool
	typingBuf string
}

func newRunEditor() runEditor {
	return runEditor{rows: []runSegRow{{zoneIdx: 0}}}
}

func (e *runEditor) addRow() {
	if len(e.rows) < 12 {
		e.rows = append(e.rows, runSegRow{zoneIdx: 0})
		e.cursor = len(e.rows) - 1
		e.col = 0
		e.typing = false
	}
}

func (e *runEditor) deleteRow() {
	if len(e.rows) <= 1 {
		return
	}
	e.rows = append(e.rows[:e.cursor], e.rows[e.cursor+1:]...)
	if e.cursor >= len(e.rows) {
		e.cursor = len(e.rows) - 1
	}
	e.typing = false
}

func (e *runEditor) currentFieldStr() string {
	row := e.rows[e.cursor]
	switch e.col {
	case runColDistance:
		return row.distanceStr
	case runColDuration:
		return row.durationStr
	}
	return ""
}

func (e *runEditor) setCurrentFieldStr(s string) {
	switch e.col {
	case runColDistance:
		e.rows[e.cursor].distanceStr = s
	case runColDuration:
		e.rows[e.cursor].durationStr = s
	}
}

func (e *runEditor) isTextCol() bool {
	return e.col == runColDistance || e.col == runColDuration
}

func (e *runEditor) update(msg tea.KeyMsg) {
	if e.typing && e.isTextCol() {
		switch msg.String() {
		case "enter", "down":
			e.typing = false
			e.col = 0
			e.cursor++
			if e.cursor >= len(e.rows) {
				e.addRow()
			}
		case "tab":
			e.typing = false
			e.col++
			if e.col > runColDuration {
				e.col = 0
			} else {
				e.typing = e.isTextCol()
				if e.typing {
					e.typingBuf = e.currentFieldStr()
				}
			}
		case "esc":
			e.typing = false
		case "backspace":
			if len(e.typingBuf) > 0 {
				e.typingBuf = e.typingBuf[:len(e.typingBuf)-1]
				e.setCurrentFieldStr(e.typingBuf)
			}
		default:
			if len(msg.String()) == 1 {
				e.typingBuf += msg.String()
				e.setCurrentFieldStr(e.typingBuf)
			}
		}
		return
	}

	switch msg.String() {
	case "j", "down":
		e.col = 0
		e.cursor++
		if e.cursor >= len(e.rows) {
			e.cursor = len(e.rows) - 1
		}
	case "k", "up":
		e.col = 0
		e.cursor--
		if e.cursor < 0 {
			e.cursor = 0
		}
	case "tab":
		e.col++
		if e.col > runColDuration {
			e.col = 0
		}
		e.typing = e.isTextCol()
		if e.typing {
			e.typingBuf = e.currentFieldStr()
		}
	case "left", "h":
		if e.col > 0 {
			e.col--
			e.typing = false
		}
	case "right", "l":
		if e.col < runColDuration {
			e.col++
			e.typing = e.isTextCol()
			if e.typing {
				e.typingBuf = e.currentFieldStr()
			}
		}
	case "enter", " ":
		if e.col == runColZone {
			// Cycle zone.
			e.rows[e.cursor].zoneIdx++
			if e.rows[e.cursor].zoneIdx >= len(runZones) {
				e.rows[e.cursor].zoneIdx = 0
			}
		} else {
			e.typing = true
			e.typingBuf = e.currentFieldStr()
		}
	case "+", "a":
		e.addRow()
	case "D":
		e.deleteRow()
	}
}

func (e *runEditor) view(width int) string {
	var b strings.Builder
	header := fmt.Sprintf("  %-10s  %-12s  %s", "Zone", "Distance (mi)", "Duration (mm:ss)")
	b.WriteString(common.TableHeaderStyle.Render(header) + "\n")
	b.WriteString(common.MutedStyle.Render("  "+strings.Repeat("─", width-4)) + "\n")

	for i, row := range e.rows {
		zone := runZoneLabels[row.zoneIdx]
		dist := row.distanceStr
		if dist == "" {
			dist = "—"
		}
		dur := row.durationStr
		if dur == "" {
			dur = "—"
		}

		zoneCell := fmt.Sprintf("%-10s", zone)
		distCell := fmt.Sprintf("%-14s", dist)
		durCell := fmt.Sprintf("%-16s", dur)

		if i == e.cursor {
			cursor := common.SelectedStyle.Render(">")
			renderCell := func(col int, text string) string {
				if e.col == col {
					if e.typing && col != runColZone {
						return common.SelectedStyle.Render(fmt.Sprintf("%-*s", len(text), e.typingBuf+"_"))
					}
					return common.SelectedStyle.Render(text)
				}
				return common.TableCellStyle.Render(text)
			}
			line := cursor + " " + renderCell(runColZone, zoneCell) + "  " +
				renderCell(runColDistance, distCell) + "  " +
				renderCell(runColDuration, durCell)
			b.WriteString(line + "\n")
		} else {
			line := "  " + common.TableCellStyle.Render(zoneCell) + "  " +
				common.TableCellStyle.Render(distCell) + "  " +
				common.TableCellStyle.Render(durCell)
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

func (e *runEditor) toSegments() []api.RunSegment {
	var out []api.RunSegment
	for _, row := range e.rows {
		dist, err := strconv.ParseFloat(strings.TrimSpace(row.distanceStr), 64)
		if err != nil || dist <= 0 {
			continue
		}
		durSecs := parseDurationToSeconds(row.durationStr)
		if durSecs <= 0 {
			continue
		}
		zone := runZones[row.zoneIdx]
		z := zone
		out = append(out, api.RunSegment{Zone: &z, DistanceMiles: dist, DurationSeconds: durSecs})
	}
	return out
}

// ── main form model ───────────────────────────────────────────────────────────

// step constants
const (
	stepType    = 0
	stepRows    = 1 // custom row editor
	stepDetails = 2 // date/notes/duration huh form
)

type detailsState struct {
	date        string
	notes       string
	durationStr string // lift only: optional minutes
}

// LogWorkoutForm is a Bubble Tea model for the log-workout huh form.
type LogWorkoutForm struct {
	client      *api.Client
	exercises   []exerciseOption
	workoutType string // "LIFTING" or "RUNNING"
	step        int
	typeForm    *huh.Form
	liftEditor  liftEditor
	runEditor   runEditor
	details     detailsState
	detailForm  *huh.Form
}

type workoutExercisesLoadedMsg struct{ exercises []exerciseOption }

// NewLogWorkoutForm creates a new log-workout form.
func NewLogWorkoutForm(client *api.Client) *LogWorkoutForm {
	lw := &LogWorkoutForm{
		client:      client,
		workoutType: "LIFTING",
		step:        stepType,
		details:     detailsState{date: common.TodayString()},
	}
	lw.typeForm = lw.buildTypeForm()
	return lw
}

func (lw *LogWorkoutForm) buildTypeForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Log Workout").
				Description("Choose workout type").
				Options(
					huh.NewOption("Lifting", "LIFTING"),
					huh.NewOption("Running", "RUNNING"),
				).
				Value(&lw.workoutType),
		),
	).WithTheme(ActiveTheme)
}

func (lw *LogWorkoutForm) buildDetailForm() *huh.Form {
	today := common.TodayString()
	if lw.workoutType == "RUNNING" {
		return huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Date (YYYY-MM-DD)").
					Placeholder(today).
					Validate(common.ValidateDate).
					Value(&lw.details.date),
				huh.NewText().
					Title("Notes").
					Description("Optional").
					Value(&lw.details.notes),
			),
		).WithTheme(ActiveTheme)
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Date (YYYY-MM-DD)").
				Placeholder(today).
				Validate(common.ValidateDate).
				Value(&lw.details.date),
			huh.NewInput().
				Title("Duration (minutes)").
				Description("Optional").
				Placeholder("60").
				Value(&lw.details.durationStr),
			huh.NewText().
				Title("Notes").
				Description("Optional").
				Value(&lw.details.notes),
		),
	).WithTheme(ActiveTheme)
}

func (lw *LogWorkoutForm) loadExercises() tea.Cmd {
	client := lw.client
	return func() tea.Msg {
		all, err := client.GetAllExercises(false)
		if err != nil {
			return workoutExercisesLoadedMsg{}
		}
		opts := make([]exerciseOption, len(all))
		for i, e := range all {
			opts[i] = exerciseOption{id: e.ID, name: e.Name}
		}
		return workoutExercisesLoadedMsg{opts}
	}
}

func (lw *LogWorkoutForm) Init() tea.Cmd {
	return tea.Batch(lw.typeForm.Init(), lw.loadExercises())
}

func (lw *LogWorkoutForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return lw, func() tea.Msg { return WorkoutLogDoneMsg{Cancelled: true} }
		}
		// esc cancels from any step
		if msg.String() == "esc" {
			if lw.step == stepRows {
				// Go back to type selection.
				lw.step = stepType
				lw.typeForm = lw.buildTypeForm()
				return lw, lw.typeForm.Init()
			}
			if lw.step == stepDetails {
				lw.step = stepRows
				return lw, nil
			}
			return lw, func() tea.Msg { return WorkoutLogDoneMsg{Cancelled: true} }
		}

	case workoutExercisesLoadedMsg:
		lw.exercises = msg.exercises
		if lw.step == stepRows && lw.workoutType == "LIFTING" {
			lw.liftEditor = newLiftEditor(lw.exercises)
		}
		return lw, nil
	}

	switch lw.step {
	case stepType:
		var cmd tea.Cmd
		var done bool
		lw.typeForm, cmd, done = updateHuhForm(lw.typeForm, msg)
		if done {
			lw.step = stepRows
			if lw.workoutType == "LIFTING" {
				lw.liftEditor = newLiftEditor(lw.exercises)
			} else {
				lw.runEditor = newRunEditor()
			}
			return lw, nil
		}
		return lw, cmd

	case stepRows:
		if key, ok := msg.(tea.KeyMsg); ok {
			// 'n' or ctrl+enter advances to details step.
			if key.String() == "n" || key.String() == "ctrl+enter" {
				lw.step = stepDetails
				lw.detailForm = lw.buildDetailForm()
				return lw, lw.detailForm.Init()
			}
			if lw.workoutType == "LIFTING" {
				lw.liftEditor.update(key)
			} else {
				lw.runEditor.update(key)
			}
		}
		return lw, nil

	case stepDetails:
		var cmd tea.Cmd
		var done bool
		lw.detailForm, cmd, done = updateHuhForm(lw.detailForm, msg)
		if done {
			return lw, lw.submit()
		}
		return lw, cmd
	}

	return lw, nil
}

func (lw *LogWorkoutForm) submit() tea.Cmd {
	return func() tea.Msg {
		var err error
		if lw.workoutType == "RUNNING" {
			err = lw.submitRun()
		} else {
			err = lw.submitLift()
		}
		if err != nil {
			return common.ToastMsg{Text: "Failed to log workout: " + err.Error(), IsError: true}
		}
		return WorkoutLogDoneMsg{Cancelled: false}
	}
}

func (lw *LogWorkoutForm) submitLift() error {
	var durPtr *int
	if lw.details.durationStr != "" {
		d, err := strconv.Atoi(strings.TrimSpace(lw.details.durationStr))
		if err == nil && d > 0 {
			durPtr = &d
		}
	}
	var notes *string
	if lw.details.notes != "" {
		n := lw.details.notes
		notes = &n
	}
	lifts := lw.liftEditor.toLiftEntries()
	return lw.client.LogLiftSession(lw.details.date, durPtr, notes, lifts)
}

func (lw *LogWorkoutForm) submitRun() error {
	segments := lw.runEditor.toSegments()
	if len(segments) == 0 {
		return fmt.Errorf("each segment needs a distance (mi) and duration (mm:ss)")
	}
	var notes *string
	if lw.details.notes != "" {
		n := lw.details.notes
		notes = &n
	}
	return lw.client.LogRunSession(lw.details.date, notes, segments)
}

// parseDurationToSeconds parses "MM:SS" or plain minutes into total seconds.
func parseDurationToSeconds(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		m, _ := strconv.Atoi(parts[0])
		sec, _ := strconv.Atoi(parts[1])
		return m*60 + sec
	}
	m, _ := strconv.Atoi(s)
	return m * 60
}

func (lw *LogWorkoutForm) View() string {
	var b strings.Builder

	switch lw.step {
	case stepType:
		return common.PopupStyle.Render(lw.typeForm.View())

	case stepRows:
		popupW := 62
		b.WriteString(common.SelectedStyle.Render("Log "+capitalize(lw.workoutType)) + "\n\n")
		if lw.workoutType == "LIFTING" {
			b.WriteString(lw.liftEditor.view(popupW))
		} else {
			b.WriteString(lw.runEditor.view(popupW))
		}
		b.WriteString("\n")
		b.WriteString(common.MutedStyle.Render(
			"  enter/space: cycle  tab: next col  j/k: rows  a: add  D: delete  n: continue  esc: back",
		))
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(common.ColorPrimary).
			Padding(1, 2).
			Width(popupW).
			Render(b.String())

	case stepDetails:
		return common.PopupStyle.Render(lw.detailForm.View())
	}

	return ""
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
