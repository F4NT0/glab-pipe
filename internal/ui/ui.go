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
	case app.ScreenJobRun:
		body = renderJobRunModal(m, w)
	}

	// Pipeline screen overlays (copy choice)
	if m.Screen() == app.ScreenPipelines {
		if m.CopyChoiceOpen() {
			body = placeOverlay(body, renderCopyChoiceOverlay(m, w), w, m.Height())
		}
	}
	// Create pipeline confirm overlay
	if m.Screen() == app.ScreenCreatePipeline && m.CreateConfirming() {
		body = placeOverlay(body, renderCreateConfirmOverlay(m, w), w, m.Height())
	}

	title := renderTitle(m, w)
	statusBar := renderStatusBar(m, w)

	return lipgloss.JoinVertical(lipgloss.Left, title, body, statusBar)
}

// ── Title bar ─────────────────────────────────────────────────────────────────

func renderTitle(m *app.Model, w int) string {
	host := gl.GetGitLabHost()
	if host == "" {
		host = "glab-pipe"
	}

	sep := " › "

	projectName := func() string {
		if p := m.SelectedProject(); p != nil {
			return p.DisplayName
		}
		return ""
	}

	var parts []string
	switch m.Screen() {
	case app.ScreenWelcome:
		parts = []string{host, "GitLab Pipeline Viewer"}
	case app.ScreenPipelines:
		parts = []string{host, projectName(), "Pipelines"}
	case app.ScreenJobs:
		pip := ""
		if d := m.Detail(); d != nil {
			pip = fmt.Sprintf("Pipeline #%d (%s)", d.ID, d.GitRef)
		}
		parts = []string{host, projectName(), pip, "Jobs"}
	case app.ScreenJobLog:
		pip := ""
		if d := m.Detail(); d != nil {
			pip = fmt.Sprintf("Pipeline #%d (%s)", d.ID, d.GitRef)
		}
		jobName := ""
		if j := m.SelectedJob(); j != nil {
			jobName = j.Name
		}
		parts = []string{host, projectName(), pip, jobName}
	case app.ScreenJobRun:
		action := "Retry Job"
		if !m.JobRunIsRetry() {
			action = "Run Job"
		}
		parts = []string{host, projectName(), action}
	case app.ScreenCreatePipeline:
		parts = []string{host, projectName(), "Create Pipeline"}
	case app.ScreenClonePrompt:
		parts = []string{host, projectName(), "Clone Project"}
	}

	// Filter empty parts
	var filtered []string
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	breadcrumb := " " + strings.Join(filtered, sep) + " "

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
		if m.CopyChoiceOpen() {
			hints = helpKey("1/b") + helpDesc(" Branch  ") +
				helpKey("2/w") + helpDesc(" URL  ") +
				helpKey("Esc") + helpDesc(" Cancel")
		} else if m.CopyFeedback() != "" {
			hints = lipgloss.NewStyle().Foreground(green).Render("  " + m.CopyFeedback())
		} else {
			hints = helpKey("↑/↓") + helpDesc(" Navigate  ") +
				helpKey("Enter") + helpDesc(" Jobs  ") +
				helpKey("n") + helpDesc(" New  ") +
				helpKey("r") + helpDesc(" Retry  ") +
				helpKey("u") + helpDesc(" Refresh  ") +
				helpKey("x") + helpDesc(" Cancel  ") +
				helpKey("c") + helpDesc(" Copy  ") +
				helpKey("Esc") + helpDesc(" Back  ") +
				helpKey("q") + helpDesc(" Quit")
		}
	case app.ScreenJobs:
		hints = helpKey("↑/↓") + helpDesc(" Navigate  ") +
			helpKey("Enter") + helpDesc(" View logs  ") +
			helpKey("r") + helpDesc(" Run/Retry job  ") +
			helpKey("u") + helpDesc(" Refresh  ") +
			helpKey("Esc") + helpDesc(" Back  ") +
			helpKey("q") + helpDesc(" Quit")
	case app.ScreenJobLog:
		hints = helpKey("↑/↓") + helpDesc(" Scroll  ") +
			helpKey("PgUp/PgDn") + helpDesc(" Half page  ") +
			helpKey("g/G") + helpDesc(" Top/Bottom  ") +
			helpKey("Esc") + helpDesc(" Back  ") +
			helpKey("q") + helpDesc(" Quit")
	case app.ScreenCreatePipeline:
		if m.CreateConfirming() {
			hints = helpKey("Enter") + helpDesc(" Confirm & Run  ") +
				helpKey("Esc") + helpDesc(" Edit")
		} else {
			hints = helpKey("Enter") + helpDesc(" Confirm  ") +
				helpKey("Tab") + helpDesc(" Next field  ") +
				helpKey("Esc") + helpDesc(" Cancel  ") +
				helpKey("q") + helpDesc(" Quit")
		}
	case app.ScreenJobRun:
		hints = helpKey("Enter") + helpDesc(" Run  ") +
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

	// Status badge — top-right of header
	var badge string
	if m.Loading() {
		badge = lipgloss.NewStyle().
			Foreground(yellow).Bold(true).
			Render(m.Spinner().View() + " Loading…")
	} else if m.HasRunningPipeline() {
		badge = lipgloss.NewStyle().
			Foreground(blue).Bold(true).
			Render("● running")
	}

	titleText := lipgloss.NewStyle().Foreground(white).Bold(true).Render("  Pipelines — " + m.SelectedProject().DisplayName)
	var titleLine string
	if badge != "" {
		badgeW := lipgloss.Width(badge)
		titleW := lipgloss.Width(titleText)
		gap := innerW - titleW - badgeW
		if gap < 1 {
			gap = 1
		}
		titleLine = titleText + strings.Repeat(" ", gap) + badge
	} else {
		titleLine = titleText
	}

	var sb strings.Builder
	sb.WriteString(titleLine + "\n")
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
		// Loading state — show badge in an empty panel
		var badge string
		if m.Loading() {
			badge = " " + m.Spinner().View() + " Loading jobs…"
		} else {
			badge = "  Loading jobs…"
		}
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(gray).
			Width(w-2).Padding(0, 1).
			Render(lipgloss.NewStyle().Foreground(yellow).Bold(true).Render(badge))
	}

	summary := renderPipelineSummary(m, d, w)

	// Narrow screen: only show summary (hide job list)
	const minWidthForJobList = 60
	if w < minWidthForJobList {
		return summary
	}

	jobs := renderJobList(m, d, w)
	return lipgloss.JoinVertical(lipgloss.Left, summary, jobs)
}

