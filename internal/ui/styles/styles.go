package styles

import "github.com/charmbracelet/lipgloss"

// Semantic color palette
var (
	// Brand
	Primary   = lipgloss.Color("#7C3AED") // Purple
	Secondary = lipgloss.Color("#A78BFA") // Light purple

	// Status
	Success = lipgloss.Color("#10B981") // Green
	Error   = lipgloss.Color("#EF4444") // Red
	Warning = lipgloss.Color("#F59E0B") // Amber

	// Text hierarchy
	FgBase   = lipgloss.Color("#E5E7EB") // Primary text
	FgMuted  = lipgloss.Color("#9CA3AF") // Secondary text
	FgSubtle = lipgloss.Color("#6B7280") // Tertiary text

	// Backgrounds/Borders
	Border = lipgloss.Color("#374151") // Gray border
)

// Pre-built styles
var (
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary)

	Muted = lipgloss.NewStyle().
		Foreground(FgMuted)

	Subtle = lipgloss.NewStyle().
		Foreground(FgSubtle)

	SuccessText = lipgloss.NewStyle().
			Foreground(Success)

	ErrorText = lipgloss.NewStyle().
			Foreground(Error)

	WarningText = lipgloss.NewStyle().
			Foreground(Warning)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(Primary)
)

// Icons
const (
	CheckIcon   = "✓"
	ErrorIcon   = "×"
	WarningIcon = "⚠"
	ToolPending = "●"
	Arrow       = "→"
	Bullet      = "•"
)
