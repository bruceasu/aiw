package flow

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"aiw/internal/session"
)

func Dispatch(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return nil
	}
	store := session.NewStore(os.Getenv("AIW_SESSION_ROOT"))
	command, args := args[0], args[1:]
	switch command {
	case "new":
		return create(store, args)
	case "run", "continue":
		return run(store, args)
	case "loop":
		return loop(store, args)
	case "grill":
		return grill(store, args)
	case "status":
		return status(store, args)
	case "list":
		return list(store)
	case "finish":
		return finish(store, args)
	case "archive":
		if len(args) != 1 {
			return errors.New("usage: aiw flow archive <session-id>")
		}
		return store.Archive(args[0])
	case "delete":
		if len(args) != 2 || args[1] != "--yes" {
			return errors.New("usage: aiw flow delete <session-id> --yes")
		}
		return store.Delete(args[0])
	case "memory":
		return memory(store, args)
	case "handoff":
		return handoff(store, args)
	default:
		return fmt.Errorf("unknown flow command: %s", command)
	}
}

func printHelp() {
	fmt.Print("aiw flow - AIW-native Session execution\n\nUsage:\n" +
		"  aiw flow new <id> [--workspace PATH] [--backend codex|copilot|openai] [--model MODEL]\n" +
		"  aiw flow run <id> --prompt TEXT [--phase NAME] [--backend NAME] [--force-new-thread]\n" +
		"  aiw flow continue <id> --prompt TEXT [--phase NAME]\n" +
		"  aiw flow loop <id> [--phase NAME]\n" +
		"  aiw flow grill <id> --workspace PATH --requirement TEXT\n" +
		"  aiw flow status <id>\n  aiw flow list\n  aiw flow finish <id>\n" + "  aiw flow archive <id>\n  aiw flow memory append <id> TEXT\n" + "  aiw flow memory show <id>\n  aiw flow handoff <id> [focus]\n" + "  aiw flow handoff show <id>\n")
}

func create(store *session.Store, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: aiw flow new <id> [options]")
	}
	id, workspace, backend, model, title := args[0], ".", "codex", "", args[0]
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--workspace", "--backend", "--model", "--title":
			if i+1 >= len(args) {
				return fmt.Errorf("%s requires a value", args[i])
			}
			i++
			switch args[i-1] {
			case "--workspace":
				workspace = args[i]
			case "--backend":
				backend = args[i]
			case "--model":
				model = args[i]
			case "--title":
				title = args[i]
			}
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}
	absolute, err := filepath.Abs(workspace)
	if err != nil {
		return err
	}
	_, err = store.Create(id, title, absolute, backend, model, "Preserve Task scope. Read the task artifacts before acting. Report validation.")
	if err == nil {
		fmt.Println("created:", id)
	}
	return err
}

func run(store *session.Store, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: aiw flow run|continue <id> --prompt TEXT")
	}
	id, prompt, phase, backend, model := args[0], "", "", "", ""
	force := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--prompt", "--prompt-file", "--phase", "--backend", "--model":
			if i+1 >= len(args) {
				return fmt.Errorf("%s requires a value", args[i])
			}
			i++
			switch args[i-1] {
			case "--prompt":
				prompt = args[i]
			case "--prompt-file":
				b, err := os.ReadFile(args[i])
				if err != nil {
					return err
				}
				prompt = string(b)
			case "--phase":
				phase = args[i]
			case "--backend":
				backend = args[i]
			case "--model":
				model = args[i]
			}
		case "--force-new-thread":
			force = true
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}
	if prompt == "" {
		b, _ := io.ReadAll(os.Stdin)
		prompt = string(b)
	}
	result, err := session.ExecuteTurn(context.Background(), store, id, phase, prompt, backend, model, force)
	if result.FinalOutput != "" {
		fmt.Println(result.FinalOutput)
	}
	return err
}

func loop(store *session.Store, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: aiw flow loop <id> [--phase NAME]")
	}
	id, phase := args[0], ""
	for i := 1; i < len(args); i++ {
		if args[i] != "--phase" || i+1 >= len(args) {
			return errors.New("usage: aiw flow loop <id> [--phase NAME]")
		}
		i++
		phase = args[i]
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err == io.EOF && strings.TrimSpace(line) == "" {
			return nil
		}
		if err != nil && err != io.EOF {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			if err == io.EOF {
				return nil
			}
			continue
		}
		switch line {
		case "/exit":
			return nil
		case "/status":
			if err := status(store, []string{id}); err != nil {
				return err
			}
			continue
		case "/handoff":
			if err := handoff(store, []string{id}); err != nil {
				return err
			}
			continue
		}
		result, runErr := session.ExecuteTurn(context.Background(), store, id, phase, line, "", "", false)
		if result.FinalOutput != "" {
			fmt.Println(result.FinalOutput)
		}
		if runErr != nil || err == io.EOF {
			return runErr
		}
	}
}

func grill(store *session.Store, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: aiw flow grill <id> --workspace PATH --requirement TEXT")
	}
	id, workspace, requirement := args[0], ".", ""
	for i := 1; i < len(args); i++ {
		if i+1 >= len(args) || (args[i] != "--workspace" && args[i] != "--requirement") {
			return errors.New("usage: aiw flow grill <id> --workspace PATH --requirement TEXT")
		}
		i++
		if args[i-1] == "--workspace" {
			workspace = args[i]
		} else {
			requirement = args[i]
		}
	}
	if strings.TrimSpace(requirement) == "" {
		return errors.New("--requirement requires a non-empty value")
	}
	absolute, err := filepath.Abs(workspace)
	if err != nil {
		return err
	}
	instructions := "Ask at most one decision question per turn. Inspect the workspace before asking discoverable questions. Finish with SUCCESS: Ready to execute.\n"
	if _, err = store.Create(id, id, absolute, "codex", "", instructions); err != nil {
		return err
	}
	result, err := session.ExecuteTurn(context.Background(), store, id, "grill", requirement, "codex", "", false)
	if result.FinalOutput != "" {
		fmt.Println(result.FinalOutput)
	}
	return err
}

func status(store *session.Store, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: aiw flow status <id>")
	}
	value, err := store.Load(args[0])
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(value, "", "  ")
	fmt.Println(string(b))
	return nil
}
func list(store *session.Store) error {
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
func finish(store *session.Store, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: aiw flow finish <id>")
	}
	_, err := store.Transition(args[0], session.StateCompleted)
	return err
}
func memory(store *session.Store, args []string) error {
	if len(args) == 2 && args[0] == "show" {
		text, err := store.ReadText(args[1], "memory.md")
		if err == nil {
			fmt.Print(text)
		}
		return err
	}
	if len(args) != 3 || args[0] != "append" {
		return errors.New("usage: aiw flow memory append <id> TEXT | show <id>")
	}
	return store.AppendMemory(args[1], strings.TrimSpace(args[2]))
}
func handoff(store *session.Store, args []string) error {
	if len(args) >= 1 && args[0] == "show" {
		if len(args) != 2 {
			return errors.New("usage: aiw flow handoff show <id>")
		}
		text, err := store.ReadArtifact(args[1], "handoff.md")
		if err == nil {
			fmt.Print(text)
		}
		return err
	}
	if len(args) < 1 || len(args) > 2 {
		return errors.New("usage: aiw flow handoff <id> [focus] | show <id>")
	}
	focus := ""
	if len(args) == 2 {
		focus = args[1]
	}
	return store.Handoff(args[0], focus)
}
