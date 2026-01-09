package stream

// StreamEvent is the base type for determining event type
type StreamEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`
}

// SystemEvent represents system messages (init, hook_response, etc.)
type SystemEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// AssistantEvent represents assistant messages with nested content
type AssistantEvent struct {
	Type    string  `json:"type"`
	Message Message `json:"message"`
}

// UserEvent represents user messages (including tool results)
type UserEvent struct {
	Type    string  `json:"type"`
	Message Message `json:"message"`
}

// Message holds the actual message content
type Message struct {
	Role    string         `json:"role,omitempty"`
	Content []ContentBlock `json:"content,omitempty"`
}

// ContentBlock represents a content item (text, tool_use, or tool_result)
type ContentBlock struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
	Content   string                 `json:"content,omitempty"`
}

// ResultEvent represents the final completion event
type ResultEvent struct {
	Type       string  `json:"type"`
	Subtype    string  `json:"subtype,omitempty"`
	IsError    bool    `json:"is_error,omitempty"`
	DurationMs int     `json:"duration_ms,omitempty"`
	NumTurns   int     `json:"num_turns,omitempty"`
	Result     string  `json:"result,omitempty"`
	SessionID  string  `json:"session_id,omitempty"`
	CostUSD    float64 `json:"total_cost_usd,omitempty"`
}
