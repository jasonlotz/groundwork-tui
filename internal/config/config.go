// Package config handles loading and saving the groundwork-tui configuration
// from ~/.config/groundwork-tui/config.toml.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	appName    = "groundwork-tui"
	configFile = "config.toml"
)

// Config holds all user configuration for the TUI.
type Config struct {
	BaseURL string `toml:"base_url"`
	APIKey  string `toml:"api_key"`
}

// DefaultBaseURL is the production Groundwork instance.
const DefaultBaseURL = "https://groundwork.lotztech.com"

// configDir returns ~/.config/groundwork-tui, creating it if needed.
// We use ~/.config explicitly (XDG convention) rather than os.UserConfigDir()
// which returns ~/Library/Application Support on macOS.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", appName), nil
}

// Path returns the full path to the config file.
func Path() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFile), nil
}

// Load reads the config file. Returns a default config and ErrNotFound if
// the file does not exist yet.
var ErrNotFound = errors.New("config file not found")

func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return &Config{BaseURL: DefaultBaseURL}, ErrNotFound
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config: %w", err)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	return &cfg, nil
}

// Save writes the config to disk, creating the directory if needed.
func Save(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	path := filepath.Join(dir, configFile)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot write config file: %w", err)
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(cfg)
}
