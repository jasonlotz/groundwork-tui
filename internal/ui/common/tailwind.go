package common

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// tailwindToHex maps Tailwind CSS background class names (e.g. "bg-violet-300")
// to their approximate hex color values for terminal rendering.
// Covers exactly the palette used by the Groundwork web app (100/200/300 shades).
var tailwindToHex = map[string]string{
	// Slate
	"bg-slate-100": "#F1F5F9",
	"bg-slate-200": "#E2E8F0",
	"bg-slate-300": "#CBD5E1",
	// Red
	"bg-red-100": "#FEE2E2",
	"bg-red-200": "#FECACA",
	"bg-red-300": "#FCA5A5",
	// Orange
	"bg-orange-100": "#FFEDD5",
	"bg-orange-200": "#FED7AA",
	"bg-orange-300": "#FDBA74",
	// Amber
	"bg-amber-100": "#FEF3C7",
	"bg-amber-200": "#FDE68A",
	"bg-amber-300": "#FCD34D",
	// Green
	"bg-green-100": "#DCFCE7",
	"bg-green-200": "#BBF7D0",
	"bg-green-300": "#86EFAC",
	// Teal
	"bg-teal-100": "#CCFBF1",
	"bg-teal-200": "#99F6E4",
	"bg-teal-300": "#5EEAD4",
	// Blue
	"bg-blue-100": "#DBEAFE",
	"bg-blue-200": "#BFDBFE",
	"bg-blue-300": "#93C5FD",
	// Indigo
	"bg-indigo-100": "#E0E7FF",
	"bg-indigo-200": "#C7D2FE",
	"bg-indigo-300": "#A5B4FC",
	// Purple
	"bg-purple-100": "#F3E8FF",
	"bg-purple-200": "#E9D5FF",
	"bg-purple-300": "#D8B4FE",
	// Pink
	"bg-pink-100": "#FCE7F3",
	"bg-pink-200": "#FBCFE8",
	"bg-pink-300": "#F9A8D4",
}

// extractBgClass pulls the first "bg-*" token from a (possibly multi-class)
// Tailwind string like "bg-violet-300 text-violet-900 dark:bg-violet-800 ...".
// Returns the input unchanged if it contains no spaces (already a single class).
func extractBgClass(class string) string {
	for _, token := range strings.Fields(class) {
		if strings.HasPrefix(token, "bg-") && !strings.HasPrefix(token, "bg-opacity") {
			return token
		}
	}
	return class
}

// TailwindToLipgloss converts a Tailwind CSS background class (e.g. "bg-violet-300"),
// or a multi-class string as stored by the web app, to a lipgloss.Color.
// Returns ColorSubtle if the class is not recognised.
func TailwindToLipgloss(class string) lipgloss.Color {
	key := extractBgClass(class)
	if hex, ok := tailwindToHex[key]; ok {
		return lipgloss.Color(hex)
	}
	return ColorSubtle
}

// ColorDot renders a small colored "●" dot using the given Tailwind class.
// Returns a plain muted dot if the class is empty or unrecognised.
func ColorDot(tailwindClass string) string {
	if tailwindClass == "" {
		return MutedStyle.Render("●")
	}
	color := TailwindToLipgloss(tailwindClass)
	return lipgloss.NewStyle().Foreground(color).Render("●")
}

// ColoredName renders text in the color matching the given Tailwind class.
// Falls back to the provided fallback style if the class is empty or unrecognised.
func ColoredName(tailwindClass, text string, fallback lipgloss.Style) string {
	if tailwindClass == "" {
		return fallback.Render(text)
	}
	color := TailwindToLipgloss(tailwindClass)
	if color == ColorSubtle {
		return fallback.Render(text)
	}
	return lipgloss.NewStyle().Foreground(color).Render(text)
}
