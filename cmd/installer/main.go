package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// getGitLabHost returns the GitLab hostname from environment variable or from glab config.
// Users can set GITLAB_HOST environment variable to configure their GitLab instance.
// If not set, tries to get the hostname from glab configuration.
func getGitLabHost() string {
	// First check environment variable
	if host := os.Getenv("GITLAB_HOST"); host != "" {
		return host
	}

	// Try to get from glab auth status - use CombinedOutput so we get output even on non-zero exit
	// (glab auth status exits non-zero when ANY configured instance has issues, even if others are fine)
	cmd := exec.Command("glab", "auth", "status")
	output, _ := cmd.CombinedOutput()
	if len(output) > 0 {
		lines := strings.Split(string(output), "\n")
		var loggedInHost string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Look for "Logged in to <hostname>" — this is the most reliable indicator
			if strings.Contains(line, "Logged in to") {
				parts := strings.Fields(line)
				for i, part := range parts {
					if part == "to" && i+1 < len(parts) {
						host := strings.Trim(parts[i+1], ".,;")
						if host != "" && host != "gitlab.com" {
							return host // Return immediately — non-public host with confirmed login
						}
						if host == "gitlab.com" && loggedInHost == "" {
							loggedInHost = host
						}
					}
				}
			}
		}
		if loggedInHost != "" {
			return loggedInHost
		}
	}

	return ""
}

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	appName    = "glab-pipe"
	binaryName = "glab-pipe.exe"
	installDir = "glab-pipe"
	version    = "1.0.0"
)

// ── Colors ────────────────────────────────────────────────────────────────────

var (
	colorBg         = lipgloss.Color("#0d1117")
	colorAccentBlue = lipgloss.Color("#58a6ff")
	colorFg         = lipgloss.Color("#c9d1d9")
	colorFgMuted    = lipgloss.Color("#6e7681")
	colorOrange     = lipgloss.Color("#e8912d")
	colorGreen      = lipgloss.Color("#3fb950")
	colorRed        = lipgloss.Color("9")
	colorYellow     = lipgloss.Color("11")
	colorGray       = lipgloss.Color("8")
	colorCyan       = lipgloss.Color("14")
	colorWhite      = lipgloss.Color("15")
)

// ── Check result types ────────────────────────────────────────────────────────

type CheckStatus int

const (
	CheckPending CheckStatus = iota
	CheckOK
	CheckWarning
	CheckError
)

type Check struct {
	Name    string
	Status  CheckStatus
	Message string
}

// ── Install step messages ─────────────────────────────────────────────────────

type checksCompleteMsg struct{ checks []Check }
type installStepMsg struct{ step string }
type installDoneMsg struct {
	targetDir string
	psProfile string
	success   bool
	errorMsg  string
}

// ── Model ─────────────────────────────────────────────────────────────────────

type Phase int

const (
	PhaseWelcome Phase = iota
	PhaseChecking
	PhaseConfirm
	PhaseInstalling
	PhaseDone
)

type Model struct {
	width   int
	height  int
	phase   Phase
	spinner spinner.Model

	checks []Check
	cursor int // confirm screen: 0=install, 1=quit

	installSteps []string
	installErr   string
	targetDir    string
	psProfile    string

	quit bool
}

func newModel() Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorOrange)

	return Model{
		phase:   PhaseWelcome,
		spinner: sp,
		cursor:  0,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, autoAdvanceWelcome())
}

