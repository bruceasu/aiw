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
	SessionID    string
	ParentThread string
	ChildThread  string
	Handoff      string
	HandoffHash  string
	StartedAt    string
	CompletedAt  string
	Status       string
	Error        string
}

func runTaskAgent(args []string) error {
	if len(args) == 3 && args[0] == "agent" && args[1] == "status" {
		b, err := os.ReadFile(filepath.Join(taskx.TaskDir(args[2]), "agent-lineage.json"))
		if err != nil {
			return fmt.Errorf("agent lineage not found for %s", args[2])
		}
		fmt.Print(string(b))
		return nil
	}
	if len(args) != 3 || args[0] != "agent" || args[1] != "next" {
		return errors.New("usage: task agent <next|status> <task-id>")
	}
	id := args[2]
	metaPath := taskx.ResolveTaskMetaPath(id)
	meta, err := taskx.ReadTaskMeta(metaPath)
	if err != nil {
		return fmt.Errorf("read task metadata: %w", err)
	}
	if strings.TrimSpace(meta.Session) == "" {
		return fmt.Errorf("task %s has no session binding (set session = \"...\" in %s)", id, metaPath)
	}
	worktree := meta.Worktree
	if worktree == "" {
		return fmt.Errorf("task %s has no worktree", id)
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

	status, err := flowStatusJSON(meta.Session)
	if err != nil {
		return err
	}
	if status.Session.State == "running" || status.Session.State == "completed" || status.Session.State == "archived" || status.Session.State == "deleted" {
		return fmt.Errorf("session %s cannot start next agent from state %q", meta.Session, status.Session.State)
	}
	lineage := agentLineage{TaskID: id, SessionID: meta.Session, ParentThread: status.Codex.ThreadID, Handoff: "artifacts/handoff.md", StartedAt: time.Now().UTC().Format(time.RFC3339), Status: "starting"}
	if err := writeLineage(id, lineage); err != nil {
		return err
	}
	handoffOutput, err := runFlow(worktree, "handoff", "create", meta.Session)
	if err != nil {
		lineage.Status, lineage.Error = "failed", err.Error()
		_ = writeLineage(id, lineage)
		return err
	}
	if marker := "Handoff saved to "; strings.Contains(handoffOutput, marker) {
		path := strings.TrimSpace(strings.SplitN(handoffOutput, marker, 2)[1])
		path = strings.TrimSpace(strings.SplitN(path, "\n", 2)[0])
		if b, readErr := os.ReadFile(path); readErr == nil {
			digest := sha256.Sum256(b)
			lineage.HandoffHash = hex.EncodeToString(digest[:])
		}
	}
	prompt := fmt.Sprintf("Continue Task %s.\n\nSession: %s\n\nRead the Session handoff at artifacts/handoff.md and referenced artifacts before taking action. Preserve the existing Task and worktree; report validation when done.\n\nTask context:\n%s", id, meta.Session, readTaskGoal(id))
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
	lineage.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	lineage.Status = "completed"
	if err := writeLineage(id, lineage); err != nil {
		return err
	}
	fmt.Printf("Task %s handed off: %s -> %s\n", id, lineage.ParentThread, lineage.ChildThread)
	return nil
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
