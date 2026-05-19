package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"gitlab-pipeline-tui/internal/app"
	gl "gitlab-pipeline-tui/internal/gitlab"
)

// ── Colour palette ────────────────────────────────────────────────────────────

var (
	colorBg         = lipgloss.Color("#0d1117")
	colorAccentBlue = lipgloss.Color("#58a6ff")
	colorFg         = lipgloss.Color("#c9d1d9")
	colorFgMuted    = lipgloss.Color("#6e7681")
	colorOrange     = lipgloss.Color("#e8912d")

	cyan   = lipgloss.Color("14")
	white  = lipgloss.Color("15")
	yellow = lipgloss.Color("11")
	red    = lipgloss.Color("9")
	green  = lipgloss.Color("10")
	blue   = lipgloss.Color("12")
	gray   = lipgloss.Color("8")
	teal   = lipgloss.Color("6")
)

func statusColor(status string) lipgloss.Color {
	switch status {
	case "success":
		return green
	case "failed":
		return red
	case "running", "pending":
		return blue
	case "created":
		return lipgloss.Color("240")
	case "canceled", "cancelled", "skipped":
		return gray
	case "manual":
		return yellow
	case "waiting_for_resource":
		return yellow
	default:
		return white
	}
}

// ── Shared styles ─────────────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(lipgloss.Color("236")).
			Padding(0, 2)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(gray).
			Padding(0, 1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(teal).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(gray)

	errorBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(red).
				Padding(1, 2)

	loadingStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(yellow).
			Padding(0, 2).
			Foreground(yellow).
			Bold(true)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cyan)

	summaryKeyStyle = lipgloss.NewStyle().Foreground(gray)
	summaryValStyle = lipgloss.NewStyle().Foreground(white)
)

// ── Main View ─────────────────────────────────────────────────────────────────

func View(m *app.Model) string {
	w := m.Width()
	if w == 0 {
		w = 80
	}

	if m.ErrText() != "" {
		return renderError(m.ErrText(), w, m.Height())
	}

	var body string
	switch m.Screen() {
	case app.ScreenWelcome:
		body = renderWelcomeScreen(m, w)
	case app.ScreenPipelines:
		body = renderPipelineTable(m, w)
	case app.ScreenJobs:
		body = renderJobsScreen(m, w)
	case app.ScreenJobLog:
		body = renderJobLogModal(m, w)
	case app.ScreenCreatePipeline:
		body = renderCreatePipelineModal(m, w)
	case app.ScreenClonePrompt:
		body = renderClonePromptModal(m, w)
	}

	if m.Loading() || m.TraceLoading() || m.ProjectCloneInProgress() {
		overlay := loadingStyle.Render(" " + m.Spinner().View() + " Loading… ")
		body = placeOverlay(body, overlay, w, m.Height())
	}

	title := renderTitle(m, w)
	statusBar := renderStatusBar(m, w)

	return lipgloss.JoinVertical(lipgloss.Left, title, body, statusBar)
}

// ── Title bar ─────────────────────────────────────────────────────────────────

