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
	cfg       *config.Config
	confirmed *bool // points to the Confirm field's bound value
	cancelled bool
	done      bool
}

// New returns a new setup wizard model pre-populated with any existing config.
func New(cfg *config.Config) Model {
	if cfg == nil {
		cfg = &config.Config{BaseURL: config.DefaultBaseURL}
	}

	baseURL := cfg.BaseURL
	apiKey := cfg.APIKey
	var confirmed bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("groundwork-tui setup").
				Description("Configure your connection to a Groundwork instance.\n\nGenerate an API key at Settings → API Keys in the web app."),

			huh.NewInput().
				Title("Base URL").
				Description("URL of your Groundwork instance").
				Placeholder(config.DefaultBaseURL).
				Value(&baseURL),

			huh.NewInput().
				Title("API Key").
				Description("Your personal API key (kept secret in ~/.config/groundwork-tui/config.toml)").
				Placeholder("gw_...").
				EchoMode(huh.EchoModePassword).
				Value(&apiKey),

			huh.NewConfirm().
				Title("Save configuration?").
				Affirmative("Save").
				Negative("Cancel").
				Value(&confirmed),
		),
	).WithTheme(forms.ActiveTheme)

	return Model{
		form:      form,
		cfg:       &config.Config{BaseURL: baseURL, APIKey: apiKey},
		confirmed: &confirmed,
	}
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		// huh updates the bound pointers directly; check confirmed before saving.
		m.done = true
		if m.confirmed == nil || !*m.confirmed {
			m.cancelled = true
		}
		return m, tea.Quit
	}

	return m, cmd
}

func (m Model) View() string {
	if m.done {
		return ""
	}
	return m.form.View()
}

// Cancelled reports whether the user cancelled setup.
func (m Model) Cancelled() bool {
	return m.cancelled
}

// Config returns the configured values after the form completes.
func (m Model) Config() *config.Config {
	return m.cfg
}