func renderPipelineSummary(m *app.Model, d *gl.PipelineDetail, w int) string {
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

	rows := []string{
		summaryRow("  ID        ", lipgloss.NewStyle().Foreground(yellow).Render(fmt.Sprintf("#%d", d.ID))),
		summaryRow("  Status    ", lipgloss.NewStyle().Foreground(col).Bold(true).Render(fmt.Sprintf("%s %s", icon, strings.ToUpper(d.Status)))),
		summaryRow("  Source    ", summaryValStyle.Render(source)),
		summaryRow("  Branch    ", lipgloss.NewStyle().Foreground(cyan).Render(d.GitRef)),
		summaryRow("  User      ", summaryValStyle.Render(author)),
		summaryRow("  Created   ", summaryValStyle.Render(fmtTime(d.CreatedAt))),
		summaryRow("  Updated   ", summaryValStyle.Render(fmtTime(d.UpdatedAt))),
	}

	content := strings.Join(rows, "\n")

	// Loading badge top-right
	innerW := w - 6
	var badge string
	if m.Loading() {
		badge = lipgloss.NewStyle().Foreground(yellow).Bold(true).Render(m.Spinner().View() + " Loading…")
	}
	titleText := lipgloss.NewStyle().Foreground(white).Bold(true).Render("  Pipeline Summary")
	var titleLine string
	if badge != "" {
		gap := innerW - lipgloss.Width(titleText) - lipgloss.Width(badge)
		if gap < 1 {
			gap = 1
		}
		titleLine = titleText + strings.Repeat(" ", gap) + badge
	} else {
		titleLine = titleText
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(statusColor(d.Status)).
		Width(w-2).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleLine, content))
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

// ── Pipeline screen overlays ─────────────────────────────────────────────────────

