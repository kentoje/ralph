package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kento/ralph/internal/config"
	"github.com/kento/ralph/internal/ui/format"
	"github.com/kento/ralph/internal/ui/styles"
)

type logItem struct {
	path        string
	displayName string
	project     string
	date        string
}

func (i logItem) FilterValue() string { return i.displayName }

type logItemDelegate struct{}

func (d logItemDelegate) Height() int                             { return 1 }
func (d logItemDelegate) Spacing() int                            { return 0 }
func (d logItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d logItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(logItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%-12s %s", i.date, i.displayName)

	fn := pickerItemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return pickerSelectedStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type logsModel struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m logsModel) Init() tea.Cmd {
	return nil
}

func (m logsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(logItem)
			if ok {
				m.choice = i.path
			}
			return m, tea.Quit

		case "j", "down":
			m.list.CursorDown()
			return m, nil
		case "k", "up":
			m.list.CursorUp()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m logsModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(pickerTitleStyle.Render("Ralph Logs"))
	b.WriteString("\n\n")
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(pickerHelpStyle.Render("↑/↓ navigate • enter view • q quit"))
	return b.String()
}

// Logs shows a picker of available logs and displays the selected one
func Logs() error {
	ralphHome, err := config.GetRalphHome()
	if err != nil {
		return err
	}

	// Find all log files
	logs, err := findAllLogs(ralphHome)
	if err != nil {
		return err
	}

	if len(logs) == 0 {
		fmt.Println(styles.Muted.Render("No logs found."))
		fmt.Println(format.FormatNextStep("ralph run", "to create some logs"))
		return nil
	}

	// Convert to list items
	items := make([]list.Item, len(logs))
	for i, log := range logs {
		items[i] = log
	}

	// Calculate list height (max 15, min needed for items)
	listHeight := len(logs) + 2
	if listHeight > 15 {
		listHeight = 15
	}

	l := list.New(items, logItemDelegate{}, 60, listHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false)

	m := logsModel{list: l}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running log picker: %w", err)
	}

	fm := finalModel.(logsModel)
	if fm.quitting || fm.choice == "" {
		return nil
	}

	// Display the selected log using cat (renders ANSI codes)
	fmt.Println()
	cmd := exec.Command("cat", fm.choice)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// findAllLogs scans all projects for log files
func findAllLogs(ralphHome string) ([]logItem, error) {
	projectsDir := filepath.Join(ralphHome, "projects")

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var logs []logItem

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectID := entry.Name()
		logsDir := filepath.Join(projectsDir, projectID, "logs")

		// Walk the logs directory recursively (handles nested branch names)
		filepath.WalkDir(logsDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".log") {
				return nil
			}

			// Extract display name from path
			relPath, _ := filepath.Rel(logsDir, path)
			displayName := strings.TrimSuffix(relPath, ".log")

			// Extract date from filename (format: name_2026-01-11.log)
			date := ""
			parts := strings.Split(filepath.Base(path), "_")
			if len(parts) >= 2 {
				date = strings.TrimSuffix(parts[len(parts)-1], ".log")
			}

			// Get project display name
			projectDisplay := extractShortProjectName(ralphHome, projectID)

			logs = append(logs, logItem{
				path:        path,
				displayName: fmt.Sprintf("%s/%s", projectDisplay, displayName),
				project:     projectID,
				date:        date,
			})

			return nil
		})
	}

	// Sort by date (newest first)
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].date > logs[j].date
	})

	return logs, nil
}

// extractShortProjectName returns a short display name for the project
func extractShortProjectName(ralphHome, projectID string) string {
	projectDir := filepath.Join(ralphHome, "projects", projectID)
	pathFile := filepath.Join(projectDir, ".path")

	if data, err := os.ReadFile(pathFile); err == nil {
		fullPath := strings.TrimSpace(string(data))
		return filepath.Base(fullPath)
	}

	// Fallback: last segment of project ID
	if len(projectID) > 20 {
		return "..." + projectID[len(projectID)-17:]
	}
	return projectID
}
