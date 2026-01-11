package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/kento/ralph/internal/config"
	"github.com/kento/ralph/internal/prd"
	"github.com/kento/ralph/internal/project"
	"github.com/kento/ralph/internal/ui/format"
	"github.com/kento/ralph/internal/ui/styles"
)

// Constants
const (
	DefaultMaxIterations = 25
	IterationSleepSecs   = 2
	MaxNameDisplayLen    = 50
	MinNameDisplayLen    = 20
)

// projectRunFiles are the files managed during a run (used by clean command)
var projectRunFiles = []string{"prd.json", "prd.md", "progress.txt", ".last-branch"}

// HelpText is the CLI help message
const HelpText = `Ralph - Autonomous Agent CLI

Usage: ralph [command] [options]

Commands:
  (no args)    Interactive command picker
  help         Show this help message
  setup        Configure RALPH_HOME path
  home         Print RALPH_HOME path
  init         Initialize Ralph for current project
  run [n]      Run autonomous loop (default: 25 iterations)
  status       Show current project status
  prd          Launch Claude for PRD creation
  list         List all projects with archive info
  logs         View run logs
  archive      Manually archive current run
  clean        Remove project data (--all for everything)

Examples:
  ralph              # Interactive mode
  ralph home         # Print RALPH_HOME path
  ralph run          # Run with 25 iterations
  ralph run 5        # Run with 5 iterations
  ralph logs         # View run logs
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
	fmt.Println(format.FormatHeader("Ralph Setup"))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	cwd, _ := os.Getwd()

	// Check for existing config
	existingCfg, _ := config.Load()
	if existingCfg != nil {
		fmt.Println(format.FormatKeyValue("Current RALPH_HOME", existingCfg.RalphHome))
		fmt.Print(styles.Muted.Render("Enter new path (or press Enter to keep current): "))
	} else {
		fmt.Println(styles.Muted.Render("Enter path for RALPH_HOME"))
		fmt.Print(styles.Muted.Render(fmt.Sprintf("(or press Enter to use current directory: %s): ", cwd)))
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	path := strings.TrimSpace(input)

	// Keep existing if empty input and config exists
	if path == "" && existingCfg != nil {
		fmt.Println(styles.Muted.Render("Keeping existing configuration."))
		// Still try to install skills
		if err := installSkills(reader); err != nil {
			fmt.Println(format.FormatWarning(fmt.Sprintf("Failed to install skills: %v", err)))
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
	fmt.Println(format.FormatKeyValue("RALPH_HOME set to", absPath))

	// Install skills
	if err := installSkills(reader); err != nil {
		fmt.Println(format.FormatWarning(fmt.Sprintf("Failed to install skills: %v", err)))
	}

	fmt.Println()
	fmt.Println(format.FormatSuccess("Setup complete!"))
	fmt.Println(format.FormatNextStep("ralph init", "in your project directory to get started"))

	return nil
}

// installSkills detects Claude config and symlinks ralph skills
func installSkills(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println(format.FormatHeader("Claude Skills Installation"))
	fmt.Println()

	// Detect Claude config directory
	claudeDir, err := config.GetClaudeConfigDir()
	if err != nil {
		return fmt.Errorf("failed to detect Claude config directory: %w", err)
	}

	// Check if Claude is installed
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		fmt.Println(styles.Muted.Render(fmt.Sprintf("Claude config not found at %s", claudeDir)))
		fmt.Println(styles.Muted.Render("Skipping skills installation. Install Claude Code first."))
		return nil
	}

	fmt.Println(format.FormatKeyValue("Claude config", claudeDir))

	// Get Claude skills directory (resolves symlinks)
	skillsDir, err := config.GetClaudeSkillsDir()
	if err != nil {
		return fmt.Errorf("failed to get Claude skills directory: %w", err)
	}

	// Create skills directory if it doesn't exist
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	fmt.Println(format.FormatKeyValue("Skills directory", skillsDir))

	// Find ralph skills directory
	ralphSkillsDir, err := findRalphSkillsDir()
	if err != nil {
		fmt.Println()
		fmt.Println(styles.Muted.Render("Could not auto-detect ralph skills directory."))
		fmt.Print(styles.Muted.Render("Enter path to ralph project (or press Enter to skip): "))

		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		ralphPath := strings.TrimSpace(input)
		if ralphPath == "" {
			fmt.Println(styles.Muted.Render("Skipping skills installation."))
			return nil
		}

		ralphSkillsDir = filepath.Join(expandTilde(ralphPath), "skills")
		if _, err := os.Stat(ralphSkillsDir); os.IsNotExist(err) {
			return fmt.Errorf("skills directory not found at %s", ralphSkillsDir)
		}
	}

	fmt.Println(format.FormatKeyValue("Ralph skills", ralphSkillsDir))
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
					fmt.Printf("  %s %s %s\n", styles.Muted.Render("="), skillName, styles.Muted.Render("(already installed)"))
					continue
				}
				// Remove old symlink
				os.Remove(dstPath)
			} else {
				// It's a regular directory, skip
				fmt.Printf("  %s %s %s\n", styles.WarningText.Render("!"), skillName, styles.Muted.Render("(skipped: directory exists)"))
				continue
			}
		}

		// Create symlink
		if err := os.Symlink(srcPath, dstPath); err != nil {
			fmt.Printf("  %s %s %s\n", styles.ErrorText.Render(styles.ErrorIcon), skillName, styles.Muted.Render(fmt.Sprintf("(failed: %v)", err)))
			continue
		}

		fmt.Printf("  %s %s\n", styles.SuccessText.Render(styles.CheckIcon), skillName)
		installedCount++
	}

	if installedCount > 0 {
		fmt.Println()
		fmt.Println(format.FormatSuccess(fmt.Sprintf("Installed %d skill(s) to Claude", installedCount)))
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

	// Display success
	fmt.Println(format.FormatSuccess("Ralph initialized"))
	fmt.Println()
	fmt.Println(format.FormatKeyValue("Project", filepath.Base(cwd)))
	fmt.Println(format.FormatKeyValue("Path", cwd))
	fmt.Println()
	fmt.Println(styles.Muted.Render("Next steps:"))
	fmt.Println(format.FormatNextStep("/prd", "in Claude to create a PRD"))
	fmt.Println(format.FormatNextStep("ralph run", "to start the autonomous loop"))

	return nil
}

// Status shows the current project status
func Status() error {
	projectDir, err := project.GetProjectDir()
	if err != nil {
		return err
	}

	cwd, _ := os.Getwd()

	fmt.Println(format.FormatHeader("Project Status"))
	fmt.Println()

	// Check if project is initialized
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		fmt.Println(format.FormatKeyValue("Project", filepath.Base(cwd)))
		fmt.Println(format.FormatKeyValue("Status", styles.WarningText.Render("Not initialized")))
		fmt.Println()
		fmt.Println(format.FormatNextStep("ralph init", "to initialize this project"))
		return nil
	}

	fmt.Println(format.FormatKeyValue("Project", filepath.Base(cwd)))

	// Load PRD if exists
	if prd.Exists(projectDir) {
		p, err := prd.Load(projectDir)
		if err != nil {
			return fmt.Errorf("failed to load prd.json: %w", err)
		}

		branch := strings.TrimPrefix(p.BranchName, "ralph/")
		fmt.Println(format.FormatKeyValue("Branch", branch))

		completed := p.CompletedCount()
		total := p.TotalCount()
		storiesText := fmt.Sprintf("%d/%d complete", completed, total)

		if p.IsComplete() {
			fmt.Println(format.FormatKeyValue("Stories", styles.SuccessText.Render(storiesText+" "+styles.CheckIcon)))
		} else {
			fmt.Println(format.FormatKeyValue("Stories", storiesText))
		}

		// Progress bar
		if total > 0 {
			prog := progress.New(progress.WithDefaultGradient(), progress.WithWidth(30), progress.WithoutPercentage())
			percent := float64(completed) / float64(total)
			fmt.Println()
			fmt.Println(prog.ViewAs(percent))
		}

		fmt.Println()

		if p.IsComplete() {
			fmt.Println(format.FormatSuccess("All stories complete!"))
			fmt.Println(format.FormatNextStep("ralph archive", "to archive and start fresh"))
		} else {
			next := p.NextIncomplete()
			if next != nil {
				fmt.Println(format.FormatKeyValue("Next", fmt.Sprintf("[%s] %s", next.ID, next.Title)))
			}
			fmt.Println()
			fmt.Println(format.FormatNextStep("ralph run", "to continue"))
		}
	} else {
		fmt.Println(format.FormatKeyValue("Status", styles.Muted.Render("No PRD")))
		fmt.Println()
		fmt.Println(format.FormatNextStep("/prd", "in Claude to create a PRD"))
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
	archivedFiles := []string{}
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
			archivedFiles = append(archivedFiles, file)
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

	// Display success
	fmt.Println(format.FormatSuccess("Archive created"))
	fmt.Println()
	fmt.Println(format.FormatKeyValue("Archived", strings.Join(archivedFiles, ", ")))
	fmt.Println(format.FormatKeyValue("Location", archiveDir))
	fmt.Println()
	fmt.Println(format.FormatNextStep("/prd", "in Claude to start a new feature"))

	return nil
}

// Clean removes project data
func Clean(all bool) error {
	projectDir, err := project.GetProjectDir()
	if err != nil {
		return err
	}

	// Show warning and list what will be deleted
	fmt.Println(format.FormatWarning("This will delete:"))
	fmt.Println()

	if all {
		fmt.Println(format.FormatBullet("All project data including archives"))
		fmt.Println(format.FormatBullet(projectDir))
	} else {
		for _, file := range projectRunFiles {
			fmt.Println(format.FormatBullet(file))
		}
		fmt.Println()
		fmt.Println(styles.Muted.Render("Archives will be preserved."))
	}

	fmt.Println()
	fmt.Println(styles.Muted.Render("This cannot be undone."))
	fmt.Print(styles.WarningText.Render("Continue? [y/N] "))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		fmt.Println(styles.Muted.Render("Cancelled."))
		return nil
	}

	fmt.Println()

	if all {
		// Remove entire project directory
		if err := os.RemoveAll(projectDir); err != nil {
			return err
		}
		fmt.Println(format.FormatSuccess("Removed all project data"))
	} else {
		// Remove only current run files (not archive)
		for _, file := range projectRunFiles {
			path := filepath.Join(projectDir, file)
			os.Remove(path)
		}
		fmt.Println(format.FormatSuccess("Cleaned current run files"))
	}

	return nil
}
