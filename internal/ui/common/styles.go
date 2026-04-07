// Package common provides shared TUI styles and components.
package common

import "github.com/charmbracelet/lipgloss"

// Palette — a small set of consistent colors.
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // violet-600
	ColorMuted     = lipgloss.Color("#6B7280") // gray-500
	ColorSuccess   = lipgloss.Color("#16A34A") // green-600
	ColorWarning   = lipgloss.Color("#D97706") // amber-600
	ColorDanger    = lipgloss.Color("#DC2626") // red-600
	ColorBorder    = lipgloss.Color("#374151") // gray-700
	ColorSubtle    = lipgloss.Color("#9CA3AF") // gray-400
	ColorHighlight = lipgloss.Color("#A78BFA") // violet-400
)

// Styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	SectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight).
			MarginTop(1)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	DangerStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	StatLabelStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StatValueStyle = lipgloss.NewStyle().
			Bold(true)
)

// ProgressBar renders a simple ASCII progress bar.
// width is the total character width of the bar.
func ProgressBar(pct float64, width int) string {
	if width <= 0 {
		width = 20
	}
	if pct > 1.0 {
		pct = 1.0
	}
	if pct < 0 {
		pct = 0
	}
	filled := int(float64(width) * pct)
	empty := width - filled

	bar := lipgloss.NewStyle().Foreground(ColorSuccess).Render(repeatChar("█", filled)) +
		lipgloss.NewStyle().Foreground(ColorBorder).Render(repeatChar("░", empty))
	return bar
}

func repeatChar(ch string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += ch
	}
	return out
}

// StatCard renders a small labeled stat block.
func StatCard(label, value string) string {
	return StatLabelStyle.Render(label) + "\n" + StatValueStyle.Render(value)
}

// KeyHelp renders a single key binding hint: "key  desc".
func KeyHelp(key, desc string) string {
	k := lipgloss.NewStyle().Foreground(ColorHighlight).Render(key)
	return k + "  " + MutedStyle.Render(desc)
}
