// Package theme defines the application-wide theme: a paired huh form theme
// and lipgloss color palette. Change Active (or call SetActive) to switch the
// entire app's look at runtime.
package theme

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Palette holds the lipgloss colors used throughout the app.
//
// Dim is the darker/more-faded color — used for archived names, help text,
// inactive tabs, and other truly de-emphasized UI elements.
//
// Muted is the lighter secondary-text color — used for table headers, stat
// labels, subtitles, and other readable-but-not-primary UI elements.
type Palette struct {
	Primary   lipgloss.Color
	Dim       lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Danger    lipgloss.Color
	Border    lipgloss.Color
	Muted     lipgloss.Color
	Highlight lipgloss.Color
}

// AppTheme pairs a huh form theme with a lipgloss palette.
type AppTheme struct {
	Name     string
	HuhTheme *huh.Theme
	Colors   Palette
}

// Active is the currently selected theme. Use SetActive to change it at runtime.
var Active = Catppuccin

// All is the ordered list of all available themes, shown in the settings screen.
var All = []AppTheme{
	Catppuccin,
	Dracula,
	Charm,
	TokyoNight,
	Gruvbox,
	Nord,
	SolarizedDark,
	Monokai,
	RosePine,
	Base,
	Base16,
}

// SetActive switches the active theme by name. Returns false if not found.
func SetActive(name string) bool {
	for _, t := range All {
		if t.Name == name {
			Active = t
			return true
		}
	}
	return false
}

// Dracula theme — purple/yellow/green huh forms, dracula lipgloss palette.
var Dracula = AppTheme{
	Name:     "Dracula",
	HuhTheme: huh.ThemeDracula(),
	Colors: Palette{
		Primary:   lipgloss.Color("#bd93f9"), // dracula purple
		Dim:       lipgloss.Color("#6272a4"), // dracula comment (darker)
		Success:   lipgloss.Color("#50fa7b"), // dracula green
		Warning:   lipgloss.Color("#f1fa8c"), // dracula yellow
		Danger:    lipgloss.Color("#ff5555"), // dracula red
		Border:    lipgloss.Color("#44475a"), // dracula current line
		Muted:     lipgloss.Color("#8892bf"), // dracula comment lightened (secondary text)
		Highlight: lipgloss.Color("#ff79c6"), // dracula pink
	},
}

// Charm theme — indigo/fuchsia/green huh forms, indigo lipgloss palette.
var Charm = AppTheme{
	Name:     "Charm",
	HuhTheme: huh.ThemeCharm(),
	Colors: Palette{
		Primary:   lipgloss.Color("#5A56E0"), // charm indigo
		Dim:       lipgloss.Color("#6B7280"), // gray-500 (darker, de-emphasized)
		Success:   lipgloss.Color("#02BA84"), // charm green
		Warning:   lipgloss.Color("#D97706"), // amber-600
		Danger:    lipgloss.Color("#ff0055"), // bright red
		Border:    lipgloss.Color("#383838"), // dark gray
		Muted:     lipgloss.Color("#9CA3AF"), // gray-400 (lighter, secondary text)
		Highlight: lipgloss.Color("#7571F9"), // charm lavender
	},
}

// Catppuccin theme — mauve/pink/green huh forms, mocha lipgloss palette.
var Catppuccin = AppTheme{
	Name:     "Catppuccin",
	HuhTheme: huh.ThemeCatppuccin(),
	Colors: Palette{
		Primary:   lipgloss.Color("#cba6f7"), // mocha mauve
		Dim:       lipgloss.Color("#6c7086"), // mocha overlay0 (darker, de-emphasized)
		Success:   lipgloss.Color("#a6e3a1"), // mocha green
		Warning:   lipgloss.Color("#f9e2af"), // mocha yellow
		Danger:    lipgloss.Color("#f38ba8"), // mocha red
		Border:    lipgloss.Color("#45475a"), // mocha surface1
		Muted:     lipgloss.Color("#9399b2"), // mocha overlay2 (lighter, secondary text)
		Highlight: lipgloss.Color("#f5c2e7"), // mocha pink
	},
}