func renderCopyChoiceOverlay(m *app.Model, w int) string {
	modalW := 36
	if modalW > w-4 {
		modalW = w - 4
	}

	var pip *gl.PipelineListItem
	if pips := m.Pipelines(); len(pips) > 0 {
		p := pips[m.PipelineCursor()]
		pip = &p
	}

	branch := "(none)"
	if pip != nil {
		branch = pip.GitRef
		if len(branch) > modalW-18 {
			branch = branch[:modalW-21] + "…"
		}
	}

	sep := lipgloss.NewStyle().Foreground(gray).Render(strings.Repeat("─", modalW-4))
	title := lipgloss.NewStyle().Foreground(white).Bold(true).Render("  Copy Options")

	opt1 := lipgloss.NewStyle().Foreground(teal).Bold(true).Render("  1 / b") +
		lipgloss.NewStyle().Foreground(white).Render("  Branch name: ") +
		lipgloss.NewStyle().Foreground(yellow).Render(branch)
	opt2 := lipgloss.NewStyle().Foreground(teal).Bold(true).Render("  2 / w") +
		lipgloss.NewStyle().Foreground(white).Render("  Pipeline URL")
	esc := lipgloss.NewStyle().Foreground(gray).Render("  Esc  Cancel")

	content := strings.Join([]string{title, sep, opt1, opt2, "", esc}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyan).
		Width(modalW).
		Padding(0, 1).
		Render(content)
}

func renderCreateConfirmOverlay(m *app.Model, w int) string {
	modalW := 56
	if modalW > w-4 {
		modalW = w - 4
	}

	displayBranch := m.CreatePipelineDisplayBranch()
	if displayBranch == "" {
		displayBranch = m.CreatePipelineBranch()
	}

	sep := lipgloss.NewStyle().Foreground(gray).Render(strings.Repeat("─", modalW-4))
	title := lipgloss.NewStyle().Foreground(white).Bold(true).Render("  Confirm Pipeline")

	branchLine := lipgloss.NewStyle().Foreground(gray).Render("  Branch: ") +
		lipgloss.NewStyle().Foreground(yellow).Bold(true).Render(displayBranch)

	var lines []string
	lines = append(lines, title, sep, branchLine)

	// Parse and list variables
	vars := m.CreatePipelineVariables()
	if vars != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(gray).Render("  Variables:"))
		for _, v := range strings.Split(vars, ",") {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			parts := strings.SplitN(v, ":", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				lines = append(lines, lipgloss.NewStyle().Foreground(white).Render(
					fmt.Sprintf("    %s = %s", k, val),
				))
			} else {
				lines = append(lines, lipgloss.NewStyle().Foreground(white).Render("    "+v))
			}
		}
	} else {
		lines = append(lines, lipgloss.NewStyle().Foreground(gray).Render("  Variables: (none)"))
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(gray).Render("  Enter to run  ·  Esc to edit"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(green).
		Width(modalW).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

// renderTitledBox draws a rounded box whose title is embedded in the top border.
func renderTitledBox(title, content string, borderColor lipgloss.Color, totalW int) string {
	innerW := totalW - 2
	titleStr := " " + title + " "
	titleLen := len([]rune(titleStr))
	rightDashes := innerW - 1 - titleLen
	if rightDashes < 0 {
		rightDashes = 0
	}

	bc := lipgloss.NewStyle().Foreground(borderColor)

	top := bc.Render("╭─" + titleStr + strings.Repeat("─", rightDashes) + "╮")
	bot := bc.Render("╰" + strings.Repeat("─", innerW) + "╯")
	padLine := bc.Render("│") + strings.Repeat(" ", innerW) + bc.Render("│")

	var rows []string
	rows = append(rows, top, padLine)
	for _, line := range strings.Split(content, "\n") {
		lw := lipgloss.Width(line)
		pad := innerW - 2 - lw
		if pad < 0 {
			pad = 0
		}
		rows = append(rows, bc.Render("│")+" "+line+strings.Repeat(" ", pad)+" "+bc.Render("│"))
	}
	rows = append(rows, padLine, bot)
	return strings.Join(rows, "\n")
}

// ── Create Pipeline Modal ───────────────────────────────────────────────────────

func renderCreatePipelineModal(m *app.Model, w int) string {
	modalW := 62
	if modalW > w-4 {
		modalW = w - 4
	}

	var lines []string

	// Branch input
	branchRaw := m.CreatePipelineBranch()
	var branchLine string
	if m.CreatePipelineInputField() == 0 {
		branchLine = lipgloss.NewStyle().Foreground(cyan).Render("Branch: ") +
			lipgloss.NewStyle().Foreground(white).Bold(true).Render(branchRaw+"█")
	} else {
		branchLine = lipgloss.NewStyle().Foreground(cyan).Render("Branch: ") +
			lipgloss.NewStyle().Foreground(white).Render(branchRaw)
	}
	lines = append(lines, branchLine)

	// Normalized branch preview
	displayBranch := m.CreatePipelineDisplayBranch()
	if displayBranch != "" && displayBranch != branchRaw {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(gray).Render("→ Will use: ")+
				lipgloss.NewStyle().Foreground(yellow).Render(displayBranch))
	}

	lines = append(lines, "")

	// Variables input
	varRaw := m.CreatePipelineVariables()
	var varLine string
	if m.CreatePipelineInputField() == 1 {
		varLine = lipgloss.NewStyle().Foreground(cyan).Render("Variables: ") +
			lipgloss.NewStyle().Foreground(white).Bold(true).Render(varRaw+"█")
	} else if varRaw == "" {
		varLine = lipgloss.NewStyle().Foreground(cyan).Render("Variables: ") +
			lipgloss.NewStyle().Foreground(gray).Render("(optional — Tab to focus)")
	} else {
		varLine = lipgloss.NewStyle().Foreground(cyan).Render("Variables: ") +
			lipgloss.NewStyle().Foreground(white).Render(varRaw)
	}
	lines = append(lines, varLine)

	// Error
	if m.CreatePipelineError() != "" {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(red).Render(m.CreatePipelineError()))
	}

	lines = append(lines, "")
	lines = append(lines,
		lipgloss.NewStyle().Foreground(gray).Render("CUC-690 → story/CUC-690   FY27_0503 → release/FY27_0503"),
		lipgloss.NewStyle().Foreground(gray).Render("Variables: key1:value1,key2:value2"),
	)

	return renderTitledBox("New Pipeline", strings.Join(lines, "\n"), white, modalW)
}

