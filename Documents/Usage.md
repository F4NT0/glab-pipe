# glab-pipe - GitLab CI/CD Pipeline TUI

## Overview

`glab-pipe` is a terminal user interface (TUI) application for monitoring and managing GitLab CI/CD pipelines. It provides real-time monitoring of pipeline status, job details, and log traces with automatic refresh capabilities.

## Before You Begin

1. Read this documentation thoroughly
2. Configure your projects in the local configuration file
3. Ensure `glab` CLI is installed and authenticated to your GitLab instance
4. A Nerd Font is recommended for proper icon display
5. PowerShell is recommended over Command Prompt for better compatibility

## Project Configuration

Configure your projects in `~/.glab-pipe/projects.json`:

```json
{
  "projects": [
    {
      "display_name": "my-project",
      "full_path": "group/subgroup/project"
    }
  ]
}
```

**Supported path formats:**
- **With hostname (recommended):** `gitlab.example.com/group/subgroup/project`
- **Without hostname:** `group/subgroup/project` (hostname auto-detected from `glab auth status`)
- **Full URL:** `https://gitlab.example.com/group/subgroup/project`

**Hostname Detection:**
The application automatically detects your GitLab instance hostname from:
1. `GITLAB_HOST` environment variable (highest priority)
2. `glab auth status` output (detects authenticated instances)
3. If neither is available, relies on glab's default configuration

## Usage

### Starting the Application

```bash
# Open main menu to select a project
glab-pipe

# Auto-detect project from current directory
glab-pipe .

# Specify project path directly
glab-pipe --source group/subgroup/project
```

### Main Screens

#### 1. Project Selection Screen

- ASCII art welcome banner
- List of configured projects
- Use `↑/↓` to navigate, `Enter` to select, `Esc` to quit

#### 2. Pipeline List Screen

Shows the last 10 pipelines for the selected project:

