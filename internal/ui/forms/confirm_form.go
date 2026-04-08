package forms

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
)

// ConfirmDoneMsg is sent when a confirm dialog completes.
type ConfirmDoneMsg struct {
	Confirmed bool
	// Tag identifies which action triggered the confirm (e.g. "archive", "delete").
	Tag string
}

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
	).WithTheme(ActiveTheme)
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

	var cmd tea.Cmd
	var done bool
	cf.form, cmd, done = updateHuhForm(cf.form, msg)
	if done {
		tag := cf.tag
		confirmed := *cf.confirmed
		return cf, func() tea.Msg { return ConfirmDoneMsg{Confirmed: confirmed, Tag: tag} }
	}
	return cf, cmd
}

func (cf ConfirmForm) View() string {
	return common.PopupStyle.Render(cf.form.View())
}
