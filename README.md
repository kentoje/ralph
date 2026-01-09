# Ralph - Autonomous AI Agent Loop

Based on [snarktank/ralph](https://github.com/snarktank/ralph?tab=readme-ov-file)

Ralph is an autonomous agent system that runs Claude Code in a loop to implement PRD (Product Requirements Document) stories one at a time.

## How It Works

1. You write a PRD describing your feature
2. Ralph converts it to a structured `prd.json` with small, implementable stories
3. Ralph runs Claude Code repeatedly, each iteration:
   - Picks the next incomplete story
   - Implements it
   - Runs quality checks
   - Commits if passing
   - Moves to the next story
4. Loop ends when all stories are complete

**Key Innovation:** Each iteration gets fresh context, preventing the AI from getting lost in large implementations. Memory persists via git history, progress.txt, and prd.json.

## Features

- **Interactive TUI** - Run `ralph` with no args for a command picker with hjkl/arrow navigation
- **Real-time Output** - Stream-JSON parsing shows tool usage and progress:
  ```
  [Read] config.go
  [Bash] go build ./...
  [Edit] main.go
  Implementing the feature...
  [Done] Success in 14.3s (3 turns)
  ```
- **Process Management** - Press `q` to cleanly kill running Claude subprocess
- **Cross-platform** - Single Go binary, no shell dependencies

## Repository Structure

```
ralph/
├── cmd/ralph/main.go         # Entry point
├── internal/
│   ├── commands/             # CLI commands (run, status, list, etc.)
│   ├── config/               # Config management
│   ├── prd/                  # PRD JSON parsing
│   ├── project/              # Project directory management
│   └── stream/               # Stream-JSON parser
├── prompt.md                 # Instructions for each Claude iteration
├── skills/                   # Claude Code skills
│   ├── prd/SKILL.md          # PRD generator skill
│   └── ralph/SKILL.md        # PRD-to-JSON converter skill
├── go.mod
└── README.md
```

## Installation

### Prerequisites

- Go 1.21+
- Claude Code CLI (`claude`)

### Build

```bash
# Clone or download this repository
cd ralph

# Build the binary
go build -o ralph ./cmd/ralph

# Move to a directory in your PATH
mv ralph ~/.local/bin/
# or
sudo mv ralph /usr/local/bin/
```

### First-time Setup

```bash
# Run setup to configure RALPH_HOME (where project data is stored)
ralph setup
```

### Install Skills (Optional but Recommended)

Copy the skills to your Claude Code skills directory:

```bash
SKILLS_DIR="$HOME/.claude/skills"
mkdir -p "$SKILLS_DIR/prd" "$SKILLS_DIR/ralph"
cp skills/prd/SKILL.md "$SKILLS_DIR/prd/"
cp skills/ralph/SKILL.md "$SKILLS_DIR/ralph/"
```

### Shell Alias (Optional)

Add to your shell config for the old `cl-ralph` command:

```bash
# ~/.bashrc, ~/.zshrc, or ~/.config/fish/config.fish
alias cl-ralph='ralph'
```

## Usage

### Workflow

```bash
# 1. Navigate to your project
cd /path/to/your/project

# 2. Initialize Ralph for this project
ralph init

# 3. Create a PRD using the /prd skill
# In Claude Code, run: /prd
# Answer the clarifying questions

# 4. Convert PRD to JSON using the /ralph skill
# In Claude Code, run: /ralph

# 5. Run the autonomous loop
ralph run        # Default: 10 iterations
ralph run 5      # Or specify max iterations

# 6. Monitor progress
ralph status

# 7. When done with a feature
ralph archive
```

### Commands

| Command | Description |
|---------|-------------|
| `ralph` | Interactive command picker (TUI) |
| `ralph help` | Show help text |
| `ralph setup` | Configure RALPH_HOME path |
| `ralph init` | Initialize Ralph for current project |
| `ralph run [n]` | Run autonomous loop (default: 10 iterations) |
| `ralph status` | Show PRD progress |
| `ralph prd` | Launch Claude for PRD creation |
| `ralph list` | List all projects with archive counts |
| `ralph archive` | Archive current run |
| `ralph clean` | Remove current project data |
| `ralph clean --all` | Remove all Ralph data |

### Run Command TUI

During `ralph run`, you'll see:
- Current iteration and story
- Real-time formatted output (tool names, assistant text)
- Progress bar showing completed stories
- Press `q` to quit and kill the Claude process

## Configuration

Config is stored at `~/.config/ralph/config.json`:

```json
{
  "ralph_home": "/path/to/ralph/projects"
}
```

## Project Data Structure

Ralph stores all data in RALPH_HOME, keeping your projects clean:

```
<ralph-home>/projects/<project-id>/
├── .path           # Original project path (for display)
├── prd.md          # Human-readable PRD
├── prd.json        # Machine-readable PRD with story status
├── progress.txt    # Learnings log
├── .last-branch    # Branch tracking
└── archive/        # Previous PRD runs
```

**Project IDs** are derived from absolute paths:
`/Users/me/code/myapp` → `users-me-code-myapp`

## Writing Good PRDs

### Story Size (Critical)

Each story must be completable in ONE iteration. If a story is too big, Claude runs out of context before finishing.

**Good sizes:**
- Add a database column and migration
- Add a UI component to an existing page
- Update a server action with new logic

**Too big (split these):**
- "Build the entire dashboard"
- "Add authentication"
- "Refactor the API"

**Rule of thumb:** If you can't describe the change in 2-3 sentences, split it.

### Story Ordering

Stories execute in priority order. Dependencies must come first:

1. Database/schema changes
2. Backend logic
3. UI components
4. Dashboard/summary views

### Acceptance Criteria

Must be verifiable, not vague:

- "Add status column with default 'pending'"
- "Clicking delete shows confirmation dialog"
- "Typecheck passes"
- ~~"Works correctly"~~
- ~~"Good UX"~~

## How Memory Works

Between iterations, Ralph remembers via:

- **Git history** - Every commit is a checkpoint
- **prd.json** - Tracks which stories are complete (`passes: true/false`)
- **progress.txt** - Consolidated learnings and patterns

## Dependencies

Go modules (automatically fetched):
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - TUI components
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/karminski/streaming-json-go` - Stream JSON parsing

## License

MIT
