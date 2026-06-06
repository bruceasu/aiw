package main

import (
	"fmt"
	"os"
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
		opts, parseErr := parseInitOptions(os.Args[2:])
		if parseErr != nil {
			err = parseErr
			break
		}
		err = initWorkspace(opts)
	case "new":
		requireArgs(3, "new <task-id>")
		err = newTask(os.Args[2])
	case "list":
		err = listTasks()
	case "show":
		requireArgs(3, "show <task-id>")
		err = showTask(os.Args[2])
	case "status":
		requireArgs(4, "status <task-id> <status>")
		err = updateStatus(os.Args[2], os.Args[3])
	case "done":
		requireArgs(3, "done <task-id>")
		err = updateStatus(os.Args[2], "DONE")
	case "archive":
		requireArgs(3, "archive <task-id> [--push] [--cleanup-wt] [--delete-branch]")
		opts, parseErr := parseArchiveOptions(os.Args[3:])
		if parseErr != nil {
			err = parseErr
			break
		}
		err = archiveTask(os.Args[2], opts)
	case "wt":
		err = dispatchWt(os.Args[2:])
	case "context":
		requireArgs(3, "context <task-id>")
		err = printContext(os.Args[2])
	case "decision":
		requireArgs(3, "decision <task-id>")
		err = createDecision(os.Args[2])
	case "spec":
		requireArgs(3, "spec <spec-id>")
		err = createSpec(os.Args[2])
	case "registry":
		err = writeRegistry()
	case "prompts":
		opts, parseErr := parsePromptOptions(os.Args[2:])
		if parseErr != nil {
			err = parseErr
			break
		}
		err = syncPrompts(opts)
	case "git":
		err = dispatchGit(os.Args[2:])
	default:
		usage()
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
`)
}
func requireArgs(n int, syntax string) {
	if len(os.Args) < n {
		fmt.Println("usage:", syntax)
		os.Exit(1)
	}
}
