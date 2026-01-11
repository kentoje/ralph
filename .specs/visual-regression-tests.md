# Visual Regression Tests

## Overview

Added snapshot/golden file testing for Ralph CLI TUI components using `github.com/charmbracelet/x/exp/teatest`.

## Changes Made

### New Files Created

```
internal/commands/ui_test.go           # TUI snapshot tests
internal/commands/testdata/*.golden    # 6 golden files for visual regression
.claude/skills/test/SKILL.md           # Skill to run tests
```

### Files Modified

```
go.mod                                 # Added teatest dependency
internal/commands/picker.go            # Added NewPickerModel() constructor
internal/commands/run.go               # Added TestRunOptions, NewRunModelForTest(), disableAnimations flag
```

## Test Coverage

### Picker TUI (3 tests)

| Test | Description |
|------|-------------|
| `TestPickerInitialView` | Initial picker state with first item selected |
| `TestPickerNavigateDown` | Navigate down once with `j` key |
| `TestPickerNavigateMultiple` | Navigate down twice |

### Run TUI (3 tests)

| Test | Description |
|------|-------------|
| `TestRunUIAt50Percent` | Progress bar at 50%, shows branch and story |
| `TestRunUIComplete` | All stories complete (100%) |
| `TestRunUINoStories` | No stories loaded state |

## Technical Details

### Deterministic Environment

- Uses `lipgloss.SetColorProfile(termenv.Ascii)` to disable ANSI colors
- Fixed terminal size (100x40) via `teatest.WithInitialTermSize()`
- `disableAnimations` flag on `runModel` to freeze spinner and animated labels

### Key Design Decisions

1. **DisableAnimations flag** - Completely disables spinner and animated label in test mode for deterministic output
2. **ASCII color profile** - Strips ANSI color codes for consistent snapshots across systems
3. **Exported constructors** - `NewPickerModel()` and `NewRunModelForTest()` return `tea.Model` interface to keep internal types unexported

### TestRunOptions

```go
type TestRunOptions struct {
    DisableAnimations bool
    Iteration         int
    MaxIterations     int
    Completed         int
    Total             int
    Branch            string
    CurrentStory      string
    Running           bool
    Width             int
    Height            int
}
```

## Usage

```bash
# Run all tests
go test ./internal/commands/... -v

# Update golden files after intentional UI changes
go test ./internal/commands/... -update

# Run tests multiple times to verify stability
for i in 1 2 3; do go test ./internal/commands/... -count=1; done
```

## Golden File Management

Golden files are stored in `internal/commands/testdata/`:
- `TestPickerInitialView.golden`
- `TestPickerNavigateDown.golden`
- `TestPickerNavigateMultiple.golden`
- `TestRunUIAt50Percent.golden`
- `TestRunUIComplete.golden`
- `TestRunUINoStories.golden`

When UI changes are intentional, regenerate with `-update` flag.