| Column | Description |
|--------|-------------|
| Status | Pipeline status with Nerd Font icons |
| ID | Pipeline ID (e.g., #56012567) |
| Branch | Branch name (e.g., story/CUC-689) |
| Started | When the pipeline started |

**Status Icons:**
- `` (\uf192) - Running (blue)
- `` (\uf05d) - Success (green)
- `` (\uf52f) - Failed (red)
- `` (\ueabd) - Canceled (gray)
- `` (\uf2be) - Manual trigger (yellow)

**Keybindings:**
- `↑/↓` - Navigate pipelines
- `Enter` - View pipeline details
- `r` - Refresh pipeline list
- `n` - Create new pipeline
- `Esc` - Return to project selection
- `q` - Quit

**Auto-refresh:** Updates every 2 seconds while any pipeline is running.

#### 3. Pipeline Detail Screen

Shows detailed information about the selected pipeline:

**Pipeline Summary (top section):**
- Pipeline ID
- Status
- Source (trigger method)
- Branch (ref)
- User who triggered
- Created timestamp
- Updated timestamp

**Job List (table):**

| Column | Description |
|--------|-------------|
| Status | Job status with icons |
| Name | Job name |

**Job Status Icons:**
- `` (\uf192) - Running (blue)
- `` (\uf05d) - Success (green)
- `` (\uf52f) - Failed (red)
- `` (\ueabd) - Canceled (gray)
- `` (\uf2be) - Manual (yellow)
- `` (\uf01d) - Waiting for manual play (yellow)

**Keybindings:**
- `↑/↓` - Navigate jobs
- `Enter` - View job logs
- `r` - Refresh job list
- `Esc` - Return to pipeline list
- `q` - Quit

**Auto-refresh:** Updates every 2 seconds while pipeline or any job is running.

#### 4. Job Log Modal

Displays log output for the selected job:

- **Modal title:** Status icon, job name, pipeline ID, branch
- **Content:** Full job log trace
- **Auto-refresh:** Updates every 2 seconds while job is running

**Keybindings:**
- `↑/↓` - Scroll log
- `PgUp/PgDn` - Scroll half page
- `g` - Jump to top
- `G` - Jump to bottom
- `/` - Search logs
- `n/N` - Next/previous search match
- `Esc` - Close modal

### Creating a New Pipeline

Press `n` on the pipeline list screen to create a new pipeline.

**What happens when you create a pipeline:**

The application uses `glab ci run` to trigger a pipeline on the GitLab server. This process:

1. **Branch Verification:** Checks if the specified branch exists on the GitLab server (not locally)
2. **Remote Trigger:** Triggers a new CI/CD pipeline on the GitLab server for the specified branch
3. **Commit Selection:** Uses the **latest commit** on the remote branch (does not use local commits)
4. **Variable Injection:** Optionally injects CI/CD variables into the pipeline
5. **Pipeline Execution:** GitLab CI/CD executes the pipeline on the remote server
6. **ID Return:** Returns the pipeline ID, which the TUI uses to refresh the pipeline list

**Important Notes:**
- **No local commit involvement:** The pipeline runs on the remote GitLab server using the remote branch's latest commit
- **Branch must exist remotely:** If the branch doesn't exist on GitLab, creation fails
- **No push operation:** Does not push local commits to the remote repository
- **Trigger-only operation:** Simply triggers the GitLab CI/CD system to execute

**Branch Name Normalization:**
- Ticket format (e.g., `CUC-639`) → Automatically prepends `story/` → `story/CUC-639`
- Long-lived branches (`develop`, `release`, `hotfix`, `main`, `master`) → Used as-is
- Branches with `/` → Used as-is
- Other inputs → Used as-is

**Pipeline Creation Modal:**

**Input Fields:**
- Branch name (required)
- Variables (optional, format: `KEY:VALUE,KEY2:VALUE2`)

**Keybindings:**
- `Tab` - Switch between input fields
- `Enter` - Create pipeline
- `Esc` - Cancel
- `q` - Quit

**Example:**
```
Branch: CUC-639
Variables: DEPLOY_ENV:staging,DEBUG:true

→ Will use: story/CUC-639
```

### Installer

The installer provides an interactive TUI setup:

1. **Prerequisites Check:**
   - Verifies `glab` CLI installation and configuration
   - Checks for Nerd Font support (warning if missing)
   - Checks shell environment (PowerShell recommended)

2. **Installation:**
   - Installs `glab-pipe.exe` to a system directory
   - Adds to PATH for both PowerShell and Command Prompt
   - Configures the `glab-pipe` command

3. **Visual Design:**
   - Modern, friendly TUI interface
   - Follows design patterns from similar tools (e.g., clidocs)

## Command Line Options

```bash
# Interactive project selection
glab-pipe

# Auto-detect from current directory
glab-pipe .

# Specify project path
glab-pipe --source group/subgroup/project
```

## Technical Details

### Data Sources

- **Pipeline list:** `glab ci list -R <project> -F json -P 10`
- **Pipeline details:** `glab ci get -R <project> -p <pipeline-id> -F json --with-job-details`
- **Job traces:** `glab ci trace <job-id> -R <project>`

### Auto-Refresh Behavior

- **Pipeline list:** Refreshes every 2 seconds while any pipeline is running
- **Job list:** Refreshes every 2 seconds while pipeline or any job is running
- **Job logs:** Refreshes every 2 seconds while the selected job is running
- **Stops auto-refresh:** When all pipelines/jobs reach terminal state (success, failed, canceled)

### Error Handling

- Displays error messages in the TUI
- Graceful handling of network issues
- Clear feedback for authentication problems
- Helpful error messages for missing branches or permissions

## Screenshots

*Placeholder for screenshots:*
- Project selection screen
- Pipeline list screen
- Pipeline detail screen
- Job log modal
- Pipeline creation modal
- Installer screens
