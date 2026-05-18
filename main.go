package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"gitlab-pipeline-tui/internal/app"
	"gitlab-pipeline-tui/internal/config"
	gl "gitlab-pipeline-tui/internal/gitlab"
	"gitlab-pipeline-tui/internal/ui"
)

// tuiWrapper adapts app.Model to the tea.Model interface.
type tuiWrapper struct {
	m app.Model
}

func (w tuiWrapper) Init() tea.Cmd {
	return w.m.Init()
}

func (w tuiWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newM, cmd := w.m.Update(msg)
	return tuiWrapper{m: newM}, cmd
}

func (w tuiWrapper) View() string {
	return ui.View(&w.m)
}

func main() {
	selectedProject := parseArgs()

	if selectedProject == nil {
		// No project pre-selected: show splash/project selector first.
		sp := ui.NewSplashModel()
		p := tea.NewProgram(sp, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		sm, ok := finalModel.(ui.SplashModel)
		if !ok || sm.Quit() {
			return
		}
		selectedProject = sm.SelectedProject()
	}

	// Run main application.
	m := app.NewWithProject(selectedProject)
	p := tea.NewProgram(
		tuiWrapper{m: m},
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// parseArgs processes CLI arguments and returns a pre-selected project (or nil).
//
//	glab-pipe              → nil (show project selector)
//	glab-pipe .            → resolve current dir to GitLab project
//	glab-pipe --source <p> → use given GitLab path directly
func parseArgs() *gl.Project {
	args := os.Args[1:]
	if len(args) == 0 {
		return nil
	}

	// --source <gitlab-path>
	for i, arg := range args {
		if arg == "--source" && i+1 < len(args) {
			fullPath := args[i+1]
			return resolveByFullPath(fullPath)
		}
		if strings.HasPrefix(arg, "--source=") {
			fullPath := strings.TrimPrefix(arg, "--source=")
			return resolveByFullPath(fullPath)
		}
	}

	// glab-pipe .  or  glab-pipe <local-path>
	if args[0] == "." || (len(args[0]) > 0 && args[0][0] != '-') {
		localPath := args[0]
		if args[0] == "." {
			var err error
			localPath, err = os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: could not determine current directory: %v\n", err)
				os.Exit(1)
			}
		}

		// Make absolute
		absPath, err := filepath.Abs(localPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not resolve path: %v\n", err)
			os.Exit(1)
		}

		// First check if the path matches a known project in config
		if proj := findInConfig(absPath); proj != nil {
			return proj
		}

		// Otherwise resolve via glab
		fullPath, err := gl.FindProjectByPath(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: project not found on GitLab: %v\n", err)
			os.Exit(1)
		}
		return &gl.Project{
			DisplayName: filepath.Base(fullPath),
			FullPath:    fullPath,
		}
	}

	return nil
}

// resolveByFullPath looks up a GitLab full path, first in config then ad-hoc.
func resolveByFullPath(fullPath string) *gl.Project {
	// Check if it matches a configured project
	cfg, _ := config.Load()
	if cfg != nil {
		for _, p := range cfg.Projects {
			if strings.EqualFold(p.FullPath, fullPath) {
				return &p
			}
		}
	}

	// Use the path directly (may fail later if user has no access)
	parts := strings.Split(fullPath, "/")
	displayName := parts[len(parts)-1]
	return &gl.Project{
		DisplayName: displayName,
		FullPath:    fullPath,
	}
}

// findInConfig checks if a local directory path matches a project in config
// by comparing the directory name against known project display names.
func findInConfig(localPath string) *gl.Project {
	dirName := filepath.Base(localPath)
	cfg, _ := config.Load()
	if cfg == nil {
		return nil
	}
	for i, p := range cfg.Projects {
		// Match by display name or last segment of full path
		parts := strings.Split(p.FullPath, "/")
		repoName := parts[len(parts)-1]
		if strings.EqualFold(p.DisplayName, dirName) || strings.EqualFold(repoName, dirName) {
			return &cfg.Projects[i]
		}
	}
	return nil
}
