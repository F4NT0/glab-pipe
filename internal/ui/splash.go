package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"gitlab-pipeline-tui/internal/config"
	gl "gitlab-pipeline-tui/internal/gitlab"
)

// splashMode tracks which sub-screen is active inside the splash.
type splashMode int

const (
	splashMenu       splashMode = iota // main two-option menu
	splashProjectList                  // list of saved projects
	splashAddRepo                      // text input for new repo path
)

// SplashModel is the initial welcome/splash screen shown before the main TUI.
type SplashModel struct {
	width  int
	height int
	quit   bool

	mode splashMode

	// main menu
	menuCursor int // 0 = Select Repository, 1 = Choose another

	// project list sub-screen
	projectList []gl.Project
	listCursor  int

	// add-repo sub-screen
	inputValue  string
	inputErr    string
	inputOK     string

	selectedProject *gl.Project
}

func NewSplashModel() SplashModel {
	return SplashModel{
		projectList: gl.ProjectList(),
		mode:        splashMenu,
	}
}

func (s SplashModel) Quit() bool             { return s.quit }
func (s SplashModel) SelectedProject() *gl.Project { return s.selectedProject }
func (s SplashModel) Init() tea.Cmd          { return nil }

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
		case splashAddRepo:
			return s.updateAddRepo(key, msg)
		}
	}
	return s, nil
}

func (s SplashModel) updateMenu(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q", "esc":
		s.quit = true
		return s, tea.Quit
	case "up", "k":
		if s.menuCursor > 0 {
			s.menuCursor--
		}
	case "down", "j":
		if s.menuCursor < 1 {
			s.menuCursor++
		}
	case "enter", " ":
		switch s.menuCursor {
		case 0:
			s.projectList = gl.ProjectList()
			s.listCursor = 0
			s.mode = splashProjectList
		case 1:
			s.inputValue = ""
			s.inputErr = ""
			s.inputOK = ""
			s.mode = splashAddRepo
		}
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

func (s SplashModel) updateAddRepo(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c":
		s.quit = true
		return s, tea.Quit
	case "esc", "backspace":
		if s.inputValue == "" {
			s.mode = splashMenu
		} else {
			// backspace one char
			r := []rune(s.inputValue)
			if len(r) > 0 {
				s.inputValue = string(r[:len(r)-1])
			}
			s.inputErr = ""
			s.inputOK = ""
		}
	case "enter":
		path := strings.TrimSpace(s.inputValue)
		if path == "" {
			s.inputErr = "Path cannot be empty."
			return s, nil
		}
		displayName, err := gl.ValidateProject(path)
		if err != nil {
			s.inputErr = "No access or project not found: " + path
			return s, nil
		}
		proj := config.Project{DisplayName: displayName, FullPath: path}
		_ = config.AddProject(proj)
		s.projectList = gl.ProjectList()
		s.inputOK = "Project \"" + displayName + "\" added!"
		s.inputValue = ""
		s.inputErr = ""
		// auto-select the newly added project
		s.selectedProject = &proj
		return s, tea.Quit
	default:
		if len(msg.Runes) > 0 {
			s.inputValue += string(msg.Runes)
			s.inputErr = ""
			s.inputOK = ""
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
	case splashAddRepo:
		modal = s.renderAddRepoModal()
	}

	var hintLine string
	switch s.mode {
	case splashMenu:
		hintLine = muted.Render("↑↓: navigate   enter: select   ctrl+c: quit")
	case splashProjectList:
		hintLine = muted.Render("↑↓: navigate   enter: open   esc: back   ctrl+c: quit")
	case splashAddRepo:
		hintLine = muted.Render("type path   enter: confirm   esc: back   ctrl+c: quit")
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
	w := 64
	if w > s.width-4 {
		w = s.width - 4
	}
	return w
}

func (s SplashModel) renderMenuModal() string {
	orange := colorOrange
	mw := s.modalWidth()

	options := []string{"Select Repository...", "Choose another..."}
	var sb strings.Builder
	titleLine := lipgloss.NewStyle().Foreground(orange).Bold(true).Render("  Projects")
	sb.WriteString(titleLine + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(gray).Render("  " + strings.Repeat("─", mw-6) + "\n"))

	for i, opt := range options {
		var row string
		if i == s.menuCursor {
			row = lipgloss.NewStyle().Foreground(orange).Bold(true).Render(" > " + opt)
		} else {
			row = lipgloss.NewStyle().Foreground(colorFg).Render("  " + opt)
		}
		sb.WriteString(row + "\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(orange).
		Width(mw).
		Padding(0, 1).
		Render(sb.String())
}

func (s SplashModel) renderProjectListModal() string {
	orange := colorOrange
	mw := s.modalWidth()

	var sb strings.Builder
	titleLine := lipgloss.NewStyle().Foreground(orange).Bold(true).Render("  Select Repository")
	sb.WriteString(titleLine + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(gray).Render("  " + strings.Repeat("─", mw-6) + "\n"))

	for i, p := range s.projectList {
		var row string
		if i == s.listCursor {
			row = lipgloss.NewStyle().Foreground(orange).Bold(true).Render("> " + p.DisplayName)
		} else {
			row = lipgloss.NewStyle().Foreground(colorFg).Render("  " + p.DisplayName)
		}
		sb.WriteString(row + "\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(orange).
		Width(mw).
		Padding(0, 1).
		Render(sb.String())
}

func (s SplashModel) renderAddRepoModal() string {
	orange := colorOrange
	mw := s.modalWidth()

	cursor := lipgloss.NewStyle().Foreground(orange).Render("_")
	inputLine := lipgloss.NewStyle().Foreground(colorFg).Render(s.inputValue) + cursor

	var statusLine string
	if s.inputErr != "" {
		statusLine = "\n" + lipgloss.NewStyle().Foreground(red).Render("  ✗ " + s.inputErr)
	} else if s.inputOK != "" {
		statusLine = "\n" + lipgloss.NewStyle().Foreground(green).Render("  ✓ " + s.inputOK)
	}

	var sb strings.Builder
	titleLine := lipgloss.NewStyle().Foreground(orange).Bold(true).Render("  Add Repository")
	sb.WriteString(titleLine + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(gray).Render("  " + strings.Repeat("─", mw-6) + "\n"))
	if s.inputValue == "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(gray).Render("  GitLab path or URL:\n"))
	}
	sb.WriteString("  " + inputLine)
	if statusLine != "" {
		sb.WriteString(statusLine)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(orange).
		Width(mw).
		Padding(0, 1).
		Render(sb.String())
}