func renderTitle(m *app.Model, w int) string {
	var breadcrumb string
	switch m.Screen() {
	case app.ScreenWelcome:
		breadcrumb = " glab-pipe  GitLab Pipeline Viewer "
	case app.ScreenPipelines:
		name := ""
		if p := m.SelectedProject(); p != nil {
			name = p.DisplayName
		}
		breadcrumb = fmt.Sprintf(" glab-pipe  ›  %s  ›  Pipelines ", name)
	case app.ScreenJobs:
		name := ""
		if p := m.SelectedProject(); p != nil {
			name = p.DisplayName
		}
		pip := ""
		if d := m.Detail(); d != nil {
			pip = fmt.Sprintf("  ›  Pipeline #%d (%s)", d.ID, d.GitRef)
		}
		breadcrumb = fmt.Sprintf(" glab-pipe  ›  %s%s  ›  Jobs ", name, pip)
	case app.ScreenJobLog:
		name := ""
		if p := m.SelectedProject(); p != nil {
			name = p.DisplayName
		}
		pip := ""
		branch := ""
		if d := m.Detail(); d != nil {
			pip = fmt.Sprintf("#%d", d.ID)
			branch = d.GitRef
		}
		jobName := ""
		if j := m.SelectedJob(); j != nil {
			jobName = j.Name
		}
		breadcrumb = fmt.Sprintf(" glab-pipe  ›  %s  ›  Pipeline %s (%s)  ›  %s ", name, pip, branch, jobName)
	case app.ScreenCreatePipeline:
		name := ""
		if p := m.SelectedProject(); p != nil {
			name = p.DisplayName
		}
		breadcrumb = fmt.Sprintf(" glab-pipe  ›  %s  ›  Create Pipeline ", name)
	case app.ScreenClonePrompt:
		name := ""
		if p := m.SelectedProject(); p != nil {
			name = p.DisplayName
		}
		breadcrumb = fmt.Sprintf(" glab-pipe  ›  %s  ›  Clone Project ", name)
	}

	return titleStyle.Width(w).Render(breadcrumb)
}

// ── Status bar ────────────────────────────────────────────────────────────────

func renderStatusBar(m *app.Model, w int) string {
	var hints string
	switch m.Screen() {
	case app.ScreenWelcome:
		if m.ShowProjectSelector() {
			hints = helpKey("↑/↓") + helpDesc(" Navigate  ") +
				helpKey("Enter") + helpDesc(" Select project  ") +
				helpKey("Esc") + helpDesc(" Quit")
		} else {
			hints = helpKey("Enter") + helpDesc(" Open project selector  ") +
				helpKey("q") + helpDesc(" Quit")
		}
	case app.ScreenPipelines:
		hints = helpKey("↑/↓") + helpDesc(" Navigate  ") +
			helpKey("Enter") + helpDesc(" View jobs  ") +
			helpKey("n") + helpDesc(" New pipeline  ") +
			helpKey("r") + helpDesc(" Refresh  ") +
			helpKey("Esc") + helpDesc(" Back  ") +
			helpKey("q") + helpDesc(" Quit")
	case app.ScreenJobs:
		hints = helpKey("↑/↓") + helpDesc(" Navigate  ") +
			helpKey("Enter") + helpDesc(" View logs  ") +
			helpKey("r") + helpDesc(" Refresh  ") +
			helpKey("Esc") + helpDesc(" Back  ") +
			helpKey("q") + helpDesc(" Quit")
	case app.ScreenJobLog:
		hints = helpKey("↑/↓") + helpDesc(" Scroll  ") +
			helpKey("PgUp/PgDn") + helpDesc(" Half page  ") +
			helpKey("g/G") + helpDesc(" Top/Bottom  ") +
			helpKey("Esc") + helpDesc(" Back  ") +
			helpKey("q") + helpDesc(" Quit")
	case app.ScreenCreatePipeline:
		hints = helpKey("Enter") + helpDesc(" Create  ") +
			helpKey("Tab") + helpDesc(" Next field  ") +
			helpKey("Esc") + helpDesc(" Cancel  ") +
			helpKey("q") + helpDesc(" Quit")
	case app.ScreenClonePrompt:
		hints = helpKey("Y") + helpDesc(" Clone  ") +
			helpKey("N") + helpDesc(" Cancel  ") +
			helpKey("q") + helpDesc(" Quit")
	}
	return statusBarStyle.Width(w).Render(hints)
}

func helpKey(s string) string  { return helpKeyStyle.Render(s) }
func helpDesc(s string) string { return helpDescStyle.Render(s) }

// ── Welcome screen ─────────────────────────────────────────────────────────────

