package commands

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kento/ralph/internal/config"
	"github.com/kento/ralph/internal/prd"
	"github.com/kento/ralph/internal/project"
	"github.com/kento/ralph/internal/stream"
)

var (
	runTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Padding(0, 1)

	runInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	runBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	runHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)
)

// runState holds shared state between TUI and runner goroutine
type runState struct {
	mu         sync.Mutex
	cancel     context.CancelFunc
	currentCmd *exec.Cmd
}

func (s *runState) setCmd(cmd *exec.Cmd) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentCmd = cmd
}

func (s *runState) killCurrentProcess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentCmd != nil && s.currentCmd.Process != nil {
		// Kill the process group to ensure all child processes are killed
		s.currentCmd.Process.Kill()
	}
	if s.cancel != nil {
		s.cancel()
	}
}

type runModel struct {
	viewport viewport.Model
	progress progress.Model
	content  *strings.Builder
	iteration     int
	maxIterations int
	currentStory  string
	branch        string
	completed     int
	total         int
	running       bool
	done          bool
	err           error
	width         int
	height        int
	projectDir    string
	workingDir    string
	state         *runState
}

type outputMsg string
type iterationCompleteMsg struct {
	success bool
}
type runDoneMsg struct {
	success bool
	err     error
}

func (m runModel) Init() tea.Cmd {
	return nil
}

func (m runModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 4
		footerHeight := 3
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - headerHeight - footerHeight - 4
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			// Kill the running process
			if m.state != nil {
				m.state.killCurrentProcess()
			}
			return m, tea.Quit
		}

	case outputMsg:
		line := string(msg)
		m.content.WriteString(line + "\n")
		m.viewport.SetContent(m.content.String())
		m.viewport.GotoBottom()

	case iterationCompleteMsg:
		if msg.success {
			m.completed++
		}
		m.iteration++
		if m.iteration >= m.maxIterations || m.completed >= m.total {
			m.done = true
			m.running = false
			return m, tea.Quit
		}

	case runDoneMsg:
		m.done = true
		m.running = false
		m.err = msg.err
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m runModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	// Viewport with output
	viewportContent := runBorderStyle.Render(m.viewport.View())
	b.WriteString(viewportContent + "\n\n")

	// Progress bar
	progress := m.renderProgressBar()
	b.WriteString(progress + "\n\n")

	// Info (moved to bottom)
	title := runTitleStyle.Render(fmt.Sprintf("Ralph - Iteration %d/%d", m.iteration+1, m.maxIterations))
	storyInfo := runInfoStyle.Render(m.currentStory)
	b.WriteString(fmt.Sprintf("%s %s\n", title, storyInfo))

	if m.branch != "" {
		b.WriteString(runInfoStyle.Render(fmt.Sprintf("Branch: %s", m.branch)) + "\n")
	}

	// Help
	help := runHelpStyle.Render("q quit")
	b.WriteString(help)

	return b.String()
}

func (m runModel) renderProgressBar() string {
	if m.total == 0 {
		return runInfoStyle.Render("Progress: No stories loaded")
	}

	percent := float64(m.completed) / float64(m.total)
	bar := m.progress.ViewAs(percent)
	return fmt.Sprintf("%s %d/%d stories complete", bar, m.completed, m.total)
}

// Run executes the autonomous loop with real-time TUI
func Run(maxIterations int) error {
	projectDir, err := project.GetProjectDir()
	if err != nil {
		return err
	}

	// Check if project is initialized
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return fmt.Errorf("project not initialized. Run 'ralph init' first")
	}

	// Check if PRD exists
	if !prd.Exists(projectDir) {
		return fmt.Errorf("no prd.json found. Run 'ralph prd' first")
	}

	workingDir, _ := os.Getwd()

	// Check for branch change and auto-archive
	if err := checkAndArchiveOnBranchChange(projectDir); err != nil {
		return err
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	state := &runState{cancel: cancel}

	// Initialize model
	vp := viewport.New(80, 20)
	vp.SetContent("")

	m := runModel{
		viewport:      vp,
		progress:      progress.New(progress.WithDefaultGradient(), progress.WithWidth(30), progress.WithoutPercentage()),
		content:       &strings.Builder{},
		maxIterations: maxIterations,
		projectDir:    projectDir,
		workingDir:    workingDir,
		running:       true,
		state:         state,
	}

	// Load initial PRD state (existence already validated above)
	prdData, _ := prd.Load(projectDir)
	if prdData != nil {
		m.branch = prdData.BranchName
		m.completed = prdData.CompletedCount()
		m.total = prdData.TotalCount()
		if next := prdData.NextIncomplete(); next != nil {
			m.currentStory = next.ID
		}
	}

	// Run in alternate screen
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Start the iteration loop in background
	go runIterationLoop(ctx, p, state, projectDir, workingDir, maxIterations)

	_, err = p.Run()

	// Ensure cleanup on exit
	cancel()

	return err
}

