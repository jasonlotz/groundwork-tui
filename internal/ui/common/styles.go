// Package common provides shared TUI styles and components.
package common

import (
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

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

// NewProgressBar returns a bubbles/progress model styled with the project palette.
// width is the total character width of the bar (excluding the percentage label).
func NewProgressBar(width int) progress.Model {
	if width <= 0 {
		width = 20
	}
	p := progress.New(
		progress.WithGradient(string(ColorPrimary), string(ColorHighlight)),
		progress.WithFillCharacters('█', '░'),
		progress.WithoutPercentage(),
		progress.WithWidth(width),
	)
	p.EmptyColor = string(ColorBorder)
	return p
}

// RenderBar renders a progress bar for the given percentage (0.0–1.0) using ViewAs.
// This is a stateless render — no animation state required.
func RenderBar(p progress.Model, pct float64) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	return p.ViewAs(pct)
}

// Truncate shortens s to at most n runes, adding "…" if truncated.
// Uses rune-aware slicing so multi-byte characters are handled correctly.
func Truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

// VisibleWindow computes the start/end slice indices to keep cursor visible
// within a window of size height over a list of total items.
func VisibleWindow(cursor, total, height int) (start, end int) {
	if total <= height {
		return 0, total
	}
	start = cursor - height/2
	if start < 0 {
		start = 0
	}
	end = start + height
	if end > total {
		end = total
		start = end - height
	}
	return start, end
}

// LoadingView returns the standard loading placeholder string.
func LoadingView() string {
	return MutedStyle.Render("\n  Loading…")
}

// ErrorView returns the standard error message string.
func ErrorView(err error) string {
	return DangerStyle.Render("\n  Error: " + err.Error() + "\n\n  Press r to retry, esc to go back.")
}

// RenderKPICards renders a horizontal row of bordered stat cards.
func RenderKPICards(cards []string) string {
	rendered := make([]string, len(cards))
	for i, c := range cards {
		rendered[i] = BorderStyle.Render(c)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
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
