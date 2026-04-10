// Package materials provides the materials list TUI screen.
package materials

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
	"github.com/jasonlotz/groundwork-tui/internal/ui/forms"
)

type materialsLoadedMsg struct{ data []model.Material }

// materialMutatedMsg is an internal result from a create/update/delete command.
type materialMutatedMsg struct{ toast string }

// preloadMsg carries skills + types fetched before opening the material form.
type preloadMsg struct {
	skills []model.Skill
	types  []model.MaterialType
	// openEdit is set when the preload was triggered by "e" — carries the material to edit.
	openEdit *model.Material
}

// OpenMaterialMsg is sent when the user presses enter on the selected material.
type OpenMaterialMsg struct{ MaterialID string }

// Model is the Bubble Tea model for the materials screen.
type Model struct {
	client      *api.Client
	materials   []model.Material
	filtered    []model.Material
	cursor      int
	activeOnly  bool
	search      string
	searching   bool
	searchInput textinput.Model
	loading     bool
	err         error
	width       int
	height      int
	spinner     spinner.Model
	keys        common.SimpleKeyMap
	overlay     tea.Model
}

func New(client *api.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "search name, skill, type…"
	ti.CharLimit = 80

	return Model{
		client:      client,
		activeOnly:  false,
		loading:     true,
		spinner:     common.NewSpinner(),
		searchInput: ti,
		keys: common.SimpleKeyMap{Bindings: []common.Binding{
			common.KBKeys("j/k", "navigate", "j", "k", "down", "up"),
			common.KB("enter", "detail"),
			common.KB("l", "log progress"),
			common.KB("n", "new"),
			common.KB("e", "edit"),
			common.KB("D", "delete"),
			common.KB("a", "toggle active"),
			common.KB("/", "search"),
			common.KB("r", "refresh"),
			common.KB("esc", "back"),
		}},
	}
}

func load(c *api.Client, activeOnly bool) tea.Cmd {
	return func() tea.Msg {
		data, err := c.GetAllMaterials(activeOnly)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return materialsLoadedMsg{data}
	}
}

// preload fetches skills and types needed to populate the material form.
// openEdit is non-nil when we're editing an existing material.
func preload(c *api.Client, openEdit *model.Material) tea.Cmd {
	return func() tea.Msg {
		skills, err := c.GetAllSkills(false)
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		types, err := c.GetAllMaterialTypes()
		if err != nil {
			return common.ErrMsg{Err: err}
		}
		return preloadMsg{skills: skills, types: types, openEdit: openEdit}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(load(m.client, m.activeOnly), m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ── overlay routing ──────────────────────────────────────────────────────
	if m.overlay != nil {
		if k, ok := msg.(tea.KeyMsg); ok && (k.String() == "ctrl+c" || k.String() == "q") {
			return m, tea.Quit
		}

		updated, cmd := m.overlay.Update(msg)
		m.overlay = updated

		switch msg := msg.(type) {
		case forms.LogDoneMsg:
			m.overlay = nil
			if !msg.Cancelled {
				return m, func() tea.Msg { return common.LearningLoggedMsg{} }
			}
			return m, nil

		case forms.MaterialFormDoneMsg:
			m.overlay = nil
			if !msg.Cancelled {
				if mf, ok := updated.(forms.MaterialForm); ok {
					return m, submitMaterialForm(m.client, m.activeOnly, mf)
				}
			}
			return m, nil

		case forms.ConfirmDoneMsg:
			m.overlay = nil
			if msg.Confirmed && msg.Tag == "delete" {
				return m, deleteMaterial(m.client, m.filtered, m.cursor, m.activeOnly)
			}
			return m, nil

		case materialsLoadedMsg:
			m.materials = msg.data
			m.resetCursor()
			m.loading = false
			return m, nil

		case common.ToastMsg:
			return m, func() tea.Msg { return msg }
		}

		return m, cmd
	}

	// ── normal update ────────────────────────────────────────────────────────
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case materialsLoadedMsg:
		m.materials = msg.data
		m.resetCursor()
		m.loading = false

	case materialMutatedMsg:
		t := msg.toast
		return m, tea.Batch(
			func() tea.Msg { return common.ToastMsg{Text: t} },
			func() tea.Msg { return common.MaterialChangedMsg{} },
		)

	case common.MaterialChangedMsg:
		return m, load(m.client, m.activeOnly)

	case preloadMsg:
		// Preload completed — open the form overlay.
		var f forms.MaterialForm
		if msg.openEdit != nil {
			f = forms.NewMaterialEditForm(msg.openEdit.ID, *msg.openEdit, msg.skills, msg.types)
		} else {
			f = forms.NewMaterialCreateForm(msg.skills, msg.types)
		}
		m.overlay = f
		return m, f.Init()

	case common.ErrMsg:
		m.err = msg.Err
		m.loading = false

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// ── search mode ──────────────────────────────────────────────────────
		if m.searching {
			switch msg.String() {
			case "enter":
				// Commit the search and exit search mode.
				m.searching = false
				m.searchInput.Blur()
			case "esc":
				// Clear search and exit search mode.
				m.searching = false
				m.searchInput.Blur()
				m.search = ""
				m.searchInput.SetValue("")
				m.applyFilter()
				m.cursor = 0
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.search = m.searchInput.Value()
				m.applyFilter()
				m.cursor = 0
				return m, cmd
			}
			return m, nil
		}

		// ── normal mode ───────────────────────────────────────────────────────
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return common.GoBackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "/":
			m.searching = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case "a":
			m.activeOnly = !m.activeOnly
			m.loading = true
			m.cursor = 0
			return m, load(m.client, m.activeOnly)
		case "r":
			m.loading = true
			m.err = nil
			return m, load(m.client, m.activeOnly)
		case "l":
			if len(m.filtered) > 0 {
				mat := m.filtered[m.cursor]
				if mat.Status == model.StatusActive {
					lf := forms.NewLogForm(m.client, mat.ID, mat.Name)
					m.overlay = lf
					return m, lf.Init()
				}
				return m, func() tea.Msg {
					return common.ToastMsg{Text: "Only active materials can be logged.", IsError: true}
				}
			}
		case "enter":
			if len(m.filtered) > 0 {
				id := m.filtered[m.cursor].ID
				return m, func() tea.Msg { return OpenMaterialMsg{MaterialID: id} }
			}
		case "n":
			// Preload skills + types, then open create form.
			return m, preload(m.client, nil)
		case "e":
			if len(m.filtered) > 0 {
				mat := m.filtered[m.cursor]
				return m, preload(m.client, &mat)
			}
		case "D":
			if len(m.filtered) > 0 {
				mat := m.filtered[m.cursor]
				f := forms.NewConfirmForm(
					"Delete material?",
					fmt.Sprintf("Permanently delete \"%s\" and all its progress logs?", common.Truncate(mat.Name, 40)),
					"delete",
				)
				m.overlay = f
				return m, f.Init()
			}
		}
	}
	return m, nil
}

