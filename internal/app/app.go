package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	gl "gitlab-pipeline-tui/internal/gitlab"
)

// ── Screens ───────────────────────────────────────────────────────────────────

type Screen int

const (
	ScreenWelcome        Screen = iota
	ScreenPipelines             // pipeline table
	ScreenJobs                  // job list for a selected pipeline
	ScreenJobLog                // modal with job trace logs
	ScreenCreatePipeline        // modal for creating new pipeline
	ScreenClonePrompt           // modal for prompting project clone
	ScreenJobRun                // modal for running/retrying a job
)

// ── Auto-refresh timing ───────────────────────────────────────────────────────

const refreshInterval = 2 * time.Second

// ── Async messages ────────────────────────────────────────────────────────────

type pipelinesLoadedMsg struct{ items []gl.PipelineListItem }
type detailLoadedMsg struct{ detail *gl.PipelineDetail }
type jobTraceLoadedMsg struct {
	jobName string
	trace   string
}
type pipelineCreatedMsg struct {
	pipelineID uint64
}
type pipelineCanceledMsg struct{}
type copyDoneMsg struct {
	feedback string
}
type jobRunMsg struct{
	success bool
	error   string
}
type tickMsg time.Time
type errMsg struct{ err error }

// Project clone messages
type projectCheckMsg struct {
	exists    bool
	localPath string
}
type projectCloneMsg struct {
	localPath string
	success   bool
	error     string
}

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	screen Screen

	// Project selection (welcome)
	projects            []gl.Project
	projectCursor       int
	showProjectSelector bool
	projectListOnly     bool // true when showing only project list (after Esc back)

	// Pipeline table
	selectedProject *gl.Project
	pipelines       []gl.PipelineListItem
	pipelineCursor  int
	pipelineScroll  int

	// Job list (detail screen)
	detail    *gl.PipelineDetail
	jobCursor int
	jobScroll int

	// Job log modal
	selectedJob   *gl.Job
	jobTrace      string
	traceLoading  bool
	traceViewport *viewport.Model

	// Pipeline creation modal
	createPipelineBranch        string
	createPipelineVariables     string
	createPipelineError         string
	createPipelineInputField    int    // 0 = branch, 1 = variables
	createPipelineDisplayBranch string // The normalized branch name that will actually be used

	// Job run modal
	jobRunJobID        uint64 // the job being run/retried
	jobRunVariables    string // variables in key:value,key:value format
	jobRunError        string // error message if job run fails
	jobRunIsRetry      bool   // true if retry, false if manual trigger
	jobRunConfirming   bool   // true when confirm step is shown

	// Pipeline screen overlays
	copyChoiceOpen bool   // whether copy-choice overlay is visible
	copyFeedback   string // brief feedback after clipboard copy

	// Create pipeline confirm step
	createConfirming bool // true when showing confirm overlay

	// Project clone functionality
	projectCloneRequested  bool   // Whether clone was requested
	projectCloneInProgress bool   // Whether clone is in progress
	projectCloneError      string // Error message if clone failed
	projectLocalPath       string // Local path of the project after clone

	// Loading / error / status
	spinner   spinner.Model
	loading   bool
	statusMsg string
	errText   string

	// Terminal dimensions
	width  int
	height int
}

func New() Model {
	return NewWithProject(nil)
}

func NewWithProject(selectedProj *gl.Project) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	vp := viewport.New(0, 0)
	vp.SetContent("")

	m := Model{
		screen:              ScreenWelcome,
		projects:            gl.ProjectList(),
		showProjectSelector: true,
		spinner:             sp,
		traceViewport:       &vp,
	}

	if selectedProj != nil {
		m.selectedProject = selectedProj
		m.screen = ScreenPipelines
		m.loading = true
		m.showProjectSelector = false
	}

	return m
}