// autoAdvanceWelcome moves from welcome to checking after a short display.
func autoAdvanceWelcome() tea.Cmd {
	// Immediately start checking (no artificial delay)
	return func() tea.Msg {
		return struct{}{}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case struct{}: // welcome advance
		if m.phase == PhaseWelcome {
			m.phase = PhaseChecking
			return m, runChecks()
		}

	case checksCompleteMsg:
		m.checks = msg.checks
		m.phase = PhaseConfirm
		return m, nil

	case installStepMsg:
		m.installSteps = append(m.installSteps, msg.step)
		return m, nil

	case installDoneMsg:
		m.targetDir = msg.targetDir
		m.psProfile = msg.psProfile
		if msg.success {
			m.installErr = ""
		} else {
			m.installErr = msg.errorMsg
		}
		m.phase = PhaseDone
		return m, nil

	case tea.KeyMsg:
		switch m.phase {
		case PhaseConfirm:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < 1 {
					m.cursor++
				}
			case "enter":
				if m.cursor == 0 {
					m.phase = PhaseInstalling
					return m, tea.Batch(m.spinner.Tick, runInstall())
				}
				m.quit = true
				return m, tea.Quit
			case "q", "ctrl+c", "esc":
				m.quit = true
				return m, tea.Quit
			}
		case PhaseDone:
			switch msg.String() {
			case "q", "ctrl+c", "esc", "enter":
				return m, tea.Quit
			}
		default:
			if msg.String() == "ctrl+c" || msg.String() == "q" {
				m.quit = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	switch m.phase {
	case PhaseWelcome:
		return m.viewWelcome()
	case PhaseChecking:
		return m.viewChecking()
	case PhaseConfirm:
		return m.viewConfirm()
	case PhaseInstalling:
		return m.viewInstalling()
	case PhaseDone:
		return m.viewDone()
	}
	return ""
}

// ── Views ─────────────────────────────────────────────────────────────────────

func (m Model) header() string {
	ascii := `  ██████╗ ██╗      █████╗ ██████╗       ██████╗ ██╗██████╗ ███████╗
 ██╔════╝ ██║     ██╔══██╗██╔══██╗      ██╔══██╗██║██╔══██╗██╔════╝
 ██║  ███╗██║     ███████║██████╔╝█████╗██████╔╝██║██████╔╝█████╗
 ██║   ██║██║     ██╔══██║██╔══██╗╚════╝██╔═══╝ ██║██╔═══╝ ██╔══╝
 ╚██████╔╝███████╗██║  ██║██████╔╝      ██║     ██║██║     ███████╗
  ╚═════╝ ╚══════╝╚═╝  ╚═╝╚═════╝       ╚═╝     ╚═╝╚═╝     ╚══════╝`

	return lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render(ascii)
}

func (m Model) subheader(sub string) string {
	return lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Foreground(colorAccentBlue).Render("GitLab Pipeline Management Tool"),
		lipgloss.NewStyle().Foreground(colorFgMuted).Render(sub),
	)
}

func (m Model) sep() string {
	return lipgloss.NewStyle().Foreground(colorFgMuted).Render(strings.Repeat("─", 54))
}

func (m Model) viewWelcome() string {
	content := lipgloss.JoinVertical(lipgloss.Center,
		m.header(),
		"",
		m.subheader("Installer — v"+version),
		"",
		m.sep(),
		"",
		lipgloss.NewStyle().Foreground(colorFgMuted).Render("Checking dependencies…"),
		m.spinner.View(),
	)
	return m.fullScreen(content)
}

func (m Model) viewChecking() string {
	content := lipgloss.JoinVertical(lipgloss.Center,
		m.header(),
		"",
		m.subheader("Checking dependencies…"),
		"",
		m.sep(),
		"",
		m.spinner.View()+" Checking…",
	)
	return m.fullScreen(content)
}

func (m Model) viewConfirm() string {
	var checksStr strings.Builder
	for _, c := range m.checks {
		var icon, col string
		switch c.Status {
		case CheckOK:
			icon = "✓"
			col = string(colorGreen)
		case CheckWarning:
			icon = "⚠"
			col = string(colorYellow)
		case CheckError:
			icon = "✗"
			col = string(colorRed)
		default:
			icon = "?"
			col = string(colorGray)
		}
		iconRendered := lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Render(icon)
		nameRendered := lipgloss.NewStyle().Foreground(colorFg).Render(c.Name)
		msgRendered := lipgloss.NewStyle().Foreground(colorFgMuted).Render(c.Message)
		checksStr.WriteString(fmt.Sprintf("  %s  %s — %s\n", iconRendered, nameRendered, msgRendered))
	}

	opt0Style := lipgloss.NewStyle().Foreground(colorFg)
	opt1Style := lipgloss.NewStyle().Foreground(colorFg)
	cursor0 := "  "
	cursor1 := "  "

	if m.cursor == 0 {
		opt0Style = lipgloss.NewStyle().Foreground(colorOrange).Bold(true)
		cursor0 = lipgloss.NewStyle().Foreground(colorOrange).Render("> ")
	} else {
		opt1Style = lipgloss.NewStyle().Foreground(colorOrange).Bold(true)
		cursor1 = lipgloss.NewStyle().Foreground(colorOrange).Render("> ")
	}

	targetDir := getInstallDir()
	infoText := lipgloss.NewStyle().Foreground(colorFgMuted).Render(
		fmt.Sprintf("  Installing to: %s", targetDir),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		m.header(),
		"",
		m.subheader("Installer — v"+version),
		"",
		m.sep(),
		"",
		checksStr.String(),
		m.sep(),
		"",
		infoText,
		"",
		cursor0+opt0Style.Render("  Install glab-pipe"),
		cursor1+opt1Style.Render("  Cancel"),
		"",
		lipgloss.NewStyle().Foreground(colorFgMuted).Render("↑↓: select   enter: confirm   ctrl+c: quit"),
	)

	return m.fullScreen(content)
}

func (m Model) viewInstalling() string {
	var stepsStr strings.Builder
	for _, s := range m.installSteps {
		stepsStr.WriteString("  " + lipgloss.NewStyle().Foreground(colorGreen).Render("✓") + "  " + s + "\n")
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		m.header(),
		"",
		m.subheader("Installing…"),
		"",
		m.sep(),
		"",
		stepsStr.String(),
		"",
		m.spinner.View()+" Installing…",
	)
	return m.fullScreen(content)
}

func (m Model) viewDone() string {
	var resultContent string
	if m.installErr != "" {
		resultContent = lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("  Installation error:"),
			"",
			lipgloss.NewStyle().Foreground(colorRed).Render(m.installErr),
		)
	} else {
		var stepsStr strings.Builder
		for _, s := range m.installSteps {
			stepsStr.WriteString("  " + lipgloss.NewStyle().Foreground(colorGreen).Render("✓") + "  " + s + "\n")
		}

		usage := lipgloss.NewStyle().Foreground(colorCyan).Render(fmt.Sprintf(`
  How to use:
    1. Restart PowerShell
    2. Run: glab-pipe

  To use immediately (PowerShell):
    $env:PATH += ";%s"

  Available commands:
    glab-pipe                      → main menu
    glab-pipe .                    → pipelines for current project
    glab-pipe --source <path>      → pipelines by GitLab path
`, m.targetDir))

		resultContent = lipgloss.JoinVertical(lipgloss.Left,
			stepsStr.String(),
			m.sep(),
			usage,
		)
	}

	title := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("  Installation Complete!")
	if m.installErr != "" {
		title = lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("  Installation Failed")
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		m.header(),
		"",
		m.subheader("Installer — v"+version),
		"",
		m.sep(),
		"",
		title,
		"",
		resultContent,
		"",
		lipgloss.NewStyle().Foreground(colorFgMuted).Render("Press any key to exit…"),
	)

	return m.fullScreen(content)
}

