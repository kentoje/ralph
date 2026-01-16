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
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kento/ralph/internal/config"
	"github.com/kento/ralph/internal/prd"
	"github.com/kento/ralph/internal/project"
	"github.com/kento/ralph/internal/stream"
	"github.com/kento/ralph/internal/ui/format"
	"github.com/kento/ralph/internal/ui/styles"
)

// Fun rotating labels for the spinner
var funLabels = []string{
	"Mijotage de neurones...",
	"Wait, I'm cooking.",
	"One sec, cooking!",
	"Calcul en cours, darling.",
	"Thinking cap: ON.",
	"Petit moment de magie.",
	"Running on croissants.",
	"Je pense, therefore... wait.",
	"Searching with baguette.",
	"Vite, vite, vite...",
	"Mode génie activé.",
	"Fast as a TGV.",
	"Freshly baked logic...",
	"C'est presque prêt, promis.",
	"Concentration maximale !",
	"Small brain, big effort.",
	"Juste pour toi...",
	"Brainstorming intense...",
	"Eiffel Tower logic loading...",
	"Fais-moi confiance, I'm fast.",
	"Baguette power: activated.",
	"Counting snails... kidding!",
	"C'est du gâteau, wait.",
	"Chef de l'IA cooking.",
	"Vitesse de la lumière, presque.",
	"T'inquiète, I got this.",
	"Fait avec amour, et code.",
	"Adding more butter...",
	"Mon cerveau chauffe !",
	"Ready in a flash.",
	"CPU au max, monsieur.",
}

// Colors for animated spinner label (light purple shades)
var spinnerColors = []lipgloss.Color{
	lipgloss.Color("#7C3AED"), // Primary purple
	lipgloss.Color("#8B5CF6"), // Lighter purple
	lipgloss.Color("#A78BFA"), // Secondary purple
	lipgloss.Color("#C4B5FD"), // Even lighter purple
}

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
	viewport          viewport.Model
	progress          progress.Model
	spinner           spinner.Model
	content           *strings.Builder
	labelIndex        int
	colorFrame        int
	iteration         int
	maxIterations     int
	currentStory      string
	currentStoryTitle string
	branch            string
	completed         int
	total             int
	running           bool
	done              bool
	err               error
	width             int
	height            int
	projectDir        string
	workingDir        string
	state             *runState
	claudeLabelShown  bool
	disableAnimations bool // For testing: disables spinner and animated label
}

type outputMsg struct {
	result stream.ParseResult
}
type promptMsg struct {
	content string
}
type iterationCompleteMsg struct {
	success bool
}
type runDoneMsg struct {
	success bool
	err     error
}