func renderWelcomeScreen(m *app.Model, w int) string {
	// If returning from pipelines via Esc, show only project list
	if m.ProjectListOnly() {
		return renderProjectListOnly(m, w)
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

	boxContent := lipgloss.JoinVertical(lipgloss.Center,
		asciiStyle.Render(ascii),
		"",
		subtitle,
		tagline,
		"",
		sep,
		"",
		muted.Render("↑↓: navigate   enter: confirm   ctrl+c: quit"),
	)

	box := lipgloss.NewStyle().
		Background(colorBg).
		Width(w).
		Height(m.Height()).
		Align(lipgloss.Center, lipgloss.Center).
		Render(boxContent)

	if m.ShowProjectSelector() {
		modal := renderProjectSelectorModal(m, w)
		box = placeOverlay(box, modal, w, m.Height())
	}

	return box
}

func renderProjectListOnly(m *app.Model, w int) string {
	projects := m.Projects()
	cursor := m.ProjectCursor()
	orange := colorOrange

	var sb strings.Builder

	for i, p := range projects {
		selected := i == cursor
		var row string
		if selected {
			row = lipgloss.NewStyle().Foreground(orange).Bold(true).Render("> " + p.DisplayName)
		} else {
			row = lipgloss.NewStyle().Foreground(colorFg).Render("  " + p.DisplayName)
		}
		sb.WriteString(row + "\n")
	}

	if len(projects) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorFgMuted).Render("  No projects configured.\n  Run glab-pipe . in a git repo to add one."))
	}

	// Center the content vertically and horizontally
	content := sb.String()
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)
	topPad := (m.Height() - contentHeight) / 2
	if topPad < 0 {
		topPad = 0
	}
	leftPad := (w - 50) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	var result strings.Builder
	result.WriteString(strings.Repeat("\n", topPad))
	for _, line := range lines {
		result.WriteString(strings.Repeat(" ", leftPad) + line + "\n")
	}

	return lipgloss.NewStyle().
		Background(colorBg).
		Width(w).
		Height(m.Height()).
		Render(result.String())
}

func renderProjectSelectorModal(m *app.Model, w int) string {
	projects := m.Projects()
	cursor := m.ProjectCursor()

	modalWidth := 40
	if modalWidth > w-4 {
		modalWidth = w - 4
	}

	orange := colorOrange
	var sb strings.Builder

	for i, p := range projects {
		selected := i == cursor
		var row string
		if selected {
			row = lipgloss.NewStyle().Foreground(orange).Bold(true).Render("> " + p.DisplayName)
		} else {
			row = lipgloss.NewStyle().Foreground(colorFg).Render("  " + p.DisplayName)
		}
		sb.WriteString(row + "\n")
	}

	modalContent := sb.String()
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(orange).
		Width(modalWidth).
		Padding(0, 1).
		Render(modalContent)

	// Center the modal
	leftPad := (w - lipgloss.Width(modal)) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := (m.Height() - lipgloss.Height(modal)) / 2
	if topPad < 0 {
		topPad = 0
	}

	leftPadStr := strings.Repeat(" ", leftPad)
	topPadStr := strings.Repeat("\n", topPad)

	lines := strings.Split(modal, "\n")
	for i := range lines {
		lines[i] = leftPadStr + lines[i]
	}
	return topPadStr + strings.Join(lines, "\n")
}

// ── Pipeline table ─────────────────────────────────────────────────────────────

