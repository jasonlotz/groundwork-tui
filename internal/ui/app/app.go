// Package app is the root Bubble Tea model that owns navigation between screens.
package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/config"
	"github.com/jasonlotz/groundwork-tui/internal/ui/activity"
	"github.com/jasonlotz/groundwork-tui/internal/ui/categories"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
	"github.com/jasonlotz/groundwork-tui/internal/ui/dashboard"
	"github.com/jasonlotz/groundwork-tui/internal/ui/fitness"
	"github.com/jasonlotz/groundwork-tui/internal/ui/materials"
	"github.com/jasonlotz/groundwork-tui/internal/ui/settings"
	"github.com/jasonlotz/groundwork-tui/internal/ui/skills"
	"github.com/jasonlotz/groundwork-tui/internal/ui/theme"
)

type screen int

const (
	screenDashboard screen = iota
	screenCategories
	screenSkills
	screenMaterials
	screenFitness
	screenActivity
	screenSettings
	screenCategoryDetail
	screenSkillDetail
	screenMaterialDetail
)

// screenState holds the current screen + its associated model pointer so we can
// push/pop a navigation stack.
type screenState struct {
	id             screen
	categoryDetail *categories.DetailModel
	skillDetail    *skills.DetailModel
	materialDetail *materials.DetailModel
}

// Model is the root application model.
type Model struct {
	client         *api.Client
	cfg            *config.Config
	current        screen
	activeTab      screen // top-level tab; does not change when pushing detail screens
	navStack       []screenState
	dashboard      dashboard.Model
	materialsList  materials.Model
	skillsList     skills.Model
	activityList   activity.Model
	categoriesList categories.Model
	settingsScreen settings.Model
	fitnessList    fitness.Model
	categoryDetail *categories.DetailModel
	skillDetail    *skills.DetailModel
	materialDetail *materials.DetailModel
	toast          string
	toastIsErr     bool
	width          int
	height         int
}

func New(client *api.Client, cfg *config.Config) Model {
	return Model{
		client:         client,
		cfg:            cfg,
		current:        screenDashboard,
		activeTab:      screenDashboard,
		dashboard:      dashboard.New(client),
		materialsList:  materials.New(client),
		skillsList:     skills.New(client),
		activityList:   activity.New(client),
		categoriesList: categories.New(client),
		settingsScreen: settings.New(client),
		fitnessList:    fitness.New(client),
	}
}

func (m Model) Init() tea.Cmd {
	return m.dashboard.Init()
}

// inputActive returns true when any screen currently owns keyboard input
// (an overlay form is open, or the materials search input is focused).
// Used to suppress global tab-switch hotkeys while the user is typing.
func (m Model) inputActive() bool {
	if m.materialsList.Searching() || m.materialsList.HasOverlay() {
		return true
	}
	if m.skillsList.HasOverlay() || m.activityList.HasOverlay() ||
		m.categoriesList.HasOverlay() {
		return true
	}
	if m.fitnessList.HasOverlay() || m.settingsScreen.HasOverlay() {
		return true
	}
	if m.categoryDetail != nil && m.categoryDetail.HasOverlay() {
		return true
	}
	if m.skillDetail != nil && m.skillDetail.HasOverlay() {
		return true
	}
	if m.materialDetail != nil && m.materialDetail.HasOverlay() {
		return true
	}
	return false
}

// pushScreen saves the current screen state onto the stack and switches to the new screen.
func (m *Model) pushScreen(s screen) {
	var state screenState
	state.id = m.current
	state.categoryDetail = m.categoryDetail
	state.skillDetail = m.skillDetail
	state.materialDetail = m.materialDetail
	m.navStack = append(m.navStack, state)
	m.current = s
}

// popScreen returns to the previous screen on the stack (or dashboard if empty).
func (m *Model) popScreen() tea.Cmd {
	if len(m.navStack) == 0 {
		m.current = screenDashboard
		m.activeTab = screenDashboard
		return nil
	}
	top := m.navStack[len(m.navStack)-1]
	m.navStack = m.navStack[:len(m.navStack)-1]
	m.current = top.id
	m.categoryDetail = top.categoryDetail
	m.skillDetail = top.skillDetail
	m.materialDetail = top.materialDetail
	// keep activeTab unchanged — it tracks the top-level tab root
	return nil
}

