package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/kento/ralph/internal/commands"
	"github.com/kento/ralph/internal/ui/format"
	"github.com/kento/ralph/internal/ui/styles"
)

func main() {
	args := os.Args[1:]

	// No args - show interactive command picker
	if len(args) == 0 {
		if err := commands.RunInteractivePicker(); err != nil {
			fmt.Fprintln(os.Stderr, format.FormatError(err.Error()))
			os.Exit(1)
		}
		return
	}

	cmd := args[0]
	cmdArgs := args[1:]

	var err error
	switch cmd {
	case "help", "--help", "-h":
		fmt.Print(commands.HelpText)
	case "setup":
		err = commands.Setup()
	case "home":
		err = commands.Home()
	case "project-dir":
		err = commands.ProjectDir()
	case "init":
		err = commands.Init()
	case "run":
		maxIterations := commands.DefaultMaxIterations
		if len(cmdArgs) > 0 {
			if n, parseErr := strconv.Atoi(cmdArgs[0]); parseErr == nil {
				maxIterations = n
			}
		}
		err = commands.Run(maxIterations)
	case "status":
		err = commands.Status()
	case "prd":
		err = commands.Prd()
	case "list":
		err = commands.List()
	case "logs":
		err = commands.Logs()
	case "archive":
		err = commands.Archive()
	case "clean":
		all := len(cmdArgs) > 0 && (cmdArgs[0] == "--all" || cmdArgs[0] == "-a")
		err = commands.Clean(all)
	default:
		fmt.Fprintln(os.Stderr, styles.ErrorText.Render(fmt.Sprintf("Unknown command: %s", cmd)))
		fmt.Println()
		fmt.Println(styles.Muted.Render("Run 'ralph' for interactive mode or 'ralph help' for usage."))
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, format.FormatError(err.Error()))
		os.Exit(1)
	}
}
