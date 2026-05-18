package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Project represents a GitLab project entry.
type Project struct {
	DisplayName string `json:"display_name"`
	FullPath    string `json:"full_path"`
}

// Config holds the application configuration.
type Config struct {
	Projects []Project `json:"projects"`
}

// defaultProjects are the built-in projects seeded when no config file exists yet.
var defaultProjects = []Project{
	{DisplayName: "account-processor-api", FullPath: "dfs/support/dfs-case-management/casemanagement/account-processor-api"},
	{DisplayName: "case-connector-api", FullPath: "dfs/support/dfs-case-management/casemanagement/case-connector-api"},
	{DisplayName: "case-gateway", FullPath: "dfs/support/dfs-case-management/casemanagement/case-gateway"},
	{DisplayName: "case-receiver-api", FullPath: "dfs/support/dfs-case-management/casemanagement/case-receiver-api"},
}

// ConfigDir returns the directory where glab-pipe stores configuration.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".glab-pipe")
	}
	return filepath.Join(home, ".glab-pipe")
}

// ConfigPath returns the full path to the projects config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "projects.json")
}

// Load reads the config file and returns the Config. If the file does not
// exist it creates one with the default projects and returns that.
func Load() (*Config, error) {
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First run: create config with defaults.
		cfg := &Config{Projects: defaultProjects}
		if err := Save(cfg); err != nil {
			// Non-fatal: just return defaults without persisting.
			return cfg, nil
		}
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the Config to disk.
func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0644)
}

// AddProject appends a project to the config file if not already present.
func AddProject(p Project) error {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{Projects: defaultProjects}
	}
	for _, existing := range cfg.Projects {
		if existing.FullPath == p.FullPath {
			return nil // already exists
		}
	}
	cfg.Projects = append(cfg.Projects, p)
	return Save(cfg)
}

// ProjectList returns the list of projects, loading from config or defaults.
func ProjectList() []Project {
	cfg, err := Load()
	if err != nil {
		return defaultProjects
	}
	if len(cfg.Projects) == 0 {
		return defaultProjects
	}
	return cfg.Projects
}
