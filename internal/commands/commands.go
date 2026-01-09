package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kento/ralph/internal/config"
	"github.com/kento/ralph/internal/prd"
	"github.com/kento/ralph/internal/project"
)

// Constants
const (
	DefaultMaxIterations = 25
	IterationSleepSecs   = 2
	MaxNameDisplayLen    = 50
	MinNameDisplayLen    = 20
)

// HelpText is the CLI help message
const HelpText = `Ralph - Autonomous Agent CLI

Usage: ralph [command] [options]

Commands:
  (no args)    Interactive command picker
  help         Show this help message
  setup        Configure RALPH_HOME path
  home         Print RALPH_HOME path
  init         Initialize Ralph for current project
  run [n]      Run autonomous loop (default: 10 iterations)
  status       Show current project status
  prd          Launch Claude for PRD creation
  list         List all projects with archive info
  archive      Manually archive current run
  clean        Remove project data (--all for everything)

Examples:
  ralph              # Interactive mode
  ralph home         # Print RALPH_HOME path
  ralph run          # Run with 10 iterations
  ralph run 5        # Run with 5 iterations
  ralph clean --all  # Remove all project data
`

// Home prints the RALPH_HOME path
func Home() error {
	home, err := config.GetRalphHome()
	if err != nil {
		return err
	}
	fmt.Println(home)
	return nil
}

