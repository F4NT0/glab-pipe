# Setup Guide

This guide explains how to configure `glab` (GitLab CLI) and how `glab-pipe` recognizes projects.

## Installing and Configuring glab

`glab-pipe` requires `glab` (GitLab CLI) to interact with GitLab CI/CD.

### 1. Install glab

**Windows (Scoop):**
```powershell
scoop install glab
```

**Windows (Chocolatey):**
```powershell
choco install glab
```

**macOS (Homebrew):**
```bash
brew install glab
```

**Linux:**
```bash
# See https://github.com/profclems/glab/blob/main/docs/install.md
```

### 2. Authenticate with GitLab

After installing, authenticate with your GitLab instance:

```bash
glab auth login
```

Follow the prompts:
- Choose your GitLab instance (e.g., `gitlab.example.com`)
- Enter your credentials or use the web browser authentication

### 3. Verify Installation

```bash
glab auth status
glab ci list -R <your-project-path> -P 1
```

## Project Configuration

`glab-pipe` stores project configuration in `~/.glab-pipe/projects.json`.

### Default Projects

On first run, `glab-pipe` creates an empty configuration file. No projects are hardcoded by default - you must add your own projects through the TUI interface or by editing the configuration file directly.

### Adding Projects

There are two ways to add projects:

#### Option 1: Via the TUI
1. Run `glab-pipe`
2. Select **"Choose another..."**
3. Enter the GitLab path or URL:
   - Path: `group/subgroup/my-project`
   - URL: `https://gitlab.example.com/group/subgroup/my-project`
4. Press Enter to validate and save

#### Option 2: Manual Configuration

Edit `~/.glab-pipe/projects.json`:

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

### Project Path Format

GitLab project paths use the format:
```
<group>/<subgroup>/<...>/<project-name>
```

For example:
- `group/subgroup/project`
- `my-org/my-group/my-project`

### Access Requirements

To use a project in `glab-pipe`, you must have:
- **Read access** to the project (to view pipelines and jobs)
- **GitLab CLI configured** with proper authentication

If you don't have access, the validation will show an error message explaining the issue.

## Troubleshooting

### "glab not found"
Ensure `glab` is installed and in your system PATH:
```bash
where glab  # Windows
which glab # macOS/Linux
```

### "No access to project"
- Verify your GitLab permissions for the project
- Check that `glab auth status` shows you're authenticated
- Ensure the project path is correct

### Configuration file location
- **Windows:** `C:\Users\<username>\.glab-pipe\projects.json`
- **macOS/Linux:** `~/.glab-pipe/projects.json`