// ── Job Run Modal ───────────────────────────────────────────────────────────────

func renderJobRunModal(m *app.Model, w int) string {
	titleText := "Run Job"
	if m.JobRunIsRetry() {
		titleText = "Retry Job"
	}

	modalW := 60
	if modalW > w-4 {
		modalW = w - 4
	}

	var lines []string

	// Job ID
	lines = append(lines,
		lipgloss.NewStyle().Foreground(gray).Render(fmt.Sprintf("Job ID: %d", m.JobRunJobID())),
	)
	lines = append(lines, "")

	if m.Loading() {
		// Loading state — spinner inline
		lines = append(lines,
			lipgloss.NewStyle().Foreground(yellow).Bold(true).Render(m.Spinner().View()+" Running job…"),
		)
	} else if m.JobRunConfirming() {
		// Confirm step
		lines = append(lines,
			lipgloss.NewStyle().Foreground(white).Bold(true).Render("Confirm:"),
		)
		if m.JobRunVariables() != "" {
			lines = append(lines,
				lipgloss.NewStyle().Foreground(gray).Render("Variables: ")+
					lipgloss.NewStyle().Foreground(white).Render(m.JobRunVariables()),
			)
		} else {
			lines = append(lines,
				lipgloss.NewStyle().Foreground(gray).Render("Variables: (none)"),
			)
		}
		lines = append(lines, "")
		lines = append(lines,
			lipgloss.NewStyle().Foreground(gray).Render("Enter to run  ·  Esc to edit"),
		)
	} else {
		// Input step
		varRaw := m.JobRunVariables()
		lines = append(lines,
			lipgloss.NewStyle().Foreground(cyan).Render("Variables: ")+
				lipgloss.NewStyle().Foreground(white).Bold(true).Render(varRaw+"█"),
		)
		if m.JobRunError() != "" {
			lines = append(lines, "")
			lines = append(lines,
				lipgloss.NewStyle().Foreground(red).Render(m.JobRunError()),
			)
		}
		lines = append(lines, "")
		lines = append(lines,
			lipgloss.NewStyle().Foreground(gray).Render("key1:value1,key2:value2  (optional)"),
		)
	}

	return renderTitledBox(titleText, strings.Join(lines, "\n"), white, modalW)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