func renderPipelineTable(m *app.Model, w int) string {
	pipelines := m.Pipelines()
	cursor := m.PipelineCursor()
	scroll := m.PipelineScroll()
	visible := m.PipelineVisibleRows()

	innerW := w - 4 // account for border + padding

	// Column widths: prefix(2) + statusW+2(icon) + sep(2) + idW + sep(2) + branchW
	statusW := 10 // displayed as statusW+2 for icon
	idW := 10
	// total fixed = 2 + (statusW+2) + 2 + idW + 2 = 28
	branchW := innerW - 28
	if branchW < 16 {
		branchW = 16
	}

	// Table header
	headerStyle := lipgloss.NewStyle().Foreground(gray).Bold(true)
	headerLine := fmt.Sprintf("  %-*s  %-*s  %-*s",
		statusW, "Status",
		idW, "ID",
		branchW, "Branch",
	)
	header := headerStyle.Render(headerLine)
	separator := lipgloss.NewStyle().Foreground(gray).Render("  " + strings.Repeat("─", innerW-2))

	var sb strings.Builder
	sb.WriteString(
		lipgloss.NewStyle().Foreground(white).Bold(true).Render("  Pipelines — "+m.SelectedProject().DisplayName) + "\n", //nolint:govet
	)
	sb.WriteString(header + "\n")
	sb.WriteString(separator + "\n")

	if len(pipelines) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(gray).Render("  No pipelines found."))
	}

	end := scroll + visible
	if end > len(pipelines) {
		end = len(pipelines)
	}

	for i := scroll; i < end; i++ {
		p := pipelines[i]
		selected := i == cursor

		col := statusColor(p.Status)
		icon := gl.StatusIcon(p.Status)
		statusStr := fmt.Sprintf("%s %-8s", icon, p.Status)
		idStr := fmt.Sprintf("#%d", p.ID)
		branch := p.GitRef
		if len(branch) > branchW {
			branch = branch[:branchW-1] + "…"
		}
		line := fmt.Sprintf("  %-*s  %-*s  %-*s",
			statusW+2, statusStr, // +2 for icon width
			idW, idStr,
			branchW, branch,
		)

		if selected {
			line = "> " + lipgloss.NewStyle().Foreground(col).Bold(true).Render(line)
		} else {
			line = "  " + lipgloss.NewStyle().Foreground(col).Render(line)
		}
		sb.WriteString(line + "\n")
	}

	// Scroll indicator
	if len(pipelines) > visible {
		pct := 0
		if len(pipelines)-visible > 0 {
			pct = scroll * 100 / (len(pipelines) - visible)
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(gray).Render(
			fmt.Sprintf("  … %d/%d  (%d%%)", cursor+1, len(pipelines), pct),
		))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(white).
		Width(w-2).Padding(0, 1).Render(sb.String())
}

func orange() lipgloss.Color { return colorOrange }

// ── Jobs screen ────────────────────────────────────────────────────────────────

func renderJobsScreen(m *app.Model, w int) string {
	d := m.Detail()
	if d == nil {
		return panelStyle.Width(w - 2).Render("  Loading jobs…")
	}

	summary := renderPipelineSummary(d, w)
	jobs := renderJobList(m, d, w)

	return lipgloss.JoinVertical(lipgloss.Left, summary, jobs)
}

func renderPipelineSummary(d *gl.PipelineDetail, w int) string {
	col := statusColor(d.Status)
	icon := gl.StatusIcon(d.Status)

	author := "unknown"
	if d.User != nil {
		if d.User.Name != nil {
			author = *d.User.Name
		} else if d.User.Username != nil {
			author = *d.User.Username
		}
	}

	source := d.Source
	if source == "" {
		source = "—"
	}

	lines := []string{
		summaryRow("  ID        ", lipgloss.NewStyle().Foreground(yellow).Render(fmt.Sprintf("#%d", d.ID))),
		summaryRow("  Status    ", lipgloss.NewStyle().Foreground(col).Bold(true).Render(fmt.Sprintf("%s %s", icon, strings.ToUpper(d.Status)))),
		summaryRow("  Source    ", summaryValStyle.Render(source)),
		summaryRow("  Branch    ", lipgloss.NewStyle().Foreground(cyan).Render(d.GitRef)),
		summaryRow("  User      ", summaryValStyle.Render(author)),
		summaryRow("  Created   ", summaryValStyle.Render(fmtTime(d.CreatedAt))),
		summaryRow("  Updated   ", summaryValStyle.Render(fmtTime(d.UpdatedAt))),
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(statusColor(d.Status)).
		Width(w-2).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(white).Bold(true).Render("  Pipeline Summary"),
			content,
		))
}

func summaryRow(key, val string) string {
	return summaryKeyStyle.Render(key+": ") + val
}