// Searching reports whether the search input is currently focused.
// app.go uses this to suppress global tab-switch keys while typing.
func (m Model) Searching() bool { return m.searching }

// HasOverlay reports whether a form or confirm dialog is currently open.
func (m Model) HasOverlay() bool { return m.overlay != nil }

func (m *Model) resetCursor() {
	m.applyFilter()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// applyFilter rebuilds m.filtered from m.materials based on the active search query.
// Sorting is handled server-side (ACTIVE → COMPLETE → INACTIVE, then alphabetical).
// activeOnly is also handled server-side (triggers a reload), so it is not re-checked here.
func (m *Model) applyFilter() {
	var filtered []model.Material
	if m.search == "" {
		filtered = make([]model.Material, len(m.materials))
		copy(filtered, m.materials)
	} else {
		q := strings.ToLower(m.search)
		for _, mat := range m.materials {
			if strings.Contains(strings.ToLower(mat.Name), q) ||
				strings.Contains(strings.ToLower(mat.SkillName()), q) ||
				strings.Contains(strings.ToLower(mat.TypeName()), q) {
				filtered = append(filtered, mat)
			}
		}
	}
	m.filtered = filtered
}

// submitMaterialForm runs the create/update API call after form completion.
func submitMaterialForm(c *api.Client, activeOnly bool, mf forms.MaterialForm) tea.Cmd {
	return func() tea.Msg {
		r := mf.Result()
		var err error
		if mf.IsEdit() {
			err = c.UpdateMaterial(api.MaterialUpdateResult{
				ID:            mf.EditID(),
				Name:          r.Name,
				SkillID:       r.SkillID,
				TypeID:        r.TypeID,
				UnitType:      r.UnitType,
				TotalUnits:    r.TotalUnits,
				URL:           r.URL,
				Notes:         r.Notes,
				StartDate:     r.StartDate,
				CompletedDate: r.CompletedDate,
				WeeklyGoal:    r.WeeklyGoal,
			})
		} else {
			err = c.CreateMaterial(api.MaterialCreateResult{
				Name:          r.Name,
				SkillID:       r.SkillID,
				TypeID:        r.TypeID,
				UnitType:      r.UnitType,
				TotalUnits:    r.TotalUnits,
				URL:           r.URL,
				Notes:         r.Notes,
				StartDate:     r.StartDate,
				CompletedDate: r.CompletedDate,
				WeeklyGoal:    r.WeeklyGoal,
			})
		}
		if err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		action := "created"
		if mf.IsEdit() {
			action = "updated"
		}
		return materialMutatedMsg{toast: "Material " + action + "!"}
	}
}

// deleteMaterial runs the delete API call after confirmation.
func deleteMaterial(c *api.Client, filtered []model.Material, cursor int, activeOnly bool) tea.Cmd {
	return func() tea.Msg {
		if cursor >= len(filtered) {
			return nil
		}
		mat := filtered[cursor]
		if err := c.DeleteMaterial(mat.ID); err != nil {
			return common.ToastMsg{Text: "Error: " + err.Error(), IsError: true}
		}
		return materialMutatedMsg{toast: fmt.Sprintf("Deleted \"%s\"", common.Truncate(mat.Name, 30))}
	}
}

func (m Model) View() string {
	if m.overlay != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.overlay.View())
	}

	if m.loading {
		return common.SpinnerView(m.spinner)
	}
	if m.err != nil {
		return common.ErrorView(m.err, m.width)
	}

	var b strings.Builder

	// Header — title with inline filter tags on the same line, rule beneath.
	var tags []string
	if m.activeOnly {
		tags = append(tags, common.MutedStyle.Render("[active only]"))
	}
	if m.search != "" {
		tags = append(tags, common.MutedStyle.Render("[search: "+m.search+"]"))
	}
	b.WriteString(common.RenderTitleWithTag("Materials", strings.Join(tags, "  "), m.width))
	b.WriteString("\n")

	// Search input (shown when actively searching)
	if m.searching {
		b.WriteString("  " + m.searchInput.View() + "\n\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(common.MutedStyle.Render("  No materials found.\n"))
	} else {
		// RenderTitle=2 + blank=1 + help(marginTop+line)=2 = 5 overhead; tab bar=3 → 8
		// search row (when visible) = 2 extra lines; always include for stability.
		searchLines := 0
		if m.searching {
			searchLines = 2
		}
		visibleItems := (m.height - 8 - searchLines) / 3
		if visibleItems < 3 {
			visibleItems = 3
		}
		start, end := common.VisibleWindow(m.cursor, len(m.filtered), visibleItems)
		for i := start; i < end; i++ {
			b.WriteString(m.renderRow(i))
			b.WriteString("\n")
		}
		if len(m.filtered) > visibleItems {
			b.WriteString(common.MutedStyle.Render(fmt.Sprintf(
				"  %d–%d of %d\n", start+1, end, len(m.filtered),
			)))
		}
	}

	b.WriteString("\n")
	b.WriteString(common.RenderHelp(m.keys, m.width))
	return b.String()
}

