package assist

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aiw/internal/session"
)

type options struct {
	action, id, topic, provider, model, flavor string
	sources []string
}

func Usage(kind string) string {
	return "Usage: aiw " + kind + " [new] [TOPIC] [--source PATH] [--provider NAME] [--model NAME]\n" +
		"       aiw " + kind + " list | resume ID | recover ID\n" +
		"No arguments: select a saved task/course or create one. Prototype: --kind logic|ui (default logic).\n" +
		"Chat: /run, /brief, /source PATH, /window 0..6, /recall TURN (0 clears), /pause, /exit, /help\n" +
		"Each line sends once. Attach articles/source via --source or /source. /run uses the displayed ready brief.\n" +
		"Teach keeps multiple independent courses. Ctrl+C/EOF saves a resumable conversation.\n" +
		"Read-only model task; host saves new artifacts, never executes prototypes or edits originals.\n" +
		"Research uses supplied local text; no automatic browsing/background agent. CLI-provider tool isolation is backend-dependent.\n" +
		"recover reads saved results only; unknown process locks/running turns are preserved for inspection.\n"
}

func parse(kind string, args []string) (options, error) {
	var o options
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--source", "--provider", "--model", "--kind":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") { return o, fmt.Errorf("missing value for %s", a) }
			i++
			switch a {
			case "--source": o.sources = append(o.sources, args[i])
			case "--provider": o.provider = args[i]
			case "--model": o.model = args[i]
			case "--kind": o.flavor = args[i]
			}
		default:
			if strings.HasPrefix(a, "-") { return o, fmt.Errorf("unknown option: %s", a) }
			positional = append(positional, a)
		}
	}
	if o.flavor != "" && (kind != "prototype" || o.flavor != "logic" && o.flavor != "ui") { return o, errors.New("--kind is prototype-only: logic or ui") }
	if len(positional) > 0 {
		switch positional[0] {
		case "list", "resume", "recover", "new": o.action, positional = positional[0], positional[1:]
		default: o.action = "new"
		}
	}
	if o.action == "" && (len(o.sources) > 0 || o.flavor != "") { o.action = "new" }
	if o.action == "resume" || o.action == "recover" {
		if len(positional) != 1 || len(o.sources) != 0 || o.flavor != "" { return o, errors.New("resume/recover needs one ID; use /source inside the conversation to attach more text") }
		o.id = positional[0]
	} else {
		o.topic = strings.Join(positional, " ")
		if len(o.topic) > maxInput { return o, errors.New("topic exceeds 8 KiB") }
		if o.action == "list" && (o.topic != "" || len(o.sources) != 0) { return o, errors.New("list does not accept topic/sources") }
	}
	return o, nil
}

type inputLine struct { text string; err error }

func inputStream(ctx context.Context) <-chan inputLine {
	ch := make(chan inputLine)
	go func() {
		defer close(ch)
		s := bufio.NewScanner(os.Stdin)
		s.Buffer(make([]byte, 1024), maxInput+1)
		for s.Scan() {
			select { case ch <- inputLine{text: s.Text()}: case <-ctx.Done(): return }
		}
		if err := s.Err(); err != nil { select { case ch <- inputLine{err: err}: case <-ctx.Done(): } }
	}()
	return ch
}

func readLine(ctx context.Context, lines <-chan inputLine) (string, error) {
	select {
	case <-ctx.Done(): return "", ctx.Err()
	case line, ok := <-lines:
		if !ok { return "", io.EOF }
		return line.text, line.err
	}
}