// ── Init ──────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	if m.selectedProject != nil && m.loading {
		return tea.Batch(m.spinner.Tick, m.loadPipelines())
	}
	return m.spinner.Tick
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update viewport dimensions if in job log screen
		if m.screen == ScreenJobLog && m.traceViewport != nil {
			vpWidth := m.width - 2
			if vpWidth < 20 {
				vpWidth = 20
			}
			// height: total - title(1) - headerBox(4) - statusbar(1)
			vpHeight := m.height - 6
			if vpHeight < 10 {
				vpHeight = 10
			}
			m.traceViewport.Width = vpWidth
			m.traceViewport.Height = vpHeight
			// Re-truncate content if it exists
			if m.jobTrace != "" {
				m.traceViewport.SetContent(truncateLines(m.jobTrace, vpWidth))
			}
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading || m.traceLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tickMsg:
		// Auto-refresh tick — only fires while something is still running.
		switch m.screen {
		case ScreenPipelines:
			if m.selectedProject != nil && m.hasRunningPipeline() {
				return m, tea.Batch(m.spinner.Tick, m.loadPipelines())
			}
		case ScreenJobs:
			if m.detail != nil && (gl.IsRunning(m.detail.Status) || gl.HasAnyRunning(m.detail.Jobs)) {
				return m, tea.Batch(m.spinner.Tick, m.loadDetail(m.detail.ID))
			}
		case ScreenJobLog:
			if m.selectedJob != nil && gl.IsRunning(m.selectedJob.Status) {
				return m, tea.Batch(m.spinner.Tick, m.loadJobTrace(m.selectedJob.ID))
			}
		}
		return m, nil

	case pipelinesLoadedMsg:
		m.loading = false
		m.pipelines = msg.items
		// Keep cursor in bounds
		if m.pipelineCursor >= len(m.pipelines) {
			m.pipelineCursor = 0
		}
		// Schedule next refresh if any pipeline is running
		var nextTick tea.Cmd
		if m.hasRunningPipeline() {
			nextTick = scheduleRefresh()
		}
		return m, nextTick

	case detailLoadedMsg:
		m.loading = false
		m.detail = msg.detail
		if m.jobCursor >= len(m.detail.Jobs) {
			m.jobCursor = 0
		}
		// Schedule next refresh if pipeline/jobs still running
		var nextTick tea.Cmd
		if gl.IsRunning(m.detail.Status) || gl.HasAnyRunning(m.detail.Jobs) {
			nextTick = scheduleRefresh()
		}
		return m, nextTick

	case jobTraceLoadedMsg:
		m.traceLoading = false
		m.jobTrace = msg.trace
		// Truncate lines to viewport width to prevent TUI border breakage
		vpWidth := m.traceViewport.Width
		if vpWidth <= 0 {
			vpWidth = m.width - 8
		}
		// Ensure minimum width to prevent issues
		if vpWidth < 20 {
			vpWidth = 20
		}
		m.traceViewport.SetContent(truncateLines(msg.trace, vpWidth))
		m.traceViewport.GotoBottom()
		// If job is still running, keep refreshing
		var nextTick tea.Cmd
		if m.selectedJob != nil && gl.IsRunning(m.selectedJob.Status) {
			nextTick = scheduleRefresh()
		}
		return m, nextTick

	case pipelineCreatedMsg:
		m.loading = false
		m.createPipelineError = ""
		m.createConfirming = false
		m.screen = ScreenPipelines
		// Reload pipelines to show the newly created one
		return m, tea.Batch(m.spinner.Tick, m.loadPipelines())

	case pipelineCanceledMsg:
		m.loading = false
		return m, tea.Batch(m.spinner.Tick, m.loadPipelines())

	case copyDoneMsg:
		m.copyFeedback = msg.feedback
		m.copyChoiceOpen = false

	case jobRunMsg:
		m.loading = false
		if msg.success {
			// Job run/retry successful, go back to jobs and refresh
			m.screen = ScreenJobs
			m.jobRunJobID = 0
			m.jobRunVariables = ""
			m.jobRunError = ""
			m.jobRunConfirming = false
			return m, tea.Batch(m.spinner.Tick, m.loadDetail(m.detail.ID))
		} else {
			m.jobRunConfirming = false
			m.jobRunError = msg.error
		}

	case projectCheckMsg:
		if msg.exists {
			// Project exists locally, proceed to pipelines
			m.screen = ScreenPipelines
			m.loading = true
			m.projectLocalPath = msg.localPath
			return m, tea.Batch(m.spinner.Tick, m.loadPipelines())
		} else {
			// Project doesn't exist locally, show clone prompt
			m.screen = ScreenClonePrompt
			m.projectCloneRequested = true
			m.projectCloneInProgress = false
			m.projectCloneError = ""
			m.projectLocalPath = ""
			return m, nil
		}

	case projectCloneMsg:
		m.projectCloneInProgress = false
		if msg.success {
			// Clone successful, proceed to pipelines
			m.projectCloneRequested = false
			m.projectLocalPath = msg.localPath
			m.screen = ScreenPipelines
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadPipelines())
		} else {
			// Clone failed
			m.projectCloneError = msg.error
			return m, nil
		}

	case errMsg:
		m.loading = false
		m.traceLoading = false
		m.errText = msg.err.Error()
		return m, nil

	case tea.KeyMsg:
		if m.errText != "" {
			m.errText = ""
			return m, nil
		}
		if m.loading && m.screen != ScreenJobLog {
			return m, nil
		}
		return m.handleKey(msg.String())
	}

	return m, nil
}

