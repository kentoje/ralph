# Enhance CLI

Let's rewrite and remove .sh files, .fish files.

As of right we use /Volumes/HomeX/kento/dotfiles/.config/claude/scripts/ralph/ralph.sh as the main script to run in "ralph" mode.
We also have fish a script in /Volumes/HomeX/kento/dotfiles/.config/fish/functions/cl-ralph.fish .

Let's create a brand new CLI which will encapsulate all the current behaviours, and enhance the user experience. We want feature parity.

To create the CLI, we will use Golang with

- https://github.com/charmbracelet/bubbles (use ref or use exa mcps, to get docs)

## Additional features

let's add:

- a setup command to set the path where we want to store our projects, and other mandatory steps
- a list command to check all projects that are registered with informations about `archive`

## Improvements

As of right now when we run the `run` command, we do not have much logs about what the agent is currently doing. We need more informations about what is the current step being implemented, what is the current command being ran.

## How to test this spec?

To test this spec, let's use /Volumes/HomeX/kento/Documents/lab/react-router-test
