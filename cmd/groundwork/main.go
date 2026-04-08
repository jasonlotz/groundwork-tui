// Package main is the entry point for groundwork-tui.
package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/config"
	"github.com/jasonlotz/groundwork-tui/internal/ui/app"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
	"github.com/jasonlotz/groundwork-tui/internal/ui/setup"
	"github.com/jasonlotz/groundwork-tui/internal/ui/theme"
)

func main() {
	cfg, err := config.Load()
	if errors.Is(err, config.ErrNotFound) || cfg.APIKey == "" {
		// First run — launch the setup wizard.
		p := tea.NewProgram(setup.New(cfg), tea.WithAltScreen())
		m, err := p.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error during setup:", err)
			os.Exit(1)
		}
		sm, ok := m.(*setup.Model)
		if !ok || sm.Cancelled() {
			fmt.Println("Setup cancelled. Run groundwork-tui again to configure.")
			os.Exit(0)
		}
		cfg = sm.Config()
		if err := config.Save(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "failed to save config:", err)
			os.Exit(1)
		}
	} else if err != nil {
		fmt.Fprintln(os.Stderr, "error loading config:", err)
		os.Exit(1)
	}

	client := api.New(cfg.BaseURL, cfg.APIKey)

	// Apply saved theme before launching.
	if cfg.Theme != "" {
		theme.SetActive(cfg.Theme)
		common.ApplyTheme()
	}

	p := tea.NewProgram(app.New(client, cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
