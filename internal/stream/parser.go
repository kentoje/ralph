package stream

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	streamingjson "github.com/karminski/streaming-json-go"
)

// Parser handles parsing of stream-json output from Claude
type Parser struct {
	lexer *streamingjson.Lexer
}

// NewParser creates a new stream parser
func NewParser() *Parser {
	return &Parser{
		lexer: streamingjson.NewLexer(),
	}
}

// ParseResult holds the formatted output from parsing
type ParseResult struct {
	Display string // Formatted string for display
	IsEmpty bool   // True if nothing to display
}

// ParseLine parses a JSON line and returns formatted output
func (p *Parser) ParseLine(line string) ParseResult {
	line = strings.TrimSpace(line)
	if line == "" {
		return ParseResult{IsEmpty: true}
	}

	// Use streaming-json-go to complete potentially incomplete JSON
	p.lexer = streamingjson.NewLexer() // Reset lexer for each line
	p.lexer.AppendString(line)
	completedJSON := p.lexer.CompleteJSON()

	// First, determine the event type
	var event StreamEvent
	if err := json.Unmarshal([]byte(completedJSON), &event); err != nil {
		// If we can't parse at all, return raw line
		return ParseResult{Display: line}
	}

	switch event.Type {
	case "assistant":
		return p.parseAssistant(completedJSON)
	case "user":
		// User messages are typically tool results - skip verbose output
		return ParseResult{IsEmpty: true}
	case "system":
		// Skip system messages (init, hooks, etc.)
		return ParseResult{IsEmpty: true}
	case "result":
		return p.parseResult(completedJSON)
	default:
		return ParseResult{IsEmpty: true}
	}
}

func (p *Parser) parseAssistant(jsonStr string) ParseResult {
	var event AssistantEvent
	if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
		return ParseResult{IsEmpty: true}
	}

	var outputs []string

	for _, block := range event.Message.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				outputs = append(outputs, block.Text)
			}
		case "tool_use":
			context := extractToolContext(block.Name, block.Input)
			if context != "" {
				outputs = append(outputs, fmt.Sprintf("[%s] %s", block.Name, context))
			} else {
				outputs = append(outputs, fmt.Sprintf("[%s]", block.Name))
			}
		}
	}

	if len(outputs) == 0 {
		return ParseResult{IsEmpty: true}
	}

	return ParseResult{Display: strings.Join(outputs, "\n")}
}

func (p *Parser) parseResult(jsonStr string) ParseResult {
	var result ResultEvent
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return ParseResult{IsEmpty: true}
	}

	status := result.Subtype
	if status == "" {
		status = "complete"
	}

	durationSec := float64(result.DurationMs) / 1000.0

	var parts []string
	parts = append(parts, fmt.Sprintf("[Done] %s", capitalize(status)))

	if result.DurationMs > 0 {
		parts = append(parts, fmt.Sprintf("in %.1fs", durationSec))
	}
	if result.NumTurns > 0 {
		parts = append(parts, fmt.Sprintf("(%d turns)", result.NumTurns))
	}

	return ParseResult{Display: strings.Join(parts, " ")}
}

// extractToolContext extracts a brief context string from tool input
func extractToolContext(toolName string, input map[string]interface{}) string {
	if input == nil {
		return ""
	}

	switch toolName {
	case "Read":
		if path, ok := input["file_path"].(string); ok {
			return shortenPath(path)
		}
	case "Write":
		if path, ok := input["file_path"].(string); ok {
			return shortenPath(path)
		}
	case "Edit":
		if path, ok := input["file_path"].(string); ok {
			return shortenPath(path)
		}
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			return truncate(cmd, 50)
		}
	case "Glob":
		if pattern, ok := input["pattern"].(string); ok {
			return pattern
		}
	case "Grep":
		if pattern, ok := input["pattern"].(string); ok {
			return truncate(pattern, 40)
		}
	case "Task":
		if desc, ok := input["description"].(string); ok {
			return truncate(desc, 40)
		}
	case "TodoWrite":
		return "updating tasks"
	case "Skill":
		if skill, ok := input["skill"].(string); ok {
			return skill
		}
	case "WebFetch":
		if url, ok := input["url"].(string); ok {
			return truncate(url, 50)
		}
	case "WebSearch":
		if query, ok := input["query"].(string); ok {
			return truncate(query, 40)
		}
	}

	// Generic fallback: try common field names
	for _, key := range []string{"file_path", "path", "command", "query", "pattern", "description", "skill"} {
		if val, ok := input[key].(string); ok && val != "" {
			return truncate(val, 40)
		}
	}

	return ""
}

// shortenPath returns the base filename or last path component
func shortenPath(path string) string {
	return filepath.Base(path)
}

// truncate shortens a string to max length with ellipsis
func truncate(s string, max int) string {
	// Remove newlines for display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)

	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
