package common

import "github.com/charmbracelet/lipgloss"

// tailwindToHex maps Tailwind CSS background class names (e.g. "bg-violet-300")
// to their approximate hex color values for terminal rendering.
// Only the palette used by Groundwork is included.
var tailwindToHex = map[string]string{
	// Slate
	"bg-slate-300": "#CBD5E1",
	"bg-slate-400": "#94A3B8",
	"bg-slate-500": "#64748B",
	// Gray
	"bg-gray-300": "#D1D5DB",
	"bg-gray-400": "#9CA3AF",
	"bg-gray-500": "#6B7280",
	// Zinc
	"bg-zinc-300": "#D4D4D8",
	"bg-zinc-400": "#A1A1AA",
	// Red
	"bg-red-300": "#FCA5A5",
	"bg-red-400": "#F87171",
	"bg-red-500": "#EF4444",
	"bg-red-600": "#DC2626",
	// Orange
	"bg-orange-300": "#FED7AA",
	"bg-orange-400": "#FB923C",
	"bg-orange-500": "#F97316",
	// Amber
	"bg-amber-300": "#FCD34D",
	"bg-amber-400": "#FBBF24",
	"bg-amber-500": "#F59E0B",
	// Yellow
	"bg-yellow-300": "#FDE047",
	"bg-yellow-400": "#FACC15",
	// Lime
	"bg-lime-300": "#BEF264",
	"bg-lime-400": "#A3E635",
	// Green
	"bg-green-300": "#86EFAC",
	"bg-green-400": "#4ADE80",
	"bg-green-500": "#22C55E",
	"bg-green-600": "#16A34A",
	// Teal
	"bg-teal-300": "#5EEAD4",
	"bg-teal-400": "#2DD4BF",
	"bg-teal-500": "#14B8A6",
	// Cyan
	"bg-cyan-300": "#67E8F9",
	"bg-cyan-400": "#22D3EE",
	// Sky
	"bg-sky-300": "#7DD3FC",
	"bg-sky-400": "#38BDF8",
	"bg-sky-500": "#0EA5E9",
	// Blue
	"bg-blue-300": "#93C5FD",
	"bg-blue-400": "#60A5FA",
	"bg-blue-500": "#3B82F6",
	"bg-blue-600": "#2563EB",
	// Indigo
	"bg-indigo-300": "#A5B4FC",
	"bg-indigo-400": "#818CF8",
	"bg-indigo-500": "#6366F1",
	// Violet
	"bg-violet-300": "#C4B5FD",
	"bg-violet-400": "#A78BFA",
	"bg-violet-500": "#8B5CF6",
	"bg-violet-600": "#7C3AED",
	// Purple
	"bg-purple-300": "#D8B4FE",
	"bg-purple-400": "#C084FC",
	"bg-purple-500": "#A855F7",
	// Fuchsia
	"bg-fuchsia-300": "#F0ABFC",
	"bg-fuchsia-400": "#E879F9",
	// Pink
	"bg-pink-300": "#F9A8D4",
	"bg-pink-400": "#F472B6",
	"bg-pink-500": "#EC4899",
	// Rose
	"bg-rose-300": "#FDA4AF",
	"bg-rose-400": "#FB7185",
	"bg-rose-500": "#F43F5E",
}

// TailwindToLipgloss converts a Tailwind CSS background class (e.g. "bg-violet-300")
// to a lipgloss.Color. Returns ColorSubtle if the class is not recognised.
func TailwindToLipgloss(class string) lipgloss.Color {
	if hex, ok := tailwindToHex[class]; ok {
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