func (m Model) handleKey(key string) (Model, tea.Cmd) {
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.screen {
	case ScreenWelcome:
		return m.handleWelcomeKey(key)
	case ScreenPipelines:
		return m.handlePipelineKey(key)
	case ScreenJobs:
		return m.handleJobsKey(key)
	case ScreenJobLog:
		return m.handleJobLogKey(key)
	case ScreenCreatePipeline:
		return m.handleCreatePipelineKey(key)
	case ScreenClonePrompt:
		return m.handleClonePromptKey(key)
	case ScreenJobRun:
		return m.handleJobRunKey(key)
	}
	return m, nil
}

// ── Screen: Welcome ───────────────────────────────────────────────────────────

func (m Model) handleWelcomeKey(key string) (Model, tea.Cmd) {
	if !m.showProjectSelector {
		switch key {
		case "enter", " ":
			m.showProjectSelector = true
		case "q", "esc":
			return m, tea.Quit
		}
		return m, nil
	}

	switch key {
	case "up", "k":
		if m.projectCursor > 0 {
			m.projectCursor--
		}
	case "down", "j":
		if m.projectCursor < len(m.projects)-1 {
			m.projectCursor++
		}
	case "enter":
		if len(m.projects) == 0 {
			return m, nil
		}
		proj := m.projects[m.projectCursor]
		m.selectedProject = &proj
		m.showProjectSelector = false
		m.projectListOnly = false
		m.pipelineCursor = 0
		m.pipelineScroll = 0
		m.screen = ScreenPipelines
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPipelines())
	case "esc", "q":
		m.showProjectSelector = false
		return m, tea.Quit
	}
	return m, nil
}

// ── Screen: Pipelines ─────────────────────────────────────────────────────────

