package main

import (
	czcmd "aiw/internal/commands/cz"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	help "aiw/internal/commands/help"
	taskcmd "aiw/internal/commands/task"

	plug "aiw/internal/plugin"
)

const (
	openspecDir   = "openspec"
	changesDir    = "openspec/changes"
	specsDir      = "openspec/specs"
	archiveDir    = "openspec/archive"
	registryFile  = "openspec/registry.json"
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
	case "registry":
		err = taskcmd.DispatchTopLevel("registry", os.Args[2:])
	case "prompts":
		err = taskcmd.DispatchTopLevel("prompts", os.Args[2:])
	case "cz":
		err = czcmd.Dispatch(os.Args[2:])
	default:
		// try plugin fallback: aiw-<subcommand>
		pluginName := os.Args[1]
		bin, err := plug.DiscoverPlugin(pluginName)
		if err != nil {
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
