package gitlab

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// getGitLabHost returns the GitLab hostname from environment variable or from glab config.
// Users can set GITLAB_HOST environment variable to configure their GitLab instance.
// If not set, tries to get the hostname from glab configuration.
// Returns empty string if no hostname can be determined, allowing glab to use its default.
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
		// Parse the output to find the authenticated host
		// Priority: prefer hosts that have a "Logged in to" confirmation
		// glab auth status output format:
		//   gitlab.example.com
		//     ✓ Logged in to gitlab.example.com as User (config.yml)
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
							loggedInHost = host // Fallback to gitlab.com if nothing better found
						}
					}
				}
			}
		}
		if loggedInHost != "" {
			return loggedInHost
		}
	}

	return "" // Let glab use its default configuration
}

// ── Raw JSON types from `glab ci list -F json` ────────────────────────────────

// PipelineListItem represents a single row returned by glab ci list.
type PipelineListItem struct {
	ID        uint64        `json:"id"`
	IID       uint64        `json:"iid"`
	Status    string        `json:"status"`
	GitRef    string        `json:"ref"`
	Source    string        `json:"source"`
	CreatedAt *string       `json:"created_at"`
	UpdatedAt *string       `json:"updated_at"`
	WebURL    *string       `json:"web_url"`
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
	host := getGitLabHost()
	if host != "" {
		cmd.Env = append(cmd.Environ(), "GITLAB_HOST="+host)
	}

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("glab exited with error:\n%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("failed to spawn glab – make sure it is installed and PATH: %w", err)
	}
	return string(out), nil
}

// normalizeProjectPath ensures the project path includes the hostname.
// It accepts:
// - Full URL: https://gitlab.example.com/group/subgroup/project
// - Hostname + path: gitlab.example.com/group/subgroup/project
// - Path only: group/subgroup/project (adds configured hostname prefix)
// Returns the path in the format: hostname/group/subgroup/project
func normalizeProjectPath(input string) string {
	path := input

	// Remove URL protocol if present
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		path = strings.TrimPrefix(input, "http://")
		path = strings.TrimPrefix(path, "https://")
	}

	// Trim trailing slash
	path = strings.TrimRight(path, "/")

	// If path already contains hostname, return as is
	if strings.Contains(path, "/") && strings.Split(path, "/")[0] != "" {
		firstPart := strings.Split(path, "/")[0]
		// Check if first part looks like a hostname (contains dot)
		if strings.Contains(firstPart, ".") {
			return path
		}
	}

	// Otherwise, prepend the configured hostname
	host := getGitLabHost()
	if host != "" {
		return host + "/" + path
	}

	// If no hostname can be determined, return path as-is
	// This allows glab to use its default configuration
	return path
}

// FetchPipelineList fetches the last 10 pipelines for a project.
func FetchPipelineList(fullPath string) ([]PipelineListItem, error) {
	normalizedPath := normalizeProjectPath(fullPath)
	raw, err := glab("ci", "list", "-R", normalizedPath, "-F", "json", "-P", "10")
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
	normalizedPath := normalizeProjectPath(fullPath)
	raw, err := glab(
		"ci", "get",
		"-R", normalizedPath,
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
	normalizedPath := normalizeProjectPath(fullPath)
	raw, err := glab(
		"ci", "trace", fmt.Sprintf("%d", jobID),
		"-R", normalizedPath,
	)
	if err != nil {
		return "", err
	}
	return raw, nil
}

// normalizeBranchName adds the "story/" prefix only when the input is a bare CUC-XXX ticket code.
// Any branch that already contains "/" or starts with a known prefix (develop, release, hotfix, etc.)
// is returned as-is.
func normalizeBranchName(branch string) string {
	branch = strings.TrimSpace(branch)

	// If branch already contains a path separator, return as-is
	if strings.Contains(branch, "/") {
		return branch
	}

	// If it starts with a known long-lived branch prefix followed by "-", return as-is
	// e.g. develop-foo, release-1.2, hotfix-bug
	for _, pfx := range []string{"develop", "release", "hotfix", "main", "master"} {
		if strings.HasPrefix(strings.ToLower(branch), pfx) {
			return branch
		}
	}

	// Only add story/ for CUC-XXX pattern
	if matched, _ := regexp.MatchString(`^CUC-\d+`, branch); matched {
		return "story/" + branch
	}

	// All other inputs are returned as-is
	return branch
}

// CreatePipeline creates a new CI/CD pipeline on the specified branch.
// Returns the pipeline ID of the created pipeline.
func CreatePipeline(fullPath string, branch string, variables string) (uint64, error) {
	normalizedPath := normalizeProjectPath(fullPath)
	normalizedBranch := normalizeBranchName(branch)
	args := []string{"ci", "run", "-R", normalizedPath, "-b", normalizedBranch}

	if variables != "" {
		args = append(args, "--variables", variables)
	}

	raw, err := glab(args...)
	if err != nil {
		return 0, err
	}

	// Parse pipeline ID from text output
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for # followed by numbers
		if strings.Contains(line, "#") {
			parts := strings.Split(line, "#")
			for _, part := range parts {
				var id uint64
				_, err := fmt.Sscanf(part, "%d", &id)
				if err == nil && id > 0 {
					return id, nil
				}
			}
		}

		// Look for "pipeline" followed by numbers
		if strings.Contains(strings.ToLower(line), "pipeline") {
			words := strings.Fields(line)
			for _, word := range words {
				var id uint64
				_, err := fmt.Sscanf(word, "%d", &id)
				if err == nil && id > 0 {
					return id, nil
				}
			}
		}

		// Try to parse the entire line as a number
		var id uint64
		_, err := fmt.Sscanf(line, "%d", &id)
		if err == nil && id > 0 {
			return id, nil
		}
	}

	return 0, fmt.Errorf("could not parse pipeline ID from glab output: %s", raw)
}

