# AGENTS.md — Project Context for AI Coding Agents

## Goal

Build and maintain groundwork-tui, a terminal UI client for the [Groundwork](https://groundwork.lotztech.com) learning tracker. Written in Go using the Charm (Bubble Tea) framework. Communicates with the Groundwork web app via its tRPC HTTP API using a personal API key.

---

## Critical Rules

- **Never run `git commit` or `git push` without explicit user permission**
- **Always run `go build ./...` before considering work done** — build must pass clean
- **Always run `go vet ./...` before suggesting a commit** — vet must pass clean
- **No local database or server** — the TUI is a pure HTTP client; all data lives in the Groundwork web app
- **Never add new packages without checking `go.mod`** — prefer the Charm stack already present

---

## Build & Run

```bash
# Build
go build ./...

# Vet
go vet ./...

# Run locally
go run ./cmd/groundwork

# Install binary
go install github.com/jasonlotz/groundwork-tui/cmd/groundwork@latest
```

---

## Tech Stack

- **Language:** Go 1.24+
- **TUI framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm Architecture)
- **Styling:** [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Components:** [Bubbles](https://github.com/charmbracelet/bubbles) (spinner, progress bar, help)
- **Forms:** [Huh](https://github.com/charmbracelet/huh)
- **Config:** [BurntSushi/toml](https://github.com/BurntSushi/toml)

---

## Project Structure

```
cmd/groundwork/
  main.go                — entry point: setup wizard or main app

internal/
  api/
    client.go            — tRPC HTTP client (generic query/mutation helpers)
  config/
    config.go            — TOML config load/save (~/.config/groundwork-tui/config.toml)
  model/
    model.go             — Go structs mirroring API JSON shapes
  ui/
    app/
      app.go             — root model; owns navigation stack + all screen models
    common/
      styles.go          — all Lip Gloss styles + shared helpers (RenderTitle, RenderBar, etc.)
      messages.go        — cross-cutting message types: GoBackMsg, ToastMsg, ErrMsg, MaterialChangedMsg, etc.
      tailwind.go        — maps Tailwind color class strings to terminal hex colors
      spinner.go         — pre-configured spinner
      help.go            — pre-configured help bar
      keys.go            — key binding helpers
      tabs.go            — RenderTabBar(activeTab, width): tab bar rendered at the top of every screen
    forms/
      category_form.go   — CategoryForm, CategoryFormDoneMsg, NewCategoryCreateForm, NewCategoryEditForm
      skill_form.go      — SkillForm, SkillFormDoneMsg, NewSkillCreateFormWithCategories, NewSkillEditForm
      material_form.go   — MaterialForm, MaterialFormDoneMsg, MaterialFormResult, NewMaterialCreateForm, NewMaterialEditForm
      confirm_form.go    — ConfirmForm, ConfirmDoneMsg, NewConfirmForm
      log_form.go        — LogForm, LogDoneMsg, NewLogForm
      colors.go          — shared colorOptions + ActiveTheme var + updateHuhForm helper
    setup/
      setup.go           — first-run wizard (Huh form for base URL + API key)
    theme/
      theme.go           — Theme struct, All slice (11 themes), Active pointer, SetActive()
    settings/
      settings.go        — theme-picker screen (press t); emits ThemeChangedMsg on selection
    dashboard/           — home screen: KPI cards + active materials list
    materials/
      materials.go       — materials list + create/edit/delete/log overlays
      material_detail.go — DetailModel, NewDetail(): single material KPI + progress log
    skills/
      skills.go          — skills list
      skill_detail.go    — DetailModel, NewDetail(): single skill KPI + materials table
    progress/            — progress log list
    categories/
      categories.go      — categories list
      category_detail.go — DetailModel, NewDetail(): single category skills list
```

---

## Architecture Conventions

### Bubble Tea pattern

Every screen is a `Model` struct satisfying `Init() tea.Cmd`, `Update(tea.Msg) (tea.Model, tea.Cmd)`, and `View() string`. State is immutable per-update — `Update` receives a copy and returns a new one. Async work is done via `tea.Cmd` (a `func() tea.Msg` run in a goroutine by the runtime).

### Navigation

`app.go` owns a `navStack []screenState`. `pushScreen(s)` saves the current screen pointers onto the stack; `popScreen()` restores them. Screens never import or reference each other — all cross-screen communication is via typed exported messages received by `app.Update()`.

### WindowSizeMsg forwarding

`tea.WindowSizeMsg` is sent once at startup (when the dashboard is active). `app.Update()` forwards it to **all** persistent screen models immediately so every screen has correct `width`/`height` before it is first displayed. Forgetting this causes list screens to show only 3 items regardless of terminal height (height stays 0, visibleItems formula bottoms out at the minimum floor).

### Visible-items formula

Each list screen computes how many rows to show:

```go
visibleItems := (m.height - overhead) / linesPerItem
```

Overhead must account for every rendered line above and below the list. Key line counts:

- `RenderTitle(s, w)` = **3 lines**: title text + implicit `MarginBottom(1)` + rule line. Always 3, never 1 or 2.
- `RenderTitleWithTag(title, tag, w)` = **2 lines**: title + tag rendered inline (no `MarginBottom`), then rule line. Use this when you need a filter tag beside the title — appending text after `RenderTitle` puts it on the line _below_ due to the `MarginBottom(1)` on `TitleStyle`.
- `HelpStyle` / `SectionStyle` both have `MarginTop(1)` = **2 lines** (margin + text).
- An explicit `b.WriteString("\n")` = **1 line**.
- **Tab bar** = **3 lines**: top-border row + label row + rule row. Add 3 to every screen's overhead.

Recount from the `View()` source every time you change layout; do not guess.

### Overlay pattern

Screens store an `overlay tea.Model` field. When non-nil, `Update` routes all messages through the overlay and `View` renders it centered via `lipgloss.Place`. Done messages (`forms.LogDoneMsg`, `forms.CategoryFormDoneMsg`, `forms.ConfirmDoneMsg`, etc.) clear the overlay and trigger a data reload. All form and done-message types live in `internal/ui/forms/` — never in `common`.

Every screen that can have an overlay must implement `HasOverlay() bool { return m.overlay != nil }`. The root `app.go` collects these via `inputActive()` to suppress global tab-switch hotkeys (`d/c/s/m/p/t`) while a form is open. If you add a new screen with an overlay, add its `HasOverlay()` check to `inputActive()` in `app.go`.

### Styles

All Lip Gloss styles are declared as package-level vars in `common/styles.go`. Do not create inline styles in screen files (minor exception: dynamically computed colors like pace-based progress bar colors). Use `common.MutedStyle.Render(text)`, `common.SuccessStyle.Render(text)`, etc.

### API client

tRPC 11 over HTTP:

- Queries: `GET /api/trpc/<procedure>?input=<url-encoded {"json": ...}>`
- Mutations: `POST /api/trpc/<procedure>` with body `{"json": ...}`
- Responses unwrapped from: `{"result": {"data": {"json": ...}}}`

Two generic functions handle all calls: `query[T]` and `mutation[T]` in `internal/api/client.go`. To add a new API call, add the model struct to `internal/model/model.go` and a one-liner method to `client.go`.

**Server-side filtering:** `GetAllSkills(includeArchived bool)` and `GetAllCategories(includeArchived bool)` pass the flag to the server — the server handles archive filtering, not the client. When the user toggles `showArchived` on a list screen, always re-fetch from the server (`load(m.client, m.showArchived)`) rather than filtering a local slice. `applyFilter()` on those screens is a pass-through (`m.filtered = m.items`) kept only for structural consistency.

### Color mapping

The web app stores colors as Tailwind CSS class strings (e.g. `"bg-violet-300 text-violet-900"`). `common/tailwind.go` maps these to terminal hex colors. Use `common.ColorDot(color)` or `common.ColoredName(name, color)` — never parse Tailwind strings in screen code.

---

## Commit Style

Conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`, `chore:`

Examples:

- `feat: add bulk delete to materials screen`
- `fix: forward WindowSizeMsg to all screens so list heights fill the terminal`
- `docs: update README architecture section`
