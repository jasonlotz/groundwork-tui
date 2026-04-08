// Package common provides shared TUI styles and components.
package common

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

	// BorderStyle is used for KPI stat cards.
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	// PopupStyle is the border/padding style for inline overlay popups (e.g. log form).
	PopupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2).
			Width(60)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	StatLabelStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StatValueStyle = lipgloss.NewStyle().
			Bold(true)

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

	// titleRuleStyle styles the horizontal rule drawn beneath the title.
	titleRuleStyle = lipgloss.NewStyle().Foreground(ColorBorder)
)

// RenderTitle renders a decorative title with a horizontal rule beneath it.
func RenderTitle(s string, width int) string {
	if width < 1 {
		width = 40
	}
	title := TitleStyle.Render(s)
	rule := titleRuleStyle.Render(strings.Repeat("─", width))
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
// width overrides the bar's width if > 0; pass 0 to use the model's existing width.
// This is a stateless render — no animation state required.
func RenderBar(p progress.Model, pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	if width > 0 {
		p.Width = width
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

// ErrorView returns the standard error message string, word-wrapped to width.
func ErrorView(err error, width int) string {
	w := width - 4
	if w < 40 {
		w = 40
	}
	return DangerStyle.Copy().Width(w).Render("\n  Error: " + err.Error() + "\n\n  Press r to retry, esc to go back.")
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

// ClampBarWidth computes a progress bar width from the terminal width,
// subtracting a fixed margin and clamping to the range [20, 60].
// The margin of 30 accounts for cursor, dot, labels, and spacing.
func ClampBarWidth(termWidth int) int {
	w := termWidth - 30
	if w < 20 {
		return 20
	}
	if w > 60 {
		return 60
	}
	return w
}

// PaceFraction returns the expected weekly progress fraction based on the current day.
// Mon=1/7, Tue=2/7, … Sun=7/7. Matches the pace logic used in the web app.
func PaceFraction() float64 {
	// time.Weekday: Sun=0, Mon=1, …, Sat=6. We remap to Mon=1 … Sun=7.
	d := int(time.Now().Weekday())
	if d == 0 {
		d = 7
	}
	return float64(d) / 7.0
}

// FormatProjectedDate parses a "YYYY-MM-DD" projected end date string and
// returns it in "Jan 2, 2006" format (e.g. "Mar 23, 2026").
// Returns the original string unchanged if parsing fails.
func FormatProjectedDate(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return t.Format("Jan 2, 2006")
}

// RenderWeeklyBar renders a fixed-width bar for weekly goal progress.
// pct is the fill fraction (0.0–1.0), pacePct is where the pace tick falls (0.0–1.0).
// The bar color is green/yellow/red based on whether pct is ahead of, near, or behind pace.
func RenderWeeklyBar(width int, pct, pacePct float64) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}

	// Choose fill color based on pace comparison (matches web app thresholds).
	var fillColor lipgloss.Color
	switch {
	case pct >= pacePct:
		fillColor = ColorSuccess
	case pct >= pacePct-0.20:
		fillColor = ColorWarning
	default:
		fillColor = ColorDanger
	}
	fillStyle := lipgloss.NewStyle().Foreground(fillColor)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorBorder)

	filled := int(pct * float64(width))

	var buf strings.Builder
	for i := range width {
		if i < filled {
			buf.WriteString(fillStyle.Render("█"))
		} else {
			buf.WriteString(emptyStyle.Render("░"))
		}
	}
	return buf.String()
}

// ValidateDate returns an error if s is not a valid YYYY-MM-DD date.
func ValidateDate(s string) error {
	if s == "" {
		return fmt.Errorf("date is required")
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return fmt.Errorf("must be YYYY-MM-DD (e.g. %s)", time.Now().Format("2006-01-02"))
	}
	return nil
}

// ValidateUnits returns an error if s is not a positive number.
func ValidateUnits(s string) error {
	if s == "" {
		return fmt.Errorf("units is required")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("must be a number (e.g. 1, 2.5)")
	}
	if v <= 0 {
		return fmt.Errorf("must be greater than 0")
	}
	return nil
}

// TodayString returns today's date as a YYYY-MM-DD string.
func TodayString() string {
	return time.Now().Format("2006-01-02")
}