func Dispatch(kind string, args []string) error {
	for _, arg := range args { if arg == "-h" || arg == "--help" { fmt.Print(Usage(kind)); return nil } }
	o, err := parse(kind, args)
	if err != nil { return err }
	workspace, err := filepath.Abs(".")
	if err != nil { return err }
	workspace, err = filepath.EvalSymlinks(workspace)
	if err != nil { return err }
	store := session.NewStore(filepath.Join(workspace, ".ai"))
	if err := rejectLinks(workspace, store.Root); err != nil { return err }
	if o.action == "list" {
		items, err := records(store, workspace, kind)
		if err != nil { return err }
		for _, r := range items { fmt.Printf("%s  %s  turns=%d\n", r.ID, r.Title, r.LastTurn) }
		if len(items) == 0 { fmt.Println("No saved " + kind + " tasks/courses in this workspace.") }
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	lines := inputStream(ctx)
	if o.action == "" {
		items, err := records(store, workspace, kind)
		if err != nil { return err }
		fmt.Println("0: new " + kind)
		for i, r := range items { fmt.Printf("%d: %s (%s)\n", i+1, r.Title, r.ID) }
		fmt.Print("Choose a number (Enter = new): ")
		choice, err := readLine(ctx, lines)
		if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) { return nil }
		if err != nil { return err }
		if choice != "" && choice != "0" {
			n, err := strconv.Atoi(strings.TrimSpace(choice))
			if err != nil || n < 1 || n > len(items) { return errors.New("invalid task/course selection") }
			o.action, o.id = "resume", items[n-1].ID
		} else { o.action = "new" }
	}
	var r *record
	if o.action == "new" {
		if kind == "prototype" && o.flavor == "" { o.flavor = "logic" }
		r, err = createRecord(store, workspace, kind, o)
	} else { r, _, err = loadRecord(store, workspace, kind, o.id) }
	if err != nil { return err }
	unlock, err := acquire(store, r.ID)
	if err != nil { return err }
	defer unlock()
	// Re-read under the auxiliary lock, never use the menu's stale state.
	r, status, err := loadRecord(store, workspace, kind, r.ID)
	if err != nil { return err }
	defer func() {
		s, loadErr := store.Load(r.ID)
		if loadErr == nil && s.Session.State == session.StateCreated {
			_, _ = store.Transition(r.ID, session.StateActive)
			s, loadErr = store.Load(r.ID)
		}
		if loadErr == nil && s.Session.State == session.StateActive { _, _ = store.Transition(r.ID, session.StatePaused) }
		fmt.Printf("Saved session: %s\nContinue: aiw %s resume %s\n", r.ID, kind, r.ID)
	}()
	if o.action == "recover" { return recoverResult(store, r) }
	if err := consistent(r, status); err != nil { return err }
	fmt.Printf("%s: %s\n", r.ID, r.Title)
	fmt.Print(Usage(kind))
	if r.Brief != "" { fmt.Println("Restored brief/progress:\n" + r.Brief) }
	if r.Ready { fmt.Println("This brief is ready; /run generates new artifacts.") }
	if o.action == "new" && o.topic != "" {
		phase := "clarify"
		if kind == "teach" { phase = "learn" }
		if err := step(ctx, store, r, phase, o.topic, o.provider, o.model); err != nil { return err }
	}
	for {
		fmt.Print(kind + "> ")
		line, err := readLine(ctx, lines)
		if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) { return nil }
		if err != nil { return err }
		line = strings.TrimSpace(line)
		if line == "" { continue }
		switch {
		case line == "/exit" || line == "/pause": return nil
		case line == "/help": fmt.Print(Usage(kind))
		case line == "/brief": fmt.Println(r.Brief)
		case strings.HasPrefix(line, "/window ") || strings.HasPrefix(line, "/recall "):
			parts := strings.Fields(line)
			if len(parts) != 2 { fmt.Println("Specify one number."); continue }
			n, err := strconv.Atoi(parts[1])
			if err != nil || n < 0 || parts[0] == "/window" && n > 6 || parts[0] == "/recall" && n > r.LastTurn { fmt.Println("Invalid history range."); continue }
			if parts[0] == "/window" { r.Window = n } else { r.Recall = n }
			if err := saveRecord(store, r); err != nil { return err }
		case strings.HasPrefix(line, "/source "):
			if err := attachSource(store, r, strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "/source ")), "\"")); err != nil { return err }
			fmt.Println("Source attached; discuss again before /run.")
		case line == "/run":
			if !r.Ready || r.Brief == "" { fmt.Println("Discuss the task until a concrete ready brief is shown first."); continue }
			if err := step(ctx, store, r, "generate", r.Brief, o.provider, o.model); err != nil { return err }
		case strings.HasPrefix(line, "/"): fmt.Println("Unknown chat command; use /help.")
		default:
			if len(line) > maxInput { fmt.Println("Input exceeds 8 KiB; use /source."); continue }
			phase := "clarify"
			if kind == "teach" { phase = "learn" }
			if err := step(ctx, store, r, phase, line, o.provider, o.model); err != nil { return err }
		}
	}
}

func attachSource(store *session.Store, r *record, path string) error {
	text, err := loadSources(r.Workspace, []string{path})
	if err != nil { return err }
	prior, err := store.ReadArtifact(r.ID, r.SourceFile)
	if err != nil { return err }
	if digest(prior) != r.SourceDigest || len(prior)+len(text) > maxSources { return errors.New("source snapshot changed or exceeds 128 KiB; start a narrower task") }
	combined := prior+text
	name := fmt.Sprintf("aux-sources-%d.md", time.Now().UnixNano())
	if err := writeImmutable(store, r.ID, name, []byte(combined)); err != nil { return err }
	r.SourceFile, r.SourceDigest, r.Ready = name, digest(combined), false
	return saveRecord(store, r)
}
