package main

import (
	czcmd "aiw/cmd/cz"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gitcmd "aiw/cmd/git"
	taskcmd "aiw/cmd/task"
	tcccmd "aiw/cmd/tcc"
	wtcmd "aiw/cmd/wt"

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
		usage()
		return
	}
	var err error
	switch os.Args[1] {
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
	case "wt":
		err = wtcmd.Dispatch(os.Args[2:])
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
	case "tcc":
		err = tcccmd.Dispatch(os.Args[2:])
	case "git":
		err = gitcmd.Dispatch(os.Args[2:])
	case "cz":
		err = czcmd.Dispatch(os.Args[2:])
	default:
		// try plugin fallback: aiw-<subcommand>
		pluginName := os.Args[1]
		bin, err := plug.DiscoverPlugin(pluginName)
		if err != nil {
			usage()
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
func usage() {
	fmt.Print(`aiw — OpenSpec-lite workspace CLI

Task management:
  init [--prompts] [--merge] [--force] [--template <name>]
  new <task-id>             Create task folder (task.toml / task.md / notes.md).
  list                      List tasks from openspec/changes.
  show <task-id>            Print task.md.
  status <task-id> <s>      Update task status (auto upper-cased).
  done <task-id>            Shortcut for: status <task-id> DONE.
  archive <task-id> [opts]  Move task to openspec/archive; supports --push / --cleanup-wt.
  context <task-id>         Show files to read before implementing.
  decision <task-id>        Create design.md when design is needed.
  spec <spec-id>            Create long-lived spec under openspec/specs.
  registry                  Rebuild openspec/registry.json.
  prompts [template] [opts] Create or merge AGENTS/CODEX/Copilot prompts.

Worktrees  (run: aiw wt help)
  wt add <task-id> [base]   Create worktree + branch feature/<task-id>.
  wt rm  <task-id> [opts]   Remove worktree.
  wt list / prune / lock / unlock / repair / ignore

Git shortcuts  (run: aiw git help)
  git <subcommand>          30+ shortcuts in groups:
                            Snapshot & Commit · History & Status · Sync & Remote
                            Branch · File · Conflicts · Tags & Export · Recovery · Guides

TCC wrapper  (run: aiw tcc help)
	tcc [args...]              Forward to tcc with auto -I/-L from the TCC root.
	tcc dll [args...]          Shortcut for shared library builds (-shared).
	tcc static [args...]       Shortcut for static linking (-static).
	tcc run [args...]          Shortcut for compile-and-run (-run).
	tcc help                   Show wrapper help and detected default paths.

Examples:
  aiw init --prompts --template go
  aiw new payment-retry
  aiw wt add payment-retry origin/main
  aiw wt lock payment-retry "hotfix in progress"
  aiw status payment-retry IN_PROGRESS
  aiw done payment-retry
  aiw archive payment-retry --finalize
  aiw git save "fix typo"
  aiw git sync
  aiw git help
	aiw tcc hello.c -o hello.exe
	aiw tcc dll hello.c -o hello.dll
	aiw tcc static hello.c -o hello.exe
	aiw tcc run hello.c
`)
}
func requireArgs(n int, syntax string) {
	if len(os.Args) < n {
		fmt.Println("usage:", syntax)
		os.Exit(1)
	}
}