// Base theme — minimal ANSI-only huh forms, neutral terminal palette.
var Base = AppTheme{
	Name:     "Base",
	HuhTheme: huh.ThemeBase(),
	Colors: Palette{
		Primary:   lipgloss.Color("4"),  // ANSI blue
		Dim:       lipgloss.Color("8"),  // ANSI bright black (dark gray, de-emphasized)
		Success:   lipgloss.Color("2"),  // ANSI green
		Warning:   lipgloss.Color("3"),  // ANSI yellow
		Danger:    lipgloss.Color("1"),  // ANSI red
		Border:    lipgloss.Color("0"),  // ANSI black
		Muted:     lipgloss.Color("7"),  // ANSI white (secondary text)
		Highlight: lipgloss.Color("12"), // ANSI bright blue
	},
}

// Base16 theme — terminal base16 palette for huh forms, matching lipgloss palette.
var Base16 = AppTheme{
	Name:     "Base16",
	HuhTheme: huh.ThemeBase16(),
	Colors: Palette{
		Primary:   lipgloss.Color("6"),  // ANSI cyan
		Dim:       lipgloss.Color("8"),  // ANSI bright black (de-emphasized)
		Success:   lipgloss.Color("2"),  // ANSI green
		Warning:   lipgloss.Color("3"),  // ANSI yellow
		Danger:    lipgloss.Color("9"),  // ANSI bright red
		Border:    lipgloss.Color("0"),  // ANSI black
		Muted:     lipgloss.Color("7"),  // ANSI white (secondary text)
		Highlight: lipgloss.Color("14"), // ANSI bright cyan
	},
}

// TokyoNight theme — deep blue/purple night sky palette.
var TokyoNight = AppTheme{
	Name:     "Tokyo Night",
	HuhTheme: tokyoNightHuh(),
	Colors: Palette{
		Primary:   lipgloss.Color("#7aa2f7"), // blue
		Dim:       lipgloss.Color("#414868"), // overlay (darker, de-emphasized)
		Success:   lipgloss.Color("#9ece6a"), // green
		Warning:   lipgloss.Color("#e0af68"), // yellow
		Danger:    lipgloss.Color("#f7768e"), // red
		Border:    lipgloss.Color("#3b4261"), // surface
		Muted:     lipgloss.Color("#9aa5ce"), // comment2 (lighter, secondary text)
		Highlight: lipgloss.Color("#bb9af7"), // purple
	},
}

func tokyoNightHuh() *huh.Theme {
	t := huh.ThemeBase()
	var (
		bg      = lipgloss.Color("#1a1b26")
		surface = lipgloss.Color("#3b4261")
		comment = lipgloss.Color("#565f89")
		fg      = lipgloss.Color("#c0caf5")
		blue    = lipgloss.Color("#7aa2f7")
		purple  = lipgloss.Color("#bb9af7")
		green   = lipgloss.Color("#9ece6a")
		yellow  = lipgloss.Color("#e0af68")
		red     = lipgloss.Color("#f7768e")
	)
	t.Focused.Base = t.Focused.Base.BorderForeground(surface)
	t.Focused.Title = t.Focused.Title.Foreground(blue)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(blue)
	t.Focused.Description = t.Focused.Description.Foreground(comment)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(purple)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(purple)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(purple)
	t.Focused.Option = t.Focused.Option.Foreground(fg)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(purple)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(fg)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(comment)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(bg).Background(blue).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(fg).Background(surface)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(yellow)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(comment)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(blue)
	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	return t
}

// Gruvbox theme — warm retro brown/orange/yellow palette.
var Gruvbox = AppTheme{
	Name:     "Gruvbox",
	HuhTheme: gruvboxHuh(),
	Colors: Palette{
		Primary:   lipgloss.Color("#d79921"), // yellow
		Dim:       lipgloss.Color("#665c54"), // bg3 (darker, de-emphasized)
		Success:   lipgloss.Color("#98971a"), // green
		Warning:   lipgloss.Color("#d65d0e"), // orange
		Danger:    lipgloss.Color("#cc241d"), // red
		Border:    lipgloss.Color("#504945"), // bg2
		Muted:     lipgloss.Color("#928374"), // gray (lighter, secondary text)
		Highlight: lipgloss.Color("#d3869b"), // purple
	},
}