func (m Model) fullScreen(content string) string {
	return lipgloss.NewStyle().
		Background(colorBg).
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

// ── Async commands ────────────────────────────────────────────────────────────

func runChecks() tea.Cmd {
	return func() tea.Msg {
		checks := []Check{
			checkGlab(),
			checkNerdFont(),
			checkPowerShell(),
		}
		return checksCompleteMsg{checks}
	}
}

func checkGlab() Check {
	c := Check{Name: "glab CLI"}
	out, err := exec.Command("glab", "version").Output()
	if err != nil {
		c.Status = CheckError
		c.Message = "Not found. Install with: scoop install glab"
		return c
	}
	// Check if configured
	authOut, err := exec.Command("glab", "auth", "status").Output()
	if err != nil {
		// glab auth status returns non-zero if not logged in
		authOut = []byte{}
	}
	host := getGitLabHost()
	if host != "" {
		if strings.Contains(string(authOut), host) || strings.Contains(string(authOut), "logged in") {
			c.Status = CheckOK
			c.Message = strings.TrimSpace(strings.Split(string(out), "\n")[0])
		} else {
			c.Status = CheckWarning
			c.Message = fmt.Sprintf("Installed but may not be configured for %s", host)
		}
	} else {
		if strings.Contains(string(authOut), "logged in") {
			c.Status = CheckOK
			c.Message = strings.TrimSpace(strings.Split(string(out), "\n")[0])
		} else {
			c.Status = CheckWarning
			c.Message = "Installed but may not be configured"
		}
	}
	return c
}

func checkNerdFont() Check {
	// Fonts cannot be reliably detected from Go — warn the user instead.
	return Check{
		Name:    "Nerd Font",
		Status:  CheckWarning,
		Message: "Cannot be verified automatically. Icons won't render correctly without a Nerd Font.",
	}
}

func checkPowerShell() Check {
	shell := os.Getenv("PSModulePath")
	if shell != "" || isPowerShell() {
		return Check{
			Name:    "PowerShell",
			Status:  CheckOK,
			Message: "Detected",
		}
	}
	return Check{
		Name:    "PowerShell",
		Status:  CheckWarning,
		Message: "Recommended for best color and icon support",
	}
}

func isPowerShell() bool {
	_, err := exec.Command("powershell", "-Command", "$PSVersionTable").Output()
	return err == nil
}

func getInstallDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "AppData", "Local", installDir)
}

