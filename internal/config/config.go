package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	RalphHome string `json:"ralph_home"`
}

// GetClaudeConfigDir returns the Claude config directory path
// On macOS/Linux: ~/.claude
// On Windows: %APPDATA%\claude
func GetClaudeConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", errors.New("APPDATA environment variable not set")
		}
		return filepath.Join(appData, "claude"), nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude"), nil
	}
}

// GetClaudeSkillsDir returns the Claude skills directory path
// Resolves symlinks if the skills directory is symlinked
func GetClaudeSkillsDir() (string, error) {
	claudeDir, err := GetClaudeConfigDir()
	if err != nil {
		return "", err
	}

	skillsDir := filepath.Join(claudeDir, "skills")

	// Resolve symlinks if present
	if resolved, err := filepath.EvalSymlinks(skillsDir); err == nil {
		return resolved, nil
	}

	return skillsDir, nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ralph", "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("ralph not configured. Run 'ralph setup' first")
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.RalphHome == "" {
		return nil, errors.New("ralph_home not set. Run 'ralph setup' first")
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func GetRalphHome() (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}
	return cfg.RalphHome, nil
}
