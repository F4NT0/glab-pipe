package gitlab

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"gitlab-pipeline-tui/internal/config"
)

// ── Project re-export ─────────────────────────────────────────────────────────

// Project is an alias so callers can use gl.Project without importing config.
type Project = config.Project

// ProjectList returns the list of projects from the external config file.
func ProjectList() []Project {
	return config.ProjectList()
}

// ── Raw JSON types from `glab ci list -F json` ────────────────────────────────

// PipelineListItem represents a single row returned by glab ci list.
type PipelineListItem struct {
	ID        uint64  `json:"id"`
	IID       uint64  `json:"iid"`
	Status    string  `json:"status"`
	GitRef    string  `json:"ref"`
	Source    string  `json:"source"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
	WebURL    *string `json:"web_url"`
	User      *PipelineUser `json:"user"`
}

// ── Raw JSON types from `glab ci get -F json` ─────────────────────────────────

// PipelineDetail holds the full pipeline information including jobs.
type PipelineDetail struct {
	ID         uint64        `json:"id"`
	IID        uint64        `json:"iid"`
	Status     string        `json:"status"`
	GitRef     string        `json:"ref"`
	Source     string        `json:"source"`
	SHA        *string       `json:"sha"`
	CreatedAt  *string       `json:"created_at"`
	StartedAt  *string       `json:"started_at"`
	FinishedAt *string       `json:"finished_at"`
	UpdatedAt  *string       `json:"updated_at"`
	Duration   *uint64       `json:"duration"`
	WebURL     *string       `json:"web_url"`
	User       *PipelineUser `json:"user"`
	Jobs       []Job         `json:"jobs"`
}

// PipelineUser holds the author information of a pipeline.
type PipelineUser struct {
	Name     *string `json:"name"`
	Username *string `json:"username"`
}

// Job represents a single CI job inside a pipeline.
type Job struct {
	ID         uint64   `json:"id"`
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	Stage      string   `json:"stage"`
	Duration   *float64 `json:"duration"`
	StartedAt  *string  `json:"started_at"`
	FinishedAt *string  `json:"finished_at"`
	AllowFail  *bool    `json:"allow_failure"`
	WebURL     *string  `json:"web_url"`
}

// ── glab wrapper ──────────────────────────────────────────────────────────────

func glab(args ...string) (string, error) {
	cmd := exec.Command("glab", args...)
	cmd.Env = append(cmd.Environ(), "GITLAB_HOST=gitlab.example.com")

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("glab exited with error:\n%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("failed to spawn glab – make sure it is installed and in PATH: %w", err)
	}
	return string(out), nil
}

// FetchPipelineList fetches the last 10 pipelines for a project.
func FetchPipelineList(fullPath string) ([]PipelineListItem, error) {
	raw, err := glab("ci", "list", "-R", fullPath, "-F", "json", "-P", "10")
	if err != nil {
		return nil, err
	}
	var items []PipelineListItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline list JSON: %w", err)
	}
	return items, nil
}

// FetchPipelineDetail fetches full information (including jobs) for a pipeline.
func FetchPipelineDetail(fullPath string, pipelineID uint64) (*PipelineDetail, error) {
	raw, err := glab(
		"ci", "get",
		"-R", fullPath,
		"-p", fmt.Sprintf("%d", pipelineID),
		"-F", "json",
		"--with-job-details",
	)
	if err != nil {
		return nil, err
	}
	var detail PipelineDetail
	if err := json.Unmarshal([]byte(raw), &detail); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline detail JSON: %w", err)
	}
	return &detail, nil
}

// FetchJobTrace fetches the log output for a specific job using its numeric ID.
func FetchJobTrace(fullPath string, jobID uint64) (string, error) {
	raw, err := glab(
		"ci", "trace", fmt.Sprintf("%d", jobID),
		"-R", fullPath,
	)
	if err != nil {
		return "", err
	}
	return raw, nil
}

// FindProjectByPath resolves the GitLab project path from a local git repo.
// It calls `glab repo view` in the given directory to get the project path.
func FindProjectByPath(localPath string) (string, error) {
	cmd := exec.Command("glab", "repo", "view", "--output", "json")
	cmd.Dir = localPath
	cmd.Env = append(cmd.Environ(), "GITLAB_HOST=gitlab.example.com")

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("glab repo view error:\n%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("failed to run glab repo view: %w", err)
	}

	// Parse JSON to extract path_with_namespace
	var result struct {
		PathWithNamespace string `json:"path_with_namespace"`
		FullPath          string `json:"full_path"`
		NameWithNamespace string `json:"name_with_namespace"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		// Try to extract from plain text output
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if strings.Contains(line, "/") && !strings.HasPrefix(line, "http") {
				return strings.TrimSpace(line), nil
			}
		}
		return "", fmt.Errorf("could not parse project path from glab output")
	}

	if result.PathWithNamespace != "" {
		return result.PathWithNamespace, nil
	}
	if result.FullPath != "" {
		return result.FullPath, nil
	}
	return "", fmt.Errorf("project path not found in glab response")
}

