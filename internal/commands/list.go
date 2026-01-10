package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/kento/ralph/internal/config"
	"github.com/kento/ralph/internal/prd"
	"github.com/kento/ralph/internal/project"
)

var listTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("170"))

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
	if maxNameLen > MaxNameDisplayLen {
		maxNameLen = MaxNameDisplayLen
	}
	if maxNameLen < MinNameDisplayLen {
		maxNameLen = MinNameDisplayLen
	}

	fmt.Println(listTitleStyle.Render("Ralph Projects"))
	fmt.Println()

	// Define table columns
	columns := []table.Column{
		{Title: "Project", Width: maxNameLen},
		{Title: "Branch", Width: 15},
		{Title: "Stories", Width: 10},
		{Title: "Archives", Width: 10},
	}

	// Build table rows
	rows := []table.Row{}
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

		rows = append(rows, table.Row{name, branch, stories, archives})
	}

	// Create and style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("170"))
	s.Cell = s.Cell.Padding(0, 1)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(rows)+1),
		table.WithStyles(s),
	)

	fmt.Println(t.View())

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
