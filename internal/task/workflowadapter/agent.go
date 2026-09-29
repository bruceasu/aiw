package workflowadapter

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
	"unicode/utf8"

	"aiw/internal/fsx"
	"aiw/internal/gitx"
	"aiw/internal/session"
	"aiw/internal/task"
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
	return RunTaskAgentWithEnvironment(args, nil)
}

// RunTaskAgentWithEnvironment is the Task-owned Agent execution entry used by
// Workflow. The root Task CLI no longer exposes turn/chat dispatch.
func RunTaskAgentWithEnvironment(args []string, environment []string) error {
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
	if !SafeID(id) {
		candidate := normalizeID(id)
		fmt.Printf("Invalid task ID %q. Candidate mapping: %s (use --yes to accept):\n", id, candidate)
		if !opts.Yes || !confirm("Create the task with this candidate name? [y/N] ") {
			return errors.New("task creation refused: invalid task ID requires confirmation")
		}
		id = candidate
	}
	metaPath := task.ResolveTaskMetaPath(id)
	existing := fsx.Exists(task.RuntimeTaskDir(id))
	var meta task.TaskMeta
	sessionMissing := false
	createdTask := false
	if existing {
		originalMeta, readErr := task.ReadTaskMeta(metaPath)
		if readErr != nil {
			return fmt.Errorf("read task metadata: %w", readErr)
		}
		sessionMissing = strings.TrimSpace(originalMeta.Session) == "" || !fsx.Exists(filepath.Join(task.RuntimeRoot(), ".ai", "sessions", originalMeta.Session, "status.json"))
		if err := task.EnsureTaskMeta(id); err != nil {
			return fmt.Errorf("repair task metadata: %w", err)
		}
		meta, err = task.ReadTaskMeta(metaPath)
		if err != nil {
			return fmt.Errorf("read task metadata: %w", err)
		}
	} else {
		handoff, source, err := resolveHandoff(opts.Handoff, "", id)
		if err != nil {
			return err
		}
		printAgentPlan(id, "create", "", "", "", source)
		if err := createTaskForAgent(id, opts.AllowUnrelatedDirty); err != nil {
			return fmt.Errorf("create task: %w", err)
		}
		createdTask = true
		metaPath = task.ResolveTaskMetaPath(id)
		meta, err = task.ReadTaskMeta(metaPath)
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
			meta, err = task.ReadTaskMeta(metaPath)
			if err != nil {
				return err
			}
		}
		meta.Session = id
		if err := task.WriteTaskMeta(metaPath, meta); err != nil {
			return rollbackNewTask(id, err)
		}
		if err := createTaskSession(id, meta.Worktree); err != nil {
			return rollbackNewTask(id, fmt.Errorf("create session: %w", err))
		}
	}
	if strings.TrimSpace(meta.Session) == "" {
		meta.Session = id
		sessionMissing = true
		if err := task.WriteTaskMeta(metaPath, meta); err != nil {
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
		meta, err = task.ReadTaskMeta(metaPath)
		if err != nil {
			return err
		}
	}
	kind := ResolveWorkspaceKind(meta)
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
	if opts.Supervised {
		prompt += "\n\nThe supervisor owns compile-only validation and its bounded repair loop. Implement the selected work, update its checkbox, and report your structured outcome; the supervisor will compile before accepting it. Do not run tests."
		instruction, instructionErr := supervisedWorkItemInstruction(id, workItemID, environment)
		if instructionErr != nil {
			return instructionErr
		}
		prompt += instruction
		prompt += "\n\nWhen you finish, return exactly one JSON object (no Markdown) with outcome=completed, blocked, or no-progress; include detail and, for blocked, blocked_category=workspace-access, authorization, dependency, validation, or unknown. Do not claim completed unless you updated the authored checklist."
	}
	if args[0] == "chat" {
		result, runErr := session.ExecuteInteractiveWithOverridesAndEnvironment(context.Background(), store, meta.Session, "handoff", prompt, opts.Provider, opts.Model, true, environment)
		if err := finalizeInteractiveAgent(id, metaPath, meta, store, lineage, attemptID, result, runErr, opts.Supervised); err != nil {
			return err
		}
		fmt.Printf("Task %s chat completed: %s\n", id, lineage.ChildThread)
		return nil
	}
	result, err := session.ExecuteTurnWithOverridesAndEnvironment(context.Background(), store, meta.Session, "handoff", prompt, opts.Provider, opts.Model, true, environment)
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
		if err := ProjectWorkflowState(id, state); err != nil {
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

func createTaskForAgent(id string, allowUnrelatedDirty bool) error {
	if err := task.CreateIssueTask(id, "", allowUnrelatedDirty); err != nil {
		return err
	}
	meta, err := task.ReadTaskMeta(task.ResolveTaskMetaPath(id))
	if err != nil {
		return err
	}
	store := workflow.NewStore("")
	if _, err := store.EnsureCompatible(WorkflowRuntimeFromMeta(meta)); err != nil {
		return err
	}
	_, err = SyncWorkflowChecklist(id, store)
	return err
}

// supervisedWorkItemInstruction supplies execution authority that is narrower
// than an Agent handoff.  A handoff can be stale after an operator repairs a
// Task binding; the current supervised request is authoritative for its own
// scoped, read-only inspection.
func supervisedWorkItemInstruction(id string, workItemID workflow.WorkItemID, environment []string) (string, error) {
	state, err := workflow.NewStore("").Load(workflow.TaskID(id))
	if err != nil {
		return "", fmt.Errorf("supervised Git preflight trust evidence: load Work Item %s: %w", workItemID, err)
	}
	var scopeReview bool
	found := false
	for _, item := range state.WorkItems {
		if item.ID != workItemID {
			continue
		}
		found = true
		scopeReview = strings.Contains(strings.ToLower(item.Title), "no unrelated changes")
		break
	}
	if !found {
		return "", fmt.Errorf("supervised Git preflight trust evidence: current Work Item %s is missing", workItemID)
	}
	trustDirectory, err := supervisedGitTrustDirectory(environment)
	if err != nil {
		return "", err
	}
	instruction := fmt.Sprintf("\n\nFor this supervised Work Item, use this exact command prefix for read-only Git inspection: `%s`. Append status, diff, staged diff, or untracked-file listing arguments. Keep the forward slashes and shell quotes as shown. The supervisor preflight approved exactly this directory. This command-local option is required because the sandbox does not inherit Git environment variables. It supersedes historical handoff notes that prohibit retrying Git. Do not change Git configuration, index, branches, or commits.", supervisedGitCommandPrefix(trustDirectory))
	if scopeReview {
		planPath, err := WorkflowChecklistPath(id)
		if err != nil { return "", err }
		instruction += fmt.Sprintf(" After collecting valid evidence, you may edit only the selected checkbox in %s; do not edit any other file.", filepath.ToSlash(planPath))
	}
	return instruction, nil
}

// SupervisedWorkItemInstruction exposes the scoped instruction builder at the
// Task-owned adapter seam.
func SupervisedWorkItemInstruction(id string, workItemID workflow.WorkItemID, environment []string) (string, error) {
	return supervisedWorkItemInstruction(id, workItemID, environment)
}

// Render shell arguments, not Go string literals: PowerShell preserves the
// doubled backslashes produced by %q instead of decoding them.
func supervisedGitCommandPrefix(directory string) string {
	directory = filepath.ToSlash(directory)
	quote := func(value string) string {
		if os.PathSeparator == '\\' {
			return "'" + strings.ReplaceAll(value, "'", "''") + "'"
		}
		return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
	}
	return "git -c " + quote("safe.directory="+directory) + " -C " + quote(directory)
}

func supervisedGitTrustDirectory(environment []string) (string, error) {
	var count, key, value string
	var countFound, keyFound, valueFound bool
	for _, entry := range environment {
		environmentKey, environmentValue, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		switch {
		case strings.EqualFold(environmentKey, "GIT_CONFIG_COUNT"):
			if countFound {
				return "", fmt.Errorf("supervised Git preflight trust evidence: duplicate GIT_CONFIG_COUNT")
			}
			count, countFound = environmentValue, true
		case strings.EqualFold(environmentKey, "GIT_CONFIG_KEY_0"):
			if keyFound {
				return "", fmt.Errorf("supervised Git preflight trust evidence: duplicate GIT_CONFIG_KEY_0")
			}
			key, keyFound = environmentValue, true
		case strings.EqualFold(environmentKey, "GIT_CONFIG_VALUE_0"):
			if valueFound {
				return "", fmt.Errorf("supervised Git preflight trust evidence: duplicate GIT_CONFIG_VALUE_0")
			}
			value, valueFound = environmentValue, true
		case strings.HasPrefix(strings.ToUpper(environmentKey), "GIT_CONFIG_KEY_") || strings.HasPrefix(strings.ToUpper(environmentKey), "GIT_CONFIG_VALUE_"):
			return "", fmt.Errorf("supervised Git preflight trust evidence: unexpected Git configuration entry %s", environmentKey)
		}
	}
	if !countFound || count != "1" {
		return "", fmt.Errorf("supervised Git preflight trust evidence: expected GIT_CONFIG_COUNT=1")
	}
	if !keyFound || key != "safe.directory" {
		return "", fmt.Errorf("supervised Git preflight trust evidence: expected GIT_CONFIG_KEY_0=safe.directory")
	}
	if !valueFound || value == "" {
		return "", fmt.Errorf("supervised Git preflight trust evidence: GIT_CONFIG_VALUE_0 is missing")
	}
	return value, nil
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
		paths = append(paths, filepath.Join(task.RuntimeTaskDir(taskID), "artifacts", "handoff.md"))
	}
	if session != "" {
		paths = append(paths, filepath.Join(task.RuntimeRoot(), ".ai", "sessions", session, "artifacts", "handoff.md"), filepath.Join("artifacts", "handoff.md"))
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
	dir := filepath.Join(task.RuntimeTaskDir(id), "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "handoff.md"), b, 0o644)
}

func createTaskSession(id, worktree string) error {
	instructions := filepath.Join(task.RuntimeTaskDir(id), "artifacts", "instructions.md")
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

// AddTaskWorktree creates the isolated Task workspace through the existing
// worktree command contract.
func AddTaskWorktree(id string) error { return addTaskWorktree(id) }

// CreateTaskSession creates the Task-owned Session binding used by Workflow.
func CreateTaskSession(id, worktree string) error { return createTaskSession(id, worktree) }

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
	_ = os.RemoveAll(task.TaskDir(id))
	_ = os.RemoveAll(task.RuntimeTaskDir(id))
	if worktreeErr != nil {
		return fmt.Errorf("%w (new Task %s was rolled back; worktree cleanup: %v)", cause, id, worktreeErr)
	}
	return fmt.Errorf("%w (new Task %s was rolled back)", cause, id)
}

func startManagedAttempt(id string, meta task.TaskMeta) (workflow.AttemptID, error) {
	store := workflow.NewStore("")
	state, err := store.EnsureCompatible(WorkflowRuntimeFromMeta(meta))
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
		// Direct turn/takeover can enter without a Supervisor sync. Refresh the
		// authored dependencies before selecting a new owner, while preserving
		// the prepared and running Attempt reuse paths above.
		state, err = SyncWorkflowChecklist(id, store)
		if err != nil {
			return "", fmt.Errorf("synchronize managed Task %s before selection: %w", id, err)
		}
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
func preparedManagedAttempt(id string, meta task.TaskMeta, state workflow.RuntimeState) (workflow.AttemptID, bool, error) {
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
func finalizeInteractiveAgent(id, metaPath string, meta task.TaskMeta, store *session.Store, lineage agentLineage, attemptID workflow.AttemptID, result session.TurnResult, runErr error, supervised bool) error {
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
		} else if err := ProjectWorkflowState(id, state); err != nil {
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

func validateTaskBindings(id string, meta task.TaskMeta) error {
	entries, err := os.ReadDir(task.RuntimeTasksPath())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == id || entry.Name() == "archive" {
			continue
		}
		other, readErr := task.ReadTaskMeta(task.ResolveTaskMetaPath(entry.Name()))
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
	path, err := WorkflowChecklistPath(id)
	if err != nil {
		return "(see the Task Feature Design)"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "(see the Task Feature Design)"
	}
	text := strings.TrimSpace(string(b))
	if len(text) > 4000 {
		end := 4000
		// Keep the byte budget without splitting a UTF-8 encoded code point.
		for end > 0 && !utf8.RuneStart(text[end]) {
			end--
		}
		text = text[:end] + "\n[truncated]"
	}
	return text
}

func writeLineage(id string, lineage agentLineage) error {
	b, err := json.MarshalIndent(lineage, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(task.RuntimeTaskDir(id), "agent-lineage.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readLineage(id string) ([]byte, error) {
	return os.ReadFile(filepath.Join(task.RuntimeTaskDir(id), "agent-lineage.json"))
}
