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

> **Windows terminal note:** Run groundwork in [Windows Terminal](https://aka.ms/terminal) or any modern terminal that supports ANSI color codes. The classic `cmd.exe` prompt will not render colors correctly. PowerShell in Windows Terminal works well.

## Install groundwork

```bash
go install github.com/jasonlotz/groundwork-tui/cmd/groundwork@latest
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
go build -o groundwork ./cmd/groundwork
```

## First run

Run the binary:

```bash
groundwork
```

On first launch you'll be prompted for:

- **Base URL** — your Groundwork instance URL (defaults to `https://groundwork.lotztech.com`)
- **API Key** — your personal key from **Settings → API Keys**

Config is saved to:

- **macOS / Linux:** `~/.config/groundwork-tui/config.toml`
- **Windows:** `%APPDATA%\groundwork-tui\config.toml`

To reconfigure, delete the config file and relaunch.

## Key bindings

### Global (all screens)

| Key | Action         |
| --- | -------------- |
| `d` | Dashboard tab  |
| `c` | Categories tab |
| `s` | Skills tab     |
| `m` | Materials tab  |
| `p` | Progress tab   |
| `q` | Quit           |

### Dashboard

| Key       | Action                        |
| --------- | ----------------------------- |
| `j` / `k` | Navigate active materials     |
| `enter`   | Open material detail          |
| `l`       | Log progress on selected item |
| `r`       | Refresh                       |

### Materials list / Skill detail

| Key         | Action                        |
| ----------- | ----------------------------- |
| `j` / `k`   | Navigate                      |
| `enter`     | Open material detail          |
| `l`         | Log progress on selected item |
| `n`         | New material                  |
| `e`         | Edit selected material        |
| `D`         | Delete selected material      |
| `a`         | Toggle active-only filter     |
| `/`         | Search by name, skill, type   |
| `r`         | Refresh                       |
| `esc` / `q` | Back                          |

### Material detail

| Key         | Action              |
| ----------- | ------------------- |
| `j` / `k`   | Scroll progress log |
| `l`         | Log progress        |
| `e`         | Edit material       |
| `D`         | Delete material     |
| `r`         | Refresh             |
| `esc` / `q` | Back                |

### Categories

| Key         | Action                                      |
| ----------- | ------------------------------------------- |
| `j` / `k`   | Navigate                                    |
| `enter`     | Open category                               |
| `n`         | New category                                |
| `e`         | Edit selected category                      |
| `A`         | Archive / unarchive selected category       |
| `D`         | Delete selected category (must be archived) |
| `a`         | Toggle show/hide archived                   |
| `r`         | Refresh                                     |
| `esc` / `q` | Back                                        |

### Skills

| Key         | Action                                   |
| ----------- | ---------------------------------------- |
| `j` / `k`   | Navigate                                 |
| `enter`     | Open skill                               |
| `n`         | New skill                                |
| `e`         | Edit selected skill                      |
| `A`         | Archive / unarchive selected skill       |
| `D`         | Delete selected skill (must be archived) |
| `a`         | Toggle show/hide archived                |
| `r`         | Refresh                                  |
| `esc` / `q` | Back                                     |

### Category detail (skills list)

| Key         | Action                                   |
| ----------- | ---------------------------------------- |
| `j` / `k`   | Navigate skills                          |
| `enter`     | Open skill                               |
| `n`         | New skill                                |
| `e`         | Edit selected skill                      |
| `A`         | Archive / unarchive selected skill       |
| `D`         | Delete selected skill (must be archived) |
| `r`         | Refresh                                  |
| `esc` / `q` | Back                                     |

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

---

## Architecture: a reader's guide to the code

This section walks through the codebase in the order that makes sense to read it — from first principles to the full picture. It is aimed at developers who want to understand how everything fits together, or who are learning Go and the Charm framework.

### The Bubble Tea mental model

Before reading any code, it helps to understand the pattern everything is built on. [Bubble Tea](https://github.com/charmbracelet/bubbletea) is a Go TUI framework based on the Elm Architecture. Every interactive component is a **model** — a plain Go struct — that satisfies three methods:

```go
Init()               tea.Cmd        // called once on startup; returns initial async work
Update(tea.Msg)      (tea.Model, tea.Cmd)  // handles events; returns new state + next cmd
View()               string         // renders the current state to a string
```

A `tea.Cmd` is just a `func() tea.Msg` — a function that Bubble Tea will run in a goroutine and deliver the result back as the next message. This is the entire concurrency model: you never touch channels or goroutines directly. All state is immutable per-update; `Update` receives a copy of the model and returns a new one.

That's it. The whole framework flows from this pattern. Once you see it in one place, you see it everywhere.

### Start here: `cmd/groundwork/main.go`

The entry point is deliberately thin. It handles two cases:

1. **No config yet** (first run or missing API key): launch a setup wizard as a standalone `tea.Program`, wait for it to finish, save the resulting config to disk, then fall through.
2. **Config exists**: construct an API client and launch the main app.

```go
cfg, err := config.Load()
if errors.Is(err, config.ErrNotFound) || cfg.APIKey == "" {
    p := tea.NewProgram(setup.New(cfg), tea.WithAltScreen())
    m, _ := p.Run()
    // ... save config ...
}
client := api.New(cfg.BaseURL, cfg.APIKey)
p := tea.NewProgram(app.New(client), tea.WithAltScreen())
p.Run()
```

The two programs are mutually exclusive — setup runs first and exits before the main app starts. `tea.WithAltScreen()` switches to the terminal's alternate screen buffer so the TUI doesn't scroll the user's history.

### Config: `internal/config/config.go`

A tiny TOML file with two fields:

```toml
base_url = "https://groundwork.lotztech.com"
api_key  = "your-key-here"
```

Stored at `~/.config/groundwork-tui/config.toml` on macOS/Linux. The code deliberately uses `~/.config` (XDG convention) rather than `~/Library/Application Support` (macOS default from `os.UserConfigDir()`). `config.Load()` returns a sentinel `ErrNotFound` error — not a wrapped I/O error — so `main.go` can distinguish "file doesn't exist yet" from "file exists but is broken".

### The API client: `internal/api/client.go`

This is the boundary between the TUI and the Groundwork web app. It's worth reading carefully because it shows how to speak tRPC from a non-JS client.

**tRPC protocol**

tRPC 11 is just HTTP under the hood:

- **Queries** (read operations): `GET /api/trpc/<procedure>?input=<url-encoded-json>`
- **Mutations** (write operations): `POST /api/trpc/<procedure>` with a JSON body
- In both cases the input is wrapped: `{"json": <your-input>}`
- The response is unwrapped from: `{"result": {"data": {"json": <output>}}}`

The double-wrapping (`json` key inside `json` key) is a tRPC convention for its "SuperJSON" serialization layer.

**Generic plumbing with Go generics**

Two generic functions do all the work:

```go
func query[T any](c *Client, procedure string, input any) (T, error)
func mutation[T any](c *Client, procedure string, input any) (T, error)
```

`query[T]` marshals the input, URL-encodes it, sends a GET, and calls `parseResponse[T]` to unwrap the result into whatever Go type `T` you ask for. `mutation[T]` does the same with a POST body. This means every typed API method is a one-liner:

```go
func (c *Client) GetOverview() (*model.Overview, error) {
    out, err := query[model.Overview](c, "dashboard.getOverview", struct{}{})
    ...
}
```

**Auth** is a single header set on every request in `doRequest`:

```go
req.Header.Set("x-api-key", c.apiKey)
```

### Data models: `internal/model/model.go`

Plain Go structs that mirror the JSON shapes returned by the API. Nothing surprising here — these are what `parseResponse[T]` deserializes into. If you add a new API call, you add the corresponding struct here first.

### The root model: `internal/ui/app/app.go`

This is the architectural center of the application. Read it alongside `main.go`.

`app.Model` owns everything:

```go
type Model struct {
    client         *api.Client
    current        screen          // which screen is showing
    activeTab      screen          // top-level tab (unchanged when drilling into details)
    navStack       []screenState   // navigation history
    dashboard      dashboard.Model
    materialsList  materials.Model
    // ... one field per screen ...
    categoryDetail *categorydetail.Model  // pointer = created on demand
    toast          string
    width, height  int
}
```

Singleton screens (`dashboard`, `materialsList`, etc.) are value fields — they're created once in `New()` and persist for the life of the app. Context-dependent screens (`categoryDetail`, `skillDetail`, `materialDetail`) are pointer fields because they're created fresh each time you navigate into one.

**Navigation with a stack and tab bar**

The `navStack []screenState` is a simple slice used as a stack. `pushScreen(s)` saves a snapshot of the current screen pointers and sets `m.current = s`. `popScreen()` restores the top snapshot. This gives you browser-style back navigation without any routing library.

`activeTab` tracks which top-level tab the user is on. It does not change when drilling into a detail screen (e.g. category → skill → material), so the correct tab stays highlighted the whole time. `switchTab(s)` handles three cases: already on that tab at the top level (no-op), on a detail screen within that tab (pop back to the list), or switching to a different tab (clear the stack, re-init).

The global `d/c/s/m/p` key bindings in `app.Update()` call `switchTab` before delegating to the active screen, so they work from anywhere — including detail screens.

**The Update two-step**

`app.Update()` has a two-phase structure. First it handles **global messages** — things that span screens — in one big type switch. Then it **delegates** to the active screen:

```go
// Phase 1: handle cross-cutting messages
switch msg := msg.(type) {
case common.GoBackMsg:   m.popScreen(); return m, nil
case skills.OpenSkillMsg: // create skill detail, push screen
    ...
}

// Phase 2: delegate to the active screen
switch m.current {
case screenDashboard:
    updated, cmd := m.dashboard.Update(msg)
    m.dashboard = updated.(dashboard.Model)
    return m, cmd
    ...
}
```

This is the key architectural insight: **screens never talk to each other directly**. A screen emits a typed message upward; the root `app.Update()` intercepts it and decides what to do. The `dashboard` package has no idea that `skilldetail` exists.

**WindowSizeMsg forwarding**

`tea.WindowSizeMsg` is only sent once at startup — when `screenDashboard` is the active screen. To ensure every screen model knows the terminal dimensions regardless of when it becomes active, the `WindowSizeMsg` case in `app.Update()` explicitly updates all persistent screen models:

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    if updated, _ := m.dashboard.Update(msg); updated != nil {
        m.dashboard = updated.(dashboard.Model)
    }
    if updated, _ := m.materialsList.Update(msg); updated != nil {
        m.materialsList = updated.(materials.Model)
    }
    // ... same for skillsList, progressList, categoriesList
```

Without this, any screen that was not active at startup would have `height == 0` and compute `visibleItems = 3` regardless of the actual terminal height.

**Tab bar and toast**

`app.View()` always composes the final output in three layers using `lipgloss.JoinVertical`: the tab bar (from `common.RenderTabBar`), the active screen's content, and — when `m.toast != ""` — a toast line below. The toast uses `common.SuccessStyle` for normal messages and `common.DangerStyle` for errors (`m.toastIsErr`). This layering means no screen needs to know about the tab bar or toasts; they render only their own content.

### A typical screen: `internal/ui/dashboard/dashboard.go`

Every screen follows the same pattern. The dashboard is a good one to read because it uses most of the features.

**Exported vs unexported messages**

At the top of the file, notice two categories of message types:

```go
// Exported — sent upward to app.go
type NavigateMsg string
type OpenMaterialMsg struct{ MaterialID string }

// Unexported — stay within this package
type overviewLoadedMsg struct{ data *model.Overview }
type activeMaterialsLoadedMsg struct{ data []model.ActiveMaterial }
```

Exported messages cross a package boundary (child → root). Unexported messages are the private result of an async load — they only need to reach this screen's own `Update`. The log form, category forms, and skill forms are handled entirely within the screen that owns them as overlays — no exported messages needed for those interactions.

**Init and async loading**

```go
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        loadOverview(m.client),
        loadActiveMaterials(m.client),
        m.spinner.Tick,
    )
}
```

`tea.Batch` runs multiple commands concurrently. Two API calls are fired in parallel, and the spinner starts ticking. Each load function is a `tea.Cmd` — a closure that calls the API and returns a message:

```go
func loadOverview(c *api.Client) tea.Cmd {
    return func() tea.Msg {
        data, err := c.GetOverview()
        if err != nil {
            return common.ErrMsg{Err: err}
        }
        return overviewLoadedMsg{data}
    }
}
```

**Update guard pattern**

Every screen's `Update` checks for the loaded message and sets `loading = false`:

```go
case activeMaterialsLoadedMsg:
    m.activeMaterials = msg.data
    m.loading = false

