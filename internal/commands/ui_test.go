package commands

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
)

// setupTestEnv configures a deterministic environment for snapshot testing.
// Uses ASCII color profile (no ANSI codes) for consistent output across systems.
func setupTestEnv(t *testing.T) {
	t.Helper()
	lipgloss.SetColorProfile(termenv.Ascii)
}

func TestPickerInitialView(t *testing.T) {
	setupTestEnv(t)

	m := NewPickerModel()
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

	// Wait for initial render
	time.Sleep(100 * time.Millisecond)

	// Get output and compare with golden file
	out := readOutput(t, tm)
	teatest.RequireEqualOutput(t, out)

	tm.Quit()
}

func TestPickerNavigateDown(t *testing.T) {
	setupTestEnv(t)

	m := NewPickerModel()
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

	// Navigate down (j key)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// Wait for update
	time.Sleep(100 * time.Millisecond)

	// Get output and compare with golden file
	out := readOutput(t, tm)
	teatest.RequireEqualOutput(t, out)

	tm.Quit()
}

func TestPickerNavigateMultiple(t *testing.T) {
	setupTestEnv(t)

	m := NewPickerModel()
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

	// Navigate down twice
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// Wait for updates
	time.Sleep(100 * time.Millisecond)

	// Get output and compare with golden file
	out := readOutput(t, tm)
	teatest.RequireEqualOutput(t, out)

	tm.Quit()
}

func TestRunUIAt50Percent(t *testing.T) {
	setupTestEnv(t)

	m := NewRunModelForTest(TestRunOptions{
		DisableAnimations: true,
		Iteration:         4,
		MaxIterations:     10,
		Completed:         5,
		Total:             10,
		Branch:            "feature/test-branch",
		CurrentStory:      "STORY-001",
		Running:           true,
		Width:             100,
		Height:            40,
	})

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

	// Send window size to trigger proper render
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 40})

	// Wait for render
	time.Sleep(100 * time.Millisecond)

	// Get output and compare with golden file
	out := readOutput(t, tm)
	teatest.RequireEqualOutput(t, out)

	tm.Quit()
}

func TestRunUIComplete(t *testing.T) {
	setupTestEnv(t)

	m := NewRunModelForTest(TestRunOptions{
		DisableAnimations: true,
		Iteration:         9,
		MaxIterations:     10,
		Completed:         10,
		Total:             10,
		Branch:            "feature/complete",
		CurrentStory:      "",
		Running:           true,
		Width:             100,
		Height:            40,
	})

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

	// Send window size to trigger proper render
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 40})

	// Wait for render
	time.Sleep(100 * time.Millisecond)

	// Get output and compare with golden file
	out := readOutput(t, tm)
	teatest.RequireEqualOutput(t, out)

	tm.Quit()
}

func TestRunUINoStories(t *testing.T) {
	setupTestEnv(t)

	m := NewRunModelForTest(TestRunOptions{
		DisableAnimations: true,
		Iteration:         0,
		MaxIterations:     10,
		Completed:         0,
		Total:             0,
		Branch:            "main",
		CurrentStory:      "",
		Running:           true,
		Width:             100,
		Height:            40,
	})

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 40))

	// Send window size to trigger proper render
	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 40})

	// Wait for render
	time.Sleep(100 * time.Millisecond)

	// Get output and compare with golden file
	out := readOutput(t, tm)
	teatest.RequireEqualOutput(t, out)

	tm.Quit()
}

// readOutput drains the test model output and returns it as bytes.
// Filters out ANSI escape sequences that might slip through.
func readOutput(t *testing.T, tm *teatest.TestModel) []byte {
	t.Helper()

	output := tm.Output()
	var buf bytes.Buffer

	// Read with timeout to avoid hanging
	done := make(chan struct{})
	go func() {
		io.Copy(&buf, output)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		// Timeout is fine, we just want current output
	}

	// Normalize output: trim trailing whitespace and normalize line endings
	result := strings.TrimRight(buf.String(), " \t\n\r")
	return []byte(result)
}
