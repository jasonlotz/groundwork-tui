// Package app is the root Bubble Tea model that owns navigation between screens.
package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/categories"
	"github.com/jasonlotz/groundwork-tui/internal/ui/categorydetail"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
	"github.com/jasonlotz/groundwork-tui/internal/ui/dashboard"
	"github.com/jasonlotz/groundwork-tui/internal/ui/materialdetail"
	"github.com/jasonlotz/groundwork-tui/internal/ui/materials"
	"github.com/jasonlotz/groundwork-tui/internal/ui/progress"
	"github.com/jasonlotz/groundwork-tui/internal/ui/skilldetail"
	"github.com/jasonlotz/groundwork-tui/internal/ui/skills"
)

type screen int

const (
	screenDashboard screen = iota
	screenMaterials
	screenSkills
	screenProgress
	screenLogForm
	screenCategories
	screenCategoryDetail
	screenSkillDetail
	screenMaterialDetail
)

// activeMaterialsReadyMsg carries active materials fetched for use in log form.
type activeMaterialsReadyMsg struct {
	data                []model.ActiveMaterial
	preselectedMaterial string // optional: pre-select this material ID
	// returnTo holds the screen to go back to after logging (instead of dashboard)
	returnTo *screenState
}

// screenState holds the current screen + its associated model pointer so we can
// push/pop a navigation stack.
type screenState struct {
	id             screen
	categoryDetail *categorydetail.Model
	skillDetail    *skilldetail.Model
	materialDetail *materialdetail.Model
}

// Model is the root application model.
type Model struct {
	client          *api.Client
	current         screen
	navStack        []screenState // previous screens for esc/back
	dashboard       dashboard.Model
	materialsList   materials.Model
	skillsList      skills.Model
	progressList    progress.Model
	logForm         *progress.LogForm
	logReturnTo     *screenState // where to go after log form
	categoriesList  categories.Model
	categoryDetail  *categorydetail.Model
	skillDetail     *skilldetail.Model
	materialDetail  *materialdetail.Model
	activeMaterials []model.ActiveMaterial
	toast           string
	toastIsErr      bool
	width           int
	height          int
}

func New(client *api.Client) Model {
	return Model{
		client:         client,
		current:        screenDashboard,
		dashboard:      dashboard.New(client),
		materialsList:  materials.New(client),
		skillsList:     skills.New(client),
		progressList:   progress.New(client),
		categoriesList: categories.New(client),
	}
}