func (m Model) handlePipelineKey(key string) (Model, tea.Cmd) {
	// Route to overlay handlers when they are open
	if m.copyChoiceOpen {
		return m.handleCopyChoiceKey(key)
	}

	// Clear copy feedback on any navigation key
	if m.copyFeedback != "" {
		m.copyFeedback = ""
	}

	switch key {
	case "up", "k":
		if m.pipelineCursor > 0 {
			m.pipelineCursor--
			m.syncPipelineScroll()
		}
	case "down", "j":
		if m.pipelineCursor < len(m.pipelines)-1 {
			m.pipelineCursor++
			m.syncPipelineScroll()
		}
	case "enter":
		return m.openPipelineDetail()
	case "r", "R":
		// Re-run same branch silently
		if len(m.pipelines) == 0 {
			return m, nil
		}
		pip := m.pipelines[m.pipelineCursor]
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.reRunPipeline(pip.GitRef))
	case "u", "U":
		// Manual refresh
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPipelines())
	case "x", "X":
		// Cancel the selected pipeline
		if len(m.pipelines) == 0 {
			return m, nil
		}
		pip := m.pipelines[m.pipelineCursor]
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.cancelPipeline(pip.ID))
	case "c", "C":
		// Open copy-choice overlay
		if len(m.pipelines) > 0 {
			m.copyChoiceOpen = true
		}
	case "n", "N":
		// Open create pipeline modal
		m.screen = ScreenCreatePipeline
		m.createPipelineBranch = ""
		m.createPipelineVariables = ""
		m.createPipelineError = ""
		m.createPipelineInputField = 0
		m.createPipelineDisplayBranch = ""
		m.createConfirming = false
	case "esc", "backspace":
		// Back to welcome / project selector
		m.screen = ScreenWelcome
		m.showProjectSelector = true
		m.projectListOnly = true
		m.selectedProject = nil
		m.pipelines = nil
		m.pipelineCursor = 0
		m.pipelineScroll = 0
		m.detail = nil
		m.jobCursor = 0
		m.jobScroll = 0
		m.selectedJob = nil
		m.jobTrace = ""
		m.traceLoading = false
		m.copyChoiceOpen = false
		m.copyFeedback = ""
		if m.traceViewport != nil {
			m.traceViewport.SetContent("")
			m.traceViewport.GotoTop()
		}
		m.loading = false
		// Reload project list to ensure fresh data
		m.projects = gl.ProjectList()
		m.projectCursor = 0
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleCopyChoiceKey(key string) (Model, tea.Cmd) {
	if len(m.pipelines) == 0 {
		m.copyChoiceOpen = false
		return m, nil
	}
	pip := m.pipelines[m.pipelineCursor]
	switch key {
	case "esc":
		m.copyChoiceOpen = false
	case "1", "b":
		// Copy branch name
		branch := pip.GitRef
		return m, m.copyToClipboard(branch, "Branch name copied!")
	case "2", "w":
		// Copy pipeline web URL
		url := ""
		if pip.WebURL != nil && *pip.WebURL != "" {
			url = *pip.WebURL
		}
		if url == "" {
			m.copyChoiceOpen = false
			return m, nil
		}
		return m, m.copyToClipboard(url, "Pipeline URL copied!")
	}
	return m, nil
}

func (m Model) openPipelineDetail() (Model, tea.Cmd) {
	if len(m.pipelines) == 0 {
		return m, nil
	}
	pip := m.pipelines[m.pipelineCursor]
	m.screen = ScreenJobs
	m.loading = true
	m.jobCursor = 0
	m.jobScroll = 0
	return m, tea.Batch(m.spinner.Tick, m.loadDetail(pip.ID))
}

// ── Screen: Jobs ──────────────────────────────────────────────────────────────

func (m Model) handleJobsKey(key string) (Model, tea.Cmd) {
	if m.detail == nil {
		if key == "esc" {
			m.screen = ScreenPipelines
		}
		return m, nil
	}
	switch key {
	case "up", "k":
		if m.jobCursor > 0 {
			m.jobCursor--
			m.syncJobScroll()
		}
	case "down", "j":
		if m.jobCursor < len(m.detail.Jobs)-1 {
			m.jobCursor++
			m.syncJobScroll()
		}
	case "enter":
		return m.openJobLog()
	case "r", "R":
		// Open job run modal for the selected job
		return m.openJobRunModal()
	case "u", "U":
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadDetail(m.detail.ID))
	case "esc", "backspace":
		m.screen = ScreenPipelines
		m.detail = nil
		m.jobCursor = 0
		// Restart pipeline refresh if any running
		if m.hasRunningPipeline() {
			return m, scheduleRefresh()
		}
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) openJobLog() (Model, tea.Cmd) {
	if m.detail == nil || len(m.detail.Jobs) == 0 {
		return m, nil
	}
	job := &m.detail.Jobs[m.jobCursor]
	m.selectedJob = job
	m.screen = ScreenJobLog
	m.jobTrace = ""
	m.traceLoading = true
	// Calculate viewport dimensions: width full, height minus header box + title + statusbar
	vpWidth := m.width - 2
	if vpWidth < 20 {
		vpWidth = 20
	}
	vpHeight := m.height - 6
	if vpHeight < 10 {
		vpHeight = 10
	}
	vp := viewport.New(vpWidth, vpHeight)
	vp.SetContent("")
	m.traceViewport = &vp
	return m, tea.Batch(m.spinner.Tick, m.loadJobTrace(job.ID))
}

