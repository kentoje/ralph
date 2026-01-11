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

Log viewport at top, all status info below:

```
╭───────────────────────────────────╮
│                                   │  ← Empty space (content bottom-aligned)
│                                   │
│ ● Read main.go                    │
│ ✓ Done                            │
╰───────────────────────────────────╯

Ralph - Iteration 1/25  ⠋ Cooking...

████████░░░░░░░░░░░░░░░░░░ 0/4 stories

Branch  ralph/spinning-button
Story   US-001

q quit • ↑/↓ scroll
```

### Bottom-Aligned Log Content

Log content appears at bottom of viewport, growing upward. Empty space fills the top when content is short. Implemented via `padContentToBottom()` helper.

### Animated Rainbow Labels

Loading labels feature a rolling color wave animation:

```go
var waveColors = []lipgloss.Color{
    "#7C3AED", // Purple
    "#3B82F6", // Blue
    "#10B981", // Green
    "#F59E0B", // Yellow
    "#EF4444", // Red
}
```

Each character gets a color based on `(position + frame) % 5`. Colors shift left on every spinner tick, creating a wave effect.

### Rotating Label Text

Fun labels rotate each time a new tool is invoked:
- Cooking...
- Brewing...
- Conjuring...
- Summoning...
- Crunching...
- Pondering...
- Tinkering...
- Wrangling...
- Scheming...
- Vibing...

### Status Section Layout

Aligned key-value format with spacing:
- Title + spinner + animated label
- Progress bar with story count
- Branch label (8-char aligned)
- Story label (8-char aligned)
- Help text

### Tool Output Formatting
- `● Read main.go` - Tool in progress
- `✓ Done in 14.3s (3 turns)` - Completion
- `── Iteration 2 ────────` - Section dividers