func (m Model) Init() tea.Cmd {
	return m.dashboard.Init()
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
// Returns the init cmd needed for the screen being returned to, if any.
func (m *Model) popScreen() tea.Cmd {
	if len(m.navStack) == 0 {
		m.current = screenDashboard
		return nil
	}
	top := m.navStack[len(m.navStack)-1]
	m.navStack = m.navStack[:len(m.navStack)-1]
	m.current = top.id
	m.categoryDetail = top.categoryDetail
	m.skillDetail = top.skillDetail
	m.materialDetail = top.materialDetail
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle global messages first.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case common.ToastMsg:
		m.toast = msg.Text
		m.toastIsErr = msg.IsError
		return m, nil

	case common.GoBackMsg:
		m.toast = ""
		m.popScreen()
		return m, nil

	// --- dashboard navigation ---
	case dashboard.NavigateMsg:
		return m.handleDashboardNav(msg)

	// --- dashboard: open material detail from active materials list ---
	case dashboard.OpenMaterialMsg:
		md := materialdetail.New(m.client, msg.MaterialID)
		m.materialDetail = &md
		m.pushScreen(screenMaterialDetail)
		return m, m.materialDetail.Init()

	// --- skills list: open skill detail ---
	case skills.OpenSkillMsg:
		sd := skilldetail.New(m.client, msg.SkillID)
		m.skillDetail = &sd
		m.pushScreen(screenSkillDetail)
		return m, m.skillDetail.Init()

	// --- categories screen: open a category detail ---
	case categories.OpenCategoryMsg:
		cd := categorydetail.New(m.client, msg.CategoryID)
		m.categoryDetail = &cd
		m.pushScreen(screenCategoryDetail)
		return m, m.categoryDetail.Init()

	// --- category detail: open a skill detail ---
	case categorydetail.OpenSkillMsg:
		sd := skilldetail.New(m.client, msg.SkillID)
		m.skillDetail = &sd
		m.pushScreen(screenSkillDetail)
		return m, m.skillDetail.Init()

	// --- skill detail: open material detail or log ---
	case skilldetail.OpenMaterialMsg:
		md := materialdetail.New(m.client, msg.MaterialID)
		m.materialDetail = &md
		m.pushScreen(screenMaterialDetail)
		return m, m.materialDetail.Init()

	case skilldetail.LogFromSkillMsg:
		materialID := msg.MaterialID
		returnState := &screenState{
			id:          m.current,
			skillDetail: m.skillDetail,
		}
		return m, func() tea.Msg {
			data, err := m.client.GetActiveMaterials()
			if err != nil {
				return common.ToastMsg{Text: "Could not load materials: " + err.Error(), IsError: true}
			}
			return activeMaterialsReadyMsg{data: data, preselectedMaterial: materialID, returnTo: returnState}
		}

	// --- material detail: log ---
	case materialdetail.LogFromDetailMsg:
		materialID := msg.MaterialID
		returnState := &screenState{
			id:             m.current,
			materialDetail: m.materialDetail,
			skillDetail:    m.skillDetail,
			categoryDetail: m.categoryDetail,
		}
		return m, func() tea.Msg {
			data, err := m.client.GetActiveMaterials()
			if err != nil {
				return common.ToastMsg{Text: "Could not load materials: " + err.Error(), IsError: true}
			}
			return activeMaterialsReadyMsg{data: data, preselectedMaterial: materialID, returnTo: returnState}
		}

	// --- materials list: open detail or log ---
	case materials.OpenMaterialMsg:
		md := materialdetail.New(m.client, msg.MaterialID)
		m.materialDetail = &md
		m.pushScreen(screenMaterialDetail)
		return m, m.materialDetail.Init()

	case materials.LogFromMaterialMsg:
		materialID := msg.MaterialID
		return m, func() tea.Msg {
			data, err := m.client.GetActiveMaterials()
			if err != nil {
				return common.ToastMsg{Text: "Could not load materials: " + err.Error(), IsError: true}
			}
			return activeMaterialsReadyMsg{data: data, preselectedMaterial: materialID}
		}

	// --- open log form ---
	case activeMaterialsReadyMsg:
		m.activeMaterials = msg.data
		lf := progress.NewLogForm(m.client, m.activeMaterials)
		if msg.preselectedMaterial != "" {
			lf.PreSelectMaterial(msg.preselectedMaterial)
		}
		m.logForm = &lf
		m.logReturnTo = msg.returnTo
		m.pushScreen(screenLogForm)
		return m, m.logForm.Init()

	// --- log form done ---
	case progress.LogDoneMsg:
		m.popScreen()
		if !msg.Cancelled {
			m.toast = "Progress logged!"
			m.toastIsErr = false
			// If we came from a detail screen, reload it.
			if m.logReturnTo != nil {
				rt := m.logReturnTo
				m.logReturnTo = nil
				m.current = rt.id
				m.categoryDetail = rt.categoryDetail
				m.skillDetail = rt.skillDetail
				m.materialDetail = rt.materialDetail
				// Re-init the detail screen to refresh data.
				switch rt.id {
				case screenMaterialDetail:
					if m.materialDetail != nil {
						return m, m.materialDetail.Init()
					}
				case screenSkillDetail:
					if m.skillDetail != nil {
						return m, m.skillDetail.Init()
					}
				}
				return m, nil
			}
		}
		m.logReturnTo = nil
		return m, nil
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

	case screenProgress:
		updated, cmd := m.progressList.Update(msg)
		m.progressList = updated.(progress.Model)
		return m, cmd

	case screenLogForm:
		if m.logForm != nil {
			updated, cmd := m.logForm.Update(msg)
			if lf, ok := updated.(progress.LogForm); ok {
				m.logForm = &lf
			}
			return m, cmd
		}

	case screenCategories:
		updated, cmd := m.categoriesList.Update(msg)
		m.categoriesList = updated.(categories.Model)
		return m, cmd

	case screenCategoryDetail:
		if m.categoryDetail != nil {
			updated, cmd := m.categoryDetail.Update(msg)
			if cd, ok := updated.(categorydetail.Model); ok {
				m.categoryDetail = &cd
			}
			return m, cmd
		}

	case screenSkillDetail:
		if m.skillDetail != nil {
			updated, cmd := m.skillDetail.Update(msg)
			if sd, ok := updated.(skilldetail.Model); ok {
				m.skillDetail = &sd
			}
			return m, cmd
		}

	case screenMaterialDetail:
		if m.materialDetail != nil {
			updated, cmd := m.materialDetail.Update(msg)
			if md, ok := updated.(materialdetail.Model); ok {
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
		m.pushScreen(screenMaterials)
		return m, m.materialsList.Init()
	case dashboard.ScreenSkills:
		m.pushScreen(screenSkills)
		return m, m.skillsList.Init()
	case dashboard.ScreenProgress:
		m.pushScreen(screenProgress)
		return m, m.progressList.Init()
	case dashboard.ScreenCategories:
		m.pushScreen(screenCategories)
		return m, m.categoriesList.Init()
	case dashboard.ScreenLogProgress:
		return m, func() tea.Msg {
			data, err := m.client.GetActiveMaterials()
			if err != nil {
				return common.ToastMsg{Text: "Could not load materials: " + err.Error(), IsError: true}
			}
			return activeMaterialsReadyMsg{data: data}
		}
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
	case screenProgress:
		content = m.progressList.View()
	case screenLogForm:
		if m.logForm != nil {
			content = m.logForm.View()
		}
	case screenCategories:
		content = m.categoriesList.View()
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

	if m.toast == "" {
		return content
	}

	// Overlay a toast at the bottom.
	toastStyle := common.SuccessStyle
	if m.toastIsErr {
		toastStyle = common.DangerStyle
	}
	toast := toastStyle.Render("  " + m.toast)

	return lipgloss.JoinVertical(lipgloss.Left, content, toast)
}