func (m Model) openJobRunModal() (Model, tea.Cmd) {
	if m.detail == nil || len(m.detail.Jobs) == 0 {
		return m, nil
	}
	job := &m.detail.Jobs[m.jobCursor]
	m.screen = ScreenJobRun
	m.jobRunJobID = job.ID
	m.jobRunVariables = ""
	m.jobRunError = ""
	// Determine if this is a retry (failed job) or manual trigger
	m.jobRunIsRetry = job.Status == "failed"
	return m, nil
}

// ── Screen: Job Log ───────────────────────────────────────────────────────────

func (m Model) handleJobLogKey(key string) (Model, tea.Cmd) {
	switch key {
	case "esc", "backspace":
		m.screen = ScreenJobs
		m.selectedJob = nil
		m.jobTrace = ""
		m.traceViewport.SetContent("")
		m.traceViewport.GotoTop()
		if m.detail != nil && (gl.IsRunning(m.detail.Status) || gl.HasAnyRunning(m.detail.Jobs)) {
			return m, scheduleRefresh()
		}
	case "up", "k":
		m.traceViewport.LineUp(1)
	case "down", "j":
		m.traceViewport.LineDown(1)
	case "pgup", "ctrl+u":
		m.traceViewport.HalfViewUp()
	case "pgdown", "ctrl+d":
		m.traceViewport.HalfViewDown()
	case "g", "home":
		m.traceViewport.GotoTop()
	case "G", "end":
		m.traceViewport.GotoBottom()
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// ── Screen: Job Run ─────────────────────────────────────────────────────────────

func (m Model) handleJobRunKey(key string) (Model, tea.Cmd) {
	// Confirm step — block editing, only accept Enter or Esc
	if m.jobRunConfirming {
		switch key {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.jobRunConfirming = false
		case "enter":
			m.loading = true
			m.jobRunError = ""
			return m, tea.Batch(m.spinner.Tick, m.runJob())
		}
		return m, nil
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		// Cancel and go back to jobs
		m.screen = ScreenJobs
		m.jobRunJobID = 0
		m.jobRunVariables = ""
		m.jobRunError = ""
		m.jobRunConfirming = false
	case "enter":
		// Go to confirm step
		m.jobRunConfirming = true
		m.jobRunError = ""
	case "backspace":
		if len(m.jobRunVariables) > 0 {
			m.jobRunVariables = m.jobRunVariables[:len(m.jobRunVariables)-1]
		}
	default:
		// Allow characters for variables input
		if len(key) == 1 {
			m.jobRunVariables += key
		}
	}
	return m, nil
}

// ── Screen: Create Pipeline ─────────────────────────────────────────────────────

func (m Model) handleCreatePipelineKey(key string) (Model, tea.Cmd) {
	// Block field editing while confirmation overlay is showing
	if m.createConfirming {
		switch key {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.createConfirming = false
		case "enter":
			m.loading = true
			m.createPipelineError = ""
			return m, tea.Batch(m.spinner.Tick, m.createPipeline())
		}
		return m, nil
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		// Cancel and go back to pipelines
		m.screen = ScreenPipelines
		m.createPipelineBranch = ""
		m.createPipelineVariables = ""
		m.createPipelineError = ""
		m.createPipelineInputField = 0
		m.createPipelineDisplayBranch = ""
		m.createConfirming = false
		m.loading = false
	case "enter":
		// Go to confirm step
		if m.createPipelineBranch == "" {
			m.createPipelineError = "Branch name is required"
			return m, nil
		}
		m.createPipelineError = ""
		m.createConfirming = true
	case "tab":
		// Switch between input fields
		m.createPipelineInputField = 1 - m.createPipelineInputField
	case "backspace":
		if m.createPipelineInputField == 0 && len(m.createPipelineBranch) > 0 {
			m.createPipelineBranch = m.createPipelineBranch[:len(m.createPipelineBranch)-1]
			m.createPipelineDisplayBranch = normalizeBranchName(m.createPipelineBranch)
		} else if m.createPipelineInputField == 1 && len(m.createPipelineVariables) > 0 {
			m.createPipelineVariables = m.createPipelineVariables[:len(m.createPipelineVariables)-1]
		}
	case "/", "-", "_", ".":
		// Explicitly allow special characters for branch names
		if m.createPipelineInputField == 0 {
			m.createPipelineBranch += key
			m.createPipelineDisplayBranch = normalizeBranchName(m.createPipelineBranch)
		} else {
			m.createPipelineVariables += key
		}
	default:
		// Allow alphanumeric characters
		if len(key) == 1 {
			if m.createPipelineInputField == 0 {
				// Allow branch name characters (alphanumeric)
				if (key >= "a" && key <= "z") || (key >= "A" && key <= "Z") ||
					(key >= "0" && key <= "9") {
					m.createPipelineBranch += key
					m.createPipelineDisplayBranch = normalizeBranchName(m.createPipelineBranch)
				}
			} else {
				// For variables, allow more characters (including :, ,, =)
				if (key >= "a" && key <= "z") || (key >= "A" && key <= "Z") ||
					(key >= "0" && key <= "9") || key == ":" || key == "," || key == "=" {
					m.createPipelineVariables += key
				}
			}
		}
	}
	return m, nil
}

// ── Screen: Clone Prompt ─────────────────────────────────────────────────────────

func (m Model) handleClonePromptKey(key string) (Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "n":
		// Cancel clone, go back to welcome
		m.screen = ScreenWelcome
		m.showProjectSelector = true
		m.projectCloneRequested = false
		m.projectCloneError = ""
		return m, nil
	case "y", "enter":
		// Confirm clone
		if m.projectCloneInProgress {
			return m, nil // Already cloning
		}
		m.projectCloneInProgress = true
		m.projectCloneError = ""
		// Clone to default directory
		home, err := os.UserHomeDir()
		if err != nil {
			m.projectCloneError = "Could not determine home directory"
			m.projectCloneInProgress = false
			return m, nil
		}
		targetDir := filepath.Join(home, "repos")
		return m, tea.Batch(m.spinner.Tick, m.cloneProject(targetDir))
	}
	return m, nil
}

// normalizeBranchName adds the "story/" prefix only when the input is a bare CUC-XXX ticket code.
func normalizeBranchName(branch string) string {
	branch = strings.TrimSpace(branch)

	// If branch already contains a path separator, return as-is
	if strings.Contains(branch, "/") {
		return branch
	}

	// If it starts with a known long-lived branch prefix, return as-is
	for _, pfx := range []string{"develop", "release", "hotfix", "main", "master"} {
		if strings.HasPrefix(strings.ToLower(branch), pfx) {
			return branch
		}
	}

	// Add release/ for FY... fiscal-year branches
	if strings.HasPrefix(strings.ToUpper(branch), "FY") {
		return "release/" + branch
	}

	// Only add story/ for CUC-XXX pattern
	if matched, _ := regexp.MatchString(`^CUC-\d+`, branch); matched {
		return "story/" + branch
	}

	// All other inputs are returned as-is
	return branch
}

// ── Async commands ────────────────────────────────────────────────────────────

func (m Model) loadPipelines() tea.Cmd {
	fp := m.selectedProject.FullPath
	return func() tea.Msg {
		items, err := gl.FetchPipelineList(fp)
		if err != nil {
			return errMsg{err}
		}
		return pipelinesLoadedMsg{items}
	}
}

func (m Model) loadDetail(pipelineID uint64) tea.Cmd {
	fp := m.selectedProject.FullPath
	return func() tea.Msg {
		d, err := gl.FetchPipelineDetail(fp, pipelineID)
		if err != nil {
			return errMsg{err}
		}
		return detailLoadedMsg{d}
	}
}

func (m Model) loadJobTrace(jobID uint64) tea.Cmd {
	fp := m.selectedProject.FullPath
	return func() tea.Msg {
		trace, err := gl.FetchJobTrace(fp, jobID)
		if err != nil {
			return errMsg{err}
		}
		return jobTraceLoadedMsg{jobName: fmt.Sprintf("%d", jobID), trace: trace}
	}
}

func (m Model) createPipeline() tea.Cmd {
	fp := m.selectedProject.FullPath
	branch := m.createPipelineBranch
	variables := m.createPipelineVariables
	return func() tea.Msg {
		pipelineID, err := gl.CreatePipeline(fp, branch, variables)
		if err != nil {
			return errMsg{err}
		}
		return pipelineCreatedMsg{pipelineID: pipelineID}
	}
}

func (m Model) reRunPipeline(branch string) tea.Cmd {
	fp := m.selectedProject.FullPath
	return func() tea.Msg {
		pipelineID, err := gl.CreatePipeline(fp, branch, "")
		if err != nil {
			return errMsg{err}
		}
		return pipelineCreatedMsg{pipelineID: pipelineID}
	}
}

func (m Model) cancelPipeline(pipelineID uint64) tea.Cmd {
	fp := m.selectedProject.FullPath
	return func() tea.Msg {
		err := gl.CancelPipeline(fp, pipelineID)
		if err != nil {
			return errMsg{err}
		}
		return pipelineCanceledMsg{}
	}
}

func (m Model) copyToClipboard(text, feedback string) tea.Cmd {
	return func() tea.Msg {
		_ = clipboard.WriteAll(text)
		return copyDoneMsg{feedback: feedback}
	}
}

func (m Model) runJob() tea.Cmd {
	fp := m.selectedProject.FullPath
	jobID := m.jobRunJobID
	isRetry := m.jobRunIsRetry
	variables := m.jobRunVariables
	return func() tea.Msg {
		err := gl.RunJob(fp, jobID, isRetry, variables)
		if err != nil {
			return jobRunMsg{success: false, error: err.Error()}
		}
		return jobRunMsg{success: true}
	}
}

func (m Model) checkProjectExists(proj gl.Project) tea.Cmd {
	return func() tea.Msg {
		// Extract project name from display name or full path
		projectName := proj.DisplayName
		if projectName == "" {
			parts := strings.Split(proj.FullPath, "/")
			projectName = parts[len(parts)-1]
		}

		localPath, err := gl.ProjectExistsLocally(projectName)
		if err != nil {
			// Project not found locally, request clone
			return projectCheckMsg{exists: false, localPath: ""}
		}

		// Project exists locally
		return projectCheckMsg{exists: true, localPath: localPath}
	}
}

func (m Model) cloneProject(targetDir string) tea.Cmd {
	fp := m.selectedProject.FullPath
	return func() tea.Msg {
		localPath, err := gl.CloneProject(fp, targetDir)
		if err != nil {
			return projectCloneMsg{localPath: "", success: false, error: err.Error()}
		}
		return projectCloneMsg{localPath: localPath, success: true, error: ""}
	}
}

// truncateLines truncates each line in the log to maxW visible characters,
// preserving ANSI codes by stripping them for width measurement only.
func truncateLines(content string, maxW int) string {
	if maxW <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		visible := 0
		inEsc := false
		var result strings.Builder
		j := 0
		runes := []rune(line)
		for j < len(runes) {
			// Detect ANSI escape sequence start
			if !inEsc && runes[j] == '\x1b' && j+1 < len(runes) && runes[j+1] == '[' {
				// Write the escape sequence without counting width
				result.WriteRune(runes[j])
				j++
				result.WriteRune(runes[j])
				j++
				inEsc = true
				continue
			}
			if inEsc {
				result.WriteRune(runes[j])
				// ANSI sequences end with a letter (a-z, A-Z)
				if (runes[j] >= 'A' && runes[j] <= 'Z') || (runes[j] >= 'a' && runes[j] <= 'z') {
					inEsc = false
				}
				j++
				continue
			}
			if visible >= maxW {
				break
			}
			result.WriteRune(runes[j])
			visible++
			j++
		}
		// Reset ANSI at end of truncated line to prevent color bleeding
		if inEsc || visible >= maxW {
			result.WriteString("\x1b[0m")
		}
		lines[i] = result.String()
	}
	return strings.Join(lines, "\n")
}

