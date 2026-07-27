package task

import "fmt"

const (
	promptsDir  = "docs/agent-templates"
	agentsFile  = "AGENTS.md"
	codexFile   = "CODEX.md"
	copilotFile = ".github/copilot-instructions.md"
)

func DispatchTopLevel(name string, args []string) error {
	if name == "new" || name == "decision" || name == "spec" || name == "archive" {
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
	case "init":
		opts, err := parseInitOptions(args)
		if err != nil {
			return err
		}
		return initWorkspace(opts)
	case "new":
		if len(args) != 1 {
			return fmt.Errorf("usage: new <task-id>")
		}
		return newTask(args[0])
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
			return fmt.Errorf("usage: archive <task-id> [--push] [--cleanup-wt] [--delete-branch]")
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
	case "registry":
		return writeRegistry()
	case "prompts":
		opts, err := parsePromptOptions(args)
		if err != nil {
			return err
		}
		return syncPrompts(opts)
	case "agent":
		return runTaskAgent(args)
	default:
		return fmt.Errorf("unknown task command: %s", name)
	}
}
