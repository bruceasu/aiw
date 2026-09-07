package task

import (
	"context"
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
	"aiw/internal/gitx"
	"aiw/internal/session"
	"aiw/internal/taskx"
)

type agentLineage struct {
	TaskID           string
	ParentTask       string
	SourceTask       string
	SessionID        string
	SourceSession    string
	ChildSession     string
	ParentThread     string
	SourceThread     string
	ChildThread      string
	Handoff          string
	SourcePath       string
	HandoffHash      string
	HandoffStatus    string
	HandoffCreatedAt string
	ConsumedAt       string
	ConsumerThread   string
	ConsumedHash     string
	ParentState      string
	ChildState       string
	StartedAt        string
	CompletedAt      string
	Status           string
	Error            string
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
		return errors.New("usage: task agent next <task-id> [--handoff PATH] [--takeover] [--isolated] [--allow-dirty] [--yes]")
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
	createdTask := false
	if existing {
		originalMeta, readErr := taskx.ReadTaskMeta(metaPath)
		if readErr != nil {
			return fmt.Errorf("read task metadata: %w", readErr)
		}
		sessionMissing = strings.TrimSpace(originalMeta.Session) == "" || !fsx.Exists(filepath.Join(".ai", "sessions", originalMeta.Session, "status.json"))
		if err := ensureTaskMeta(id); err != nil {
			return fmt.Errorf("repair task metadata: %w", err)
		}
		meta, err = taskx.ReadTaskMeta(metaPath)
		if err != nil {
			return fmt.Errorf("read task metadata: %w", err)
		}
	} else {
		handoff, source, err := resolveHandoff(opts.Handoff, "", id)
		if err != nil {
			return err
		}
		printAgentPlan(id, "create", "", "", "", source)
		if err := newTask(id, opts.AllowDirty); err != nil {
			return fmt.Errorf("create task: %w", err)
		}
		createdTask = true
		metaPath = taskx.ResolveTaskMetaPath(id)
		meta, err = taskx.ReadTaskMeta(metaPath)
		if err != nil {
			return err
		}
		if err := copyHandoff(id, handoff, source); err != nil {
			return rollbackNewTask(id, fmt.Errorf("copy handoff: %w", err))
		}
		if opts.Isolated {
			if err := addTaskWorktree(id); err != nil {
				return rollbackNewTask(id, fmt.Errorf("isolation failed: %w", err))
			}
			meta, err = taskx.ReadTaskMeta(metaPath)
			if err != nil {
				return err
			}
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
	if err := validateTaskBindings(id, meta); err != nil {
		return err
	}
	if opts.Isolated && meta.WorkspaceKind != "isolated" {
		if err := addTaskWorktree(id); err != nil {
			return err
		}
		meta, err = taskx.ReadTaskMeta(metaPath)
		if err != nil {
			return err
		}
	}
	kind := resolvedWorkspaceKind(meta)
	if kind == "unassigned" || kind == "unknown" {
		return fmt.Errorf("task %s workspace is %s", id, kind)
	}
	worktree := meta.Worktree
	if strings.TrimSpace(worktree) == "" {
		return fmt.Errorf("task %s workspace is unassigned", id)
	}
	if !filepath.IsAbs(worktree) {
		root, rootErr := gitx.ProjectRoot()
		if rootErr != nil {
			return rootErr
		}
		worktree = filepath.Join(root, filepath.FromSlash(worktree))
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
		if err := createTaskSession(id, meta.Worktree); err != nil {
			return err
		}
	}
	store := session.NewStore("")
	status, err := store.Load(meta.Session)
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
	printAgentPlan(id, "reuse", meta.Session, meta.Branch, meta.Worktree, handoff)
	lineage := agentLineage{TaskID: id, ParentTask: id, SessionID: meta.Session, ChildSession: meta.Session, ParentThread: status.Backend.ThreadID, Handoff: handoff, HandoffStatus: "pending", ParentState: "active", ChildState: "starting", StartedAt: time.Now().UTC().Format(time.RFC3339), Status: "starting"}
	lineage.SourcePath = handoff
	lineage.SourceSession = meta.Session
	lineage.SourceThread = status.Backend.ThreadID
	lineage.HandoffCreatedAt = lineage.StartedAt
	if b, readErr := os.ReadFile(handoff); readErr == nil {
		digest := sha256.Sum256(b)
		lineage.HandoffHash = hex.EncodeToString(digest[:])
	}
	if err := writeLineage(id, lineage); err != nil {
		return err
	}
	prompt := fmt.Sprintf("Continue Task %s.\n\nSession: %s\n\nRead the handoff at %s and referenced artifacts before taking action. Preserve the existing Task and worktree; report validation when done.\n\nTask context:\n%s", id, meta.Session, handoff, readTaskGoal(id))
	result, err := session.ExecuteTurn(context.Background(), store, meta.Session, "handoff", prompt, status.Backend.Name, status.Backend.Model, true)
	if err != nil {
		_ = recordSessionHandoff(store, meta.Session, lineage)
		lineage.Status, lineage.ChildState, lineage.Error = "failed", "failed", err.Error()
		_ = writeLineage(id, lineage)
		if createdTask {
			return rollbackNewTask(id, err)
		}
		return err
	}
	lineage.ChildThread = result.ThreadID
	lineage.HandoffStatus = "consumed"
	lineage.ConsumedAt = time.Now().UTC().Format(time.RFC3339)
	lineage.ConsumerThread = result.ThreadID
	lineage.ConsumedHash = lineage.HandoffHash
	lineage.ParentState = "handed-off"
	lineage.ChildState = "completed"
	lineage.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	lineage.Status = "completed"
	if err := recordSessionHandoff(store, meta.Session, lineage); err != nil {
		return err
	}
	if err := markTaskRunning(id); err != nil {
		lineage.Status, lineage.Error = "failed", err.Error()
		_ = writeLineage(id, lineage)
		if createdTask {
			return rollbackNewTask(id, err)
		}
		return err
	}
	if err := writeLineage(id, lineage); err != nil {
		return err
	}
	fmt.Printf("Task %s handed off: %s -> %s\n", id, lineage.ParentThread, lineage.ChildThread)
	return nil
}

type agentOptions struct {
	Handoff    string
	Takeover   bool
	Isolated   bool
	AllowDirty bool
	Yes        bool
}

func parseAgentOptions(args []string) (agentOptions, error) {
	var opts agentOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--takeover":
			opts.Takeover = true
		case "--isolated":
			opts.Isolated = true
		case "--allow-dirty":
			opts.AllowDirty = true
		case "--yes":
			opts.Yes = true
		case "--handoff":
			if i+1 >= len(args) {
				return opts, errors.New("--handoff requires a path")
			}
			i++
			opts.Handoff = args[i]
		default:
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
	}
	return opts, nil
}

func normalizeID(id string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(id) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
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
	if explicit != "" {
		paths = append(paths, explicit)
	}
	if taskID != "" {
		paths = append(paths, filepath.Join(taskx.TaskDir(taskID), "artifacts", "handoff.md"))
	}
	if session != "" {
		paths = append(paths, filepath.Join(".ai", "sessions", session, "artifacts", "handoff.md"), filepath.Join("artifacts", "handoff.md"))
	}
	for _, path := range paths {
		if fsx.Exists(path) {
			absolute, err := filepath.Abs(path)
			if err != nil {
				return "", "", err
			}
			return absolute, absolute, nil
		}
	}
	return "", "", errors.New("handoff not found; use --handoff PATH or create a Session handoff first")
}

func copyHandoff(id, source, sourcePath string) error {
	b, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	dir := filepath.Join(taskx.TaskDir(id), "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "handoff.md"), b, 0o644)
}

func createTaskSession(id, worktree string) error {
	instructions := filepath.Join(taskx.TaskDir(id), "artifacts", "instructions.md")
	content := "Read artifacts/handoff.md before acting. Preserve the Task scope and report validation.\n"
	if err := os.WriteFile(instructions, []byte(content), 0o644); err != nil {
		return err
	}
	if !filepath.IsAbs(worktree) {
		root, err := gitx.ProjectRoot()
		if err != nil {
			return err
		}
		worktree = filepath.Join(root, filepath.FromSlash(worktree))
	}
	_, err := session.NewStore("").Create(id, id, worktree, "codex", "", content)
	return err
}

func addTaskWorktree(id string) error { return runCommand("aiw", "wt", "add", id) }

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func rollbackNewTask(id string, cause error) error {
	_ = session.NewStore("").Delete(id)
	worktreeErr := runCommand("aiw", "wt", "rm", id, "--force")
	_ = os.RemoveAll(taskx.TaskDir(id))
	if worktreeErr != nil {
		return fmt.Errorf("%w (new Task %s was rolled back; worktree cleanup: %v)", cause, id, worktreeErr)
	}
	return fmt.Errorf("%w (new Task %s was rolled back)", cause, id)
}

func markTaskRunning(id string) error {
	path := taskx.ResolveTaskMetaPath(id)
	meta, err := taskx.ReadTaskMeta(path)
	if err != nil {
		return err
	}
	meta.Status = "RUNNING"
	meta.Updated = taskx.Today()
	if err := taskx.WriteTaskMeta(path, meta); err != nil {
		return err
	}
	return taskx.WriteRegistry()
}

func recordSessionHandoff(store *session.Store, sessionID string, lineage agentLineage) error {
	_, err := store.Update(sessionID, func(status *session.Status) error {
		status.Task = map[string]interface{}{
			"task_id": lineage.TaskID, "handoff": lineage.Handoff,
			"handoff_hash": lineage.HandoffHash, "handoff_status": lineage.HandoffStatus,
			"handoff_created_at": lineage.HandoffCreatedAt, "consumed_hash": lineage.ConsumedHash,
			"parent_thread": lineage.ParentThread, "child_thread": lineage.ChildThread,
			"consumed_at": lineage.ConsumedAt, "consumer_thread": lineage.ConsumerThread,
			"parent_state": lineage.ParentState, "child_state": lineage.ChildState,
		}
		return nil
	})
	return err
}

func validateTaskBindings(id string, meta taskx.TaskMeta) error {
	entries, err := os.ReadDir(taskx.ChangesDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == id || entry.Name() == "archive" {
			continue
		}
		other, readErr := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPathInDir(filepath.Join(taskx.ChangesDir, entry.Name())))
		if readErr != nil {
			continue
		}
		if strings.TrimSpace(meta.Session) != "" && meta.Session == other.Session {
			return fmt.Errorf("session %s is already bound to Task %s", meta.Session, other.ID)
		}
		if strings.TrimSpace(meta.Worktree) != "" && meta.Worktree != "." && meta.Worktree == other.Worktree {
			return fmt.Errorf("worktree %s is already bound to Task %s", meta.Worktree, other.ID)
		}
	}
	return nil
}

func printAgentPlan(id, path, sessionID, branch, worktree, handoff string) {
	fmt.Printf("Task agent plan: task=%s path=%s session=%s branch=%s worktree=%s handoff=%s thread=fresh\n", id, path, sessionID, branch, worktree, handoff)
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
