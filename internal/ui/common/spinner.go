package common

import (
	"github.com/charmbracelet/bubbles/spinner"
)

// NewSpinner returns a spinner.Model configured with the project palette.
func NewSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle
	return s
}

// SpinnerView renders the loading state with an animated spinner.
func SpinnerView(s spinner.Model) string {
	return "\n  " + s.View() + DimStyle.Render(" Loading…")
}
