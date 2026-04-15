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

// WorkoutLogDoneMsg is sent when the log-workout form completes or is cancelled.
type WorkoutLogDoneMsg struct{ Cancelled bool }

// ── exercise option ───────────────────────────────────────────────────────────

type exerciseOption struct {
	id   string
	name string
}

// ── subtype option ────────────────────────────────────────────────────────────

type subtypeOption struct {
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

func (e *liftEditor) isTyping() bool { return e.typing }

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
		case "tab", "down":
			e.typing = false
			e.col = 0
			if msg.String() == "down" {
				if e.cursor < len(e.rows)-1 {
					e.cursor++
				}
			}
		case "esc":
			e.typing = false
		case "enter":
			// enter commits the value and exits typing mode only
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
			// Move focus to weight column.
			e.col = 1
			e.typing = true
			e.typingBuf = e.rows[e.cursor].weightStr
		}
		// Weight col: enter is intercepted by parent before reaching here.
	case " ":
		// Space cycles exercise when on exercise col.
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
	b.WriteString(common.DimStyle.Render("  "+strings.Repeat("─", width-4)) + "\n")

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
			continue
		}
		out = append(out, api.LiftEntry{
			ExerciseID: e.exercises[row.exerciseIdx].id,
			WeightLbs:  w,
		})
	}
	return out
}

// ── cardio segment editor ───────────────────────────────────────────────────

var cardioZones = []string{"FREE", "Z1", "Z2", "Z3", "Z4", "Z5"}
var cardioZoneLabels = []string{"Freestyle", "Zone 1", "Zone 2", "Zone 3", "Zone 4", "Zone 5"}

type cardioSegRow struct {
	zoneIdx        int // index into cardioZones
	distanceStr    string
	durationStr    string // mm:ss or plain minutes
	elevationStr   string // elevation gain in ft
	stepsStr       string // steps count
}

// cardioCol constants
const (
	cardioColZone      = 0
	cardioColDistance   = 1
	cardioColDuration  = 2
	cardioColElevation = 3
	cardioColSteps     = 4
	cardioColCount     = 5
)

type cardioEditor struct {
	rows      []cardioSegRow
	cursor    int
	col       int
	typing    bool
	typingBuf string
}

func newCardioEditor() cardioEditor {
	return cardioEditor{rows: []cardioSegRow{{zoneIdx: 0}}}
}

func (e *cardioEditor) isTyping() bool { return e.typing }

func (e *cardioEditor) addRow() {
	if len(e.rows) < 12 {
		e.rows = append(e.rows, cardioSegRow{zoneIdx: 0})
		e.cursor = len(e.rows) - 1
		e.col = 0
		e.typing = false
	}
}

func (e *cardioEditor) deleteRow() {
	if len(e.rows) <= 1 {
		return
	}
	e.rows = append(e.rows[:e.cursor], e.rows[e.cursor+1:]...)
	if e.cursor >= len(e.rows) {
		e.cursor = len(e.rows) - 1
	}
	e.typing = false
}

func (e *cardioEditor) currentFieldStr() string {
	row := e.rows[e.cursor]
	switch e.col {
	case cardioColDistance:
		return row.distanceStr
	case cardioColDuration:
		return row.durationStr
	case cardioColElevation:
		return row.elevationStr
	case cardioColSteps:
		return row.stepsStr
	}
	return ""
}

func (e *cardioEditor) setCurrentFieldStr(s string) {
	switch e.col {
	case cardioColDistance:
		e.rows[e.cursor].distanceStr = s
	case cardioColDuration:
		e.rows[e.cursor].durationStr = s
	case cardioColElevation:
		e.rows[e.cursor].elevationStr = s
	case cardioColSteps:
		e.rows[e.cursor].stepsStr = s
	}
}

func (e *cardioEditor) isTextCol() bool {
	return e.col >= cardioColDistance && e.col <= cardioColSteps
}

