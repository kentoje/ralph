# Go CLI Implementation

## Overview

Rewrote the Ralph CLI from shell scripts (ralph.sh + cl-ralph.fish) to a Go binary using charmbracelet/bubbletea for TUI.

## Changes Made

### New Files Created

```
cmd/ralph/main.go                    # Entry point, command routing
internal/config/config.go            # Config management (~/.config/ralph/config.json)
internal/project/project.go          # Project directory management
internal/prd/prd.go                   # PRD JSON parsing and status
internal/commands/picker.go          # Interactive command picker TUI
internal/commands/commands.go        # Core commands (setup, init, status, prd, archive, clean)
internal/commands/list.go            # List command with table output
internal/commands/logs.go            # Logs viewer with picker TUI
internal/commands/run.go             # Run command with real-time TUI
internal/ui/styles/styles.go         # Semantic colors, icons, pre-built styles
internal/ui/format/format.go         # Reusable formatting helpers
```

### Files Modified

- `~/.config/fish/functions/ralph.fish` - Fish function reads `ralph_home` from config, calls `$ralph_home/ralph`

### Files Deleted

- `ralph.sh` - Replaced by Go binary

## Commands

| Command | Description |
|---------|-------------|
| `ralph` (no args) | Interactive command picker (hjkl/arrows navigation) |
| `ralph help` | Show help text |
| `ralph setup` | Configure RALPH_HOME path (defaults to current directory) |
| `ralph home` | Print RALPH_HOME path |
| `ralph init` | Initialize project (creates .path file for display) |
| `ralph status` | Show project status |
| `ralph prd` | Launch Claude for PRD creation |
| `ralph run [n]` | Run autonomous loop (default: 10 iterations) |
| `ralph list` | List all projects with archive counts |
| `ralph logs` | View run logs (with ANSI colors) |
| `ralph archive` | Archive current run |
| `ralph clean [--all]` | Remove project data |

## Key Features

1. **Interactive Command Picker** - Run `ralph` without args for TUI with hjkl/arrow navigation
2. **Process Management** - Pressing `q` during `run` kills the claude subprocess properly using context cancellation
3. **Backward Compatible** - Works with existing project data structure
4. **Path Display** - Stores original path in `.path` file during init for readable project names in list

## Technical Details

### Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - TUI components (viewport, list, progress, spinner, table)
- `github.com/charmbracelet/lipgloss` - Styling

### Internal UI Packages

- `internal/ui/styles` - Semantic color palette (Primary, Secondary, Success, Error, Warning) and icon constants
- `internal/ui/format` - Formatting helpers for consistent output across all commands

### Config Location

```
~/.config/ralph/config.json
{
  "ralph_home": "/path/to/ralph/repo"
}
```

`ralph_home` points to the Ralph repository root:
- Binary: `$ralph_home/ralph`
- Prompt: `$ralph_home/prompt.md`
- Projects: `$ralph_home/projects/<project-id>/`

### Project Data Structure (unchanged)

```
$RALPH_HOME/projects/<project-id>/
├── .path              # NEW: stores original path for display
├── prd.json
├── prd.md
├── progress.txt
├── .last-branch
└── archive/
```

### Run Command

- Uses `claude --dangerously-skip-permissions -p --output-format stream-json`
- Pipes prompt via stdin
- Streams output to viewport in real-time
- Context-based cancellation for clean process termination
- Chat-like UI with "Ralph →" / "Claude →" headers
- Bottom-aligned viewport (content grows upward)
- Animated spinner with purple wave effect and rotating labels
- Status section showing iteration, branch, and story info

## Completed

- [x] Parse stream-json output and display nicely (extract tool names, show progress)
  - Added `internal/stream/` package with types and parser
  - Uses `streaming-json-go` library for robust JSON handling
  - Displays: `[ToolName] context` for tools, plain text for messages, `[Done] status` for results

