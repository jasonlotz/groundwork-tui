package common

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Tab index constants — must match the screen iota in app/app.go.
const (
	TabDashboard  = 0
	TabCategories = 1
	TabSkills     = 2
	TabMaterials  = 3
	TabFitness    = 4
	TabActivity   = 5
	TabSettings   = 6
)

// tabDef describes one tab entry.
type tabDef struct {
	key    string // single letter, lowercase
	prefix string // text before the key letter (empty for most tabs)
	suffix string // text after the key letter
}

var tabDefs = []tabDef{
	{"d", "", "ashboard"},
	{"c", "", "ategories"},
	{"s", "", "kills"},
	{"m", "", "aterials"},
	{"f", "", "itness"},
	{"a", "", "ctivity"},
	{"i", "Sett", "ngs"},
}

var (
	// Active tab: top+left+right border in highlight color, no bottom border —
	// visually "sits on" the rule beneath it.
	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHighlight).
			Border(lipgloss.Border{
			Top:         "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "╯",
			BottomRight: "╰",
		}).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(false).
		BorderForeground(ColorHighlight).
		PaddingLeft(1).
		PaddingRight(1)

	// Inactive tab: flat, dim — no border.
	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(ColorDim).
				PaddingLeft(1).
				PaddingRight(1).
				MarginTop(1) // push down 1 line so tops align with active tab border

	// Underlined key letter styles.
	tabKeyActiveStyle   = lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true).Underline(true)
	tabKeyInactiveStyle = lipgloss.NewStyle().Foreground(ColorMuted).Underline(true)

	// Suffix text styles.
	tabSuffixActiveStyle   = lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true)
	tabSuffixInactiveStyle = lipgloss.NewStyle().Foreground(ColorDim)

	tabRuleStyle = lipgloss.NewStyle().Foreground(ColorBorder)
)

// rebuildTabStyles recreates all tab bar styles from the current palette.
// Called by ApplyTheme() after a theme switch.
func rebuildTabStyles() {
	tabActiveStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorHighlight).
		Border(lipgloss.Border{
			Top:         "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "╯",
			BottomRight: "╰",
		}).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(false).
		BorderForeground(ColorHighlight).
		PaddingLeft(1).
		PaddingRight(1)
	tabInactiveStyle = lipgloss.NewStyle().
		Foreground(ColorDim).
		PaddingLeft(1).
		PaddingRight(1).
		MarginTop(1)
	tabKeyActiveStyle = lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true).Underline(true)
	tabKeyInactiveStyle = lipgloss.NewStyle().Foreground(ColorMuted).Underline(true)
	tabSuffixActiveStyle = lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true)
	tabSuffixInactiveStyle = lipgloss.NewStyle().Foreground(ColorDim)
	tabRuleStyle = lipgloss.NewStyle().Foreground(ColorBorder)
}

// RenderTabBar renders a tab bar with a rule beneath (3 lines total:
// top border row, label row, rule row — but active tab has no bottom border
// so the rule acts as its floor).
func RenderTabBar(activeTab int, width int) string {
	var parts []string
	for i, t := range tabDefs {
		active := i == activeTab
		keyStyle := tabKeyInactiveStyle
		suffixStyle := tabSuffixInactiveStyle
		if active {
			keyStyle = tabKeyActiveStyle
			suffixStyle = tabSuffixActiveStyle
		}
		keyText := strings.ToUpper(t.key)
		if t.prefix != "" {
			keyText = t.key // don't capitalize mid-word keys
		}
		label := suffixStyle.Render(t.prefix) + keyStyle.Render(keyText) + suffixStyle.Render(t.suffix)
		if active {
			parts = append(parts, tabActiveStyle.Render(label))
		} else {
			parts = append(parts, tabInactiveStyle.Render(label))
		}
	}

	// Join tabs horizontally, aligned to bottom so the rule sits flush.
	row := lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)

	rule := tabRuleStyle.Render(strings.Repeat("─", width))

	return row + "\n" + rule
}