// FindProjectByPath resolves the GitLab project path from a local git repo.
// It calls `glab repo view` in the given directory to get the project path.
func FindProjectByPath(localPath string) (string, error) {
	cmd := exec.Command("glab", "repo", "view", "--output", "json")
	cmd.Dir = localPath
	host := getGitLabHost()
	if host != "" {
		cmd.Env = append(cmd.Environ(), "GITLAB_HOST="+host)
	}

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
// Accepts either a path (group/subgroup/project), hostname+path (gitlab.example.com/group/subgroup/project),
// or a full URL (https://gitlab.example.com/group/subgroup/project).
// Returns the last path component as display name on success, or an error.
func ValidateProject(input string) (displayName string, err error) {
	if input == "" {
		return "", fmt.Errorf("Path cannot be empty")
	}

	// Normalize the path to include hostname
	normalizedPath := normalizeProjectPath(input)

	_, err = glab("ci", "list", "-R", normalizedPath, "-F", "json", "-P", "1")
	if err != nil {
		return "", fmt.Errorf("You do not have access to this project or it does not exist.\nEnsure the path is correct and you have the necessary GitLab permissions.")
	}

	// Extract display name from the last part of the path
	parts := strings.Split(normalizedPath, "/")
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

// ProjectExistsLocally checks if a GitLab project exists in the local filesystem.
// It searches in a configurable base directory (default: ~/repos or ~/source/repos).
// Returns the local path if found, or an error if not found.
func ProjectExistsLocally(projectName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	// Common repository directories to search
	searchPaths := []string{
		filepath.Join(home, "repos"),
		filepath.Join(home, "source", "repos"),
		filepath.Join(home, "source"),
		filepath.Join(home, "projects"),
		filepath.Join(home, "development"),
	}

	// Also search in current directory and parent directories
	cwd, err := os.Getwd()
	if err == nil {
		searchPaths = append(searchPaths, cwd)
		for i := 0; i < 3; i++ { // Check up to 3 parent levels
			cwd = filepath.Dir(cwd)
			searchPaths = append(searchPaths, cwd)
		}
	}

	// Search for the project directory
	for _, basePath := range searchPaths {
		projectPath := filepath.Join(basePath, projectName)
		if _, err := os.Stat(projectPath); err == nil {
			// Check if it's a git repository
			gitDir := filepath.Join(projectPath, ".git")
			if _, err := os.Stat(gitDir); err == nil {
				return projectPath, nil
			}
		}
	}

	return "", fmt.Errorf("project '%s' not found in local directories", projectName)
}

// CloneProject clones a GitLab project to a local directory using glab.
// targetDir is the directory where the project should be cloned.
// Returns the path to the cloned project or an error.
func CloneProject(fullPath string, targetDir string) (string, error) {
	normalizedPath := normalizeProjectPath(fullPath)

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	// Extract project name from full path
	parts := strings.Split(normalizedPath, "/")
	projectName := parts[len(parts)-1]
	clonePath := filepath.Join(targetDir, projectName)

	// Check if already exists
	if _, err := os.Stat(clonePath); err == nil {
		return clonePath, nil // Already exists
	}

	// Clone using glab
	cmd := exec.Command("glab", "repo", "clone", normalizedPath, clonePath)
	host := getGitLabHost()
	if host != "" {
		cmd.Env = append(cmd.Environ(), "GITLAB_HOST="+host)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to clone project: %w\nOutput: %s", err, string(output))
	}

	return clonePath, nil
}