func (m runModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m runModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 0
		footerHeight := 8
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - headerHeight - footerHeight - 4
		m.viewport.SetContent(m.padContentToBottom(m.content.String()))
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

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.colorFrame++
		return m, cmd

	case promptMsg:
		// Display the prompt with "Ralph →" header
		line := format.FormatPrompt(msg.content)
		m.content.WriteString(line + "\n\n")
		m.viewport.SetContent(m.padContentToBottom(m.content.String()))
		m.viewport.GotoBottom()
		// Reset Claude label for new prompt
		m.claudeLabelShown = false

	case outputMsg:
		result := msg.result

		// Add "Claude →" header before first output
		if !m.claudeLabelShown {
			m.content.WriteString(format.FormatClaudeHeader() + "\n")
			m.claudeLabelShown = true
		}

		// Rotate label on tool calls
		if result.Type == stream.OutputToolCall {
			m.labelIndex = (m.labelIndex + 1) % len(funLabels)
		}

		// Format the output based on type
		var line string
		switch result.Type {
		case stream.OutputToolCall:
			line = format.FormatToolCall(result.ToolName, result.Context)
		case stream.OutputResult:
			line = format.FormatDone(result.Display)
		case stream.OutputError:
			line = format.FormatError(result.Display)
		default:
			line = result.Display
		}

		if line != "" {
			m.content.WriteString(line + "\n")
			m.viewport.SetContent(m.padContentToBottom(m.content.String()))
			m.viewport.GotoBottom()
		}

	case iterationCompleteMsg:
		if msg.success {
			m.completed++
			// Reload PRD and update current story to show the next incomplete one
			if prdData, err := prd.Load(m.projectDir); err == nil && prdData != nil {
				if next := prdData.NextIncomplete(); next != nil {
					m.currentStory = next.ID
					m.currentStoryTitle = next.Title
				} else {
					// All stories complete
					m.currentStory = ""
					m.currentStoryTitle = ""
				}
			}
		}
		m.iteration++

		// Add iteration separator
		if m.iteration < m.maxIterations && m.completed < m.total {
			separator := format.FormatSection(fmt.Sprintf("Iteration %d", m.iteration+1), m.width-4)
			m.content.WriteString("\n" + separator + "\n\n")
			m.viewport.SetContent(m.padContentToBottom(m.content.String()))
			m.viewport.GotoBottom()
		}

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

	// Build viewport content with optional loading text at bottom
	content := m.content.String()
	if m.running && !m.disableAnimations {
		label := funLabels[m.labelIndex]
		loadingText := m.spinner.View() + " " + m.renderAnimatedLabel(label)
		content += loadingText
	}

	// Create a temporary viewport with the content including loading text
	tempViewport := m.viewport
	tempViewport.SetContent(m.padContentToBottom(content))

	// Viewport with output (at top)
	b.WriteString(tempViewport.View() + "\n")

	// Separator line between logs and status
	separator := styles.Subtle.Render(strings.Repeat("─", m.width))
	b.WriteString(separator + "\n\n")

	// Title (without spinner - it's now in viewport)
	title := styles.Title.Render(fmt.Sprintf("Ralph - Iteration %d/%d", m.iteration+1, m.maxIterations))
	b.WriteString(title + "\n\n")

	// Progress bar
	progressBar := m.renderProgressBar()
	b.WriteString(progressBar + "\n\n")

	// Info section with aligned labels
	if m.branch != "" {
		b.WriteString(styles.Muted.Render(fmt.Sprintf("%-8s", "Branch")) + m.branch + "\n")
	}
	if m.currentStory != "" {
		storyInfo := m.currentStory
		if m.currentStoryTitle != "" {
			storyInfo += " - " + m.currentStoryTitle
		}
		b.WriteString(styles.Muted.Render(fmt.Sprintf("%-8s", "Story")) + storyInfo + "\n")
	}
	b.WriteString("\n")

	// Help
	help := styles.Subtle.Render("q quit • ↑/↓ scroll")
	b.WriteString(help)

	return b.String()
}

func (m runModel) renderProgressBar() string {
	if m.total == 0 {
		return styles.Muted.Render("Progress: No stories loaded")
	}

	percent := float64(m.completed) / float64(m.total)
	bar := m.progress.ViewAs(percent)
	return fmt.Sprintf("%s %d/%d stories complete", bar, m.completed, m.total)
}

func (m runModel) renderAnimatedLabel(label string) string {
	var result strings.Builder
	runes := []rune(label)
	labelLen := len(runes)

	if labelLen == 0 {
		return ""
	}

	// Wave head position moves right to left
	// As colorFrame increases, position decreases (wrapping around)
	waveHeadPos := (labelLen - 1) - (m.colorFrame % labelLen)

	for i, char := range runes {
		// Calculate distance from wave head (considering wrap-around)
		// Positive = behind the wave (already passed), Negative = ahead of wave
		dist := waveDistance(i, waveHeadPos, labelLen)

		var colorIdx int
		if dist >= 0 && dist < 3 {
			// Inside the 3-char wave head - brightest
			colorIdx = len(spinnerColors) - 1
		} else if dist >= 3 {
			// Trailing behind the wave - fade gradient
			fadeSteps := dist - 3
			colorIdx = len(spinnerColors) - 1 - (fadeSteps / 2)
			if colorIdx < 0 {
				colorIdx = 0
			}
		} else {
			// Ahead of wave - base color
			colorIdx = 0
		}

		style := lipgloss.NewStyle().Foreground(spinnerColors[colorIdx])
		result.WriteString(style.Render(string(char)))
	}
	return result.String()
}

// waveDistance calculates how far behind the wave head a position is
// Returns: 0-2 = in wave head, 3+ = trailing behind, negative = ahead
func waveDistance(pos, waveHeadPos, labelLen int) int {
	diff := pos - waveHeadPos
	if diff < 0 {
		diff += labelLen
	}
	return diff
}

