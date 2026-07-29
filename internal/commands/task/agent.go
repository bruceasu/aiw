package task

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"aiw/internal/fsx"
	"aiw/internal/taskx"
)

type flowStatus struct {
	Session struct{ State string }
	Codex   struct{ ThreadID string }
}

type agentLineage struct {
	TaskID       string
	ParentTask   string
	SessionID    string
	ChildSession string
	ParentThread string
	ChildThread  string
	Handoff      string
	HandoffHash  string
	HandoffStatus string
	StartedAt    string
	CompletedAt  string
	Status       string
	Error        string
}

func runTaskAgent(args []string) error {
	if len(args) >= 3 && args[0] == "agent" && args[1] == "status" {
		b, err := os.ReadFile(filepath.Join(taskx.TaskDir(args[2]), "agent-lineage.json"))
		if err != nil {
			return fmt.Errorf("agent lineage not found for %s", args[2])
		}
		fmt.Print(string(b))
		return nil
	}
	if len(args) < 3 || args[0] != "agent" || args[1] != "next" {
		return errors.New("usage: task agent next <task-id> [--handoff PATH] [--takeover] [--yes]")
	}
	id := args[2]
	opts, err := parseAgentOptions(args[3:])
	if err != nil {
		return err
	}
	if !safeID(id) {
		candidate := normalizeID(id)
		fmt.Printf("Invalid task ID %q. Candidate mapping: %s (use --yes to accept):\n", id, candidate)
		if !opts.Yes || !confirm("Create the task with this candidate name? [y/N] ") {
			return errors.New("task creation refused: invalid task ID requires confirmation")
		}
		id = candidate
	}
	metaPath := taskx.ResolveTaskMetaPath(id)
	existing := fsx.Exists(taskx.TaskDir(id))
	var meta taskx.TaskMeta
	sessionMissing := false
	if existing {
		meta, err = taskx.ReadTaskMeta(metaPath)
		if err != nil {
			return fmt.Errorf("read task metadata: %w", err)
		}
	} else {
		handoff, source, err := resolveHandoff(opts.Handoff, "", id)
		if err != nil {
			return err
		}
		if err := newTask(id); err != nil {
			return fmt.Errorf("create task: %w", err)
		}
		metaPath = taskx.ResolveTaskMetaPath(id)
		meta, err = taskx.ReadTaskMeta(metaPath)
		if err != nil {
			return err
		}
		if err := copyHandoff(id, handoff, source); err != nil {
			return rollbackNewTask(id, fmt.Errorf("copy handoff: %w", err))
		}
		if err := addTaskWorktree(id); err != nil {
			return rollbackNewTask(id, fmt.Errorf("create worktree: %w", err))
		}
		meta, err = taskx.ReadTaskMeta(metaPath)
		if err != nil {
			return rollbackNewTask(id, err)
		}
		meta.Session = id
		if err := taskx.WriteTaskMeta(metaPath, meta); err != nil {
			return rollbackNewTask(id, err)
		}
		if err := createTaskSession(id, meta.Worktree); err != nil {
			return rollbackNewTask(id, fmt.Errorf("create session: %w", err))
		}
	}
	if strings.TrimSpace(meta.Session) == "" {
		meta.Session = id
		sessionMissing = true
		if err := taskx.WriteTaskMeta(metaPath, meta); err != nil {
			return err
		}
	}
	worktree := meta.Worktree
	if worktree == "" {
		if err := addTaskWorktree(id); err != nil {
			return err
		}
		meta, err = taskx.ReadTaskMeta(metaPath)
		if err != nil {
			return err
		}
		worktree = meta.Worktree
	}
	if !filepath.IsAbs(worktree) {
		worktree = filepath.Join(".", filepath.FromSlash(worktree))
	}
	worktree, err = filepath.Abs(worktree)
	if err != nil || !fsx.Exists(worktree) {
		return fmt.Errorf("task %s worktree does not exist: %s", id, worktree)
	}

	leaseDir := filepath.Join(".aiw", "agent-leases")
	if err := os.MkdirAll(leaseDir, 0o755); err != nil {
		return err
	}
	leasePath := filepath.Join(leaseDir, id+".lock")
	lease, err := os.OpenFile(leasePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("task worktree is already leased: %s", id)
	}
	lease.Close()
	defer os.Remove(leasePath)

	if sessionMissing {
		if err := createTaskSession(id, meta.Worktree); err != nil { return err }
	}
	status, err := flowStatusJSON(meta.Session)
	if err != nil {
		return err
	}
	if status.Session.State == "running" && !opts.Takeover {
		return fmt.Errorf("session %s is running; use --takeover to continue", meta.Session)
	}
	if status.Session.State == "completed" || status.Session.State == "archived" || status.Session.State == "deleted" {
		return fmt.Errorf("session %s cannot start next agent from state %q", meta.Session, status.Session.State)
	}
	handoff, _, err := resolveHandoff(opts.Handoff, meta.Session, id)
	if err != nil {
		return err
	}
	lineage := agentLineage{TaskID: id, ParentTask: id, SessionID: meta.Session, ChildSession: meta.Session, ParentThread: status.Codex.ThreadID, Handoff: handoff, HandoffStatus: "pending", StartedAt: time.Now().UTC().Format(time.RFC3339), Status: "starting"}
	if b, readErr := os.ReadFile(handoff); readErr == nil {
		digest := sha256.Sum256(b)
		lineage.HandoffHash = hex.EncodeToString(digest[:])
	}
	if err := writeLineage(id, lineage); err != nil { return err }
	prompt := fmt.Sprintf("Continue Task %s.\n\nSession: %s\n\nRead the handoff at %s and referenced artifacts before taking action. Preserve the existing Task and worktree; report validation when done.\n\nTask context:\n%s", id, meta.Session, handoff, readTaskGoal(id))
	if _, err := runFlow(worktree, "run", meta.Session, "--force-new-thread", "--prompt", prompt); err != nil {
		lineage.Status, lineage.Error = "failed", err.Error()
		_ = writeLineage(id, lineage)
		return err
	}
	child, err := flowStatusJSON(meta.Session)
	if err != nil {
		return err
	}
	lineage.ChildThread = child.Codex.ThreadID
	lineage.HandoffStatus = "consumed"
	lineage.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	lineage.Status = "completed"
	if err := writeLineage(id, lineage); err != nil {
		return err
	}
	fmt.Printf("Task %s handed off: %s -> %s\n", id, lineage.ParentThread, lineage.ChildThread)
	return nil
}

