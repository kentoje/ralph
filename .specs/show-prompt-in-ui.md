# Show Prompt Text in UI

## Problem

The Ralph TUI only shows filtered output from Claude (tool calls, results, errors). The actual prompt being sent to Claude is hidden, making it difficult to:

- Debug issues with prompt substitution
- Understand what instructions Claude received
- Verify the prompt template is being loaded correctly

## Proposed Solution

Display the prompt text in the log window at the start of each iteration, before Claude's output begins.

## Requirements

- Show the full prompt text at the beginning of each iteration
- Visually distinguish the prompt from Claude's output (different styling/color)
- Include a clear header (e.g., "Prompt sent to Claude:")
- Show the **substituted** prompt (with actual paths, not placeholders like `{{PROJECT_DIR}}`)
