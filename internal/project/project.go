package project

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kento/ralph/internal/config"
)

// GetProjectID derives the project ID from a path
// Path: /Volumes/HomeX/kento/Documents/gitlab/assets-page
// Project ID: volumes-homex-kento-documents-gitlab-assets-page
func GetProjectID(path string) string {
	// Remove leading slash
	id := strings.TrimPrefix(path, "/")
	// Replace / with -
	id = strings.ReplaceAll(id, "/", "-")
	// Lowercase
	id = strings.ToLower(id)
	return id
}

// GetProjectDir returns the project data directory for the current working directory
func GetProjectDir() (string, error) {
	ralphHome, err := config.GetRalphHome()
	if err != nil {
		return "", err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	projectID := GetProjectID(cwd)
	return filepath.Join(ralphHome, "projects", projectID), nil
}

// EnsureProjectDir creates the project directory if it doesn't exist
func EnsureProjectDir() (string, error) {
	dir, err := GetProjectDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Create archive subdirectory
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", err
	}

	return dir, nil
}

// ListProjects returns all project directories
func ListProjects() ([]string, error) {
	ralphHome, err := config.GetRalphHome()
	if err != nil {
		return nil, err
	}

	projectsDir := filepath.Join(ralphHome, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var projects []string
	for _, entry := range entries {
		if entry.IsDir() {
			projects = append(projects, entry.Name())
		}
	}

	return projects, nil
}

// ProjectExists checks if a project directory exists
func ProjectExists() (bool, error) {
	dir, err := GetProjectDir()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