func (m Model) renderRow(i int) string {
	mat := m.filtered[i]

	selected := i == m.cursor
	cursorStr := "  "
	nameStyle := common.DefaultNameStyle
	switch {
	case selected:
		cursorStr = common.SelectedStyle.Render("▶ ")
		nameStyle = common.SelectedStyle
	case mat.Status == model.StatusComplete:
		nameStyle = common.CompletedNameStyle
	}

	// Progress
	pct := 0.0
	if mat.TotalUnits > 0 {
		pct = mat.CompletedUnits / mat.TotalUnits
	}
	bar := common.RenderOverallBar(18, pct)
	progressText := common.MutedStyle.Render(fmt.Sprintf(
		"%.4g / %.4g %s", mat.CompletedUnits, mat.TotalUnits, mat.UnitType.Label(),
	))

	// Status badge
	statusStyle := common.MutedStyle
	statusStr := "inactive"
	switch mat.Status {
	case model.StatusActive:
		statusStyle = common.SuccessStyle
		statusStr = "active"
	case model.StatusComplete:
		statusStyle = common.CompletedStatusStyle
		statusStr = "done"
	}
	status := statusStyle.Render(statusStr)

	// Type + skill
	meta := common.MutedStyle.Render(common.Truncate(mat.TypeName(), 14)) +
		"  " + common.MutedStyle.Render(common.Truncate(mat.SkillName(), 18))

	// Weekly goal indicator (only for active)
	weeklyInfo := ""
	if mat.Status == model.StatusActive && mat.WeeklyUnitGoal != nil && *mat.WeeklyUnitGoal > 0 {
		weeklyInfo = "  " + common.MutedStyle.Render(fmt.Sprintf("goal: %d/%s", *mat.WeeklyUnitGoal, mat.UnitType.Label()))
	}

	name := nameStyle.Render(common.Truncate(mat.Name, 36))

	line1 := cursorStr + name + "  " + status
	line2 := "    " + bar + "  " + progressText + "  " + meta + weeklyInfo

	return line1 + "\n" + line2
}