func scheduleRefresh() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (m Model) hasRunningPipeline() bool {
	for _, p := range m.pipelines {
		if gl.IsRunning(p.Status) {
			return true
		}
	}
	return false
}

func (m *Model) syncPipelineScroll() {
	visible := m.pipelineVisibleRows()
	if m.pipelineCursor < m.pipelineScroll {
		m.pipelineScroll = m.pipelineCursor
	} else if m.pipelineCursor >= m.pipelineScroll+visible {
		m.pipelineScroll = m.pipelineCursor + 1 - visible
	}
}

func (m *Model) syncJobScroll() {
	visible := m.jobVisibleRows()
	if m.jobCursor < m.jobScroll {
		m.jobScroll = m.jobCursor
	} else if m.jobCursor >= m.jobScroll+visible {
		m.jobScroll = m.jobCursor + 1 - visible
	}
}

func (m *Model) pipelineVisibleRows() int {
	// header(3) + borders(2) + statusbar(1) + title(1) = ~7
	v := m.height - 7
	if v < 3 {
		v = 3
	}
	return v
}

func (m *Model) jobVisibleRows() int {
	// summary(8) + header(2) + borders(2) + statusbar(1) + title(1) = ~14
	v := m.height - 14
	if v < 3 {
		v = 3
	}
	return v
}

