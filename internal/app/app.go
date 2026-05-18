package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	gl "gitlab-pipeline-tui/internal/gitlab"
)

// ── Screens ───────────────────────────────────────────────────────────────────

type Screen int

const (
	ScreenWelcome   Screen = iota
	ScreenPipelines        // pipeline table
	ScreenJobs             // job list for a selected pipeline
	ScreenJobLog           // modal with job trace logs
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
type tickMsg time.Time
type errMsg struct{ err error }

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	screen Screen

	// Project selection (welcome)
	projects            []gl.Project
	projectCursor       int
	showProjectSelector bool

	// Pipeline table
	selectedProject *gl.Project
	pipelines       []gl.PipelineListItem
	pipelineCursor  int
	pipelineScroll  int

	// Job list (detail screen)
	detail      *gl.PipelineDetail
	jobCursor   int
	jobScroll   int

	// Job log modal
	selectedJob   *gl.Job
	jobTrace      string
	traceLoading  bool
	traceViewport *viewport.Model

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
		m.traceViewport.SetContent(msg.trace)
		m.traceViewport.GotoBottom()
		// If job is still running, keep refreshing
		var nextTick tea.Cmd
		if m.selectedJob != nil && gl.IsRunning(m.selectedJob.Status) {
			nextTick = scheduleRefresh()
		}
		return m, nextTick

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
		if m.loading {
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
		m.screen = ScreenPipelines
		m.loading = true
		m.showProjectSelector = false
		m.pipelineCursor = 0
		m.pipelineScroll = 0
		return m, tea.Batch(m.spinner.Tick, m.loadPipelines())
	case "esc", "q":
		m.showProjectSelector = false
		return m, tea.Quit
	}
	return m, nil
}

// ── Screen: Pipelines ─────────────────────────────────────────────────────────

func (m Model) handlePipelineKey(key string) (Model, tea.Cmd) {
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
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadPipelines())
	case "esc", "backspace":
		// Back to welcome / project selector
		m.screen = ScreenWelcome
		m.showProjectSelector = true
		m.selectedProject = nil
		m.pipelines = nil
		m.pipelineCursor = 0
	case "q":
		return m, tea.Quit
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
	vp := viewport.New(m.width-4, m.height-8)
	vp.SetContent("")
	m.traceViewport = &vp
	return m, tea.Batch(m.spinner.Tick, m.loadJobTrace(job.ID))
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
		// Resume job-list refresh if still running
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

func (m *Model) Screen() Screen              { return m.screen }
func (m *Model) Loading() bool               { return m.loading }
func (m *Model) TraceLoading() bool          { return m.traceLoading }
func (m *Model) ErrText() string             { return m.errText }
func (m *Model) StatusMsg() string           { return m.statusMsg }
func (m *Model) Spinner() spinner.Model      { return m.spinner }
func (m *Model) SelectedProject() *gl.Project { return m.selectedProject }
func (m *Model) Width() int                  { return m.width }
func (m *Model) Height() int                 { return m.height }

// Welcome
func (m *Model) Projects() []gl.Project    { return m.projects }
func (m *Model) ProjectCursor() int        { return m.projectCursor }
func (m *Model) ShowProjectSelector() bool { return m.showProjectSelector }

// Pipelines
func (m *Model) Pipelines() []gl.PipelineListItem { return m.pipelines }
func (m *Model) PipelineCursor() int              { return m.pipelineCursor }
func (m *Model) PipelineScroll() int              { return m.pipelineScroll }
func (m *Model) PipelineVisibleRows() int         { return m.pipelineVisibleRows() }

// Jobs
func (m *Model) Detail() *gl.PipelineDetail { return m.detail }
func (m *Model) JobCursor() int             { return m.jobCursor }
func (m *Model) JobScroll() int             { return m.jobScroll }
func (m *Model) JobVisibleRows() int        { return m.jobVisibleRows() }

// Job Log
func (m *Model) SelectedJob() *gl.Job      { return m.selectedJob }
func (m *Model) JobTrace() string          { return m.jobTrace }
func (m *Model) TraceViewport() *viewport.Model { return m.traceViewport }

// FmtPipelineID returns the formatted pipeline ID string.
func FmtPipelineID(id uint64) string {
	return fmt.Sprintf("#%d", id)
}
