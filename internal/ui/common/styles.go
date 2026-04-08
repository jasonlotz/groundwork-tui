// Package common provides shared TUI styles and components.
package common

import (
	"fmt"
	"strings"

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
	ColorCardBg    = lipgloss.Color("#1F1635") // very dark violet tint for stat cards
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

	// BorderStyle is used for KPI stat cards — subtle background tint.
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Background(ColorCardBg).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	StatLabelStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(ColorCardBg)

	StatValueStyle = lipgloss.NewStyle().
			Bold(true).
			Background(ColorCardBg)

	// CompletedNameStyle renders completed material names with strikethrough.
	CompletedNameStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Strikethrough(true)

	// InactiveNameStyle renders inactive material names in italic muted text.
	InactiveNameStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Italic(true)

	// ArchivedNameStyle renders archived item names in italic muted text.
	ArchivedNameStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Italic(true)

	// DefaultNameStyle is a plain unstyled style used as the default for list row names.
	DefaultNameStyle = lipgloss.NewStyle()

	// CompletedStatusStyle renders a "done" status label in the primary violet color.
	CompletedStatusStyle = lipgloss.NewStyle().Foreground(ColorPrimary)

	// SpinnerStyle is the foreground color applied to the loading spinner.
	SpinnerStyle = lipgloss.NewStyle().Foreground(ColorHighlight)

	// TableBorderStyle styles the separator lines in lipgloss/table renders.
	TableBorderStyle = lipgloss.NewStyle().Foreground(ColorBorder)

	// TableHeaderStyle styles the header row in lipgloss/table renders.
	TableHeaderStyle = lipgloss.NewStyle().Foreground(ColorMuted).Bold(true)

	// TableSelectedStyle highlights the selected row in lipgloss/table renders.
	TableSelectedStyle = lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true)

	// TableCellStyle is the default cell style in lipgloss/table renders.
	TableCellStyle = lipgloss.NewStyle().Foreground(ColorSubtle)
)

// RenderTitle renders a decorative title with a violet rule beneath it.
func RenderTitle(s string, width int) string {
	if width < 1 {
		width = 40
	}
	title := TitleStyle.Render(s)
	rule := lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", width))
	return fmt.Sprintf("%s\n%s", title, rule)
}

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