func (m runModel) padContentToBottom(content string) string {
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)
	if contentHeight >= m.viewport.Height {
		return content
	}
	padding := strings.Repeat("\n", m.viewport.Height-contentHeight)
	return padding + content
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

	// Initialize spinner with slower animation
	s := spinner.New()
	s.Spinner = spinner.Spinner{
		Frames: []string{"·", "✻", "✽", "✶", "✳", "✢"},
		FPS:    time.Second / 7,
	}
	s.Style = styles.SpinnerStyle

	m := runModel{
		viewport:      vp,
		progress:      progress.New(progress.WithDefaultGradient(), progress.WithWidth(30), progress.WithoutPercentage()),
		spinner:       s,
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
			m.currentStoryTitle = next.Title
		}
	}

	// Run in alternate screen
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Start the iteration loop in background
	go runIterationLoop(ctx, p, state, projectDir, workingDir, maxIterations)

	finalModel, err := p.Run()

	// Ensure cleanup on exit
	cancel()

	// Check if run completed successfully (all tasks done)
	var runSuccess bool
	if fm, ok := finalModel.(runModel); ok {
		runSuccess = fm.completed >= fm.total && fm.total > 0 && fm.err == nil
	}

	// Persist log before exiting
	if m, ok := finalModel.(runModel); ok && m.content != nil {
		logContent := m.content.String()
		if logContent != "" {
			branchName := m.branch
			if branchName == "" {
				branchName = "unknown"
			}
			if logErr := saveRunLog(projectDir, branchName, logContent); logErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save run log: %v\n", logErr)
			}
		}
	}

	// Auto-archive on successful completion
	if runSuccess {
		fmt.Println()
		if archiveErr := Archive(); archiveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-archive failed: %v\n", archiveErr)
		}
	}

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

		// Send prompt to TUI for display
		p.Send(promptMsg{content: prompt})

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
				p.Send(outputMsg{result: stream.ParseResult{
					Display: fmt.Sprintf("Command error: %v", err),
					Type:    stream.OutputError,
				}})
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
					p.Send(outputMsg{result: stream.ParseResult{
						Display: "All stories complete!",
						Type:    stream.OutputResult,
					}})
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
		if !result.IsEmpty {
			p.Send(outputMsg{result: result})
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

// TestRunOptions configures a runModel for testing with deterministic state.
type TestRunOptions struct {
	DisableAnimations bool // Disables spinner and animated label entirely
	Iteration         int
	MaxIterations     int
	Completed         int
	Total             int
	Branch            string
	CurrentStory      string
	Running           bool
	Width             int
	Height            int
}

// NewRunModelForTest creates a runModel for snapshot testing.
// Returns tea.Model interface to keep runModel unexported.
func NewRunModelForTest(opts TestRunOptions) tea.Model {
	vp := viewport.New(opts.Width, opts.Height-12) // Account for header/footer
	vp.SetContent("")

	s := spinner.New()
	s.Spinner = spinner.Spinner{
		Frames: []string{"·", "✻", "✽", "✶", "✳", "✢"},
		FPS:    time.Second / 7,
	}
	s.Style = styles.SpinnerStyle

	return runModel{
		viewport:          vp,
		progress:          progress.New(progress.WithDefaultGradient(), progress.WithWidth(30), progress.WithoutPercentage()),
		spinner:           s,
		content:           &strings.Builder{},
		iteration:         opts.Iteration,
		maxIterations:     opts.MaxIterations,
		currentStory:      opts.CurrentStory,
		branch:            opts.Branch,
		completed:         opts.Completed,
		total:             opts.Total,
		running:           opts.Running,
		width:             opts.Width,
		height:            opts.Height,
		disableAnimations: opts.DisableAnimations,
	}
}

func saveRunLog(projectDir, branchName, content string) error {
	// Format: feature-name_2026-01-11-15-04-05.log
	date := time.Now().Format("2006-01-02-15-04-05")
	logPath := filepath.Join(projectDir, "logs", fmt.Sprintf("%s_%s.log", branchName, date))

	// Create all parent directories (handles branch names with slashes like "ralph/feature")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(logPath, []byte(content), 0644)
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
