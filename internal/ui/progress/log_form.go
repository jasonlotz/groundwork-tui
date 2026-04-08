// Package progress — log form for logging units against a material.
package progress

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

// LogDoneMsg is sent when the log form completes (or is cancelled).
type LogDoneMsg struct{ Cancelled bool }

// LogForm is a Bubble Tea model for the log-progress huh form.
type LogForm struct {
	client    *api.Client
	materials []model.ActiveMaterial
	form      *huh.Form

	// bound form values
	materialID string
	dateStr    string
	unitsStr   string
	notes      string
}

// NewLogForm creates a log-progress form pre-populated with active materials.
func NewLogForm(client *api.Client, activeMaterials []model.ActiveMaterial) LogForm {
	today := time.Now().Format("2006-01-02")

	lf := LogForm{
		client:    client,
		materials: activeMaterials,
		dateStr:   today,
		unitsStr:  "",
	}

	// Build material options for the select field.
	matOptions := make([]huh.Option[string], 0, len(activeMaterials))
	for _, m := range activeMaterials {
		label := fmt.Sprintf("%-30s  %s", common.Truncate(m.Name, 30), m.SkillName())
		matOptions = append(matOptions, huh.NewOption(label, m.ID))
	}
	if len(matOptions) > 0 {
		lf.materialID = activeMaterials[0].ID
	}

	lf.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Material").
				Options(matOptions...).
				Value(&lf.materialID),

			huh.NewInput().
				Title("Date").
				Description("YYYY-MM-DD").
				Placeholder(today).
				Value(&lf.dateStr),

			huh.NewInput().
				Title("Units").
				Description("How many units did you complete?").
				Placeholder("1").
				Value(&lf.unitsStr),

			huh.NewText().
				Title("Notes").
				Description("Optional").
				Value(&lf.notes),
		),
	).WithTheme(huh.ThemeDracula())

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

	form, cmd := lf.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		lf.form = f
	}

	if lf.form.State == huh.StateCompleted {
		return lf, lf.submit()
	}

	return lf, cmd
}

func (lf LogForm) submit() tea.Cmd {
	return func() tea.Msg {
		units, err := strconv.ParseFloat(lf.unitsStr, 64)
		if err != nil || units <= 0 {
			return common.ToastMsg{Text: "Invalid units value", IsError: true}
		}
		var notes *string
		if lf.notes != "" {
			n := lf.notes
			notes = &n
		}
		if err := lf.client.LogUnits(lf.materialID, lf.dateStr, units, notes); err != nil {
			return common.ToastMsg{Text: "Failed to log: " + err.Error(), IsError: true}
		}
		return LogDoneMsg{Cancelled: false}
	}
}

func (lf LogForm) View() string {
	return lf.form.View()
}

// PreSelectMaterial sets the selected material ID if it exists in the materials list.
func (lf *LogForm) PreSelectMaterial(materialID string) {
	for _, m := range lf.materials {
		if m.ID == materialID {
			lf.materialID = materialID
			return
		}
	}
}