func runInstall() tea.Cmd {
	return func() tea.Msg {
		steps := []string{}
		targetDir := getInstallDir()

		// 1. Create install directory
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return installDoneMsg{success: false, errorMsg: fmt.Sprintf("Error creating install directory: %v", err)}
		}
		steps = append(steps, fmt.Sprintf("Install directory: %s", targetDir))

		// 2. Find source binary
		installerPath, err := os.Executable()
		if err != nil {
			return installDoneMsg{success: false, errorMsg: fmt.Sprintf("Error locating installer: %v", err)}
		}
		installerDir := filepath.Dir(installerPath)

		sourceBinary := filepath.Join(installerDir, binaryName)
		if _, err := os.Stat(sourceBinary); os.IsNotExist(err) {
			// Try dist/ subdirectory
			sourceBinary = filepath.Join(installerDir, "dist", binaryName)
			if _, err := os.Stat(sourceBinary); os.IsNotExist(err) {
				// Try gitlab-pipeline.exe as fallback name
				sourceBinary = filepath.Join(installerDir, "gitlab-pipeline.exe")
				if _, err := os.Stat(sourceBinary); os.IsNotExist(err) {
					return installDoneMsg{
						success:  false,
						errorMsg: fmt.Sprintf("Binary not found in: %s\nBuild it first with: build.bat", installerDir),
					}
				}
			}
		}

		// 3. Copy binary
		targetBinary := filepath.Join(targetDir, binaryName)
		if err := copyFile(sourceBinary, targetBinary); err != nil {
			return installDoneMsg{success: false, errorMsg: fmt.Sprintf("Error copying binary: %v", err)}
		}
		steps = append(steps, fmt.Sprintf("Binary copied to: %s", targetBinary))

		// 4. Add to PowerShell PATH via profile
		home, err := os.UserHomeDir()
		if err != nil {
			return installDoneMsg{success: false, errorMsg: fmt.Sprintf("Error locating home directory: %v", err)}
		}

		psProfileDir := filepath.Join(home, "Documents", "PowerShell")
		psProfilePath := filepath.Join(psProfileDir, "Microsoft.PowerShell_profile.ps1")

		if err := os.MkdirAll(psProfileDir, 0755); err != nil {
			return installDoneMsg{success: false, errorMsg: fmt.Sprintf("Error creating PowerShell profile directory: %v", err)}
		}

		profileContent := ""
		if data, err := os.ReadFile(psProfilePath); err == nil {
			profileContent = string(data)
		}

		pathEntry := fmt.Sprintf(`
# glab-pipe — GitLab Pipeline Viewer
if ($env:PATH -notlike "*%s*") {
    $env:PATH = "%s;" + $env:PATH
}
function glab-pipe { & "%s" @args }
`, targetDir, targetDir, targetBinary)

		if !strings.Contains(profileContent, "glab-pipe") {
			newContent := profileContent
			if newContent != "" && !strings.HasSuffix(newContent, "\n") {
				newContent += "\n"
			}
			newContent += pathEntry + "\n"
			if err := os.WriteFile(psProfilePath, []byte(newContent), 0644); err != nil {
				return installDoneMsg{success: false, errorMsg: fmt.Sprintf("Error updating PowerShell profile: %v", err)}
			}
			steps = append(steps, "PowerShell profile updated: "+psProfilePath)
		} else {
			steps = append(steps, "PowerShell profile already configured")
		}

		// 5. Also add to CMD via user PATH in registry (Windows)
		addToRegistryPath(targetDir)
		steps = append(steps, "User PATH updated")

		return installDoneMsg{
			targetDir: targetDir,
			psProfile: psProfilePath,
			success:   true,
		}
	}
}

// addToRegistryPath adds targetDir to the user PATH in Windows registry.
// Uses PowerShell + .NET to read/write only the User PATH from the registry,
// avoiding the setx 1024-char truncation bug that destroys PATH when the
// combined User+System PATH is accidentally passed as the new value.
func addToRegistryPath(targetDir string) {
	script := fmt.Sprintf(`
$regPath = 'HKCU:\Environment'
$current = (Get-ItemProperty -Path $regPath -Name PATH -ErrorAction SilentlyContinue).PATH
if ($null -eq $current) { $current = '' }
if ($current -notlike '*%s*') {
    $newPath = '%s;' + $current
    Set-ItemProperty -Path $regPath -Name PATH -Value $newPath
}
`, targetDir, targetDir)
	exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Run()
}

// ── File helpers ──────────────────────────────────────────────────────────────

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	m := newModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Installer error: %v\n", err)
		os.Exit(1)
	}
}
