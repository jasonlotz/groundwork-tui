// Package app is the root Bubble Tea model that owns navigation between screens.
package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jasonlotz/groundwork-tui/internal/api"
	"github.com/jasonlotz/groundwork-tui/internal/model"
	"github.com/jasonlotz/groundwork-tui/internal/ui/common"
	"github.com/jasonlotz/groundwork-tui/internal/ui/dashboard"
	"github.com/jasonlotz/groundwork-tui/internal/ui/materials"
	"github.com/jasonlotz/groundwork-tui/internal/ui/progress"
	"github.com/jasonlotz/groundwork-tui/internal/ui/skills"
)

type screen int

const (
	screenDashboard screen = iota
	screenMaterials
	screenSkills
	screenProgress
	screenLogForm
)

// activeMaterialsReadyMsg carries active materials fetched by dashboard for use in log form.
type activeMaterialsReadyMsg struct{ data []model.ActiveMaterial }

// Model is the root application model.
type Model struct {
	client          *api.Client
	current         screen
	dashboard       dashboard.Model
	materials       materials.Model
	skills          skills.Model
	progress        progress.Model
	logForm         *progress.LogForm
	activeMaterials []model.ActiveMaterial
	toast           string
	toastIsErr      bool
	width           int
	height          int
}

func New(client *api.Client) Model {
	return Model{
		client:    client,
		current:   screenDashboard,
		dashboard: dashboard.New(client),
		materials: materials.New(client),
		skills:    skills.New(client),
		progress:  progress.New(client),
	}
}

func (m Model) Init() tea.Cmd {
	return m.dashboard.Init()
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
		m.current = screenDashboard
		m.toast = ""
		return m, nil

	case dashboard.NavigateMsg:
		return m, m.navigateTo(msg)

	case activeMaterialsReadyMsg:
		m.activeMaterials = msg.data
		lf := progress.NewLogForm(m.client, m.activeMaterials)
		m.logForm = &lf
		m.current = screenLogForm
		return m, m.logForm.Init()

	case progress.LogDoneMsg:
		m.current = screenDashboard
		if !msg.Cancelled {
			m.toast = "Progress logged!"
			m.toastIsErr = false
		}
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
		updated, cmd := m.materials.Update(msg)
		m.materials = updated.(materials.Model)
		return m, cmd

	case screenSkills:
		updated, cmd := m.skills.Update(msg)
		m.skills = updated.(skills.Model)
		return m, cmd

	case screenProgress:
		updated, cmd := m.progress.Update(msg)
		m.progress = updated.(progress.Model)
		return m, cmd

	case screenLogForm:
		if m.logForm != nil {
			updated, cmd := m.logForm.Update(msg)
			if lf, ok := updated.(progress.LogForm); ok {
				m.logForm = &lf
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) navigateTo(nav dashboard.NavigateMsg) tea.Cmd {
	switch nav {
	case dashboard.ScreenMaterials:
		m.current = screenMaterials
		return m.materials.Init()
	case dashboard.ScreenSkills:
		m.current = screenSkills
		return m.skills.Init()
	case dashboard.ScreenProgress:
		m.current = screenProgress
		return m.progress.Init()
	case dashboard.ScreenLogProgress:
		// Fetch active materials then open form.
		return func() tea.Msg {
			data, err := m.client.GetActiveMaterials()
			if err != nil {
				return common.ToastMsg{Text: "Could not load materials: " + err.Error(), IsError: true}
			}
			return activeMaterialsReadyMsg{data}
		}
	}
	return nil
}

func (m Model) View() string {
	var content string

	switch m.current {
	case screenDashboard:
		content = m.dashboard.View()
	case screenMaterials:
		content = m.materials.View()
	case screenSkills:
		content = m.skills.View()
	case screenProgress:
		content = m.progress.View()
	case screenLogForm:
		if m.logForm != nil {
			content = m.logForm.View()
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
