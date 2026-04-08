# groundwork-tui

A terminal UI for [Groundwork](https://groundwork.lotztech.com) — built with Go and the [Charm](https://charm.sh) framework.

## Requirements

- A running Groundwork instance
- A personal API key (generate one at **Settings → API Keys** in the web app)
- Go 1.22+ (see below if you don't have it)

## Installing Go

### macOS

The easiest way is via [Homebrew](https://brew.sh):

```bash
brew install go
```

Or download the macOS installer from [go.dev/dl](https://go.dev/dl/) and follow the prompts. After installing, open a new terminal and verify:

```bash
go version
```

### Windows

Download the Windows installer (`.msi`) from [go.dev/dl](https://go.dev/dl/) and run it. It will install Go and add it to your `PATH` automatically. Open a new terminal and verify:

```
go version
```

> **Windows terminal note:** Run groundwork-tui in [Windows Terminal](https://aka.ms/terminal) or any modern terminal that supports ANSI color codes. The classic `cmd.exe` prompt will not render colors correctly. PowerShell in Windows Terminal works well.

## Install groundwork-tui

```bash
go install github.com/jasonlotz/groundwork-tui/cmd/groundwork-tui@latest
```

This downloads and compiles the binary into your Go bin directory (`~/go/bin` on macOS/Linux, `%USERPROFILE%\go\bin` on Windows). Make sure that directory is in your `PATH` — the Go installer adds it automatically on Windows; on macOS you may need to add it to your shell profile:

```bash
# Add to ~/.zshrc or ~/.bashrc
export PATH="$PATH:$HOME/go/bin"
```

Or build from source:

```bash
git clone https://github.com/jasonlotz/groundwork-tui
cd groundwork-tui
go build -o groundwork-tui ./cmd/groundwork-tui
```

## First run

Run the binary:

```bash
groundwork-tui
```

On first launch you'll be prompted for:

- **Base URL** — your Groundwork instance URL (defaults to `https://groundwork.lotztech.com`)
- **API Key** — your personal key from **Settings → API Keys**

Config is saved to:

- **macOS / Linux:** `~/.config/groundwork-tui/config.toml`
- **Windows:** `%APPDATA%\groundwork-tui\config.toml`

To reconfigure, delete the config file and relaunch.

## Key bindings

### Dashboard

| Key       | Action                        |
| --------- | ----------------------------- |
| `j` / `k` | Navigate active materials     |
| `enter`   | Open material detail          |
| `l`       | Log progress on selected item |
| `c`       | Categories screen             |
| `s`       | Skills screen                 |
| `m`       | Materials screen              |
| `p`       | Progress log                  |
| `r`       | Refresh                       |
| `q`       | Quit                          |

### Materials / Skill detail

| Key         | Action                        |
| ----------- | ----------------------------- |
| `j` / `k`   | Navigate                      |
| `enter`     | Open material detail          |
| `l`         | Log progress on selected item |
| `r`         | Refresh                       |
| `esc` / `q` | Back                          |

### All other screens

| Key         | Action   |
| ----------- | -------- |
| `j` / `k`   | Navigate |
| `r`         | Refresh  |
| `esc` / `q` | Back     |

## Tech stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — styling
- [Bubbles](https://github.com/charmbracelet/bubbles) — components
- [Huh](https://github.com/charmbracelet/huh) — forms
- [BurntSushi/toml](https://github.com/BurntSushi/toml) — config file
