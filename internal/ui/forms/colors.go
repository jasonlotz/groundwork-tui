package forms

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/jasonlotz/groundwork-tui/internal/ui/theme"
)

// ActiveTheme is the huh form theme drawn from the active app theme.
// To change the theme for the whole app, edit theme.Active in internal/ui/theme/theme.go.
var ActiveTheme = theme.Active.HuhTheme

// UpdateHuhForm forwards msg to the huh form and returns the updated form,
// a Tea command, and whether the form just completed. Callers use the
// completed bool to decide which DoneMsg to emit.
func UpdateHuhForm(f *huh.Form, msg tea.Msg) (*huh.Form, tea.Cmd, bool) {
	updated, cmd := f.Update(msg)
	if nf, ok := updated.(*huh.Form); ok {
		f = nf
	}
	return f, cmd, f.State == huh.StateCompleted
}

// updateHuhForm is the unexported alias used within the forms package.
func updateHuhForm(f *huh.Form, msg tea.Msg) (*huh.Form, tea.Cmd, bool) {
	return UpdateHuhForm(f, msg)
}

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