// ── Exported accessors (used by ui package) ────────────────────────────────────

func (m *Model) Screen() Screen               { return m.screen }
func (m *Model) Loading() bool                { return m.loading }
func (m *Model) TraceLoading() bool           { return m.traceLoading }
func (m *Model) ErrText() string              { return m.errText }
func (m *Model) StatusMsg() string            { return m.statusMsg }
func (m *Model) Spinner() spinner.Model       { return m.spinner }
func (m *Model) SelectedProject() *gl.Project { return m.selectedProject }
func (m *Model) Width() int                   { return m.width }
func (m *Model) Height() int                  { return m.height }

// Welcome
func (m *Model) Projects() []gl.Project    { return m.projects }
func (m *Model) ProjectCursor() int        { return m.projectCursor }
func (m *Model) ShowProjectSelector() bool { return m.showProjectSelector }
func (m *Model) ProjectListOnly() bool     { return m.projectListOnly }

// Pipelines
func (m *Model) Pipelines() []gl.PipelineListItem { return m.pipelines }
func (m *Model) PipelineCursor() int              { return m.pipelineCursor }
func (m *Model) PipelineScroll() int              { return m.pipelineScroll }
func (m *Model) PipelineVisibleRows() int         { return m.pipelineVisibleRows() }

// Create Pipeline
func (m *Model) CreatePipelineBranch() string        { return m.createPipelineBranch }
func (m *Model) CreatePipelineVariables() string     { return m.createPipelineVariables }
func (m *Model) CreatePipelineError() string         { return m.createPipelineError }
func (m *Model) CreatePipelineInputField() int       { return m.createPipelineInputField }
func (m *Model) CreatePipelineDisplayBranch() string { return m.createPipelineDisplayBranch }
func (m *Model) CreateConfirming() bool              { return m.createConfirming }

