// Package settings provides the settings screen: theme picker.
package settings

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
	"github.com/jasonlotz/groundwork-tui/internal/ui/theme"
)

// Model is the Bubble Tea model for the settings screen.
type Model struct {
	client      interface{} // retained for interface compatibility
	themeCursor int
	width       int
	height      int
	keys        common.SimpleKeyMap
}

func New(client interface{}) Model {
	// Start cursor on the currently active theme.
	cursor := 0
	for i, t := range theme.All {
		if t.Name == theme.Active.Name {
			cursor = i
			break
		}
	}
	return Model{
		client:      client,
		themeCursor: cursor,
		keys: common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("enter", "select theme"),
			common.KB("esc", "back"),
		}},
	}
}

// HasOverlay reports whether a form overlay is currently active.
func (m Model) HasOverlay() bool { return false }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return common.GoBackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.themeCursor < len(theme.All)-1 {
				m.themeCursor++
			}
		case "k", "up":
			if m.themeCursor > 0 {
				m.themeCursor--
			}
		case "enter", " ":
			selected := theme.All[m.themeCursor]
			name := selected.Name
			return m, func() tea.Msg { return common.ThemeChangedMsg{ThemeName: name} }
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(common.RenderTitle("Settings", m.width))
	b.WriteString("\n")

	for i, t := range theme.All {
		active := t.Name == theme.Active.Name
		selected := i == m.themeCursor

		prefix := "  "
		if selected {
			prefix = common.SelectedStyle.Render("▶ ")
		}

		var nameStr string
		switch {
		case active:
			nameStr = common.SuccessStyle.Render("✓ " + t.Name)
		case selected:
			nameStr = common.SelectedStyle.Render(t.Name)
		default:
			nameStr = common.DimStyle.Render(t.Name)
		}

		swatch := renderSwatch(t)
		b.WriteString(fmt.Sprintf("%s%-14s  %s\n", prefix, nameStr, swatch))
	}

	b.WriteString("\n")
	b.WriteString(common.RenderHelp(m.keys, m.width))
	return b.String()
}

// renderSwatch renders a row of colored blocks showing the theme palette.
func renderSwatch(t theme.AppTheme) string {
	swatchColors := []lipgloss.Color{
		t.Colors.Primary,
		t.Colors.Highlight,
		t.Colors.Success,
		t.Colors.Warning,
		t.Colors.Danger,
		t.Colors.Dim,
	}
	var parts []string
	for _, c := range swatchColors {
		parts = append(parts, lipgloss.NewStyle().Foreground(c).Render("■"))
	}
	return strings.Join(parts, "")
}
