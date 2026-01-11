# UI Improvements Spec

Consistent styling and improved UX across all Ralph CLI commands.

## Design System

### Colors (`internal/ui/styles/styles.go`)

| Color | Hex | Usage |
|-------|-----|-------|
| Primary | `#7C3AED` | Titles, selected items, brand |
| Secondary | `#A78BFA` | Tool names, accents |
| Success | `#10B981` | Checkmarks, success messages |
| Error | `#EF4444` | Errors, failures |
| Warning | `#F59E0B` | Warnings, destructive actions |
| FgMuted | `#9CA3AF` | Secondary text |
| FgSubtle | `#6B7280` | Help text, hints |
| Border | `#374151` | Borders, dividers |

### Icons

| Icon | Symbol | Usage |
|------|--------|-------|
| CheckIcon | ✓ | Success, completion |
| ErrorIcon | × | Failure, error |
| WarningIcon | ⚠ | Destructive actions |
| Arrow | → | Next steps |
| Bullet | • | List items |
| ToolPending | ● | Tool in progress |

## Format Helpers (`internal/ui/format/format.go`)

```go
FormatHeader(title)           // Styled title
FormatSuccess(msg)            // ✓ Success message
FormatWarning(msg)            // ⚠ Warning message
FormatError(msg)              // ERROR label + message
FormatNextStep(cmd, desc)     // → cmd description
FormatKeyValue(key, value)    // Key  Value
FormatBullet(item)            // • Item
FormatSection(title, width)   // ── Title ────────
FormatToolCall(name, ctx)     // ● ToolName context
FormatDone(msg)               // ✓ msg
FormatPrompt(prompt)          // Ralph → + truncated prompt (5 lines)
FormatClaudeHeader()          // Claude → header
```

## Command Output Patterns

### Success Pattern
```
✓ Action completed

Key    Value
Key    Value

Next steps:
  → command description
```

### Warning/Confirmation Pattern
```
⚠ This will do something dangerous:

  • item 1
  • item 2

This cannot be undone.
Continue? [y/N]
```

### Status Pattern
```
Header Title

Key      Value
Key      Value
Progress ████████░░░░░░░░

→ next action
```

## Files Modified

| File | Changes |
|------|---------|
| `internal/ui/styles/styles.go` | Added WarningIcon, Bullet |
| `internal/ui/format/format.go` | Added FormatHeader, FormatSuccess, FormatWarning, FormatNextStep, FormatKeyValue, FormatBullet |
| `internal/commands/picker.go` | Styled header, help footer, semantic colors |
| `internal/commands/list.go` | Semantic colors, empty state guidance |
| `internal/commands/commands.go` | Restyled init, status, archive, clean, setup |
| `internal/commands/run.go` | Spinner with fun labels, styled output |
| `cmd/ralph/main.go` | Styled error messages |

## Run Command Special Features

### Window Layout

Chat-like UI with log viewport at top (no border), separator line, then status info below:

```
Ralph →
You are an autonomous coding agent...
Read the PRD at /path/to/prd.json
...
[+127 lines]

Claude →
● Read prd.json
⠋ Vite, vite, vite...

────────────────────────────────────────

Ralph - Iteration 1/25

████████░░░░░░░░░░░░░░░░░░ 0/4 stories

Branch  ralph/spinning-button
Story   US-001

q quit • ↑/↓ scroll
```

### Chat-Like Interface

- **Ralph →** (Primary purple, bold): Shows the prompt sent to Claude
- **Claude →** (Secondary purple, bold): Shows Claude's responses
- Prompt is truncated to first 5 lines with `[+N lines]` indicator
- Claude header appears before first tool call output

### Bottom-Aligned Log Content

Log content appears at bottom of viewport, growing upward. Empty space fills the top when content is short. Implemented via `padContentToBottom()` helper.

### Animated Purple Wave Labels

Loading labels feature a rolling color wave animation using purple shades:

```go
var spinnerColors = []lipgloss.Color{
    "#7C3AED", // Primary purple
    "#8B5CF6", // Lighter purple
    "#A78BFA", // Secondary purple
    "#C4B5FD", // Even lighter purple
}
```

Each character gets a color based on `(position + frame) % 4`. Colors shift on every spinner tick (7 FPS), creating a wave effect.

### Spinner Configuration

- Uses `MiniDot` spinner (`⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏`)
- Slowed to 7 FPS for smoother animation
- Loading text appears inside viewport (at bottom)

### Rotating Label Text

Fun French/English labels rotate each time a new tool is invoked:
- Mijotage de neurones...
- Wait, I'm cooking.
- One sec, cooking!
- Calcul en cours, darling.
- Thinking cap: ON.
- Petit moment de magie.
- Running on croissants.
- Je pense, therefore... wait.
- Searching with baguette.
- Vite, vite, vite...
- Mode génie activé.
- Fast as a TGV.
- Freshly baked logic...
- C'est presque prêt, promis.
- Concentration maximale !
- Small brain, big effort.
- Juste pour toi...
- Brainstorming intense...
- Eiffel Tower logic loading...
- Fais-moi confiance, I'm fast.

### Status Section Layout

Aligned key-value format with spacing:
- Title (iteration count)
- Progress bar with story count
- Branch label (8-char aligned)
- Story label (8-char aligned)
- Help text

### Tool Output Formatting
- `● Read main.go` - Tool in progress
- `✓ Done in 14.3s (3 turns)` - Completion
- `── Iteration 2 ────────` - Section dividers