func gruvboxHuh() *huh.Theme {
	t := huh.ThemeBase()
	var (
		bg     = lipgloss.Color("#282828")
		bg2    = lipgloss.Color("#504945")
		gray   = lipgloss.Color("#928374")
		fg     = lipgloss.Color("#ebdbb2")
		yellow = lipgloss.Color("#d79921")
		orange = lipgloss.Color("#d65d0e")
		green  = lipgloss.Color("#98971a")
		red    = lipgloss.Color("#cc241d")
		purple = lipgloss.Color("#b16286")
	)
	t.Focused.Base = t.Focused.Base.BorderForeground(bg2)
	t.Focused.Title = t.Focused.Title.Foreground(yellow)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(yellow)
	t.Focused.Description = t.Focused.Description.Foreground(gray)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(orange)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(orange)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(orange)
	t.Focused.Option = t.Focused.Option.Foreground(fg)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(orange)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(fg)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(gray)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(bg).Background(yellow).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(fg).Background(bg2)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(orange)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(gray)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(yellow)
	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(gray)
	t.Blurred.Title = t.Blurred.Title.Foreground(gray)
	_ = purple
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	return t
}

// Nord theme — cool arctic blues and soft gray palette.
var Nord = AppTheme{
	Name:     "Nord",
	HuhTheme: nordHuh(),
	Colors: Palette{
		Primary:   lipgloss.Color("#88c0d0"), // frost cyan
		Dim:       lipgloss.Color("#434c5e"), // polar night 2 (darker, de-emphasized)
		Success:   lipgloss.Color("#a3be8c"), // aurora green
		Warning:   lipgloss.Color("#ebcb8b"), // aurora yellow
		Danger:    lipgloss.Color("#bf616a"), // aurora red
		Border:    lipgloss.Color("#3b4252"), // polar night 1
		Muted:     lipgloss.Color("#4c566a"), // polar night 3 (lighter, secondary text)
		Highlight: lipgloss.Color("#b48ead"), // aurora purple
	},
}

func nordHuh() *huh.Theme {
	t := huh.ThemeBase()
	var (
		bg     = lipgloss.Color("#2e3440")
		pn1    = lipgloss.Color("#3b4252")
		pn3    = lipgloss.Color("#4c566a")
		fg     = lipgloss.Color("#eceff4")
		cyan   = lipgloss.Color("#88c0d0")
		blue   = lipgloss.Color("#81a1c1")
		green  = lipgloss.Color("#a3be8c")
		yellow = lipgloss.Color("#ebcb8b")
		red    = lipgloss.Color("#bf616a")
		purple = lipgloss.Color("#b48ead")
	)
	_ = blue
	t.Focused.Base = t.Focused.Base.BorderForeground(pn1)
	t.Focused.Title = t.Focused.Title.Foreground(cyan)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(cyan)
	t.Focused.Description = t.Focused.Description.Foreground(pn3)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(purple)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(purple)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(purple)
	t.Focused.Option = t.Focused.Option.Foreground(fg)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(purple)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(fg)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(pn3)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(bg).Background(cyan).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(fg).Background(pn1)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(yellow)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(pn3)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(cyan)
	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	return t
}

// SolarizedDark theme — classic muted cyan/blue on dark background.
var SolarizedDark = AppTheme{
	Name:     "Solarized Dark",
	HuhTheme: solarizedDarkHuh(),
	Colors: Palette{
		Primary:   lipgloss.Color("#268bd2"), // blue
		Dim:       lipgloss.Color("#586e75"), // base01 (darker, de-emphasized)
		Success:   lipgloss.Color("#859900"), // green
		Warning:   lipgloss.Color("#b58900"), // yellow
		Danger:    lipgloss.Color("#dc322f"), // red
		Border:    lipgloss.Color("#073642"), // base02
		Muted:     lipgloss.Color("#839496"), // base0 (lighter, secondary text)
		Highlight: lipgloss.Color("#2aa198"), // cyan
	},
}

func solarizedDarkHuh() *huh.Theme {
	t := huh.ThemeBase()
	var (
		base02 = lipgloss.Color("#073642")
		base01 = lipgloss.Color("#586e75")
		base0  = lipgloss.Color("#839496")
		base3  = lipgloss.Color("#fdf6e3")
		blue   = lipgloss.Color("#268bd2")
		cyan   = lipgloss.Color("#2aa198")
		green  = lipgloss.Color("#859900")
		yellow = lipgloss.Color("#b58900")
		red    = lipgloss.Color("#dc322f")
		violet = lipgloss.Color("#6c71c4")
	)
	_ = base3
	_ = violet
	t.Focused.Base = t.Focused.Base.BorderForeground(base02)
	t.Focused.Title = t.Focused.Title.Foreground(blue)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(blue)
	t.Focused.Description = t.Focused.Description.Foreground(base01)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(cyan)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(cyan)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(cyan)
	t.Focused.Option = t.Focused.Option.Foreground(base0)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(cyan)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(base0)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(base01)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(base02).Background(blue).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(base0).Background(base02)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(yellow)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(base01)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(blue)
	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	return t
}

