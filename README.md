# groundwork-tui

A terminal UI for [Groundwork](https://groundwork.lotztech.com) — built with Go and the [Charm](https://charm.sh) framework.

## Requirements

- Go 1.21+
- A running Groundwork instance
- A personal API key (generate one at Settings → API Keys in the web app)

## Install

```bash
go install github.com/jasonlotz/groundwork-tui/cmd/groundwork-tui@latest
```

Or build locally:

```bash
git clone https://github.com/jasonlotz/groundwork-tui
cd groundwork-tui
go build -o groundwork-tui ./cmd/groundwork-tui
```

## First run

On first launch you'll be prompted for:

- **Base URL** — your Groundwork instance (defaults to `https://groundwork.lotztech.com`)
- **API Key** — your personal key from Settings → API Keys

Config is saved to `~/.config/groundwork-tui/config.toml`.

## Key bindings

### Dashboard

| Key       | Action                    |
| --------- | ------------------------- |
| `j` / `k` | Navigate active materials |
| `l`       | Log progress              |
| `m`       | Materials screen          |
| `s`       | Skills screen             |
| `p`       | Progress log              |
| `r`       | Refresh                   |
| `q`       | Quit                      |

### All other screens

| Key         | Action            |
| ----------- | ----------------- |
| `j` / `k`   | Navigate          |
| `r`         | Refresh           |
| `esc` / `q` | Back to dashboard |

## Tech stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — styling
- [Bubbles](https://github.com/charmbracelet/bubbles) — components
- [Huh](https://github.com/charmbracelet/huh) — forms
- [BurntSushi/toml](https://github.com/BurntSushi/toml) — config file
