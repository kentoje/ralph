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

// expandTilde expands ~ to the user's home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}

// Setup configures the RALPH_HOME path and installs skills
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
		// Still try to install skills
		if err := installSkills(reader); err != nil {
			fmt.Printf("Warning: failed to install skills: %v\n", err)
		}
		return nil
	}

	// Use current directory if empty input and no existing config
	if path == "" {
		path = cwd
	}

	// Expand ~ and convert to absolute path
	path = expandTilde(path)
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

	// Install skills
	if err := installSkills(reader); err != nil {
		fmt.Printf("Warning: failed to install skills: %v\n", err)
	}

	fmt.Println()
	fmt.Println("Setup complete! You can now use 'ralph init' in your project directory.")

	return nil
}

// installSkills detects Claude config and symlinks ralph skills
func installSkills(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("Claude Skills Installation")
	fmt.Println("--------------------------")

	// Detect Claude config directory
	claudeDir, err := config.GetClaudeConfigDir()
	if err != nil {
		return fmt.Errorf("failed to detect Claude config directory: %w", err)
	}

	// Check if Claude is installed
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		fmt.Printf("Claude config directory not found at %s\n", claudeDir)
		fmt.Println("Skipping skills installation. Install Claude Code first, then run 'ralph setup' again.")
		return nil
	}

	fmt.Printf("Detected Claude config: %s\n", claudeDir)

	// Get Claude skills directory (resolves symlinks)
	skillsDir, err := config.GetClaudeSkillsDir()
	if err != nil {
		return fmt.Errorf("failed to get Claude skills directory: %w", err)
	}

	// Create skills directory if it doesn't exist
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	fmt.Printf("Skills directory: %s\n", skillsDir)

	// Find ralph skills directory
	ralphSkillsDir, err := findRalphSkillsDir()
	if err != nil {
		fmt.Println()
		fmt.Println("Could not auto-detect ralph skills directory.")
		fmt.Print("Enter path to ralph project (or press Enter to skip): ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		ralphPath := strings.TrimSpace(input)
		if ralphPath == "" {
			fmt.Println("Skipping skills installation.")
			return nil
		}

		ralphSkillsDir = filepath.Join(expandTilde(ralphPath), "skills")
		if _, err := os.Stat(ralphSkillsDir); os.IsNotExist(err) {
			return fmt.Errorf("skills directory not found at %s", ralphSkillsDir)
		}
	}

	fmt.Printf("Ralph skills found: %s\n", ralphSkillsDir)
	fmt.Println()

	// List and symlink each skill
	entries, err := os.ReadDir(ralphSkillsDir)
	if err != nil {
		return fmt.Errorf("failed to read skills directory: %w", err)
	}

	installedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()
		srcPath := filepath.Join(ralphSkillsDir, skillName)
		dstPath := filepath.Join(skillsDir, skillName)

		// Check if skill already exists
		if info, err := os.Lstat(dstPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				// It's a symlink, check if it points to the same place
				existing, _ := os.Readlink(dstPath)
				if existing == srcPath {
					fmt.Printf("  [=] %s (already installed)\n", skillName)
					continue
				}
				// Remove old symlink
				os.Remove(dstPath)
			} else {
				// It's a regular directory, skip
				fmt.Printf("  [!] %s (skipped: directory already exists)\n", skillName)
				continue
			}
		}

		// Create symlink
		if err := os.Symlink(srcPath, dstPath); err != nil {
			fmt.Printf("  [x] %s (failed: %v)\n", skillName, err)
			continue
		}

		fmt.Printf("  [+] %s\n", skillName)
		installedCount++
	}

	if installedCount > 0 {
		fmt.Printf("\nInstalled %d skill(s) to Claude.\n", installedCount)
	}

	return nil
}

// findRalphSkillsDir attempts to find the ralph skills directory
func findRalphSkillsDir() (string, error) {
	var candidates []string

	// Current working directory
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "skills"))
	}

	// Relative to executable
	if execPath, err := os.Executable(); err == nil {
		execPath, _ = filepath.EvalSymlinks(execPath)
		execDir := filepath.Dir(execPath)
		candidates = append(candidates,
			filepath.Join(execDir, "skills"),
			filepath.Join(execDir, "..", "skills"),
		)
	}

	for _, dir := range candidates {
		if isValidSkillsDir(dir) {
			return filepath.Abs(dir)
		}
	}

	return "", fmt.Errorf("could not find ralph skills directory")
}

// isValidSkillsDir checks if a directory contains valid skills
func isValidSkillsDir(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}

	// Check if it contains at least one skill directory with SKILL.md
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				return true
			}
		}
	}

	return false
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