// Pipeline overlays
func (m *Model) CopyChoiceOpen() bool      { return m.copyChoiceOpen }
func (m *Model) CopyFeedback() string      { return m.copyFeedback }
func (m *Model) HasRunningPipeline() bool  { return m.hasRunningPipeline() }

// Job Run
func (m *Model) JobRunJobID() uint64      { return m.jobRunJobID }
func (m *Model) JobRunVariables() string  { return m.jobRunVariables }
func (m *Model) JobRunError() string      { return m.jobRunError }
func (m *Model) JobRunIsRetry() bool      { return m.jobRunIsRetry }
func (m *Model) JobRunConfirming() bool   { return m.jobRunConfirming }

// Clone Prompt
func (m *Model) ProjectCloneRequested() bool  { return m.projectCloneRequested }
func (m *Model) ProjectCloneInProgress() bool { return m.projectCloneInProgress }
func (m *Model) ProjectCloneError() string    { return m.projectCloneError }

// Jobs
func (m *Model) Detail() *gl.PipelineDetail { return m.detail }
func (m *Model) JobCursor() int             { return m.jobCursor }
func (m *Model) JobScroll() int             { return m.jobScroll }
func (m *Model) JobVisibleRows() int        { return m.jobVisibleRows() }

// Job Log
func (m *Model) SelectedJob() *gl.Job           { return m.selectedJob }
func (m *Model) JobTrace() string               { return m.jobTrace }
func (m *Model) TraceViewport() *viewport.Model { return m.traceViewport }

// FmtPipelineID returns the formatted pipeline ID string.
func FmtPipelineID(id uint64) string {
	return fmt.Sprintf("#%d", id)
}
