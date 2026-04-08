package forms

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

// LogDoneMsg is sent when the log form completes (or is cancelled).
type LogDoneMsg struct{ Cancelled bool }

// LogForm is a Bubble Tea model for the log-progress huh form.
type LogForm struct {
	client       *api.Client
	materialID   string
	materialName string
	form         *huh.Form

	// bound form values — stored behind a pointer so copies remain valid.
	state *logFormState
}

type logFormState struct {
	dateStr  string
	unitsStr string
	notes    string
}

// NewLogForm creates a log-progress form pre-selected on the given material.
func NewLogForm(client *api.Client, materialID, materialName string) LogForm {
	today := fmt.Sprintf("%s", common.TodayString())
	st := &logFormState{dateStr: today}

	lf := LogForm{
		client:       client,
		materialID:   materialID,
		materialName: materialName,
		state:        st,
	}

	lf.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Log progress — %s", common.Truncate(materialName, 40))).
				Description("Date (YYYY-MM-DD)").
				Placeholder(today).
				Validate(common.ValidateDate).
				Value(&st.dateStr),

			huh.NewInput().
				Title("Units").
				Description("How many units did you complete?").
				Placeholder("1").
				Validate(common.ValidateUnits).
				Value(&st.unitsStr),

			huh.NewText().
				Title("Notes").
				Description("Optional").
				Value(&st.notes),
		),
	).WithTheme(ActiveTheme)

	return lf
}

func (lf LogForm) Init() tea.Cmd {
	return lf.form.Init()
}

func (lf LogForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "esc" {
			return lf, func() tea.Msg { return LogDoneMsg{Cancelled: true} }
		}
	}

	var cmd tea.Cmd
	var done bool
	lf.form, cmd, done = updateHuhForm(lf.form, msg)
	if done {
		return lf, lf.submit()
	}
	return lf, cmd
}

func (lf LogForm) submit() tea.Cmd {
	return func() tea.Msg {
		// Validation already passed in the form; these parses are safe.
		units, _ := strconv.ParseFloat(lf.state.unitsStr, 64)
		var notes *string
		if lf.state.notes != "" {
			n := lf.state.notes
			notes = &n
		}
		if err := lf.client.LogUnits(lf.materialID, lf.state.dateStr, units, notes); err != nil {
			return common.ToastMsg{Text: "Failed to log: " + err.Error(), IsError: true}
		}
		return LogDoneMsg{Cancelled: false}
	}
}

func (lf LogForm) View() string {
	return common.PopupStyle.Render(lf.form.View())
}
