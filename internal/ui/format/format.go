package format

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kento/ralph/internal/ui/styles"
)

// FormatToolCall formats a tool invocation with pending icon
func FormatToolCall(name, context string) string {
	icon := styles.Muted.Render(styles.ToolPending)
	toolName := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Bold(true).
		Render(name)

	if context != "" {
		return fmt.Sprintf("%s %s %s", icon, toolName, styles.Muted.Render(context))
	}
	return fmt.Sprintf("%s %s", icon, toolName)
}

// FormatError formats an error message with a prominent label
func FormatError(msg string) string {
	label := lipgloss.NewStyle().
		Background(styles.Error).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Render("ERROR")

	return label + " " + styles.ErrorText.Render(msg)
}

// FormatSection formats a section header with a line
func FormatSection(title string, width int) string {
	titleRendered := styles.Title.Render(title)
	titleWidth := lipgloss.Width(titleRendered)

	// Calculate remaining space for the line
	lineWidth := max(width-titleWidth-3, 4)

	line := styles.Subtle.Render(strings.Repeat("─", lineWidth))
	prefix := styles.Subtle.Render("──")

	return fmt.Sprintf("%s %s %s", prefix, titleRendered, line)
}

// FormatDone formats a completion message
func FormatDone(msg string) string {
	return styles.SuccessText.Render(styles.CheckIcon + " " + msg)
}

// FormatHeader renders a styled command header
func FormatHeader(title string) string {
	return styles.Title.Render(title)
}

// FormatSuccess renders a success message with ✓ icon
func FormatSuccess(msg string) string {
	return styles.SuccessText.Render(styles.CheckIcon + " " + msg)
}

// FormatWarning renders a warning message with ⚠ icon
func FormatWarning(msg string) string {
	return styles.WarningText.Render(styles.WarningIcon + " " + msg)
}

// FormatNextStep renders a next step hint with → arrow
func FormatNextStep(cmd, description string) string {
	arrow := styles.Muted.Render(styles.Arrow)
	cmdStyled := styles.Title.Render(cmd)
	if description != "" {
		return fmt.Sprintf("  %s %s %s", arrow, cmdStyled, styles.Muted.Render(description))
	}
	return fmt.Sprintf("  %s %s", arrow, cmdStyled)
}

// FormatKeyValue renders a key-value pair with aligned formatting
func FormatKeyValue(key, value string) string {
	keyStyled := styles.Muted.Render(key)
	return fmt.Sprintf("%s  %s", keyStyled, value)
}

// FormatBullet renders a bullet point item
func FormatBullet(item string) string {
	return fmt.Sprintf("  %s %s", styles.Muted.Render(styles.Bullet), item)
}

// FormatPrompt formats the prompt sent to Claude with "Ralph →" header
// Shows first 5 lines with a "[+N lines]" indicator if truncated
func FormatPrompt(prompt string) string {
	header := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Render("Ralph " + styles.Arrow)

	lines := strings.Split(prompt, "\n")
	const maxLines = 5

	var content string
	if len(lines) <= maxLines {
		content = styles.Muted.Render(prompt)
	} else {
		preview := strings.Join(lines[:maxLines], "\n")
		content = styles.Muted.Render(preview)
		remaining := len(lines) - maxLines
		content += "\n" + styles.Subtle.Render(fmt.Sprintf("[+%d lines]", remaining))
	}

	return header + "\n" + content
}

// FormatClaudeHeader formats the "Claude →" header for Claude's responses
func FormatClaudeHeader() string {
	return lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Bold(true).
		Render("Claude " + styles.Arrow)
}
