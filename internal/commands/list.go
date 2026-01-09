package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kento/ralph/internal/config"
	"github.com/kento/ralph/internal/prd"
	"github.com/kento/ralph/internal/project"
)

var (
	listTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	listHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252"))

	listCompleteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))

	listDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type projectInfo struct {
	id           string
	displayName  string
	branch       string
	stories      string
	archiveCount int
	isComplete   bool
}

// List shows all projects with archive info
func List() error {
	ralphHome, err := config.GetRalphHome()
	if err != nil {
		return err
	}

	projects, err := project.ListProjects()
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		fmt.Println("Run 'ralph init' in a project directory to get started.")
		return nil
	}

	// Gather project info
	var infos []projectInfo
	maxNameLen := 0
	for _, projectID := range projects {
		info := getProjectInfo(ralphHome, projectID)
		if len(info.displayName) > maxNameLen {
			maxNameLen = len(info.displayName)
		}
		infos = append(infos, info)
	}

	// Cap max name length
	if maxNameLen > 50 {
		maxNameLen = 50
	}
	if maxNameLen < 20 {
		maxNameLen = 20
	}

	fmt.Println(listTitleStyle.Render("Ralph Projects"))
	fmt.Println()

	// Table header (no styling to keep alignment)
	headerFmt := fmt.Sprintf("%%-%ds  %%-15s  %%-8s  %%s\n", maxNameLen)
	fmt.Printf(headerFmt, "Project", "Branch", "Stories", "Archives")
	fmt.Println(strings.Repeat("─", maxNameLen+2+15+2+8+2+8))

	// Rows (no styling on column values to keep alignment)
	rowFmt := fmt.Sprintf("%%-%ds  %%-15s  %%-8s  %%s\n", maxNameLen)
	for _, info := range infos {
		name := info.displayName
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		branch := info.branch
		if branch == "" {
			branch = "-"
		} else if len(branch) > 15 {
			branch = branch[:12] + "..."
		}

		stories := info.stories
		if stories == "" {
			stories = "-"
		} else if info.isComplete {
			stories = stories + " ✓"
		}

		archives := fmt.Sprintf("%d", info.archiveCount)

		fmt.Printf(rowFmt, name, branch, stories, archives)
	}

	return nil
}

func getProjectInfo(ralphHome, projectID string) projectInfo {
	projectDir := filepath.Join(ralphHome, "projects", projectID)

	info := projectInfo{
		id:          projectID,
		displayName: extractDisplayName(projectDir, projectID),
	}

	// Load PRD info
	if prd.Exists(projectDir) {
		p, err := prd.Load(projectDir)
		if err == nil {
			// Extract branch name without ralph/ prefix
			info.branch = strings.TrimPrefix(p.BranchName, "ralph/")

			completed := p.CompletedCount()
			total := p.TotalCount()
			info.stories = fmt.Sprintf("%d/%d", completed, total)
			info.isComplete = completed == total && total > 0
		}
	}

	// Count archives
	archiveDir := filepath.Join(projectDir, "archive")
	entries, err := os.ReadDir(archiveDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				info.archiveCount++
			}
		}
	}

	return info
}

// extractDisplayName shows the last 3 path segments
// Reads from .path file if available, otherwise falls back to project ID
func extractDisplayName(projectDir, projectID string) string {
	// Try to read original path from .path file
	pathFile := filepath.Join(projectDir, ".path")
	if data, err := os.ReadFile(pathFile); err == nil {
		fullPath := strings.TrimSpace(string(data))
		parts := strings.Split(fullPath, "/")

		// Take last 3 segments
		if len(parts) > 3 {
			return ".../" + strings.Join(parts[len(parts)-3:], "/")
		}
		return fullPath
	}

	// Fallback: just show the project ID (can't reliably parse)
	if len(projectID) > 30 {
		return "..." + projectID[len(projectID)-27:]
	}
	return projectID
}
