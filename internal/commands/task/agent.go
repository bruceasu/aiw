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
	"aiw/internal/workflow"
)

type agentLineage struct {
	TaskID           string
	AttemptID        string
	WorkItemID       string
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
	if len(args) >= 2 && (args[1] == "help" || args[1] == "--help" || args[1] == "-h") {
		printAgentHelp()
		return nil
	}
	if len(args) < 2 || (args[0] != "turn" && args[0] != "chat") {
		return errors.New("usage: aiw turn|chat <task-id> [--handoff PATH] [--provider NAME] [--model MODEL] [options]")
	}
	id := args[1]
	opts, err := parseAgentOptions(args[2:])
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
	existing := fsx.Exists(taskx.RuntimeTaskDir(id))
	var meta taskx.TaskMeta
	sessionMissing := false
	createdTask := false
	if existing {
		originalMeta, readErr := taskx.ReadTaskMeta(metaPath)
		if readErr != nil {
			return fmt.Errorf("read task metadata: %w", readErr)
		}
		sessionMissing = strings.TrimSpace(originalMeta.Session) == "" || !fsx.Exists(filepath.Join(taskx.RuntimeRoot(), ".ai", "sessions", originalMeta.Session, "status.json"))
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
		if err := newTask(id, opts.AllowUnrelatedDirty); err != nil {
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
	attemptID, err := startManagedAttempt(id, meta)
	if err != nil {
		return err
	}
	workItemID, err := managedAttemptWorkItem(id, attemptID)
	if err != nil {
		return err
	}
	lineage := agentLineage{TaskID: id, AttemptID: string(attemptID), WorkItemID: string(workItemID), ParentTask: id, SessionID: meta.Session, ChildSession: meta.Session, ParentThread: status.Backend.ThreadID, Handoff: handoff, HandoffStatus: "pending", ParentState: "active", ChildState: "starting", StartedAt: time.Now().UTC().Format(time.RFC3339), Status: "starting"}
	lineage.SourcePath = handoff
	lineage.SourceSession = meta.Session
	lineage.SourceThread = status.Backend.ThreadID
	lineage.HandoffCreatedAt = lineage.StartedAt
	if b, readErr := os.ReadFile(handoff); readErr == nil {
		digest := sha256.Sum256(b)
		lineage.HandoffHash = hex.EncodeToString(digest[:])
	}
	if err := writeLineage(id, lineage); err != nil {
		if !opts.Supervised {
			_, _ = workflow.NewStore("").RecordAttemptOutcome(workflow.TaskID(id), attemptID, false)
		}
		return err
	}
	if err := recordSessionHandoff(store, meta.Session, lineage); err != nil {
		if !opts.Supervised {
			_, _ = workflow.NewStore("").RecordAttemptOutcome(workflow.TaskID(id), attemptID, false)
		}
		return err
	}
	prompt := fmt.Sprintf("Continue Task %s.\n\nSession: %s\n\nRead the handoff at %s and referenced artifacts before taking action. Preserve the existing Task and worktree; report validation when done.\n\nTask context:\n%s", id, meta.Session, handoff, readTaskGoal(id))
	if args[0] == "chat" {
		result, runErr := session.ExecuteInteractiveWithOverrides(context.Background(), store, meta.Session, "handoff", prompt, opts.Provider, opts.Model, true)
		if err := finalizeInteractiveAgent(id, metaPath, meta, store, lineage, attemptID, result, runErr, opts.Supervised); err != nil {
			return err
		}
		fmt.Printf("Task %s chat completed: %s\n", id, lineage.ChildThread)
		return nil
	}
	result, err := session.ExecuteTurnWithOverrides(context.Background(), store, meta.Session, "handoff", prompt, opts.Provider, opts.Model, true)
	if err != nil {
		if !opts.Supervised {
			_, _ = workflow.NewStore("").RecordAttemptOutcome(workflow.TaskID(id), attemptID, false)
		}
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
	state := workflow.RuntimeState{}
	if !opts.Supervised {
		state, err = workflow.NewStore("").RecordAttemptOutcome(workflow.TaskID(id), attemptID, true)
		if err != nil {
			return err
		}
	}
	if err := recordSessionHandoff(store, meta.Session, lineage); err != nil {
		return err
	}
	if !opts.Supervised {
		if err := reportWorkflowState(meta, state); err != nil {
			lineage.Status, lineage.Error = "failed", err.Error()
			_ = writeLineage(id, lineage)
			if createdTask {
				return rollbackNewTask(id, err)
			}
			return err
		}
	}
	if err := writeLineage(id, lineage); err != nil {
		return err
	}
	fmt.Printf("Task %s handed off: %s -> %s\n", id, lineage.ParentThread, lineage.ChildThread)
	return nil
}

func printAgentHelp() {
	fmt.Print("aiw turn|chat - Managed Task Agent execution\n\n" +
		"Usage:\n" + "  aiw turn <task-id> [options]\n" + "  aiw chat <task-id> [options]\n\n" + "Options:\n" + "  --handoff PATH              Use an explicit handoff file.\n" + "  --provider NAME             Override the LLM provider for this call.\n" + "  --model MODEL               Override the LLM model for this call.\n" + "  --isolated                  Create or use an isolated worktree.\n" + "  --takeover                  Take over a running Session.\n" + "  --allow-unrelated-dirty    Allow unrelated dirty paths during creation.\n" + "  --yes                       Confirm normalized Task IDs.\n\n" + "The managed workflow normally prepares the handoff and workspace before\n" + "calling this command.\n")
}

type agentOptions struct {
	Handoff             string
	Provider            string
	Model               string
	Takeover            bool
	Isolated            bool
	AllowUnrelatedDirty bool
	Yes                 bool
	Supervised          bool
}

func parseAgentOptions(args []string) (agentOptions, error) {
	var opts agentOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--takeover":
			opts.Takeover = true
		case "--isolated":
			opts.Isolated = true
		case "--allow-unrelated-dirty", "--allow-dirty":
			opts.AllowUnrelatedDirty = true
		case "--yes":
			opts.Yes = true
		case "--supervised":
			opts.Supervised = true
		case "--handoff":
			if i+1 >= len(args) {
				return opts, errors.New("--handoff requires a path")
			}
			i++
			opts.Handoff = args[i]
		case "--provider":
			if i+1 >= len(args) {
				return opts, errors.New("--provider requires a name")
			}
			i++
			opts.Provider = args[i]
		case "--model":
			if i+1 >= len(args) {
				return opts, errors.New("--model requires a model")
			}
			i++
			opts.Model = args[i]
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
		paths = append(paths, filepath.Join(taskx.RuntimeTaskDir(taskID), "artifacts", "handoff.md"))
	}
	if session != "" {
		paths = append(paths, filepath.Join(taskx.RuntimeRoot(), ".ai", "sessions", session, "artifacts", "handoff.md"), filepath.Join("artifacts", "handoff.md"))
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
	dir := filepath.Join(taskx.RuntimeTaskDir(id), "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "handoff.md"), b, 0o644)
}

func createTaskSession(id, worktree string) error {
	instructions := filepath.Join(taskx.RuntimeTaskDir(id), "artifacts", "instructions.md")
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
	_ = os.RemoveAll(taskx.RuntimeTaskDir(id))
	if worktreeErr != nil {
		return fmt.Errorf("%w (new Task %s was rolled back; worktree cleanup: %v)", cause, id, worktreeErr)
	}
	return fmt.Errorf("%w (new Task %s was rolled back)", cause, id)
}

func startManagedAttempt(id string, meta taskx.TaskMeta) (workflow.AttemptID, error) {
	store := workflow.NewStore("")
	state, err := store.EnsureCompatible(taskx.WorkflowRuntimeFromMeta(meta))
	if err != nil {
		return "", err
	}
	if attemptID, prepared, err := preparedManagedAttempt(id, meta, state); err != nil {
		return "", err
	} else if prepared {
		return attemptID, nil
	}
	for _, attempt := range state.Attempts {
		if attempt.State == workflow.AttemptRunning && attempt.SessionID == meta.Session {
			return attempt.ID, nil
		}
	}
	if workflow.HasMappedWorkItems(state) {
		item, err := workflow.SelectReadyMappedWorkItem(state)
		if err != nil {
			return "", fmt.Errorf("managed Task %s has no executable mapped Work Item: %w", id, err)
		}
		attemptID := workflow.AttemptID(fmt.Sprintf("attempt-%d", time.Now().UTC().UnixNano()))
		_, err = store.StartAttempt(workflow.TaskID(id), workflow.Attempt{ID: attemptID, WorkItemID: item.ID, SessionID: meta.Session, Workspace: meta.Worktree})
		return attemptID, err
	}
	fmt.Fprintln(os.Stderr, "workflow compatibility fallback: no mapped checklist Work Item exists; using legacy managed execution item")
	workItemID := workflow.WorkItemID("wi-0001")
	if len(state.WorkItems) == 0 {
		state, workItemID, err = store.EnsureLegacyManagedWorkItem(workflow.TaskID(id))
		if err != nil {
			return "", err
		}
	}
	for _, item := range state.WorkItems {
		if item.State == workflow.WorkItemReady {
			workItemID = item.ID
			attemptID := workflow.AttemptID(fmt.Sprintf("attempt-%d", time.Now().UTC().UnixNano()))
			_, err := store.StartAttempt(workflow.TaskID(id), workflow.Attempt{ID: attemptID, WorkItemID: workItemID, SessionID: meta.Session, Workspace: meta.Worktree})
			return attemptID, err
		}
	}
	return "", fmt.Errorf("managed Task %s has no ready Work Item", id)
}

// preparedManagedAttempt validates an advance-created request against the
// current managed binding before the existing agent path consumes it.
func preparedManagedAttempt(id string, meta taskx.TaskMeta, state workflow.RuntimeState) (workflow.AttemptID, bool, error) {
	request := state.Automation.PreparedRequest
	if request == nil || state.WriteLease == nil || state.WriteLease.AttemptID != request.AttemptID {
		return "", false, nil
	}
	if request.TaskID != workflow.TaskID(id) || request.SessionID != meta.Session || request.Workspace != meta.Worktree {
		return "", true, fmt.Errorf("prepared agent request is stale or conflicts with the Task Session/workspace binding")
	}
	for _, attempt := range state.Attempts {
		if attempt.ID == request.AttemptID && attempt.WorkItemID == request.WorkItemID && attempt.State == workflow.AttemptRunning {
			return attempt.ID, true, nil
		}
	}
	return "", true, fmt.Errorf("prepared agent request references a stale Attempt %s", request.AttemptID)
}

func managedAttemptWorkItem(id string, attemptID workflow.AttemptID) (workflow.WorkItemID, error) {
	state, err := workflow.NewStore("").Load(workflow.TaskID(id))
	if err != nil {
		return "", err
	}
	for _, attempt := range state.Attempts {
		if attempt.ID == attemptID {
			return attempt.WorkItemID, nil
		}
	}
	return "", fmt.Errorf("managed Attempt %s was not persisted", attemptID)
}

func recordSessionHandoff(store *session.Store, sessionID string, lineage agentLineage) error {
	_, err := store.Update(sessionID, func(status *session.Status) error {
		status.Task = &session.ManagedExecutionRef{
			SchemaVersion:    1,
			TaskID:           lineage.TaskID,
			WorkItemID:       lineage.WorkItemID,
			AttemptID:        lineage.AttemptID,
			Handoff:          lineage.Handoff,
			HandoffHash:      lineage.HandoffHash,
			HandoffStatus:    lineage.HandoffStatus,
			HandoffCreatedAt: lineage.HandoffCreatedAt,
			ConsumedHash:     lineage.ConsumedHash,
			ParentThread:     lineage.ParentThread,
			ChildThread:      lineage.ChildThread,
			ConsumedAt:       lineage.ConsumedAt,
			ConsumerThread:   lineage.ConsumerThread,
			ParentState:      lineage.ParentState,
			ChildState:       lineage.ChildState,
		}
		return nil
	})
	return err
}

// finalizeInteractiveAgent closes every durable execution boundary after the
// provider-owned process returns. The provider error is preserved while the
// Session reference, Attempt/lease, workflow projection, and lineage are
// still recorded for failed and interrupted exits.
func finalizeInteractiveAgent(id, metaPath string, meta taskx.TaskMeta, store *session.Store, lineage agentLineage, attemptID workflow.AttemptID, result session.TurnResult, runErr error, supervised bool) error {
	succeeded := runErr == nil && result.ExitCode == 0
	now := time.Now().UTC().Format(time.RFC3339)
	lineage.ChildThread = result.ThreadID
	lineage.HandoffStatus = "consumed"
	lineage.ConsumedAt = now
	lineage.ConsumerThread = result.ThreadID
	lineage.ConsumedHash = lineage.HandoffHash
	lineage.ParentState = "handed-off"
	lineage.CompletedAt = now
	if succeeded {
		lineage.ChildState, lineage.Status = "completed", "completed"
	} else {
		lineage.ChildState, lineage.Status = "failed", "failed"
		if runErr != nil {
			lineage.Error = runErr.Error()
		} else {
			lineage.Error = fmt.Sprintf("interactive provider exited with code %d", result.ExitCode)
		}
	}

	var errs []error
	if err := recordSessionHandoff(store, meta.Session, lineage); err != nil {
		errs = append(errs, fmt.Errorf("record interactive Session state: %w", err))
	}
	if !supervised {
		state, err := workflow.NewStore("").RecordAttemptOutcome(workflow.TaskID(id), attemptID, succeeded)
		if err != nil {
			errs = append(errs, fmt.Errorf("record interactive Attempt outcome: %w", err))
		} else if err := reportWorkflowState(meta, state); err != nil {
			errs = append(errs, fmt.Errorf("render interactive workflow state: %w", err))
		}
	}
	if err := writeLineage(id, lineage); err != nil {
		errs = append(errs, fmt.Errorf("record interactive lineage: %w", err))
	}
	if runErr != nil {
		errs = append([]error{runErr}, errs...)
	} else if !succeeded {
		errs = append(errs, fmt.Errorf("interactive provider exited with code %d", result.ExitCode))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func validateTaskBindings(id string, meta taskx.TaskMeta) error {
	entries, err := os.ReadDir(taskx.RuntimeTasksPath())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == id || entry.Name() == "archive" {
			continue
		}
		other, readErr := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(entry.Name()))
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
	path, err := workflowChecklistPath(id)
	if err != nil {
		return "(see tasks.md)"
	}
	b, err := os.ReadFile(path)
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
	path := filepath.Join(taskx.RuntimeTaskDir(id), "agent-lineage.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readLineage(id string) ([]byte, error) {
	return os.ReadFile(filepath.Join(taskx.RuntimeTaskDir(id), "agent-lineage.json"))
}