case common.ErrMsg:
    m.err = msg.Err
    m.loading = false
```

Note that `overviewLoadedMsg` does not set `loading = false` — only the last expected message does. This is a simple coordination mechanism: wait for both responses before clearing the spinner.

**View guard pattern**

```go
func (m Model) View() string {
    if m.loading { return common.SpinnerView(m.spinner) }
    if m.err != nil { return common.ErrorView(m.err) }
    // ... render content ...
}
```

**Navigating upward**

When the user presses `enter` on an active material, the dashboard doesn't navigate anywhere itself — it just emits a message:

```go
case "enter":
    return m, func() tea.Msg { return OpenMaterialMsg{MaterialID: id} }
```

`app.Update()` receives `dashboard.OpenMaterialMsg`, creates a `materialdetail.Model`, and pushes the detail screen. Tab switches (`d/c/s/m/p`) are intercepted globally in `app.Update()` via `switchTab()` before delegation, so screens never handle them directly.

### The `common` package: `internal/ui/common/`

Shared infrastructure used by every screen. Seven files:

**`messages.go`** — Cross-cutting message types. `GoBackMsg` tells the root to pop the stack. `ToastMsg` tells the root to show a transient notification. `ErrMsg` wraps a load error so it can travel as a `tea.Msg`. Four **domain event** types (`MaterialChangedMsg`, `ProgressLoggedMsg`, `SkillChangedMsg`, `CategoryChangedMsg`) are broadcast globally by any screen that performs a mutation — `app.Update()` fans them out to every persistent screen so background screens stay in sync without polling.

**`styles.go`** — All Lip Gloss styles declared as package-level variables, acting as a stylesheet. No screen creates styles inline (with minor exceptions for dynamically computed colors). A screen simply calls `common.MutedStyle.Render(text)`. This file also contains helper functions like `RenderTitle`, `RenderKPICards`, `RenderBar`, and `RenderWeeklyBar`. `PopupStyle` (rounded violet border, 60-char wide) is the shared style applied by all popup `View()` methods.

**`tabs.go`** — `RenderTabBar(activeTab, width)` renders the tab bar shown at the top of every screen. The root `app.View()` prepends it above the active screen's content using `lipgloss.JoinVertical`.

**`crudforms.go`** — Reusable Huh-based popup models for all CRUD operations: `CategoryForm`, `SkillForm`, `ConfirmForm`, and `MaterialForm`. Each satisfies `tea.Model` and wraps its content in `PopupStyle`. See the Forms section below for details.

**`tailwind.go`** — The web app stores category and skill colors as Tailwind CSS class strings like `"bg-violet-300 text-violet-900 dark:bg-violet-800"`. This file maps those strings to terminal hex colors. `extractBgClass` pulls the first `bg-*` token from a multi-class string. `TailwindToLipgloss` returns `(lipgloss.Color, bool)` — the `bool` follows Go's idiomatic "ok" pattern so callers can distinguish "no color" from "color happens to match the fallback". `ColorDot` and `ColoredName` are the two helpers screens actually call.

**`spinner.go`** and **`help.go`** — Thin wrappers that pre-configure the `bubbles/spinner` and `bubbles/help` components with the project's color palette, so every screen gets consistent styling without repeating configuration.

### Forms: `internal/ui/progress/log_form.go` and `internal/ui/common/crudforms.go`

The log-progress form uses [Huh](https://github.com/charmbracelet/huh), Charm's form library. `LogForm` is itself a `tea.Model` — it wraps a `*huh.Form` and satisfies `Init`/`Update`/`View`.

The key technique is **pointer binding**: form field values are bound to pointers on the `LogForm` struct at construction time, and Huh updates them directly as the user types:

```go
lf.form = huh.NewForm(
    huh.NewGroup(
        huh.NewInput().Title("Units").Value(&lf.unitsStr),
        huh.NewText().Title("Notes").Value(&lf.notes),
    ),
).WithTheme(huh.ThemeDracula())
```

When the form completes (`lf.form.State == huh.StateCompleted`), `submit()` is called — a `tea.Cmd` that reads the bound values, calls `client.LogUnits`, and returns either a `ToastMsg` on error or a `LogDoneMsg{Cancelled: false}` on success.

**Inline overlay pattern**

Forms are rendered as **inline overlays** — the screen that owns them stores an `overlay tea.Model` field and routes all messages through it when non-nil:

```go
if m.overlay != nil {
    updated, cmd := m.overlay.Update(msg)
    m.overlay = updated
    // handle done messages (LogDoneMsg, CategoryFormDoneMsg, etc.)
    return m, cmd
}
```

`View()` uses `lipgloss.Place` to center the popup over a blank background:

```go
if m.overlay != nil {
    return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.overlay.View())
}
```

Each form's `View()` wraps its content in `common.PopupStyle` (rounded violet border, fixed 60-char width), so callers don't need to apply any additional styling.

`crudforms.go` in the `common` package provides four reusable popup models built on the same pattern:

- **`CategoryForm`** — name + color picker for create/edit (`NewCategoryCreateForm` / `NewCategoryEditForm`). Sends `CategoryFormDoneMsg` when complete.
- **`SkillForm`** — same structure for skills, also carries the parent `categoryID`. Supports an optional category picker (`NewSkillCreateFormWithCategories`). Sends `SkillFormDoneMsg`.
- **`MaterialForm`** — create/edit form for materials (name, skill, type, unit type, total units, URL, dates). Sends `MaterialFormDoneMsg`.
- **`ConfirmForm`** — a small `huh.NewConfirm` yes/no dialog used for archive and delete confirmations. Carries a `tag` string (e.g. `"archive"`, `"delete"`) so the screen knows which action to execute. Sends `ConfirmDoneMsg`.

The setup wizard in `internal/ui/setup/setup.go` uses the exact same Huh pattern, with `EchoModePassword` on the API key field and a `huh.NewConfirm` for the save prompt.

### Styling: Lip Gloss

[Lip Gloss](https://github.com/charmbracelet/lipgloss) is a styling library for terminal output. It works like CSS: you build a `lipgloss.Style` with chained method calls, then call `.Render(string)` to apply it.

```go
var MutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

