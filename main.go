package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	completioncmd "aiw/internal/commands/completion"
	askcmd "aiw/internal/commands/ask"
	czcmd "aiw/internal/commands/cz"
	help "aiw/internal/commands/help"
	sessioncmd "aiw/internal/commands/session"
	taskcmd "aiw/internal/commands/task"

	plug "aiw/internal/plugin"
)

const (
	openspecDir   = "openspec"
	changesDir    = "openspec/changes"
	specsDir      = "openspec/specs"
	archiveDir    = "openspec/archive"
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
	case "requirement":
		err = taskcmd.DispatchTopLevel("requirement", os.Args[2:])
	case "prompts":
		err = taskcmd.DispatchTopLevel("prompts", os.Args[2:])
	case "turn", "chat":
		err = taskcmd.DispatchTopLevel(os.Args[1], os.Args[2:])
	case "workflow", "workspace":
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
	case "cz":
		err = czcmd.Dispatch(os.Args[2:])
	default:
		// try plugin fallback: aiw-<subcommand>
		pluginName := os.Args[1]
		bin, err := plug.DiscoverPlugin(pluginName)
		if err != nil {
			fmt.Println("plugin discovery error:", err)
			help.Dispatch([]string{})
		} else {
			// prepare env
			env := map[string]string{
				"AIW_PLUGIN_NAME": pluginName,
				"AIW_PLUGIN_PATH": bin,
				"AIW_CMDLINE":     strings.Join(os.Args[1:], " "),
			}
			if home := os.Getenv("HOME"); home != "" {
				env["AIW_HOME"] = home
			} else if uhome := os.Getenv("USERPROFILE"); uhome != "" {
				env["AIW_HOME"] = uhome
			}
			// pass through additional aiw-related roots
			if exe, err := os.Executable(); err == nil {
				env["AIW_ROOT"] = filepath.Dir(exe)
			}

			code, err := plug.ExecPlugin(bin, os.Args[2:], env)
			if err != nil {
				fmt.Fprintln(os.Stderr, "plugin execution error:", err)
				os.Exit(1)
			}
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

func requireArgs(n int, syntax string) {
	if len(os.Args) < n {
		fmt.Println("usage:", syntax)
		os.Exit(1)
	}
}
