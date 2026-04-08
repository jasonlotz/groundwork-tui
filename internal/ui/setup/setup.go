// Package setup provides the first-run configuration wizard.
package setup

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/jasonlotz/groundwork-tui/internal/config"
	"github.com/jasonlotz/groundwork-tui/internal/ui/forms"
)

// Model is the Bubble Tea model for the setup wizard.
type Model struct {
	form      *huh.Form
	baseURL   string
	apiKey    string
	confirmed bool
	cancelled bool
	done      bool
}

// New returns a new setup wizard model pre-populated with any existing config.
// Returns *Model so that huh can bind directly to the struct fields.
func New(cfg *config.Config) *Model {
	if cfg == nil {
		cfg = &config.Config{BaseURL: config.DefaultBaseURL}
	}

	m := &Model{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("groundwork-tui setup").
				Description("Configure your connection to a Groundwork instance.\n\nGenerate an API key at Settings → API Keys in the web app."),

			huh.NewInput().
				Title("Base URL").
				Description("URL of your Groundwork instance").
				Placeholder(config.DefaultBaseURL).
				Value(&m.baseURL),

			huh.NewInput().
				Title("API Key").
				Description("Your personal API key (kept secret in ~/.config/groundwork-tui/config.toml)").
				Placeholder("gw_...").
				EchoMode(huh.EchoModePassword).
				Value(&m.apiKey),

			huh.NewConfirm().
				Title("Save configuration?").
				Affirmative("Save").
				Negative("Cancel").
				Value(&m.confirmed),
		),
	).WithTheme(forms.ActiveTheme)

	return m
}

func (m *Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.cancelled = true
			return m, tea.Quit
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		m.done = true
		if !m.confirmed {
			m.cancelled = true
		}
		return m, tea.Quit
	}

	return m, cmd
}

func (m *Model) View() string {
	if m.done {
		return ""
	}
	return m.form.View()
}

// Cancelled reports whether the user cancelled setup.
func (m *Model) Cancelled() bool {
	return m.cancelled
}

// Config returns the configured values after the form completes.
func (m *Model) Config() *config.Config {
	return &config.Config{BaseURL: m.baseURL, APIKey: m.apiKey}
}
