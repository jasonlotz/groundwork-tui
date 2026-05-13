# Groundwork TUI

Terminal UI client for the [Groundwork](https://groundwork.lotztech.com) learning, fitness, and habit tracker. Go + Charm (Bubble Tea). Talks to the Groundwork web app via its tRPC HTTP API using a personal API key.

The Groundwork web app source (Next.js / tRPC) is at `../groundwork`. Refer to it when adding or modifying API calls to verify procedure names, input shapes, and response types.

## Rules (shared)

- Never run `git commit` or `git push` without explicit user permission. Invoking `/ship-it` (or any equivalent "ship this" / "commit and push" instruction in the user's message) counts as that explicit permission — proceed without re-asking
- `go build ./...` and `go vet ./...` must pass clean before suggesting a commit
- Conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`, `chore:`

## Rules (project)

- No local DB or server — pure HTTP client; all data lives in the Groundwork web app
- Never add new packages without checking `go.mod` — prefer the Charm stack already present

## Stack

- Go 1.24+
- TUI: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm Architecture)
- Styling: [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- Components: [Bubbles](https://github.com/charmbracelet/bubbles) (spinner, progress, help)
- Forms: [Huh](https://github.com/charmbracelet/huh)
- Config: [BurntSushi/toml](https://github.com/BurntSushi/toml) (`~/.config/groundwork-tui/config.toml`)

## Build & run

```bash
go build ./...                                                            # build
go vet ./...                                                              # vet
go run ./cmd/groundwork                                                   # run locally (preferred during dev)
go install github.com/jasonlotz/groundwork-tui/cmd/groundwork@latest      # install (commit/push only)
```

## Architecture

### Bubble Tea pattern

Every screen is a `Model` satisfying `Init() tea.Cmd`, `Update(tea.Msg) (tea.Model, tea.Cmd)`, `View() string`. State is immutable per-update. Async work via `tea.Cmd`.

### Navigation

`app.go` owns a `navStack []screenState`. `pushScreen(s)` saves current pointers; `popScreen()` restores. **Screens never import each other** — all cross-screen communication is via typed exported messages received by `app.Update()`.

### WindowSizeMsg forwarding

`tea.WindowSizeMsg` is sent once at startup. `app.Update()` must forward it to **all** persistent screen models immediately so every screen has correct `width`/`height` before first display. Forgetting this causes list screens to show 3 items regardless of terminal height.

### Visible-items formula

```go
visibleItems := (m.height - overhead) / linesPerItem
```

Overhead must account for every rendered line. Key counts:

- `RenderTitle(s, w)` = **3 lines** (title + implicit `MarginBottom(1)` + rule). Always 3
- `RenderTitleWithTag(title, tag, w)` = **2 lines** (title + tag inline, no margin, then rule). Use when you need a tag beside the title
- `HelpStyle` / `SectionStyle` `MarginTop(1)` = **2 lines**
- Explicit `b.WriteString("\n")` = **1 line**
- **Tab bar = 3 lines** (top-border + label + rule). Add 3 to every screen's overhead

Recount from `View()` source on every layout change — don't guess.

### Overlay pattern

Screens hold an `overlay tea.Model` field. When non-nil, `Update` routes all messages through it; `View` centers via `lipgloss.Place`. Done messages (e.g. `forms.LogDoneMsg`, `forms.CategoryFormDoneMsg`, `forms.ConfirmDoneMsg`) clear the overlay and trigger reload. **All form + done-message types live in `internal/ui/forms/`** — never in `common`.

Every screen with an overlay must implement `HasOverlay() bool { return m.overlay != nil }`. `app.go` collects these via `inputActive()` to suppress global tab-switch hotkeys (`d/c/s/m/a/t/f`) while a form is open. New overlay screens must add their `HasOverlay()` to `inputActive()`. The `settings` screen is theme-only (no overlays).

### Styles

All Lip Gloss styles are package-level vars in `common/styles.go`. No inline styles in screen files (minor exception: dynamically computed colors like pace-based progress). Use `common.DimStyle.Render` for de-emphasized text, `common.SuccessStyle.Render`, etc. `TableHeaderStyle` / `StatLabelStyle` use `ColorMuted` (lighter); `DimStyle` uses `ColorDim` (darker).

### API client

tRPC 11 over HTTP:

- Queries: `GET /api/trpc/<procedure>?input=<url-encoded {"json": ...}>`
- Mutations: `POST /api/trpc/<procedure>` with body `{"json": ...}`
- Responses unwrap from `{"result": {"data": {"json": ...}}}`

Generic `query[T]` and `mutation[T]` in `internal/api/client.go`. New API call = add a struct to `internal/model/model.go` + a one-liner method to `client.go`.

**Server-side filtering:** `GetAllSkills(includeArchived)` and `GetAllCategories(includeArchived)` pass the flag — the server filters, not the client. When the user toggles `showArchived`, re-fetch from the server (`load(m.client, m.showArchived)`) — don't filter a local slice. `applyFilter()` on those screens is a pass-through kept for structural consistency.

### Color mapping

The web app stores colors as Tailwind class strings (e.g. `"bg-violet-300 text-violet-900"`). `common/tailwind.go` maps them to terminal hex. Use `common.ColorDot(color)` or `common.ColoredName(name, color)` — never parse Tailwind strings in screen code.

## Notable files

- `cmd/groundwork/main.go` — entry: setup wizard or main app
- `internal/api/client.go` — generic tRPC `query[T]` / `mutation[T]`
- `internal/ui/app/app.go` — root model, navigation stack, `inputActive()` hotkey gate
- `internal/ui/common/messages.go` — cross-cutting message types (GoBackMsg, ToastMsg, ErrMsg, MaterialChangedMsg, LearningLoggedMsg, WorkoutLoggedMsg, ExerciseChangedMsg, SubtypeChangedMsg, HabitChangedMsg, …)
- `internal/ui/common/styles.go` — all Lip Gloss styles + helpers (RenderTitle, RenderBar, etc.)
- `internal/ui/common/tabs.go` — `RenderTabBar(activeTab, width)` (tabs: d=Dashboard c=Categories s=Skills m=Materials f=Fitness h=Habits a=Activity i=Settings)
- `internal/ui/common/tailwind.go` — Tailwind class string → terminal hex
- `internal/ui/forms/` — every form + its `*DoneMsg` (category, skill, material, confirm, log, log_workout, edit_workout, habit). Multi-step forms: `log_workout_form.go` (type → subtype → details → row editor); `edit_workout_form.go` (subtype → details → row editor)
- `internal/ui/theme/theme.go` — Theme struct, 11-theme `All` slice, `Active` pointer, `SetActive()`
