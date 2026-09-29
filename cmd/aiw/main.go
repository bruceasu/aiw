package main

import (
	"fmt"
	"os"
	"strings"

	askcmd "aiw/internal/commands/ask"
	completioncmd "aiw/internal/commands/completion"
	help "aiw/internal/commands/help"
	sessioncmd "aiw/internal/commands/session"
	taskcmd "aiw/internal/commands/task"
	taskadapter "aiw/internal/task/workflowadapter"
	"aiw/internal/version"
	workflowcli "aiw/internal/workflow/cli"
	workflowcmd "aiw/internal/workflow/facade"

	plug "aiw/internal/plugin"
)

const (
	openspecDir   = "openspec"
	changesDir    = "openspec/changes"
	specsDir      = "openspec/specs"
	archiveDir    = "openspec/changes/archive"
	worktreeDir   = ".wt"
	gitignoreFile = ".gitignore"
	promptsDir    = "docs/agent-templates"
	agentsFile    = "AGENTS.md"
	codexFile     = "CODEX.md"
	copilotFile   = ".github/copilot-instructions.md"
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
		err = taskcmd.DispatchTopLevel("init", os.Args[2:])
	case "new":
		err = taskcmd.DispatchTopLevel("new", os.Args[2:])
	case "list":
		err = taskcmd.DispatchTopLevel("list", os.Args[2:])
	case "show":
		err = taskcmd.DispatchTopLevel("show", os.Args[2:])
	case "status":
		err = taskcmd.DispatchTopLevel("status", os.Args[2:])
	case "done":
		err = taskcmd.DispatchTopLevel("done", os.Args[2:])
	case "archive":
		err = taskcmd.DispatchTopLevel("archive", os.Args[2:])
	// `wt` is implemented as an external plugin (aiw-wt.py) and will be
	// handled by the plugin fallback below. Do not dispatch a built-in handler.
	case "context":
		err = taskcmd.DispatchTopLevel("context", os.Args[2:])
	case "decision":
		err = taskcmd.DispatchTopLevel("decision", os.Args[2:])
	case "spec":
		err = taskcmd.DispatchTopLevel("spec", os.Args[2:])
	case "prompts":
		err = taskcmd.DispatchTopLevel("prompts", os.Args[2:])
	case "workspace":
		err = taskcmd.DispatchTopLevel(os.Args[1], os.Args[2:])
	case "completion":
		err = completioncmd.Dispatch(os.Args[2:])
	case "ask":
		err = askcmd.Dispatch(os.Args[2:])
	case "task":
		if len(os.Args) < 3 {
			err = taskcmd.DispatchTopLevel("help", nil)
		} else {
			err = taskcmd.DispatchTopLevel(os.Args[2], os.Args[3:])
		}
	case "session":
		err = sessioncmd.Dispatch(os.Args[2:])
	case "wf":
		facade := workflowcmd.New(
			func(args []string) error {
				return workflowcli.RunWorkflowCommand(taskadapter.New(taskadapter.DefaultOperations{}), args)
			},
			workflowcli.PrintWorkflowHelp,
		)
		err = facade.Dispatch(os.Args[2:])
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
