package main

import (
	"fmt"
	"os"
	"strings"

	askcmd "aiw/internal/commands/ask"
	completioncmd "aiw/internal/commands/completion"
	help "aiw/internal/commands/help"
	initcmd "aiw/internal/commands/initcmd"
	"aiw/internal/version"

	plug "aiw/internal/plugin"
)

const (
	openspecDir       = "openspec"
	changesDir        = "openspec/changes"
	specsDir          = "openspec/specs"
	archiveDir        = "openspec/changes/archive"
	worktreeDir       = ".wt"
	gitignoreFile     = ".gitignore"
	agentTemplatesDir = "agent-templates"
	agentsFile        = "AGENTS.md"
	copilotFile       = ".github/copilot-instructions.md"
)

func main() {
	if len(os.Args) < 2 {
		help.Dispatch([]string{})
		return
	}
	var err error
	switch os.Args[1] {
	case "-h", "--help":
		err = help.Dispatch([]string{})
	case "-v", "--version", "version":
		fmt.Println("aiw " + version.Label())
	case "help":
		err = help.Dispatch(os.Args[2:])
	case "init":
		err = initcmd.Dispatch(os.Args[2:])
	// FD worktree commands are provided by the aiw-git subcommand dispatcher
	// as `aiw git wt`; do not dispatch standalone `wt` as a built-in command.
	case "completion":
		err = completioncmd.Dispatch(os.Args[2:])
	case "ask":
		err = askcmd.Dispatch(os.Args[2:])
	case "issue":
		// The existing req plugin remains the compatibility implementation.
		code, pluginErr := dispatchPlugin("req", os.Args[2:])
		if pluginErr != nil {
			err = pluginErr
		} else if code != 0 {
			os.Exit(code)
		}
	default:
		// try plugin fallback: aiw-<subcommand>
		pluginName := os.Args[1]
		code, pluginErr := dispatchPlugin(pluginName, os.Args[2:])
		if pluginErr != nil {
			fmt.Println(pluginErr)
			help.Dispatch([]string{})
			err = pluginErr
		} else {
			if code != 0 {
				os.Exit(code)
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func dispatchPlugin(pluginName string, args []string) (int, error) {
	bin, err := plug.DiscoverPlugin(pluginName)
	if err != nil {
		return 0, fmt.Errorf("plugin discovery error: %w", err)
	}
	env := plug.InvocationEnvironment(pluginName, bin, strings.Join(append([]string{pluginName}, args...), " "))
	code, err := plug.ExecPlugin(bin, args, env)
	if err != nil {
		return 0, fmt.Errorf("plugin execution error: %w", err)
	}
	return code, nil
}

func requireArgs(n int, syntax string) {
	if len(os.Args) < n {
		fmt.Println("usage:", syntax)
		os.Exit(1)
	}
}