func (e *cardioEditor) update(msg tea.KeyMsg) {
	if e.typing && e.isTextCol() {
		switch msg.String() {
		case "tab":
			e.typing = false
			e.col++
			if e.col >= cardioColCount {
				e.col = 0
			} else {
				e.typing = e.isTextCol()
				if e.typing {
					e.typingBuf = e.currentFieldStr()
				}
			}
		case "esc":
			e.typing = false
		case "enter":
			// enter commits the value and exits typing mode only
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
		if e.col >= cardioColCount {
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
		if e.col < cardioColSteps {
			e.col++
			e.typing = e.isTextCol()
			if e.typing {
				e.typingBuf = e.currentFieldStr()
			}
		}
	case "enter":
		if e.col == cardioColZone {
			// Enter moves to distance column.
			e.col = cardioColDistance
			e.typing = true
			e.typingBuf = e.currentFieldStr()
		}
		// Text cols: enter is intercepted by parent before reaching here.
	case " ":
		if e.col == cardioColZone {
			// Space cycles zone.
			e.rows[e.cursor].zoneIdx++
			if e.rows[e.cursor].zoneIdx >= len(cardioZones) {
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

func (e *cardioEditor) view(width int) string {
	var b strings.Builder
	header := fmt.Sprintf("  %-10s  %-10s  %-10s  %-10s  %s", "Zone", "Dist (mi)", "Dur (mm:ss)", "Elev (ft)", "Steps")
	b.WriteString(common.TableHeaderStyle.Render(header) + "\n")
	b.WriteString(common.DimStyle.Render("  "+strings.Repeat("─", width-4)) + "\n")

	for i, row := range e.rows {
		zone := cardioZoneLabels[row.zoneIdx]
		dist := row.distanceStr
		if dist == "" {
			dist = "—"
		}
		dur := row.durationStr
		if dur == "" {
			dur = "—"
		}
		elev := row.elevationStr
		if elev == "" {
			elev = "—"
		}
		steps := row.stepsStr
		if steps == "" {
			steps = "—"
		}

		zoneCell := fmt.Sprintf("%-10s", zone)
		distCell := fmt.Sprintf("%-10s", dist)
		durCell := fmt.Sprintf("%-11s", dur)
		elevCell := fmt.Sprintf("%-10s", elev)
		stepsCell := fmt.Sprintf("%-6s", steps)

		if i == e.cursor {
			cursor := common.SelectedStyle.Render(">")
			renderCell := func(col int, text string) string {
				if e.col == col {
					if e.typing && col != cardioColZone {
						return common.SelectedStyle.Render(fmt.Sprintf("%-*s", len(text), e.typingBuf+"_"))
					}
					return common.SelectedStyle.Render(text)
				}
				return common.TableCellStyle.Render(text)
			}
			line := cursor + " " + renderCell(cardioColZone, zoneCell) + "  " +
				renderCell(cardioColDistance, distCell) + "  " +
				renderCell(cardioColDuration, durCell) + "  " +
				renderCell(cardioColElevation, elevCell) + "  " +
				renderCell(cardioColSteps, stepsCell)
			b.WriteString(line + "\n")
		} else {
			line := "  " + common.TableCellStyle.Render(zoneCell) + "  " +
				common.TableCellStyle.Render(distCell) + "  " +
				common.TableCellStyle.Render(durCell) + "  " +
				common.TableCellStyle.Render(elevCell) + "  " +
				common.TableCellStyle.Render(stepsCell)
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

func (e *cardioEditor) toSegments() []api.CardioSegment {
	var out []api.CardioSegment
	for _, row := range e.rows {
		durSecs := parseDurationToSeconds(row.durationStr)
		if durSecs <= 0 {
			continue
		}
		zone := cardioZones[row.zoneIdx]
		seg := api.CardioSegment{
			Zone:            &zone,
			DurationSeconds: durSecs,
		}
		if dist, err := strconv.ParseFloat(strings.TrimSpace(row.distanceStr), 64); err == nil && dist > 0 {
			seg.DistanceMiles = &dist
		}
		if elev, err := strconv.ParseFloat(strings.TrimSpace(row.elevationStr), 64); err == nil && elev > 0 {
			seg.ElevationGainFt = &elev
		}
		if steps, err := strconv.Atoi(strings.TrimSpace(row.stepsStr)); err == nil && steps > 0 {
			seg.Steps = &steps
		}
		out = append(out, seg)
	}
	return out
}

// ── main form model ───────────────────────────────────────────────────────────

// step constants
const (
	stepType    = 0
	stepSubtype = 1 // subtype selection
	stepDetails = 2 // date/notes/duration huh form
	stepRows    = 3 // custom row editor
)

type detailsState struct {
	date        string
	notes       string
	durationStr string // lift only: required minutes
}

// LogWorkoutForm is a Bubble Tea model for the log-workout huh form.
type LogWorkoutForm struct {
	client      *api.Client
	exercises   []exerciseOption
	subtypes    []subtypeOption
	workoutType string // "LIFTING" or "CARDIO"
	subtypeID   string
	step        int
	typeForm    *huh.Form
	subtypeForm *huh.Form
	liftEditor  liftEditor
	cardioEditor cardioEditor
	details     detailsState
	detailForm  *huh.Form
}

type workoutExercisesLoadedMsg struct{ exercises []exerciseOption }
type workoutSubtypesLoadedMsg struct {
	subtypes []subtypeOption
}

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
					huh.NewOption("Cardio", "CARDIO"),
				).
				Value(&lw.workoutType),
		),
	).WithTheme(ActiveTheme)
}

func (lw *LogWorkoutForm) buildSubtypeForm() *huh.Form {
	opts := make([]huh.Option[string], len(lw.subtypes))
	for i, st := range lw.subtypes {
		opts[i] = huh.NewOption(st.name, st.id)
	}
	if len(opts) == 0 {
		opts = []huh.Option[string]{huh.NewOption("(no subtypes — create in Settings)", "")}
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose Subtype").
				Description(common.TitleCase(lw.workoutType) + " subtype").
				Options(opts...).
				Value(&lw.subtypeID),
		),
	).WithTheme(ActiveTheme)
}

func (lw *LogWorkoutForm) buildDetailForm() *huh.Form {
	today := common.TodayString()
	if lw.workoutType == "CARDIO" {
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
				Placeholder("60").
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

func (lw *LogWorkoutForm) loadSubtypes() tea.Cmd {
	client := lw.client
	wt := lw.workoutType
	return func() tea.Msg {
		all, err := client.GetAllSubtypes(&wt, false)
		if err != nil {
			return workoutSubtypesLoadedMsg{}
		}
		opts := make([]subtypeOption, len(all))
		for i, st := range all {
			opts[i] = subtypeOption{id: st.ID, name: st.Name}
		}
		return workoutSubtypesLoadedMsg{opts}
	}
}

// SubtypeOptions returns the unexported subtypeOption slice so callers
// can pass subtypes fetched separately into forms.
func SubtypeOptions(subtypes []model.WorkoutSubtype) []subtypeOption {
	opts := make([]subtypeOption, len(subtypes))
	for i, st := range subtypes {
		opts[i] = subtypeOption{id: st.ID, name: st.Name}
	}
	return opts
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
				// rows → back to details
				lw.step = stepDetails
				lw.detailForm = lw.buildDetailForm()
				return lw, lw.detailForm.Init()
			}
			if lw.step == stepDetails {
				// details → back to subtype
				lw.step = stepSubtype
				lw.subtypeForm = lw.buildSubtypeForm()
				return lw, lw.subtypeForm.Init()
			}
			if lw.step == stepSubtype {
				// subtype → back to type
				lw.step = stepType
				lw.typeForm = lw.buildTypeForm()
				return lw, lw.typeForm.Init()
			}
			return lw, func() tea.Msg { return WorkoutLogDoneMsg{Cancelled: true} }
		}

	case workoutExercisesLoadedMsg:
		lw.exercises = msg.exercises
		if lw.step == stepRows && lw.workoutType == "LIFTING" && lw.liftEditor.rows == nil {
			lw.liftEditor = newLiftEditor(lw.exercises)
		}
		return lw, nil

	case workoutSubtypesLoadedMsg:
		lw.subtypes = msg.subtypes
		// Build and show subtype form now that we have data
		if lw.step == stepSubtype {
			lw.subtypeForm = lw.buildSubtypeForm()
			return lw, lw.subtypeForm.Init()
		}
		return lw, nil
	}

	switch lw.step {
	case stepType:
		var cmd tea.Cmd
		var done bool
		lw.typeForm, cmd, done = updateHuhForm(lw.typeForm, msg)
		if done {
			// Advance to subtype selection — load subtypes for this type
			lw.step = stepSubtype
			return lw, lw.loadSubtypes()
		}
		return lw, cmd

	case stepSubtype:
		if lw.subtypeForm == nil {
			// Still loading subtypes
			return lw, nil
		}
		var cmd tea.Cmd
		var done bool
		lw.subtypeForm, cmd, done = updateHuhForm(lw.subtypeForm, msg)
		if done {
			// Advance to details
			lw.step = stepDetails
			lw.detailForm = lw.buildDetailForm()
			return lw, lw.detailForm.Init()
		}
		return lw, cmd

	case stepDetails:
		var cmd tea.Cmd
		var done bool
		lw.detailForm, cmd, done = updateHuhForm(lw.detailForm, msg)
		if done {
			// Advance to row editor
			lw.step = stepRows
			if lw.workoutType == "LIFTING" {
				if lw.liftEditor.rows == nil {
					lw.liftEditor = newLiftEditor(lw.exercises)
				}
			} else {
				if lw.cardioEditor.rows == nil {
					lw.cardioEditor = newCardioEditor()
				}
			}
			return lw, nil
		}
		return lw, cmd

	case stepRows:
		if key, ok := msg.(tea.KeyMsg); ok {
			// enter submits only when not in a text-typing field
			if key.String() == "enter" {
				typing := false
				if lw.workoutType == "LIFTING" {
					typing = lw.liftEditor.isTyping()
				} else {
					typing = lw.cardioEditor.isTyping()
				}
				if !typing {
					return lw, lw.submit()
				}
			}
			if lw.workoutType == "LIFTING" {
				lw.liftEditor.update(key)
			} else {
				lw.cardioEditor.update(key)
			}
		}
		return lw, nil
	}

	return lw, nil
}

func (lw *LogWorkoutForm) submit() tea.Cmd {
	if lw.subtypeID == "" {
		return func() tea.Msg {
			return common.ToastMsg{Text: "No subtype selected — create one in Settings first", IsError: true}
		}
	}
	return func() tea.Msg {
		var err error
		if lw.workoutType == "CARDIO" {
			err = lw.submitCardio()
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
	dur, err := strconv.Atoi(strings.TrimSpace(lw.details.durationStr))
	if err != nil || dur <= 0 {
		return fmt.Errorf("duration is required and must be > 0")
	}
	var notes *string
	if lw.details.notes != "" {
		n := lw.details.notes
		notes = &n
	}
	lifts := lw.liftEditor.toLiftEntries()
	return lw.client.LogLiftSession(lw.details.date, dur, notes, lw.subtypeID, lifts)
}

func (lw *LogWorkoutForm) submitCardio() error {
	segments := lw.cardioEditor.toSegments()
	if len(segments) == 0 {
		return fmt.Errorf("each segment needs a duration (mm:ss)")
	}
	var notes *string
	if lw.details.notes != "" {
		n := lw.details.notes
		notes = &n
	}
	return lw.client.LogCardioSession(lw.details.date, notes, lw.subtypeID, segments)
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

	case stepSubtype:
		if lw.subtypeForm == nil {
			return common.PopupStyle.Render(common.DimStyle.Render("Loading subtypes…"))
		}
		return common.PopupStyle.Render(lw.subtypeForm.View())

	case stepRows:
		popupW := 76
		b.WriteString(common.SelectedStyle.Render("Log "+common.TitleCase(lw.workoutType)) + "\n\n")
		if lw.workoutType == "LIFTING" {
			b.WriteString(common.DimStyle.Render("  Exercises (optional)") + "\n\n")
			b.WriteString(lw.liftEditor.view(popupW))
		} else {
			b.WriteString(common.DimStyle.Render("  Segments") + "\n\n")
			b.WriteString(lw.cardioEditor.view(popupW))
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

	case stepDetails:
		return common.PopupStyle.Render(lw.detailForm.View())
	}

	return ""
}

