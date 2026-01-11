# Ralph Installation Prompt

Use this prompt to instruct an LLM to install Ralph on a new machine.

---

## Prompt

```
I want you to install the Ralph autonomous agent system on my machine. Ralph is a Go-based tool that runs Claude Code in a loop to implement PRD stories autonomously.

Before we start, I need to know:

1. Where should I clone/place the Ralph repository?
   (This will be RALPH_HOME - contains go.mod, cmd/, internal/, prompt.md, etc.)

2. Where are your Claude Code skills located?
   Example: ~/.claude/skills/

Based on my answers, please:

1. Clone or move Ralph to the chosen location
2. Build the Ralph binary (stays in repo)
3. Run `ralph setup` (press Enter to use current directory)
4. Copy the skills from the `skills/` directory to my Claude skills location
5. Set up a shell function to call ralph from anywhere

## Build Steps

```bash
# Navigate to Ralph repo
cd <ralph-repo-dir>

# Build the binary (stays in repo directory)
go build -o ralph ./cmd/ralph

# Run first-time setup (press Enter to use current directory as RALPH_HOME)
./ralph setup

# Verify
./ralph home  # Should print the ralph home path
```

## Skills to Copy

```
<ralph-repo>/skills/
├── prd/SKILL.md      # Copy to: <skills-dir>/prd/SKILL.md
└── ralph/SKILL.md    # Copy to: <skills-dir>/ralph/SKILL.md
```

## Shell Function Setup

The binary stays in the repo. Set up a shell function to call it from anywhere:

**Fish** (`~/.config/fish/functions/ralph.fish`):
```fish
function ralph -d "Global Ralph autonomous agent for Claude Code"
    set -l CONFIG_FILE "$HOME/.config/ralph/config.json"
    if not test -f "$CONFIG_FILE"
        echo "Error: Ralph not configured. Run 'ralph setup' from the ralph repo first."
        return 1
    end
    set -l RALPH_HOME (cat "$CONFIG_FILE" | jq -r '.ralph_home')
    if test -z "$RALPH_HOME" -o "$RALPH_HOME" = "null"
        echo "Error: ralph_home not set in config."
        return 1
    end
    set -l RALPH_BIN "$RALPH_HOME/ralph"
    if not test -x "$RALPH_BIN"
        echo "Error: ralph binary not found at $RALPH_BIN"
        echo "Build it with: cd $RALPH_HOME && go build -o ralph ./cmd/ralph"
        return 1
    end
    $RALPH_BIN $argv
end
```

**Bash/Zsh** (add to `~/.bashrc` or `~/.zshrc`):
```bash
ralph() {
    local config="$HOME/.config/ralph/config.json"
    if [[ -f "$config" ]]; then
        local home=$(jq -r '.ralph_home' "$config")
        "$home/ralph" "$@"
    else
        echo "Error: Ralph not configured. Run 'ralph setup' first."
    fi
}
```

## After Installation

Verify with:
1. `ralph help` - Should show available commands
2. `ralph home` - Should print the RALPH_HOME path
3. `cd /path/to/any/project`
4. `ralph init`
5. `ralph status`
```

---

## Quick Install (Copy-Paste)

For users who know their paths:

```bash
# Set your paths
RALPH_REPO="/path/to/ralph"  # Where you want the ralph repo
SKILLS_DIR="$HOME/.claude/skills"

# Clone or navigate to Ralph repo
cd "$RALPH_REPO"

# Build (binary stays in repo)
go build -o ralph ./cmd/ralph

# Setup (press Enter to use current directory as RALPH_HOME)
./ralph setup

# Install skills
mkdir -p "$SKILLS_DIR/prd" "$SKILLS_DIR/ralph"
cp skills/prd/SKILL.md "$SKILLS_DIR/prd/"
cp skills/ralph/SKILL.md "$SKILLS_DIR/ralph/"

# Verify
./ralph help
./ralph home
```

---

## Commands Reference

| Command | Description |
|---------|-------------|
| `ralph` | Interactive command picker (TUI with hjkl/arrows) |
| `ralph help` | Show help text |
| `ralph setup` | Configure RALPH_HOME path (defaults to current directory) |
| `ralph home` | Print RALPH_HOME path |
| `ralph init` | Initialize Ralph for current project |
| `ralph run [n]` | Run autonomous loop (default: 10 iterations) |
| `ralph status` | Show PRD progress |
| `ralph prd` | Launch Claude for PRD creation |
| `ralph list` | List all projects |
| `ralph logs` | View run logs (with colors) |
| `ralph archive` | Archive current run |
| `ralph clean` | Remove current project data |

---

## Configuration

After running `ralph setup`, config is stored at:

```
~/.config/ralph/config.json
{
  "ralph_home": "/path/to/ralph/repo"
}
```

`ralph_home` points to the Ralph repository root. Binary is at `$ralph_home/ralph`, project data at `$ralph_home/projects/<project-id>/`.
