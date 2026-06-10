<table align="center"><tr><td align="center" width="9999">

<img src="docs/images/logo.png" alt="glab-pipe main interface" width="900">

**A terminal-native gitlab pipeline manager built with Go**

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00add8?style=flat-square&logo=go)](https://go.dev)
[![Platform](https://img.shields.io/badge/Platform-Windows%2011-0078d4?style=flat-square&logo=windows)](https://www.microsoft.com/windows)
[![Shell](https://img.shields.io/badge/Shell-PowerShell-5391fe?style=flat-square&logo=powershell)](https://learn.microsoft.com/powershell)

</td></tr></table>

---

## Preview

### Welcome Screen — Project Selector

<table align="center"><tr><td align="center" width="9999">

<img src="docs/images/welcome.png" alt="glab-pipe main interface" width="500">

</td></tr></table>

### Pipeline Table

<table align="center"><tr><td align="center" width="9999">

<img src="docs/images/pipelines.png" alt="glab-pipe main interface" width="500">

</td></tr></table>

### Job List

<table align="center"><tr><td align="center" width="9999">

<img src="docs/images/jobs.png" alt="glab-pipe main interface" width="500">

</td></tr></table>

### Job Log Modal

<table align="center"><tr><td align="center" width="9999">

<img src="docs/images/jobs-logs.png" alt="glab-pipe main interface" width="500">

</td></tr></table>

### Run pipeline modal

<table align="center"><tr><td align="center" width="9999">

<img src="docs/images/run-pipeline.png" alt="glab-pipe main interface" width="500">

</td></tr></table>

---

## Features

- **Welcome screen** with ASCII art and modal project selector
- **Pipeline table** with columns: Status, ID, Branch, Started At
- **Nerd Font icons** color-coded by status on all screens
- **Auto-refresh** every 2 seconds when pipelines or jobs are running
- **Job screen** with full job list per pipeline, plus a pipeline summary header
- **Log modal** (near full-screen) powered by `glab ci trace`, auto-updated while the job is running
- **Full Esc navigation** at every level
- **CLI arguments** for opening a project directly
- **External project config file** at `~/.glab-pipe/projects.json`
- **TUI installer** with dependency checks (glab, Nerd Font, PowerShell)

---

## Requirements

- **Go 1.22+** (for building only)
- **`glab` CLI** installed and authenticated with your GitLab instance
- **Nerd Font** in your terminal (for correct icons — e.g. JetBrainsMono Nerd Font)
- **PowerShell** (recommended for best color support)

### Install glab
```powershell
scoop install glab
# or
winget install glab

# Authenticate with your GitLab instance
glab auth login --hostname your-gitlab-instance.com
```

### Install a Nerd Font
Download and install a Nerd Font:
- [JetBrainsMono Nerd Font](https://github.com/ryanoasis/nerd-fonts/releases)
- [FiraCode Nerd Font](https://github.com/ryanoasis/nerd-fonts/releases)
- [CascadiaCode Nerd Font](https://github.com/ryanoasis/nerd-fonts/releases)

Then configure your terminal (Windows Terminal, PowerShell) to use the installed font.

---

## Configuration

### GitLab Hostname

By default, `glab-pipe` uses the GitLab instance configured in your `glab` CLI. If you need to specify a different GitLab hostname, set the `GITLAB_HOST` environment variable:

**PowerShell:**
```powershell
$env:GITLAB_HOST = "your-gitlab-instance.com"
```

**To make it permanent:**
```powershell
[System.Environment]::SetEnvironmentVariable('GITLAB_HOST', 'your-gitlab-instance.com', 'User')
```

**CMD:**
```cmd
set GITLAB_HOST=your-gitlab-instance.com
```

### Project Configuration

Projects are configured in `%USERPROFILE%\.glab-pipe\projects.json`. This file is created automatically on first run. No projects are hardcoded by default - you must add your own projects either through the TUI interface or by editing the configuration file directly.

---

## Installation

### Using the TUI Installer (recommended)

1. Build the project:
   ```batch
   build.bat
   ```

2. Run the installer:
   ```powershell
   dist\installer.exe
   ```
   The installer will:
   - Check that `glab` is installed and configured
   - Warn if a Nerd Font is not detected
   - Warn if PowerShell is not being used
   - Copy the binary to `%LOCALAPPDATA%\glab-pipe\glab-pipe.exe`
   - Add the directory to the user PATH (registry) and PowerShell profile
   - Create a `glab-pipe` function in the PowerShell profile

3. Restart PowerShell and run:
   ```powershell
   glab-pipe
   ```

### Manual Installation

1. Build:
   ```batch
   go build -ldflags="-s -w" -o dist\glab-pipe.exe .
   ```

2. Copy the binary to a directory in your PATH:
   ```powershell
   Copy-Item dist\glab-pipe.exe "$env:LOCALAPPDATA\glab-pipe\glab-pipe.exe"
   $env:PATH += ";$env:LOCALAPPDATA\glab-pipe"
   ```

3. Run:
   ```powershell
   glab-pipe
   ```

---

## Usage

### Commands

| Command | Description |
|---------|-------------|
| `glab-pipe` | Opens the main menu with project selector |
| `glab-pipe .` | Detects the GitLab project in the current directory and opens its pipelines |
| `glab-pipe --source <path>` | Opens pipelines for the given GitLab path |

**Examples:**
```powershell
# Main menu
glab-pipe

# Pipelines for the project in the current directory
cd C:\repos\my-project
glab-pipe .

# Pipelines by GitLab path (with hostname - recommended)
glab-pipe --source gitlab.example.com/group/subgroup/project

# Pipelines by GitLab path (without hostname - will use GITLAB_HOST if set)
glab-pipe --source group/subgroup/project

# Pipelines by full URL
glab-pipe --source https://gitlab.example.com/group/subgroup/project
```

### Keyboard Shortcuts

#### Welcome Screen / Project Selector
| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Navigate the list |
| `Enter` | Select project |
| `Esc` / `q` | Quit |

**Automatic Project Clone**: When you select a project that doesn't exist locally, `glab-pipe` will automatically prompt you to clone it to `~/repos`. This ensures you always have local access to the projects you want to monitor.

#### Pipeline Table
| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Navigate pipelines |
| `Enter` | Open pipeline jobs |
| `r` | Manual refresh |
| `n` | Create new pipeline |
| `Esc` | Back to main menu |
| `q` | Quit |

**Automatic Branch Prefix**: When creating a new pipeline, if you enter a ticket number like `CUC-639`, the system will automatically add the `story/` prefix, creating the pipeline on branch `story/CUC-639`. If you enter a full branch name with `/`, it will be used as-is.

#### Job List
| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Navigate jobs |
| `Enter` | Open job logs |
| `r` | Manual refresh |
| `Esc` | Back to pipeline table |
| `q` | Quit |

#### Job Log Modal
| Key | Action |
|-----|--------|
| `Esc` | Close modal (returns to job list) |
| `q` | Quit |

---

## Navigation Flow

```
Welcome Screen (splash + project selector)
    │
    │ Enter (select project)
    ▼
Pipeline Table  ◄─────── Esc (back)
    │
    │ Enter (select pipeline)
    ▼
Job List ───────────────── Esc (back to pipeline table)
    │
    │ Enter (select job)
    ▼
Job Log Modal ──────────── Esc (close modal)
```

---

## Status Icons

| Icon | Color | Status |
|------|-------|--------|
| `` (\uf192) | Blue | Running |
| `` (\uf05d) | Green | Success |
| `` (\uf52f) | Red | Failed |
| `` (\ueabd) | Gray | Canceled |
| `` (\uf2be) | Yellow | Manual trigger |
| `` (\uf01d) | Yellow | Waiting for manual play |

> **Note:** Icons require a Nerd Font installed in your terminal.

---

## Automatic Features

### Smart Branch Naming
When creating a new pipeline, the system intelligently handles branch names:
- **Ticket numbers**: Enter `CUC-639` → automatically uses `story/CUC-639`
- **Full branch names**: Enter `feature/new-api` → uses exactly as provided
- **Branch names with `/`: Enter `hotfix/urgent-fix` → uses exactly as provided

This saves time and ensures consistent branch naming conventions for ticket-based development.

### Automatic Project Clone
When you select a project from the welcome screen, `glab-pipe` automatically:
1. Searches for the project in common local directories (`~/repos`, `~/source/repos`, etc.)
2. If not found locally, prompts you to clone the project
3. Clones to `~/repos/<project-name>` using `glab repo clone`
4. Proceeds to show pipelines after successful clone

This ensures you always have local access to monitor pipelines without manual project management.

---

## Auto-Refresh

The app auto-refreshes every **2 seconds** under the following conditions:

- **Pipeline table:** when any pipeline has status `running` or `pending`
- **Job list:** when the pipeline or any of its jobs is still running
- **Log modal:** when the selected job is still running

Auto-refresh stops automatically when no active processes remain.

---

## Adding Projects

### Via the TUI Interface

When you launch `glab-pipe`, you'll see a welcome screen with two options:

1. **Select Repository...** - Choose from your configured projects
2. **Choose another...** - Add a new project by entering its path

When you select "Choose another...", you can enter the project path in any of these formats:

- **With hostname** (recommended): `gitlab.example.com/group/subgroup/project`
- **Without hostname**: `group/subgroup/project` (hostname will be auto-added from GITLAB_HOST if set)
- **Full URL**: `https://gitlab.example.com/group/subgroup/project`

The system will validate that you have access to the project before adding it to your configuration.

### Via Manual Configuration

You can also edit the configuration file directly (see Project Configuration section below).

---

## Project Configuration

Projects are loaded from `%USERPROFILE%\.glab-pipe\projects.json`.

The file is created automatically on first run with an empty projects list. You must add your own projects:

```json
{
  "projects": [
    {
      "display_name": "my-project",
      "full_path": "gitlab.example.com/group/subgroup/my-project"
    }
  ]
}
```

To add more projects, simply edit this JSON file. Project paths can be specified in three formats:
- **With hostname** (recommended): `gitlab.example.com/group/subgroup/project`
- **Without hostname**: `group/subgroup/project` (hostname will be auto-added from GITLAB_HOST if set)
- **Full URL**: `https://gitlab.example.com/group/subgroup/project`

---

## Building

```batch
# Build both binaries
build.bat

# Manually:
go build -ldflags="-s -w" -o dist\glab-pipe.exe .
go build -ldflags="-s -w" -o dist\installer.exe .\cmd\installer
```

### Manual debugging

```powershell
# Run directly without installing
dist\glab-pipe.exe

# Open current directory's project
dist\glab-pipe.exe .

# Open project by GitLab path (with hostname - recommended)
dist\glab-pipe.exe --source gitlab.example.com/group/subgroup/project

# Open project by GitLab path (without hostname - will use GITLAB_HOST if set)
dist\glab-pipe.exe --source group/subgroup/project
```

---

## Project Structure

```
gitlab-pipelines/
├── main.go                    # Entry point + CLI argument parsing
├── build.bat                  # Build script
├── go.mod / go.sum
├── cmd/
│   └── installer/
│       └── main.go            # TUI installer
├── internal/
│   ├── config/
│   │   └── config.go          # Read/write ~/.glab-pipe/projects.json
│   ├── gitlab/
│   │   └── gitlab.go          # glab CLI wrapper (ci list, ci get, ci trace)
│   ├── app/
│   │   └── app.go             # Main BubbleTea model + business logic
│   └── ui/
│       ├── ui.go              # All screen renderers
│       └── splash.go          # Welcome screen / project selector
├── dist/
│   ├── glab-pipe.exe          # Main application binary
│   └── installer.exe          # TUI installer binary
└── Documents/
    └── Usage.md               # Requirements specification
```

---

## Uninstalling

```powershell
# 1. Remove alias from PowerShell profile
notepad $PROFILE
# Delete the lines containing "glab-pipe"

# 2. Remove the install directory
Remove-Item -Recurse -Force "$env:LOCALAPPDATA\glab-pipe"

# 3. Remove the config file (optional)
Remove-Item -Recurse -Force "$env:USERPROFILE\.glab-pipe"
```

---

## License

MIT
