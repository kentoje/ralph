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
internal/commands/run.go             # Run command with real-time TUI
```

### Files Modified

- `/Volumes/HomeX/kento/dotfiles/.config/fish/functions/cl-ralph.fish` - Updated to call the new `ralph` binary

### Files Deleted

- `ralph.sh` - Replaced by Go binary

## Commands

| Command | Description |
|---------|-------------|
| `ralph` (no args) | Interactive command picker (hjkl/arrows navigation) |
| `ralph help` | Show help text |
| `ralph setup` | Configure RALPH_HOME path |
| `ralph init` | Initialize project (creates .path file for display) |
| `ralph status` | Show project status |
| `ralph prd` | Launch Claude for PRD creation |
| `ralph run [n]` | Run autonomous loop (default: 10 iterations) |
| `ralph list` | List all projects with archive counts |
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
- `github.com/charmbracelet/bubbles` - TUI components (spinner, viewport, list)
- `github.com/charmbracelet/lipgloss` - Styling

### Config Location

```
~/.config/ralph/config.json
{
  "ralph_home": "/path/to/ralph/projects"
}
```

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

## Completed

- [x] Parse stream-json output and display nicely (extract tool names, show progress)
  - Added `internal/stream/` package with types and parser
  - Uses `streaming-json-go` library for robust JSON handling
  - Displays: `[ToolName] context` for tools, plain text for messages, `[Done] status` for results
