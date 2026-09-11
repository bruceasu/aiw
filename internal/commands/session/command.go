package sessioncmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	storex "aiw/internal/session"
	"aiw/internal/taskx"
)

// Dispatch exposes persisted Session operations only. Task Workflow and
// Requirement Management own Session creation and AI execution.
func Dispatch(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return nil
	}
	store := storex.NewStore(os.Getenv("AIW_SESSION_ROOT"))
	command, args := args[0], args[1:]
	switch command {
	case "status":
		return status(store, args)
	case "list":
		return list(store)
	case "get":
		return get(store, args)
	case "finish":
		return finish(store, args)
	case "archive":
		if len(args) != 1 {
			return errors.New("usage: aiw session archive <task-id>")
		}
		id, err := taskSessionID(args[0])
		if err != nil { return err }
		return store.Archive(id)
	case "delete":
		if len(args) != 2 || args[1] != "--yes" {
			return errors.New("usage: aiw session delete <task-id> --yes")
		}
		id, err := taskSessionID(args[0])
		if err != nil { return err }
		return store.Delete(id)
	case "memory":
		return memory(store, args)
	case "handoff":
		return handoff(store, args)
	default:
		return fmt.Errorf("unknown session command: %s", command)
	}
}

func printHelp() {
	fmt.Print("aiw session - Persisted AIW Session management\n\nUsage:\n" +
		"  aiw session status <task-id>\n  aiw session list\n" +
		"  aiw session get <task-id>\n  aiw session finish <task-id>\n" +
		"  aiw session archive <task-id>\n  aiw session delete <task-id> --yes\n" +
		"  aiw session memory append <task-id> TEXT\n  aiw session memory show <task-id>\n" +
		"  aiw session handoff <task-id> [focus]\n  aiw session handoff show <task-id>\n")
}

func status(store *storex.Store, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: aiw session status <task-id>")
	}
	id, err := taskSessionID(args[0])
	if err != nil { return err }
	value, err := store.Load(id)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(value, "", "  ")
	fmt.Println(string(b))
	return nil
}

func list(store *storex.Store) error {
	entries, err := os.ReadDir(filepath.Join(store.Root, "sessions"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if value, err := store.Load(entry.Name()); err == nil {
				fmt.Printf("%s\t%s\t%s\n", value.Session.ID, value.Session.State, value.Workspace.Path)
			}
		}
	}
	return nil
}

// get resolves a Task's internal Session binding and prints only the
// backend-native ID that Codex or Copilot accepts for resume.
func get(store *storex.Store, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: aiw session get <task-id>")
	}
	id, err := taskSessionID(args[0])
	if err != nil { return err }
	value, err := store.Load(id)
	if err != nil {
		return fmt.Errorf("load Session for task %s: %w", args[0], err)
	}
	if value.Backend.Name != "codex" && value.Backend.Name != "copilot" {
		return fmt.Errorf("task %s backend %q has no supported CLI resume ID", args[0], value.Backend.Name)
	}
	if strings.TrimSpace(value.Backend.ThreadID) == "" {
		return fmt.Errorf("task %s has no backend resume ID; complete a backend turn first", args[0])
	}
	fmt.Println(value.Backend.ThreadID)
	return nil
}

func taskSessionID(taskID string) (string, error) {
	meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(taskID))
	if err != nil {
		return "", fmt.Errorf("read task %s: %w", taskID, err)
	}
	if strings.TrimSpace(meta.Session) == "" {
		return "", fmt.Errorf("task %s has no bound AIW Session", taskID)
	}
	return meta.Session, nil
}

func finish(store *storex.Store, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: aiw session finish <task-id>")
	}
	id, err := taskSessionID(args[0])
	if err != nil { return err }
	_, err = store.Transition(id, storex.StateCompleted)
	return err
}

func memory(store *storex.Store, args []string) error {
	if len(args) == 2 && args[0] == "show" {
		id, err := taskSessionID(args[1])
		if err != nil { return err }
		text, err := store.ReadText(id, "memory.md")
		if err == nil {
			fmt.Print(text)
		}
		return err
	}
	if len(args) != 3 || args[0] != "append" {
		return errors.New("usage: aiw session memory append <task-id> TEXT | show <task-id>")
	}
	id, err := taskSessionID(args[1])
	if err != nil { return err }
	return store.AppendMemory(id, strings.TrimSpace(args[2]))
}

func handoff(store *storex.Store, args []string) error {
	if len(args) >= 1 && args[0] == "show" {
		if len(args) != 2 {
			return errors.New("usage: aiw session handoff show <task-id>")
		}
		id, err := taskSessionID(args[1])
		if err != nil { return err }
		text, err := store.ReadArtifact(id, "handoff.md")
		if err == nil {
			fmt.Print(text)
		}
		return err
	}
	if len(args) < 1 || len(args) > 2 {
		return errors.New("usage: aiw session handoff <task-id> [focus] | show <task-id>")
	}
	focus := ""
	if len(args) == 2 {
		focus = args[1]
	}
	id, err := taskSessionID(args[0])
	if err != nil { return err }
	return store.Handoff(id, focus)
}