// Setup configures the RALPH_HOME path
func Setup() error {
	fmt.Println("Ralph Setup")
	fmt.Println("===========")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	cwd, _ := os.Getwd()

	// Check for existing config
	existingCfg, _ := config.Load()
	if existingCfg != nil {
		fmt.Printf("Current RALPH_HOME: %s\n", existingCfg.RalphHome)
		fmt.Print("Enter new path (or press Enter to keep current): ")
	} else {
		fmt.Printf("Enter path for RALPH_HOME (or press Enter to use current directory: %s): ", cwd)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	path := strings.TrimSpace(input)

	// Keep existing if empty input and config exists
	if path == "" && existingCfg != nil {
		fmt.Println("Keeping existing configuration.")
		return nil
	}

	// Use current directory if empty input and no existing config
	if path == "" {
		path = cwd
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(home, path[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create projects subdirectory
	projectsDir := filepath.Join(absPath, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		return fmt.Errorf("failed to create projects directory: %w", err)
	}

	// Save config
	cfg := &config.Config{RalphHome: absPath}
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("RALPH_HOME set to: %s\n", absPath)
	fmt.Println("Setup complete! You can now use 'ralph init' in your project directory.")

	return nil
}

// Init initializes Ralph for the current project
func Init() error {
	projectDir, err := project.EnsureProjectDir()
	if err != nil {
		return err
	}

	cwd, _ := os.Getwd()
	fmt.Printf("Initialized Ralph for: %s\n", cwd)
	fmt.Printf("Project data directory: %s\n", projectDir)

	// Store original path for display purposes
	pathFile := filepath.Join(projectDir, ".path")
	if err := os.WriteFile(pathFile, []byte(cwd), 0644); err != nil {
		return err
	}

	// Create empty progress.txt with header
	progressPath := filepath.Join(projectDir, "progress.txt")
	if _, err := os.Stat(progressPath); os.IsNotExist(err) {
		header := fmt.Sprintf("# Progress Log\n# Project: %s\n# Initialized: %s\n\n", cwd, time.Now().Format("2006-01-02 15:04:05"))
		if err := os.WriteFile(progressPath, []byte(header), 0644); err != nil {
			return err
		}
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Run 'ralph prd' to create a PRD")
	fmt.Println("  2. Run 'ralph run' to start the autonomous loop")

	return nil
}

// Status shows the current project status
func Status() error {
	projectDir, err := project.GetProjectDir()
	if err != nil {
		return err
	}

	cwd, _ := os.Getwd()
	fmt.Printf("Project: %s\n", cwd)
	fmt.Printf("Data dir: %s\n", projectDir)
	fmt.Println()

	// Check if project is initialized
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		fmt.Println("Status: Not initialized")
		fmt.Println("Run 'ralph init' to initialize this project.")
		return nil
	}

	// Load PRD if exists
	if prd.Exists(projectDir) {
		p, err := prd.Load(projectDir)
		if err != nil {
			return fmt.Errorf("failed to load prd.json: %w", err)
		}

		fmt.Printf("Branch: %s\n", p.BranchName)
		fmt.Printf("Stories: %d/%d complete\n", p.CompletedCount(), p.TotalCount())

		if p.IsComplete() {
			fmt.Println("Status: All stories complete!")
		} else {
			next := p.NextIncomplete()
			if next != nil {
				fmt.Printf("Next: [%s] %s\n", next.ID, next.Title)
			}
		}
	} else {
		fmt.Println("No prd.json found.")
		fmt.Println("Run 'ralph prd' to create a PRD.")
	}

	return nil
}

// Prd launches Claude for PRD creation
func Prd() error {
	projectDir, err := project.GetProjectDir()
	if err != nil {
		return err
	}

	cwd, _ := os.Getwd()

	systemPrompt := fmt.Sprintf(`You are helping to create a PRD (Product Requirements Document) for an autonomous agent.

Project directory: %s
PRD data will be saved to: %s

Use the /prd skill to interactively create a PRD, then use the /ralph skill to convert it to prd.json format.

The prd.json should be saved to: %s/prd.json
`, cwd, projectDir, projectDir)

	fmt.Println("Launching Claude for PRD creation...")
	fmt.Println()

	cmd := exec.Command("claude", "--system-prompt", systemPrompt)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = cwd

	return cmd.Run()
}

// Archive manually archives the current run
func Archive() error {
	projectDir, err := project.GetProjectDir()
	if err != nil {
		return err
	}

	if !prd.Exists(projectDir) {
		return fmt.Errorf("no prd.json found - nothing to archive")
	}

	p, err := prd.Load(projectDir)
	if err != nil {
		return err
	}

	// Create archive directory with timestamp and branch name
	branchName := strings.TrimPrefix(p.BranchName, "ralph/")
	branchName = strings.ReplaceAll(branchName, "/", "-")
	timestamp := time.Now().Format("2006-01-02")
	archiveName := fmt.Sprintf("%s-%s", timestamp, branchName)
	archiveDir := filepath.Join(projectDir, "archive", archiveName)

	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return err
	}

	// Copy files to archive
	filesToArchive := []string{"prd.json", "progress.txt", "prd.md"}
	for _, file := range filesToArchive {
		src := filepath.Join(projectDir, file)
		if _, err := os.Stat(src); err == nil {
			data, err := os.ReadFile(src)
			if err != nil {
				continue
			}
			dst := filepath.Join(archiveDir, file)
			if err := os.WriteFile(dst, data, 0644); err != nil {
				return err
			}
		}
	}

	// Reset progress.txt
	cwd, _ := os.Getwd()
	header := fmt.Sprintf("# Progress Log\n# Project: %s\n# Reset: %s\n\n", cwd, time.Now().Format("2006-01-02 15:04:05"))
	progressPath := filepath.Join(projectDir, "progress.txt")
	if err := os.WriteFile(progressPath, []byte(header), 0644); err != nil {
		return err
	}

	// Remove PRD files
	os.Remove(filepath.Join(projectDir, "prd.json"))
	os.Remove(filepath.Join(projectDir, "prd.md"))
	os.Remove(filepath.Join(projectDir, ".last-branch"))

	fmt.Printf("Archived to: %s\n", archiveDir)
	fmt.Println("PRD files removed. Ready for a new feature.")

	return nil
}

// Clean removes project data
func Clean(all bool) error {
	projectDir, err := project.GetProjectDir()
	if err != nil {
		return err
	}

	if all {
		// Remove entire project directory
		if err := os.RemoveAll(projectDir); err != nil {
			return err
		}
		fmt.Printf("Removed all project data: %s\n", projectDir)
	} else {
		// Remove only current run files (not archive)
		files := []string{"prd.json", "prd.md", "progress.txt", ".last-branch"}
		for _, file := range files {
			path := filepath.Join(projectDir, file)
			os.Remove(path)
		}
		fmt.Println("Removed current run files (archive preserved).")
	}

	return nil
}
