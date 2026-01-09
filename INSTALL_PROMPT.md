# Ralph Installation Prompt

Use this prompt to instruct an LLM to install Ralph on a new machine.

---

## Prompt

```
I want you to install the Ralph autonomous agent system on my machine. Ralph is a Go-based tool that runs Claude Code in a loop to implement PRD stories autonomously.

Before we start, I need to know:

1. Where is the Ralph source code located?
   (The directory containing go.mod, cmd/, internal/, etc.)

2. Where should the `ralph` binary be installed?
   Examples: ~/.local/bin/, /usr/local/bin/

3. Where are your Claude Code skills located?
   Example: ~/.claude/skills/

Based on my answers, please:

1. Build the Ralph binary from source
2. Move it to my PATH
3. Run `ralph setup` to configure RALPH_HOME
4. Copy the skills from the `skills/` directory to my Claude skills location

## Build Steps

```bash
# Navigate to Ralph source
cd <ralph-source-dir>

# Build the binary
go build -o ralph ./cmd/ralph

# Move to PATH
mv ralph <install-dir>/ralph

# Run first-time setup
ralph setup
```

## Skills to Copy

```
<ralph-source>/skills/
├── prd/SKILL.md      # Copy to: <skills-dir>/prd/SKILL.md
└── ralph/SKILL.md    # Copy to: <skills-dir>/ralph/SKILL.md
```

## After Installation

Verify with:
1. `ralph help` - Should show available commands
2. `cd /path/to/any/project`
3. `ralph init`
4. `ralph status`

## Optional: Shell Alias

If you want to use the old `cl-ralph` command name:

**Bash/Zsh:**
```bash
echo 'alias cl-ralph="ralph"' >> ~/.bashrc  # or ~/.zshrc
```

**Fish:**
```fish
alias cl-ralph='ralph' --save
```
```

---

## Quick Install (Copy-Paste)

For users who know their paths:

```bash
# Set your paths
RALPH_SRC="$HOME/.config/claude/scripts/ralph"
INSTALL_DIR="$HOME/.local/bin"
SKILLS_DIR="$HOME/.claude/skills"

# Build and install
cd "$RALPH_SRC"
go build -o ralph ./cmd/ralph
mv ralph "$INSTALL_DIR/"

# Setup (interactive - configures RALPH_HOME)
ralph setup

# Install skills
mkdir -p "$SKILLS_DIR/prd" "$SKILLS_DIR/ralph"
cp skills/prd/SKILL.md "$SKILLS_DIR/prd/"
cp skills/ralph/SKILL.md "$SKILLS_DIR/ralph/"

# Verify
ralph help
```

---

## Commands Reference

| Command | Description |
|---------|-------------|
| `ralph` | Interactive command picker (TUI with hjkl/arrows) |
| `ralph help` | Show help text |
| `ralph setup` | Configure RALPH_HOME path |
| `ralph init` | Initialize Ralph for current project |
| `ralph run [n]` | Run autonomous loop (default: 10 iterations) |
| `ralph status` | Show PRD progress |
| `ralph prd` | Launch Claude for PRD creation |
| `ralph list` | List all projects |
| `ralph archive` | Archive current run |
| `ralph clean` | Remove current project data |

---

## Configuration

After running `ralph setup`, config is stored at:

```
~/.config/ralph/config.json
{
  "ralph_home": "/path/to/ralph/projects"
}
```

Project data is stored in `<ralph_home>/projects/<project-id>/`.
