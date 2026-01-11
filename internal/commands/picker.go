package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kento/ralph/internal/ui/styles"
)

var (
	pickerTitleStyle    = lipgloss.NewStyle().MarginLeft(2).Bold(true).Foreground(styles.Primary)
	pickerItemStyle     = lipgloss.NewStyle().PaddingLeft(4)
	pickerSelectedStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(styles.Primary)
	pickerHelpStyle     = lipgloss.NewStyle().PaddingLeft(4).PaddingBottom(1).Foreground(styles.FgSubtle)
)

// commandItems returns the list of available commands for the picker
func commandItems() []list.Item {
	return []list.Item{
		commandItem{name: "run", description: "Run autonomous loop"},
		commandItem{name: "init", description: "Initialize project"},
		commandItem{name: "status", description: "Show project status"},
		commandItem{name: "prd", description: "Create/edit PRD"},
		commandItem{name: "list", description: "List all projects"},
		commandItem{name: "logs", description: "View run logs"},
		commandItem{name: "archive", description: "Archive current run"},
		commandItem{name: "clean", description: "Remove project data"},
		commandItem{name: "setup", description: "Configure RALPH_HOME"},
		commandItem{name: "help", description: "Show help"},
	}
}

type commandItem struct {
	name        string
	description string
}

func (i commandItem) FilterValue() string { return i.name }

type commandItemDelegate struct{}

func (d commandItemDelegate) Height() int                             { return 1 }
func (d commandItemDelegate) Spacing() int                            { return 0 }
func (d commandItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d commandItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(commandItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%-10s %s", i.name, i.description)

	fn := pickerItemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return pickerSelectedStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type pickerModel struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m pickerModel) Init() tea.Cmd {
	return nil
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			i, ok := m.list.SelectedItem().(commandItem)
			if ok {
				m.choice = i.name
			}
			return m, tea.Quit

		// vim-style navigation
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

func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(pickerTitleStyle.Render("Ralph Commands"))
	b.WriteString("\n\n")
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(pickerHelpStyle.Render("↑/↓ navigate • enter select • q quit"))
	return b.String()
}

func RunInteractivePicker() error {
	const listHeight = 14

	l := list.New(commandItems(), commandItemDelegate{}, 40, listHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false)

	m := pickerModel{list: l}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running picker: %w", err)
	}

	fm := finalModel.(pickerModel)
	if fm.quitting || fm.choice == "" {
		return nil
	}

	// Execute the selected command
	fmt.Printf("\n")
	return executeCommand(fm.choice)
}

func executeCommand(cmd string) error {
	switch cmd {
	case "run":
		return Run(DefaultMaxIterations)
	case "init":
		return Init()
	case "status":
		return Status()
	case "prd":
		return Prd()
	case "list":
		return List()
	case "logs":
		return Logs()
	case "archive":
		return Archive()
	case "clean":
		return Clean(false)
	case "setup":
		return Setup()
	case "help":
		printHelpFromPicker()
		return nil
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func printHelpFromPicker() {
	fmt.Print(HelpText)
}

// NewPickerModel creates a new picker model for testing purposes.
// Returns tea.Model interface to keep pickerModel unexported.
func NewPickerModel() tea.Model {
	const listHeight = 14

	l := list.New(commandItems(), commandItemDelegate{}, 40, listHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false)

	return pickerModel{list: l}
}