func renderJobList(m *app.Model, d *gl.PipelineDetail, w int) string {
	jobs := d.Jobs
	if len(jobs) == 0 {
		return panelStyle.Width(w - 2).Render("  No jobs found.")
	}

	cursor := m.JobCursor()
	scroll := m.JobScroll()
	visible := m.JobVisibleRows()

	innerW := w - 6

	// Column widths
	statusW := 14
	nameW := innerW - statusW

	headerStyle := lipgloss.NewStyle().Foreground(gray).Bold(true)
	header := headerStyle.Render(fmt.Sprintf("  %-*s  %-*s", statusW, "Status", nameW, "Job Name"))
	separator := lipgloss.NewStyle().Foreground(gray).Render("  " + strings.Repeat("─", innerW))

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(white).Bold(true).Render("  Jobs") + "\n")
	sb.WriteString(header + "\n")
	sb.WriteString(separator + "\n")

	end := scroll + visible
	if end > len(jobs) {
		end = len(jobs)
	}

	for i := scroll; i < end; i++ {
		j := jobs[i]
		selected := i == cursor

		col := statusColor(j.Status)
		icon := gl.StatusIcon(j.Status)
		statusStr := fmt.Sprintf("%s %-8s", icon, j.Status)
		nameStr := j.Name
		if len([]rune(nameStr)) > nameW {
			nameStr = string([]rune(nameStr)[:nameW-1]) + "…"
		}
		line := fmt.Sprintf("%-*s  %-*s",
			statusW+2, statusStr,
			nameW, nameStr,
		)

		if selected {
			line = "> " + lipgloss.NewStyle().Foreground(col).Render(line)
		} else {
			line = "  " + lipgloss.NewStyle().Foreground(col).Render(line)
		}
		sb.WriteString(line + "\n")
	}

	if len(jobs) > visible {
		pct := 0
		if len(jobs)-visible > 0 {
			pct = scroll * 100 / (len(jobs) - visible)
		}
		sb.WriteString(lipgloss.NewStyle().Foreground(gray).Render(
			fmt.Sprintf("  … %d/%d  (%d%%)", cursor+1, len(jobs), pct),
		))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(white).
		Width(w-2).Padding(0, 1).Render(sb.String())
}

// ── Job Log Modal ──────────────────────────────────────────────────────────────

func renderJobLogModal(m *app.Model, w int) string {
	job := m.SelectedJob()
	d := m.Detail()

	// Header box width
	headerW := w - 4
	if headerW < 40 {
		headerW = 40
	}
	innerW := headerW - 4

	// Build title
	titleStr := " Job Log "
	if job != nil && d != nil {
		col := statusColor(job.Status)
		icon := gl.StatusIcon(job.Status)
		statusRendered := lipgloss.NewStyle().Foreground(col).Bold(true).Render(fmt.Sprintf("%s %s", icon, strings.ToUpper(job.Status)))
		titleStr = fmt.Sprintf(" %s  %s  |  Pipeline #%d  |  %s ", statusRendered, job.Name, d.ID, d.GitRef)
	}

	// Line count
	totalLines := len(strings.Split(m.JobTrace(), "\n"))
	scrollInfo := lipgloss.NewStyle().Foreground(gray).Render(fmt.Sprintf(" %d lines", totalLines))

	logTitleStyle := lipgloss.NewStyle().Foreground(white).Bold(true)
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top,
		logTitleStyle.Render(titleStr),
		scrollInfo,
	)
	sep := lipgloss.NewStyle().Foreground(gray).Render(strings.Repeat("─", innerW))

	// Header in a compact bordered box
	headerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(white).
		Width(headerW).
		Padding(0, 1).
		Render(strings.Join([]string{headerRow, sep}, "\n"))

	// Log content rendered freely (viewport), no outer box
	vp := m.TraceViewport()
	traceContent := vp.View()
	if m.TraceLoading() && traceContent == "" {
		traceContent = m.Spinner().View() + " Loading logs…"
	}
	if traceContent == "" {
		traceContent = "  No log output available."
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerBox, traceContent)
}

// ── Create Pipeline Modal ───────────────────────────────────────────────────────