- [x] Code simplification and maintainability improvements (2025-01)
  - **Consolidated help text** - Single `HelpText` constant in `commands.go`, used by both main.go and picker.go
  - **Added named constants** - `DefaultMaxIterations`, `IterationSleepSecs`, `MaxNameDisplayLen`, `MinNameDisplayLen`
  - **Simplified parser** - Removed unused lexer field from Parser struct, create lexer locally in ParseLine
  - **Simplified switch statements** - Grouped similar tools (Read/Write/Edit, Glob/Grep) in extractToolContext
  - **Removed redundant code** - PRD existence check, hard-coded paths, unused style variables
  - **Fixed strings.Builder bug** - Changed to pointer `*strings.Builder` to prevent copy-by-value panic in bubbletea

- [x] Progress bar upgrade (2025-01)
  - **Replaced custom progress bar** - Now uses `github.com/charmbracelet/bubbles/progress` component
  - **Added gradient styling** - Uses `progress.WithDefaultGradient()` for purple-to-pink gradient fill
  - **Removed unused code** - Deleted `runProgressStyle`, `lines` field from runModel, and unused append
  - **Static rendering** - Uses `ViewAs(percent)` for simple percentage-based display without animation

- [x] Dynamic path configuration (2025-01)
  - **Added `ralph home` command** - Prints the configured `ralph_home` path
  - **Setup defaults to current directory** - Running `ralph setup` and pressing Enter uses `pwd` as RALPH_HOME
  - **Dynamic prompt.md location** - `run.go` now uses `config.GetRalphHome()` to find `prompt.md`
  - **Updated fish function** - Reads `ralph_home` from config, binary is at `$ralph_home/ralph`
  - **Updated SKILL.md files** - Use `$(ralph home)/projects/...` instead of hardcoded paths

- [x] UI improvements and design system (2025-01)
  - **Design system** - Semantic color palette (Primary `#7C3AED`, Secondary `#A78BFA`, Success, Error, Warning) in `internal/ui/styles/`
  - **Icon constants** - CheckIcon (✓), ErrorIcon (×), WarningIcon (⚠), Arrow (→), Bullet (•), ToolPending (●)
  - **Format helpers** - Reusable functions in `internal/ui/format/` (FormatHeader, FormatSuccess, FormatWarning, FormatNextStep, FormatKeyValue, FormatBullet, FormatSection, FormatToolCall, FormatPrompt, FormatClaudeHeader)
  - **Chat-like run UI** - "Ralph →" and "Claude →" headers, bottom-aligned log content growing upward
  - **Animated spinner** - MiniDot spinner at 7 FPS with purple wave animation (4-color gradient shift per character)
  - **Rotating labels** - 20 fun French/English loading phrases ("Running on croissants", "Vite, vite, vite...")
  - **Styled commands** - All commands (init, status, archive, clean, setup) using new format helpers for consistent output

- [x] Logs viewer and sub-agent visibility (2026-01)
  - **`ralph logs` command** - Interactive picker showing all log files sorted by date, uses `cat` to render ANSI colors
  - **Task tool highlighting** - Sub-agent spawning (Task tool) now displays in orange (`#F97316`) instead of purple for visibility
  - **Log directory fix** - Branch names with slashes (e.g., `ralph/feature`) now create nested log directories correctly

## Code Architecture

### Constants (commands.go)

```go
const (
    DefaultMaxIterations = 10   // Default run iterations
    IterationSleepSecs   = 2    // Sleep between iterations
    MaxNameDisplayLen    = 50   // Max project name length in list
    MinNameDisplayLen    = 20   // Min project name length in list
)
```

### Key Design Decisions

1. **strings.Builder as pointer** - bubbletea passes models by value; `strings.Builder` cannot be copied after use
2. **Lexer created per-line** - No benefit to reusing lexer between lines in stream parser
3. **Help text as constant** - Single source of truth, used by both CLI and interactive picker
4. **Separate command routing** - main.go handles CLI args (flags, numeric args), picker.go uses defaults