type agentOptions struct { Handoff string; Takeover bool; Yes bool }

func parseAgentOptions(args []string) (agentOptions, error) {
	var opts agentOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--takeover": opts.Takeover = true
		case "--yes": opts.Yes = true
		case "--handoff":
			if i+1 >= len(args) { return opts, errors.New("--handoff requires a path") }
			i++; opts.Handoff = args[i]
		default: return opts, fmt.Errorf("unknown option: %s", args[i])
		}
	}
	return opts, nil
}

func normalizeID(id string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(id) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' { b.WriteRune(r) } else { b.WriteRune('-') }
	}
	return strings.Trim(b.String(), "-_.")
}

func confirm(prompt string) bool {
	fmt.Print(prompt)
	var answer string
	_, err := fmt.Scanln(&answer)
	return err == nil && strings.EqualFold(strings.TrimSpace(answer), "y")
}

func resolveHandoff(explicit, session, taskID string) (string, string, error) {
	paths := []string{}
	if explicit != "" { paths = append(paths, explicit) }
	if taskID != "" { paths = append(paths, filepath.Join(taskx.TaskDir(taskID), "artifacts", "handoff.md")) }
	if session != "" { paths = append(paths, filepath.Join(".aiw", "sessions", session, "artifacts", "handoff.md"), filepath.Join("artifacts", "handoff.md")) }
	for _, path := range paths {
		if fsx.Exists(path) { absolute, err := filepath.Abs(path); if err != nil { return "", "", err }; return absolute, absolute, nil }
	}
	return "", "", errors.New("handoff not found; use --handoff PATH or create a Session handoff first")
}

func copyHandoff(id, source, sourcePath string) error {
	b, err := os.ReadFile(source); if err != nil { return err }
	dir := filepath.Join(taskx.TaskDir(id), "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil { return err }
	return os.WriteFile(filepath.Join(dir, "handoff.md"), b, 0o644)
}

func createTaskSession(id, worktree string) error {
	instructions := filepath.Join(taskx.TaskDir(id), "artifacts", "instructions.md")
	content := "Read artifacts/handoff.md before acting. Preserve the Task scope and report validation.\n"
	if err := os.WriteFile(instructions, []byte(content), 0o644); err != nil { return err }
	return runCommand("aiw-flow", "new", "--id", id, "--title", id, "--workspace", worktree, "--instructions", instructions)
}

func addTaskWorktree(id string) error { return runCommand("aiw", "wt", "add", id, "main") }

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...); cmd.Stdout = os.Stdout; cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil { return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err) }
	return nil
}

func rollbackNewTask(id string, cause error) error {
	_ = runCommand("aiw-flow", "delete", id, "--yes")
	worktreeErr := runCommand("aiw", "wt", "rm", id, "--force")
	if worktreeErr == nil {
		_ = os.RemoveAll(taskx.TaskDir(id))
		return fmt.Errorf("%w (new Task %s was rolled back)", cause, id)
	}
	return fmt.Errorf("%w (new Task %s needs manual cleanup: %v)", cause, id, worktreeErr)
}

func flowStatusJSON(session string) (flowStatus, error) {
	cmd := exec.Command("aiw-flow", "status", session, "--json")
	out, err := cmd.Output()
	if err != nil {
		return flowStatus{}, fmt.Errorf("read aiw-flow session %s: %w", session, err)
	}
	var status flowStatus
	if err := json.Unmarshal(out, &status); err != nil {
		return flowStatus{}, fmt.Errorf("decode aiw-flow status: %w", err)
	}
	return status, nil
}

func runFlow(worktree string, args ...string) (string, error) {
	cmd := exec.Command("aiw-flow", args...)
	cmd.Dir = worktree
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("aiw-flow %s: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func readTaskGoal(id string) string {
	b, err := os.ReadFile(filepath.Join(taskx.TaskDir(id), "tasks.md"))
	if err != nil {
		return "(see tasks.md)"
	}
	text := strings.TrimSpace(string(b))
	if len(text) > 4000 {
		text = text[:4000] + "\n[truncated]"
	}
	return text
}

func writeLineage(id string, lineage agentLineage) error {
	b, err := json.MarshalIndent(lineage, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(taskx.TaskDir(id), "agent-lineage.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readLineage(id string) ([]byte, error) {
	return os.ReadFile(filepath.Join(taskx.TaskDir(id), "agent-lineage.json"))
}