// switchTab jumps to a top-level tab. If already on that tab at the top level,
// it's a no-op. If on a detail screen within that tab, it pops back to the list.
func (m *Model) switchTab(s screen) (tea.Model, tea.Cmd) {
	// Already on this tab at the top level — do nothing.
	if m.activeTab == s && m.current == s {
		return m, nil
	}
	// Already on this tab but inside a detail screen — pop back to the list.
	if m.activeTab == s {
		m.navStack = nil
		m.current = s
		return m, nil
	}
	// Switching to a different tab — clear stack, re-init.
	m.navStack = nil
	m.activeTab = s
	m.current = s
	m.toast = ""
	switch s {
	case screenDashboard:
		return m, m.dashboard.Init()
	case screenMaterials:
		return m, m.materialsList.Init()
	case screenSkills:
		return m, m.skillsList.Init()
	case screenActivity:
		return m, m.activityList.Init()
	case screenCategories:
		return m, m.categoriesList.Init()
	case screenSettings:
		m.settingsScreen = settings.New(m.client)
		return m, m.settingsScreen.Init()
	case screenFitness:
		return m, m.fitnessList.Init()
	}
	return m, nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle global messages first.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward size to all persistent screen models.
		if updated, _ := m.dashboard.Update(msg); updated != nil {
			m.dashboard = updated.(dashboard.Model)
		}
		if updated, _ := m.materialsList.Update(msg); updated != nil {
			m.materialsList = updated.(materials.Model)
		}
		if updated, _ := m.skillsList.Update(msg); updated != nil {
			m.skillsList = updated.(skills.Model)
		}
		if updated, _ := m.activityList.Update(msg); updated != nil {
			m.activityList = updated.(activity.Model)
		}
		if updated, _ := m.categoriesList.Update(msg); updated != nil {
			m.categoriesList = updated.(categories.Model)
		}
		if updated, _ := m.settingsScreen.Update(msg); updated != nil {
			m.settingsScreen = updated.(settings.Model)
		}
		if updated, _ := m.fitnessList.Update(msg); updated != nil {
			m.fitnessList = updated.(fitness.Model)
		}
		return m, nil

	case common.ToastMsg:
		m.toast = msg.Text
		m.toastIsErr = msg.IsError
		return m, nil

	case common.ThemeChangedMsg:
		if theme.SetActive(msg.ThemeName) {
			common.ApplyTheme()
			// Persist to config.
			if m.cfg != nil {
				m.cfg.Theme = msg.ThemeName
				_ = config.Save(m.cfg)
			}
			// Refresh the settings cursor to reflect new active theme.
			m.settingsScreen = settings.New(m.client)
			m.toast = "Theme changed to " + msg.ThemeName
		}
		return m, nil

	case common.GoBackMsg:
		m.toast = ""
		m.popScreen()
		return m, nil

	// --- domain events: fan out to all persistent screens ---
	case common.MaterialChangedMsg, common.LearningLoggedMsg, common.SkillChangedMsg, common.CategoryChangedMsg:
		var cmds []tea.Cmd
		if updated, cmd := m.dashboard.Update(msg); updated != nil {
			m.dashboard = updated.(dashboard.Model)
			cmds = append(cmds, cmd)
		}
		if updated, cmd := m.materialsList.Update(msg); updated != nil {
			m.materialsList = updated.(materials.Model)
			cmds = append(cmds, cmd)
		}
		if updated, cmd := m.skillsList.Update(msg); updated != nil {
			m.skillsList = updated.(skills.Model)
			cmds = append(cmds, cmd)
		}
		if updated, cmd := m.activityList.Update(msg); updated != nil {
			m.activityList = updated.(activity.Model)
			cmds = append(cmds, cmd)
		}
		if updated, cmd := m.categoriesList.Update(msg); updated != nil {
			m.categoriesList = updated.(categories.Model)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case common.WorkoutLoggedMsg:
		var cmds []tea.Cmd
		if updated, cmd := m.dashboard.Update(msg); updated != nil {
			m.dashboard = updated.(dashboard.Model)
			cmds = append(cmds, cmd)
		}
		if updated, cmd := m.fitnessList.Update(msg); updated != nil {
			m.fitnessList = updated.(fitness.Model)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case common.ExerciseChangedMsg:
		var cmds []tea.Cmd
		if updated, cmd := m.fitnessList.Update(msg); updated != nil {
			m.fitnessList = updated.(fitness.Model)
			cmds = append(cmds, cmd)
		}
		if updated, cmd := m.settingsScreen.Update(msg); updated != nil {
			m.settingsScreen = updated.(settings.Model)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- dashboard navigation ---
	case dashboard.NavigateMsg:
		return m.handleDashboardNav(msg)

	// --- dashboard: open material detail from active materials list ---
	case dashboard.OpenMaterialMsg:
		md := materials.NewDetail(m.client, msg.MaterialID)
		m.materialDetail = &md
		m.pushScreen(screenMaterialDetail)
		return m, m.materialDetail.Init()

	// --- skills list: open skill detail ---
	case skills.OpenSkillMsg:
		sd := skills.NewDetail(m.client, msg.SkillID)
		m.skillDetail = &sd
		m.pushScreen(screenSkillDetail)
		return m, m.skillDetail.Init()

	// --- categories screen: open a category detail ---
	case categories.OpenCategoryMsg:
		cd := categories.NewDetail(m.client, msg.CategoryID)
		m.categoryDetail = &cd
		m.pushScreen(screenCategoryDetail)
		return m, m.categoryDetail.Init()

	// --- category detail: open a skill detail ---
	case categories.OpenSkillMsg:
		sd := skills.NewDetail(m.client, msg.SkillID)
		m.skillDetail = &sd
		m.pushScreen(screenSkillDetail)
		return m, m.skillDetail.Init()

	// --- skill detail: open material detail ---
	case skills.OpenMaterialMsg:
		md := materials.NewDetail(m.client, msg.MaterialID)
		m.materialDetail = &md
		m.pushScreen(screenMaterialDetail)
		return m, m.materialDetail.Init()

	// --- materials list: open detail ---
	case materials.OpenMaterialMsg:
		md := materials.NewDetail(m.client, msg.MaterialID)
		m.materialDetail = &md
		m.pushScreen(screenMaterialDetail)
		return m, m.materialDetail.Init()
	}

	// --- global tab-switch keys (work from any screen, unless a screen owns the input) ---
	if key, ok := msg.(tea.KeyMsg); ok && !m.inputActive() {
		switch key.String() {
		case "d":
			return m.switchTab(screenDashboard)
		case "m":
			return m.switchTab(screenMaterials)
		case "s":
			return m.switchTab(screenSkills)
		case "a":
			return m.switchTab(screenActivity)
		case "c":
			return m.switchTab(screenCategories)
		case "t":
			return m.switchTab(screenSettings)
		case "f":
			return m.switchTab(screenFitness)
		}
	}

	// Clear toast on any key.
	if _, ok := msg.(tea.KeyMsg); ok && m.toast != "" {
		m.toast = ""
	}

	// Delegate to the active screen.
	switch m.current {
	case screenDashboard:
		updated, cmd := m.dashboard.Update(msg)
		m.dashboard = updated.(dashboard.Model)
		return m, cmd

	case screenMaterials:
		updated, cmd := m.materialsList.Update(msg)
		m.materialsList = updated.(materials.Model)
		return m, cmd

	case screenSkills:
		updated, cmd := m.skillsList.Update(msg)
		m.skillsList = updated.(skills.Model)
		return m, cmd

	case screenActivity:
		updated, cmd := m.activityList.Update(msg)
		m.activityList = updated.(activity.Model)
		return m, cmd

	case screenCategories:
		updated, cmd := m.categoriesList.Update(msg)
		m.categoriesList = updated.(categories.Model)
		return m, cmd

	case screenSettings:
		updated, cmd := m.settingsScreen.Update(msg)
		m.settingsScreen = updated.(settings.Model)
		return m, cmd

	case screenFitness:
		updated, cmd := m.fitnessList.Update(msg)
		m.fitnessList = updated.(fitness.Model)
		return m, cmd

	case screenCategoryDetail:
		if m.categoryDetail != nil {
			updated, cmd := m.categoryDetail.Update(msg)
			if cd, ok := updated.(categories.DetailModel); ok {
				m.categoryDetail = &cd
			}
			return m, cmd
		}

	case screenSkillDetail:
		if m.skillDetail != nil {
			updated, cmd := m.skillDetail.Update(msg)
			if sd, ok := updated.(skills.DetailModel); ok {
				m.skillDetail = &sd
			}
			return m, cmd
		}

	case screenMaterialDetail:
		if m.materialDetail != nil {
			updated, cmd := m.materialDetail.Update(msg)
			if md, ok := updated.(materials.DetailModel); ok {
				m.materialDetail = &md
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) handleDashboardNav(nav dashboard.NavigateMsg) (tea.Model, tea.Cmd) {
	switch nav {
	case dashboard.ScreenMaterials:
		return m.switchTab(screenMaterials)
	case dashboard.ScreenSkills:
		return m.switchTab(screenSkills)
	case dashboard.ScreenActivity:
		return m.switchTab(screenActivity)
	case dashboard.ScreenCategories:
		return m.switchTab(screenCategories)
	}
	return m, nil
}

func (m Model) View() string {
	var content string

	switch m.current {
	case screenDashboard:
		content = m.dashboard.View()
	case screenMaterials:
		content = m.materialsList.View()
	case screenSkills:
		content = m.skillsList.View()
	case screenActivity:
		content = m.activityList.View()
	case screenCategories:
		content = m.categoriesList.View()
	case screenSettings:
		content = m.settingsScreen.View()
	case screenFitness:
		content = m.fitnessList.View()
	case screenCategoryDetail:
		if m.categoryDetail != nil {
			content = m.categoryDetail.View()
		}
	case screenSkillDetail:
		if m.skillDetail != nil {
			content = m.skillDetail.View()
		}
	case screenMaterialDetail:
		if m.materialDetail != nil {
			content = m.materialDetail.View()
		}
	}

	tabBar := common.RenderTabBar(int(m.activeTab), m.width)

	if m.toast == "" {
		return lipgloss.JoinVertical(lipgloss.Left, tabBar, content)
	}

	toastStyle := common.SuccessStyle
	if m.toastIsErr {
		toastStyle = common.DangerStyle
	}
	toast := toastStyle.Render("  " + m.toast)

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content, toast)
}
