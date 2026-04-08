package common

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// Binding is a re-export alias so callers only need to import common.
type Binding = key.Binding

// KB creates a key.Binding with a single key label and description.
func KB(keyStr, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keyStr), key.WithHelp(keyStr, desc))
}

// KBKeys creates a key.Binding with multiple underlying keys but a single display label.
func KBKeys(display, desc string, keys ...string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(display, desc))
}

// SimpleKeyMap implements help.KeyMap for a flat list of bindings.
// The first N bindings are shown in short mode; all in full mode.
type SimpleKeyMap struct {
	Bindings []key.Binding
	ShortN   int // how many bindings to show in short (collapsed) mode; 0 = all
}

func (k SimpleKeyMap) ShortHelp() []key.Binding {
	if k.ShortN <= 0 || k.ShortN >= len(k.Bindings) {
		return k.Bindings
	}
	return k.Bindings[:k.ShortN]
}

func (k SimpleKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.Bindings}
}

// NewHelp returns a help.Model styled with the project palette.
func NewHelp() help.Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(ColorHighlight)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(ColorMuted)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(ColorBorder)
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(ColorHighlight)
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(ColorMuted)
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(ColorBorder)
	h.Styles.Ellipsis = lipgloss.NewStyle().Foreground(ColorBorder)
	return h
}