func renderCreatePipelineModal(m *app.Model, w int) string {
	modalW := 60
	if modalW > w-4 {
		modalW = w - 4
	}

	var sb strings.Builder

	// Title
	title := lipgloss.NewStyle().Foreground(white).Bold(true).Render("  Create New Pipeline  ")
	sb.WriteString(title + "\n")

	// Separator
	sep := lipgloss.NewStyle().Foreground(gray).Render(strings.Repeat("─", modalW-4))
	sb.WriteString(sep + "\n")

	// Branch input
	branchLabel := lipgloss.NewStyle().Foreground(cyan).Render("  Branch:  ")
	branchInput := m.CreatePipelineBranch()
	var branchDisplay string
	if m.CreatePipelineInputField() == 0 {
		branchInput += "█"
		branchDisplay = lipgloss.NewStyle().Foreground(white).Bold(true).Render(branchInput)
	} else {
		branchDisplay = lipgloss.NewStyle().Foreground(white).Render(branchInput)
	}
	sb.WriteString(branchLabel + branchDisplay + "\n")

	// Show normalized branch name if different from input
	displayBranch := m.CreatePipelineDisplayBranch()
	if displayBranch != "" && displayBranch != branchInput {
		normalizedText := lipgloss.NewStyle().Foreground(gray).Render("  → Will use: " + displayBranch)
		sb.WriteString(normalizedText + "\n")
	}

	// Variables input (optional)
	varLabel := lipgloss.NewStyle().Foreground(cyan).Render("  Variables (optional):  ")
	varInput := m.CreatePipelineVariables()
	if varInput == "" {
		varInput = "(press Enter to skip)"
	}
	var varDisplay string
	if m.CreatePipelineInputField() == 1 {
		if m.CreatePipelineVariables() == "" {
			varInput = "█"
		} else {
			varInput += "█"
		}
		varDisplay = lipgloss.NewStyle().Foreground(white).Bold(true).Render(varInput)
	} else {
		varDisplay = lipgloss.NewStyle().Foreground(gray).Render(varInput)
	}
	sb.WriteString(varLabel + varDisplay + "\n")

	// Error message
	if m.CreatePipelineError() != "" {
		errorMsg := lipgloss.NewStyle().Foreground(red).Render("  " + m.CreatePipelineError())
		sb.WriteString("\n" + errorMsg + "\n")
	}

	// Help text
	helpText := lipgloss.NewStyle().Foreground(gray).Render("  Variables format: key1:value1,key2:value2")
	sb.WriteString("\n" + helpText + "\n")
	branchHelp := lipgloss.NewStyle().Foreground(gray).Render("  Enter only the ticket code (e.g. CUC-690) → becomes story/CUC-690")
	sb.WriteString(branchHelp + "\n")

	modalContent := sb.String()

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyan).
		Width(modalW).
		Padding(1, 1).
		Render(modalContent)
}

// ── Clone Prompt Modal ───────────────────────────────────────────────────────────