// ValidateProject checks if the user has access to a GitLab project by its full path.
// Accepts either a path (group/subgroup/project) or a full URL (https://gitlab.example.com/group/subgroup/project).
// Returns the last path component as display name on success, or an error.
func ValidateProject(input string) (displayName string, err error) {
	path := input
	// Normalize URL to path
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		u := strings.TrimPrefix(input, "http://")
		u = strings.TrimPrefix(u, "https://")
		parts := strings.SplitN(u, "/", 2)
		if len(parts) < 2 {
			return "", fmt.Errorf("Invalid URL format. Expected: https://gitlab.example.com/group/subgroup/project")
		}
		path = parts[1]
	}
	path = strings.TrimRight(path, "/")
	if path == "" {
		return "", fmt.Errorf("Path cannot be empty")
	}

	_, err = glab("ci", "list", "-R", path, "-F", "json", "-P", "1")
	if err != nil {
		return "", fmt.Errorf("You do not have access to this project or it does not exist.\nEnsure the path is correct and you have the necessary GitLab permissions.")
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1], nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// StageGroup groups jobs under an ordered stage name.
type StageGroup struct {
	Name string
	Jobs []*Job
}

// JobsByStage groups jobs by their stage, preserving insertion order.
func JobsByStage(jobs []Job) []StageGroup {
	seen := make(map[string]int) // stage name → index in result
	var result []StageGroup

	for i := range jobs {
		j := &jobs[i]
		if idx, ok := seen[j.Stage]; ok {
			result[idx].Jobs = append(result[idx].Jobs, j)
		} else {
			seen[j.Stage] = len(result)
			result = append(result, StageGroup{Name: j.Stage, Jobs: []*Job{j}})
		}
	}
	return result
}

// StatusIcon returns the Nerd Font icon for a given pipeline/job status.
// Icons match the specification from Usage.md:
//
//	\uf192 running  (blue)
//	\uf05d success  (green)
//	\uf52f failed   (red)
//	\ueabd canceled (gray)
//	\uf2be manual   (yellow) — pipeline-level manual trigger
//	\uf01d waiting  (yellow) — job waiting for manual play
func StatusIcon(status string) string {
	switch status {
	case "success":
		return "\uf05d" // nf-fa-check_circle
	case "failed":
		return "\uf52f" // nf-fa-times_circle (cancel/fail)
	case "running":
		return "\uf192" // nf-fa-dot_circle_o
	case "pending", "created":
		return "\uf192" // treat pending like running (spinning)
	case "canceled", "cancelled", "skipped":
		return "\ueabd" // nf-cod-circle_slash
	case "manual":
		return "\uf2be" // nf-fa-user_circle_o
	case "waiting_for_resource":
		return "\uf01d" // nf-fa-play_circle_o
	default:
		return "?"
	}
}

// IsRunning reports whether a pipeline or job status is considered "active"
// (i.e., the TUI should keep auto-refreshing).
func IsRunning(status string) bool {
	switch status {
	case "running", "pending", "created", "waiting_for_resource":
		return true
	}
	return false
}

// HasAnyRunning reports whether any job in the slice is in an active state.
func HasAnyRunning(jobs []Job) bool {
	for _, j := range jobs {
		if IsRunning(j.Status) {
			return true
		}
	}
	return false
}

// FmtDuration formats seconds into a human-readable duration string.
func FmtDuration(secs float64) string {
	s := uint64(secs)
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm%ds", s/60, s%60)
}

// DerefStr safely dereferences a *string, returning fallback if nil.
func DerefStr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}