func runIterationLoop(ctx context.Context, p *tea.Program, state *runState, projectDir, workingDir string, maxIterations int) {
	// Get ralph home from config
	ralphHome, err := config.GetRalphHome()
	if err != nil {
		p.Send(runDoneMsg{err: fmt.Errorf("failed to get ralph home: %w", err)})
		return
	}

	promptPath := filepath.Join(ralphHome, "prompt.md")
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		p.Send(runDoneMsg{err: fmt.Errorf("prompt.md not found at %s", promptPath)})
		return
	}

	for i := 0; i < maxIterations; i++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read and substitute prompt
		promptContent, err := os.ReadFile(promptPath)
		if err != nil {
			p.Send(runDoneMsg{err: err})
			return
		}

		prompt := string(promptContent)
		prompt = strings.ReplaceAll(prompt, "{{PROJECT_DIR}}", projectDir)
		prompt = strings.ReplaceAll(prompt, "{{WORKING_DIR}}", workingDir)

		// Run claude with context - pipe prompt via stdin with streaming output
		cmd := exec.CommandContext(ctx, "claude", "--dangerously-skip-permissions", "-p", "--output-format", "stream-json")
		cmd.Dir = workingDir

		// Set up stdin pipe for the prompt
		stdin, err := cmd.StdinPipe()
		if err != nil {
			p.Send(runDoneMsg{err: err})
			return
		}

		// Store the command so it can be killed
		state.setCmd(cmd)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			p.Send(runDoneMsg{err: err})
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			p.Send(runDoneMsg{err: err})
			return
		}

		if err := cmd.Start(); err != nil {
			p.Send(runDoneMsg{err: err})
			return
		}

		// Write prompt to stdin and close
		go func() {
			defer stdin.Close()
			io.WriteString(stdin, prompt)
		}()

		// Stream output
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			streamOutput(p, stdout)
		}()
		go func() {
			defer wg.Done()
			streamOutput(p, stderr)
		}()

		// Wait for output streams to close
		wg.Wait()

		err = cmd.Wait()
		state.setCmd(nil)

		// Check if cancelled
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err != nil {
			// Don't report error if we were cancelled
			if ctx.Err() == nil {
				p.Send(outputMsg(fmt.Sprintf("Command error: %v", err)))
			}
		}

		// Check for completion signal
		complete := checkForCompletion(projectDir)
		p.Send(iterationCompleteMsg{success: complete})

		if complete {
			// Reload PRD to check if all done
			if prd.Exists(projectDir) {
				prdData, _ := prd.Load(projectDir)
				if prdData != nil && prdData.IsComplete() {
					p.Send(outputMsg("All stories complete!"))
					p.Send(runDoneMsg{success: true})
					return
				}
			}
		}

		// Sleep with context check
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(IterationSleepSecs) * time.Second):
		}
	}

	p.Send(runDoneMsg{success: false, err: fmt.Errorf("max iterations reached")})
}

func streamOutput(p *tea.Program, r io.Reader) {
	scanner := bufio.NewScanner(r)
	parser := stream.NewParser()
	for scanner.Scan() {
		result := parser.ParseLine(scanner.Text())
		if !result.IsEmpty && result.Display != "" {
			p.Send(outputMsg(result.Display))
		}
	}
}

func checkForCompletion(projectDir string) bool {
	// Check if any story was marked as complete in prd.json
	if prd.Exists(projectDir) {
		p, _ := prd.Load(projectDir)
		if p != nil {
			return p.CompletedCount() > 0
		}
	}
	return false
}

func checkAndArchiveOnBranchChange(projectDir string) error {
	lastBranchPath := filepath.Join(projectDir, ".last-branch")

	// Read current branch from prd.json
	var currentBranch string
	if prd.Exists(projectDir) {
		p, err := prd.Load(projectDir)
		if err == nil && p != nil {
			currentBranch = p.BranchName
		}
	}

	if currentBranch == "" {
		return nil
	}

	// Read last branch
	lastBranchData, err := os.ReadFile(lastBranchPath)
	if err != nil {
		// No last branch - just save current
		return os.WriteFile(lastBranchPath, []byte(currentBranch), 0644)
	}

	lastBranch := strings.TrimSpace(string(lastBranchData))
	if lastBranch != currentBranch {
		// Branch changed - auto-archive
		fmt.Printf("Branch changed from %s to %s. Auto-archiving...\n", lastBranch, currentBranch)
		if err := Archive(); err != nil {
			return fmt.Errorf("auto-archive failed: %w", err)
		}
	}

	// Update last branch
	return os.WriteFile(lastBranchPath, []byte(currentBranch), 0644)
}