func renderClonePromptModal(m *app.Model, w int) string {
	modalW := 60
	if modalW > w-4 {
		modalW = w - 4
	}

	var sb strings.Builder

	// Title
	title := lipgloss.NewStyle().Foreground(white).Bold(true).Render("  Project Not Found Locally  ")
	sb.WriteString(title + "\n")

	// Separator
	sep := lipgloss.NewStyle().Foreground(gray).Render(strings.Repeat("─", modalW-4))
	sb.WriteString(sep + "\n")

	// Message
	projectName := ""
	if m.SelectedProject() != nil {
		projectName = m.SelectedProject().DisplayName
	}
	message := fmt.Sprintf("  Project '%s' not found in local directories.\n\n  Clone it to ~/repos?", projectName)
	sb.WriteString(lipgloss.NewStyle().Foreground(colorFg).Render(message) + "\n")

	// Error message if any
	if m.ProjectCloneError() != "" {
		errorText := lipgloss.NewStyle().Foreground(red).Render("  Error: " + m.ProjectCloneError())
		sb.WriteString("\n" + errorText + "\n")
	}

	// Options
	sb.WriteString("\n")
	yesStyle := lipgloss.NewStyle().Foreground(green).Bold(true)
	noStyle := lipgloss.NewStyle().Foreground(red)
	sb.WriteString("  " + yesStyle.Render("Y") + "/Enter - Clone to ~/repos\n")
	sb.WriteString("  " + noStyle.Render("N") + "/Esc - Cancel\n")

	modalContent := sb.String()

	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyan).
		Width(modalW).
		Padding(1, 1).
		Render(modalContent)

	// Center the modal
	leftPad := (w - lipgloss.Width(modal)) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := (m.Height() - lipgloss.Height(modal)) / 2
	if topPad < 0 {
		topPad = 0
	}

	leftPadStr := strings.Repeat(" ", leftPad)
	topPadStr := strings.Repeat("\n", topPad)

	lines := strings.Split(modal, "\n")
	for i := range lines {
		lines[i] = leftPadStr + lines[i]
	}
	return topPadStr + strings.Join(lines, "\n")
}

// ── Error overlay ─────────────────────────────────────────────────────────────

func renderError(msg string, w, h int) string {
	box := errorBorderStyle.
		Width(w / 2).
		Render(
			lipgloss.NewStyle().Foreground(red).Bold(true).Render("  Error  (press any key to dismiss)\n\n") +
				lipgloss.NewStyle().Foreground(red).Render(msg),
		)

	bw := lipgloss.Width(box)
	bh := lipgloss.Height(box)
	padTop := (h - bh) / 2
	padLeft := (w - bw) / 2
	if padTop < 0 {
		padTop = 0
	}
	if padLeft < 0 {
		padLeft = 0
	}

	top := strings.Repeat("\n", padTop)
	left := strings.Repeat(" ", padLeft)

	var lines []string
	for _, l := range strings.Split(box, "\n") {
		lines = append(lines, left+l)
	}
	return top + strings.Join(lines, "\n")
}

// ── Overlay helper ─────────────────────────────────────────────────────────────

func placeOverlay(body, overlay string, w, h int) string {
	ow := lipgloss.Width(overlay)
	oh := lipgloss.Height(overlay)

	padTop := (h - oh) / 2
	padLeft := (w - ow) / 2
	if padTop < 0 {
		padTop = 0
	}
	if padLeft < 0 {
		padLeft = 0
	}

	bodyLines := strings.Split(body, "\n")
	overlayLines := strings.Split(overlay, "\n")

	result := make([]string, len(bodyLines))
	copy(result, bodyLines)

	for i, ol := range overlayLines {
		row := padTop + i
		if row >= len(result) {
			break
		}
		rl := []rune(result[row])
		olRunes := []rune(ol)
		for len(rl) < padLeft+len(olRunes) {
			rl = append(rl, ' ')
		}
		copy(rl[padLeft:], olRunes)
		result[row] = string(rl)
	}
	return strings.Join(result, "\n")
}

// ── Time helpers ──────────────────────────────────────────────────────────────

// fmtTimeShort formats as "05/18 14:23 UTC" (15 chars) for compact table columns.
func fmtTimeShort(s *string) string {
	if s == nil || *s == "" {
		return "—"
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05 UTC",
	} {
		if t, err := time.Parse(layout, *s); err == nil {
			return t.Format("01/02 15:04 UTC")
		}
	}
	raw := *s
	if len(raw) > 15 {
		raw = raw[:15]
	}
	return raw
}

func fmtTime(s *string) string {
	if s == nil || *s == "" {
		return "—"
	}
	// Try common ISO8601 formats (GitLab returns UTC)
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05 UTC",
	} {
		if t, err := time.Parse(layout, *s); err == nil {
			return t.Format("01/02/2006 15:04 UTC")
		}
	}
	// Fallback: return raw string trimmed
	raw := *s
	if len(raw) > 19 {
		raw = raw[:19]
	}
	return raw
}
