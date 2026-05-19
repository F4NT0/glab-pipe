package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	gl "gitlab-pipeline-tui/internal/gitlab"
)

// splashMode tracks which sub-screen is active inside the splash.
type splashMode int

const (
	splashMenu        splashMode = iota // main menu
	splashProjectList                   // list of saved projects
)

// SplashModel is the initial welcome/splash screen shown before the main TUI.
type SplashModel struct {
	width  int
	height int
	quit   bool

	mode splashMode

	// main menu (only one option now)
	menuCursor int // 0 = Select Repository

	// project list sub-screen
	projectList []gl.Project
	listCursor  int

	selectedProject *gl.Project
}

func NewSplashModel() SplashModel {
	return SplashModel{
		projectList: gl.ProjectList(),
		mode:        splashMenu,
	}
}

func (s SplashModel) Quit() bool                   { return s.quit }
func (s SplashModel) SelectedProject() *gl.Project { return s.selectedProject }
func (s SplashModel) Init() tea.Cmd                { return nil }

func (s SplashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case tea.KeyMsg:
		key := msg.String()
		switch s.mode {
		case splashMenu:
			return s.updateMenu(key)
		case splashProjectList:
			return s.updateProjectList(key)
		}
	}
	return s, nil
}

func (s SplashModel) updateMenu(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q", "esc":
		s.quit = true
		return s, tea.Quit
	case "enter", " ":
		s.projectList = gl.ProjectList()
		s.listCursor = 0
		if len(s.projectList) == 0 {
			return s, nil // stay in menu if no projects
		}
		s.mode = splashProjectList
	}
	return s, nil
}

func (s SplashModel) updateProjectList(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c":
		s.quit = true
		return s, tea.Quit
	case "esc", "q", "backspace":
		s.mode = splashMenu
	case "up", "k":
		if s.listCursor > 0 {
			s.listCursor--
		}
	case "down", "j":
		if s.listCursor < len(s.projectList)-1 {
			s.listCursor++
		}
	case "enter":
		if len(s.projectList) > 0 {
			proj := s.projectList[s.listCursor]
			s.selectedProject = &proj
			return s, tea.Quit
		}
	}
	return s, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (s SplashModel) View() string {
	if s.width == 0 {
		return ""
	}

	muted := lipgloss.NewStyle().Foreground(colorFgMuted)
	orange := colorOrange

	ascii := `  ██████╗ ██╗      █████╗ ██████╗       ██████╗ ██╗██████╗ ███████╗
 ██╔════╝ ██║     ██╔══██╗██╔══██╗      ██╔══██╗██║██╔══██╗██╔════╝
 ██║  ███╗██║     ███████║██████╔╝█████╗██████╔╝██║██████╔╝█████╗
 ██║   ██║██║     ██╔══██║██╔══██╗╚════╝██╔═══╝ ██║██╔═══╝ ██╔══╝
 ╚██████╔╝███████╗██║  ██║██████╔╝      ██║     ██║██║     ███████╗
  ╚═════╝ ╚══════╝╚═╝  ╚═╝╚═════╝       ╚═╝     ╚═╝╚═╝     ╚══════╝`

	asciiStyle := lipgloss.NewStyle().Foreground(orange).Bold(true)
	subtitle := lipgloss.NewStyle().Foreground(colorAccentBlue).Render("GitLab Pipeline Management Tool")
	tagline := muted.Render("Interactive TUI for managing GitLab CI/CD pipelines")
	sep := muted.Render(strings.Repeat("─", 54))

	var modal string
	switch s.mode {
	case splashMenu:
		modal = s.renderMenuModal()
	case splashProjectList:
		modal = s.renderProjectListModal()
	}

	var hintLine string
	switch s.mode {
	case splashMenu:
		hintLine = muted.Render("enter: select   ctrl+c: quit")
	case splashProjectList:
		hintLine = muted.Render("↑↓: navigate   enter: open   esc: back   ctrl+c: quit")
	}

	parts := []string{
		asciiStyle.Render(ascii),
		"",
		subtitle,
		tagline,
		"",
		sep,
		"",
		modal,
		"",
		hintLine,
	}

	boxContent := lipgloss.JoinVertical(lipgloss.Center, parts...)

	return lipgloss.NewStyle().
		Background(colorBg).
		Width(s.width).
		Height(s.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(boxContent)
}

func (s SplashModel) modalWidth() int {
	// Use a more responsive width based on screen size
	w := s.width / 2
	if w < 30 {
		w = 30
	}
	if w > 50 {
		w = 50
	}
	if w > s.width-4 {
		w = s.width - 4
	}
	return w
}

func (s SplashModel) renderMenuModal() string {
	orange := colorOrange
	mw := s.modalWidth()

	var sb strings.Builder
	row := lipgloss.NewStyle().Foreground(orange).Bold(true).Render("> Select Repository...")
	sb.WriteString(row + "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(orange).
		Width(mw).
		Padding(1, 1).
		Render(sb.String())
}

func (s SplashModel) renderProjectListModal() string {
	orange := colorOrange
	mw := s.modalWidth()

	var sb strings.Builder

	if len(s.projectList) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorFgMuted).Render("  No projects configured.\n  Run glab-pipe . in a git repo to add one."))
	} else {
		for i, p := range s.projectList {
			var row string
			if i == s.listCursor {
				row = lipgloss.NewStyle().Foreground(orange).Bold(true).Render("> " + p.DisplayName)
			} else {
				row = lipgloss.NewStyle().Foreground(colorFg).Render("  " + p.DisplayName)
			}
			sb.WriteString(row + "\n")
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(orange).
		Width(mw).
		Padding(1, 1).
		Render(sb.String())
}