// Monokai theme — vivid pink/orange/green on dark background.
var Monokai = AppTheme{
	Name:     "Monokai",
	HuhTheme: monokaiHuh(),
	Colors: Palette{
		Primary:   lipgloss.Color("#f92672"), // pink/red
		Dim:       lipgloss.Color("#75715e"), // comment brown (darker, de-emphasized)
		Success:   lipgloss.Color("#a6e22e"), // green
		Warning:   lipgloss.Color("#e6db74"), // yellow
		Danger:    lipgloss.Color("#f92672"), // pink/red
		Border:    lipgloss.Color("#3e3d32"), // bg light
		Muted:     lipgloss.Color("#908b83"), // warm gray (lighter, secondary text)
		Highlight: lipgloss.Color("#66d9e8"), // cyan
	},
}

func monokaiHuh() *huh.Theme {
	t := huh.ThemeBase()
	var (
		bg      = lipgloss.Color("#272822")
		bgLight = lipgloss.Color("#3e3d32")
		comment = lipgloss.Color("#75715e")
		fg      = lipgloss.Color("#f8f8f2")
		pink    = lipgloss.Color("#f92672")
		orange  = lipgloss.Color("#fd971f")
		green   = lipgloss.Color("#a6e22e")
		yellow  = lipgloss.Color("#e6db74")
		cyan    = lipgloss.Color("#66d9e8")
		purple  = lipgloss.Color("#ae81ff")
	)
	_ = orange
	_ = purple
	t.Focused.Base = t.Focused.Base.BorderForeground(bgLight)
	t.Focused.Title = t.Focused.Title.Foreground(pink)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(pink)
	t.Focused.Description = t.Focused.Description.Foreground(comment)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(pink)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(pink)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(cyan)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(cyan)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(cyan)
	t.Focused.Option = t.Focused.Option.Foreground(fg)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(cyan)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(fg)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(comment)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(bg).Background(pink).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(fg).Background(bgLight)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(yellow)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(comment)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(pink)
	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	return t
}

// RosePine theme — muted mauve/rose/gold on deep dark background.
var RosePine = AppTheme{
	Name:     "Rosé Pine",
	HuhTheme: rosePineHuh(),
	Colors: Palette{
		Primary:   lipgloss.Color("#c4a7e7"), // iris/purple
		Dim:       lipgloss.Color("#6e6a86"), // muted (darker, de-emphasized)
		Success:   lipgloss.Color("#31748f"), // pine/teal
		Warning:   lipgloss.Color("#f6c177"), // gold
		Danger:    lipgloss.Color("#eb6f92"), // love/rose
		Border:    lipgloss.Color("#26233a"), // overlay
		Muted:     lipgloss.Color("#9893a5"), // subtle (lighter, secondary text)
		Highlight: lipgloss.Color("#ebbcba"), // rose
	},
}

func rosePineHuh() *huh.Theme {
	t := huh.ThemeBase()
	var (
		bg      = lipgloss.Color("#191724")
		overlay = lipgloss.Color("#26233a")
		muted   = lipgloss.Color("#6e6a86")
		text    = lipgloss.Color("#e0def4")
		rose    = lipgloss.Color("#ebbcba")
		iris    = lipgloss.Color("#c4a7e7")
		pine    = lipgloss.Color("#31748f")
		gold    = lipgloss.Color("#f6c177")
		love    = lipgloss.Color("#eb6f92")
	)
	_ = rose
	t.Focused.Base = t.Focused.Base.BorderForeground(overlay)
	t.Focused.Title = t.Focused.Title.Foreground(iris)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(iris)
	t.Focused.Description = t.Focused.Description.Foreground(muted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(love)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(love)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(iris)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(iris)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(iris)
	t.Focused.Option = t.Focused.Option.Foreground(text)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(iris)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(pine)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(pine)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(text)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(muted)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(bg).Background(iris).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(text).Background(overlay)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(gold)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(muted)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(iris)
	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	return t
}
