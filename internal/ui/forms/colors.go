package forms

import "github.com/charmbracelet/huh"

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
