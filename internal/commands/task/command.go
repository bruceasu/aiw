package task

import "fmt"

const (
	promptsDir  = "docs/agent-templates"
	agentsFile  = "AGENTS.md"
	codexFile   = "CODEX.md"
	copilotFile = ".github/copilot-instructions.md"
)

func DispatchTopLevel(name string, args []string) error {
	if name == "help" || name == "--help" || name == "-h" {
		printTaskHelp()
		return nil
	}
	if name == "new" || name == "decision" || name == "spec" {
		mode, routedArgs, err := selectBackend(name, args)
		if err != nil {
			return err
		}
		if mode == backendOpenSpec {
			return runOpenSpec(routedArgs[0], name, routedArgs[1:])
		}
		args = routedArgs
	}
	switch name {
	case "workflow":
		return runWorkflowCommand(args)
	case "turn", "chat":
		return runTaskAgent(append([]string{name}, args...))
	case "requirement":
		return DispatchRequirement(args)
	case "init":
		opts, err := parseInitOptions(args)
		if err != nil {
			return err
		}
		return initWorkspace(opts)
	case "new":
		id, allowUnrelatedDirty, err := parseNewArgs(args)
		if err != nil {
			return err
		}
		return newTask(id, allowUnrelatedDirty)
	case "list":
		return listTasks()
	case "show":
		if len(args) != 1 {
			return fmt.Errorf("usage: show <task-id>")
		}
		return showTask(args[0])
	case "status":
		if len(args) != 2 {
			return fmt.Errorf("usage: status <task-id> <status>")
		}
		return updateStatus(args[0], args[1])
	case "done":
		if len(args) != 1 {
			return fmt.Errorf("usage: done <task-id>")
		}
		return updateStatus(args[0], "DONE")
	case "archive":
		if len(args) < 1 {
			return fmt.Errorf("usage: archive <task-id> [--push] [--cleanup-wt] [--delete-branch] [--force]")
		}
		opts, err := parseArchiveOptions(args[1:])
		if err != nil {
			return err
		}
		return archiveTask(args[0], opts)
	case "context":
		if len(args) != 1 {
			return fmt.Errorf("usage: context <task-id>")
		}
		return printContext(args[0])
	case "decision":
		if len(args) != 1 {
			return fmt.Errorf("usage: decision <task-id>")
		}
		return createDecision(args[0])
	case "spec":
		if len(args) != 1 {
			return fmt.Errorf("usage: spec <spec-id>")
		}
		return createSpec(args[0])
	case "prompts":
		opts, err := parsePromptOptions(args)
		if err != nil {
			return err
		}
		return syncPrompts(opts)
	case "workspace":
		return bindTaskWorkspace(args)
	default:
		return fmt.Errorf("unknown task command: %s", name)
	}
}

func printTaskHelp() {
	fmt.Print("aiw task - Task lifecycle and managed execution\n\n" +
		"Usage:\n" +
		"  aiw task <command> [args...]\n" +
		"  aiw task help\n" + "  aiw task --help\n\n" +
		"Task lifecycle:\n" +
		"  new <task-id> [--allow-unrelated-dirty]\n" +
		"  list\n" +
		"  show <task-id>\n" +
		"  status <task-id> <status>\n" +
		"  done <task-id>\n" + "  archive <task-id> [options]\n\n" +
		"Execution:\n" +
		"  aiw turn <task-id> [options]\n" +
		"  aiw chat <task-id> [options]\n" +
		"  workflow --help\n" +
		"  workspace bind <task-id> --primary\n\n" +
		"Examples:\n" +
		"  aiw task new payment-retry\n" +
		"  aiw task workflow run payment-retry --execute\n" + "  aiw task workflow supervise payment-retry start\n\n" + "Run `aiw task <command> --help` for command-specific help.\n")
}

func parseNewArgs(args []string) (string, bool, error) {
	if len(args) < 1 || len(args) > 2 {
		return "", false, fmt.Errorf("usage: new <task-id> [--allow-unrelated-dirty]")
	}
	if len(args) == 2 && args[1] != "--allow-unrelated-dirty" {
		return "", false, fmt.Errorf("unknown new option: %s", args[1])
	}
	return args[0], len(args) == 2, nil
}