MutedStyle.Render("hello")  // renders "hello" in gray
```

A few patterns worth noting:

- **Width + padding before styling**: when building table columns, pad the plain string to the desired width _before_ applying a style. Applying `fmt.Sprintf("%-Ns", ...)` to an already-styled string breaks because ANSI escape codes add invisible characters that confuse `fmt`'s width calculation.
- **`lipgloss/table`**: used on list screens (progress, categories, skill detail, etc.) for consistent column alignment. The `StyleFunc` callback receives `(row, col int)` and returns a style, which is how the selected row highlight is applied.
- **`lipgloss.JoinVertical` / `JoinHorizontal`**: used to compose sections. KPI cards are joined horizontally; the toast overlay is joined vertically below the screen content.

### Tracing a full user interaction

To tie it all together, here is what happens when a user presses `l` (log progress) on the dashboard:

1. **`dashboard.Update`** sees `"l"`, creates a `progress.LogForm` and stores it in `m.overlay`. Returns `m.overlay.Init()` as the next command.
2. **`LogForm.Init`** returns `lf.form.Init()` — Huh initializes its cursor state.
3. `dashboard.View()` now sees `m.overlay != nil` and returns `lipgloss.Place(...)` centered around `m.overlay.View()` — the form popup floats over the dashboard.
4. The user fills in the form. Each keystroke is routed through the overlay block in `dashboard.Update` → `m.overlay.Update(msg)` → `LogForm.Update` → `lf.form.Update(msg)` → Huh handles it internally.
5. User presses Enter on the last field. `lf.form.State == huh.StateCompleted`. `LogForm.Update` returns `lf.submit()` as the next command.
6. **`submit()`** runs in a goroutine: parses units, calls `client.LogUnits(...)` (HTTP POST), returns `LogDoneMsg{Cancelled: false}` or `ToastMsg{IsError: true}`.
7. **`dashboard.Update`** (inside the overlay routing block) receives `LogDoneMsg`. Clears `m.overlay`. Fires a reload command and a `ToastMsg{"Progress logged!"}`.
8. **`app.Update`** receives the `ToastMsg` (forwarded upward). Sets `m.toast`.
9. **`app.View`** renders `m.dashboard.View()` and appends the green toast below it via `lipgloss.JoinVertical`.
10. On the next keypress, **`app.Update`** clears `m.toast`.

The same overlay pattern applies to category/skill CRUD. For example, pressing `n` on the categories screen creates a `common.CategoryForm`, stores it as `m.overlay`, and routes messages through it until `CategoryFormDoneMsg` arrives — at which point the screen calls the API and reloads its data, all without `app.go` being involved at all.

### Package dependency map

```
cmd/groundwork
    └── internal/ui/app          ← root model; imports all screens
            ├── internal/ui/dashboard
            ├── internal/ui/materials
            ├── internal/ui/skills
            ├── internal/ui/categories
            ├── internal/ui/categorydetail
            ├── internal/ui/skilldetail
            ├── internal/ui/materialdetail
            ├── internal/ui/progress    ← owns LogForm; used as inline overlay
            └── internal/ui/common     ← styles, messages, shared form models (crudforms.go)
                                          imported by all screens; imports nothing from ui

    └── internal/api             ← HTTP client; imported by all screens via constructor
    └── internal/config          ← TOML config; only imported by main + setup
    └── internal/model           ← data structs; imported by api + screens
```

No screen package imports another screen package. All cross-screen communication goes through `app.go` via typed messages. `common` flows inward only — it imports nothing from `ui`.
