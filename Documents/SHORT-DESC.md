# glab-pipe - GitLab CI/CD Pipeline Monitor

**glab-pipe** is a terminal-based interactive application for monitoring and managing GitLab CI/CD pipelines. It provides a real-time, visual interface to track pipeline status, view job details, and inspect logs without leaving the command line.

## What It Does

- **Monitor Pipelines**: View pipeline status, IDs, branches, and start times in a clean table format
- **Track Jobs**: Drill down into individual jobs within pipelines to see their status and logs
- **Real-time Updates**: Auto-refreshes every 2 seconds when pipelines or jobs are running
- **Create Pipelines**: Trigger new CI/CD pipelines directly from the terminal
- **Project Management**: Easily switch between multiple GitLab projects

## Who It's For

- **Developers**: Quickly check if your CI/CD pipeline passed without opening a web browser
- **DevOps Engineers**: Monitor multiple projects and troubleshoot failing jobs from the terminal
- **Teams**: Streamline pipeline monitoring with a consistent, keyboard-driven interface

## How It Works

Built with Go using the BubbleTea TUI framework, glab-pipe wraps the GitLab CLI (`glab`) to provide an enhanced visual experience. It connects to your GitLab instance (e.g., `gitlab.example.com`) and displays pipeline information in an intuitive, color-coded interface with Nerd Font icons.

## Quick Start

```bash
# Install and run
glab-pipe

# Monitor current directory's project
glab-pipe .

# Monitor a specific project
glab-pipe --source gitlab.example.com/group/project
```

## Key Features

- Color-coded status icons (running, success, failed, canceled)
- Keyboard navigation (no mouse required)
- Smart branch naming (e.g., `CUC-639` → `story/CUC-639`)
- Automatic project cloning
- Full-screen log viewer with search
- Cross-platform (Windows, Linux, macOS)
