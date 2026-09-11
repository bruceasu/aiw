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

	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

const maxFocusedTestOutputBytes = 64 * 1024

// FocusedTestRun is the immutable execution input resolved by the controlled
// runner. It has no field for an Agent-provided command: argv and directory
// always come from the active Verification Plan.
//
// It carries only Plan-derived execution values; callers must use the
// controlled execution seam below rather than invoking a process directly.
type FocusedTestRun struct {
	TaskID             workflow.TaskID
	WorkItemID         workflow.WorkItemID
	AttemptID          workflow.AttemptID
	PlanDigest         string
	CheckID            string
	Argv               []string
	Worktree           string
	WorkingDirectory   string
	Timeout            time.Duration
	ExpectedExitCode   int
	EvidenceDestination string
	NetworkPolicy       string
	AllowedEnvironment []string
}

// FocusedTestNetworkEnforcer is the platform-specific boundary used by the
// controlled executor. Implementations must establish a technically enforced
// no-network boundary for this exact run before a child process is created.
// An advisory environment variable or command flag is not sufficient.
type FocusedTestNetworkEnforcer interface {
	EnforceNoNetwork(FocusedTestRun) error
}

// FocusedTestProcessRunner is the sole seam that may create the selected
// child process. Tests can replace it without granting a test-agent command
// construction authority.
type FocusedTestProcessRunner interface {
	Run(context.Context, FocusedTestRun) ([]byte, int, error)
}

type osFocusedTestProcessRunner struct{}

func (osFocusedTestProcessRunner) Run(ctx context.Context, run FocusedTestRun) ([]byte, int, error) {
	command := exec.CommandContext(ctx, run.Argv[0], run.Argv[1:]...)
	command.Dir = run.WorkingDirectory
	command.Env = focusedTestEnvironment(run.AllowedEnvironment)
	output, err := command.CombinedOutput()
	if err == nil {
		return output, 0, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return output, exitError.ExitCode(), nil
	}
	return output, -1, err
}

// FocusedTestExecution is the durable outcome returned after the result
// artifact and terminal Workflow Core Evidence have been recorded.
type FocusedTestExecution struct {
	Result        workflow.FocusedTestResult
	EvidenceID    workflow.EvidenceID
	EvidenceState workflow.EvidenceState
	RuntimeState  workflow.RuntimeState
}

// ResolveFocusedTestRun loads the authored Plan and persisted test-agent
// selection, then resolves a single immutable check inside the owning Attempt
// worktree. It never falls back to the primary workspace and never starts a
// process.
func ResolveFocusedTestRun(meta taskx.TaskMeta, state workflow.RuntimeState, attemptID workflow.AttemptID) (FocusedTestRun, error) {
	if strings.TrimSpace(string(attemptID)) == "" {
		return FocusedTestRun{}, fmt.Errorf("focused-test attempt ID is required")
	}
	if state.Task.ID != workflow.TaskID(meta.ID) {
		return FocusedTestRun{}, fmt.Errorf("focused-test runtime Task ID does not match Task metadata")
	}
	planData, err := os.ReadFile(taskx.VerificationPlanPath(meta.ID))
	if err != nil {
		return FocusedTestRun{}, fmt.Errorf("read verification plan: %w", err)
	}
	plan, err := workflow.ParseVerificationPlan(planData)
	if err != nil {
		return FocusedTestRun{}, fmt.Errorf("parse verification plan: %w", err)
	}
	if plan.TaskID != workflow.TaskID(meta.ID) {
		return FocusedTestRun{}, fmt.Errorf("verification plan Task ID does not match Task metadata")
	}

	selectionData, err := os.ReadFile(taskx.VerificationSelectionPath(meta.ID))
	if err != nil {
		return FocusedTestRun{}, fmt.Errorf("read focused-test selection: %w", err)
	}
	selection, err := workflow.ParseVerificationPlanSelection(selectionData, plan)
	if err != nil {
		return FocusedTestRun{}, fmt.Errorf("parse focused-test selection: %w", err)
	}

	attempt, ok := focusedTestAttempt(state.Attempts, attemptID)
	if !ok {
		return FocusedTestRun{}, fmt.Errorf("focused-test attempt %s is not owned by Task %s", attemptID, meta.ID)
	}
	if strings.TrimSpace(attempt.Workspace) == "" {
		return FocusedTestRun{}, fmt.Errorf("focused-test attempt %s has no worktree", attemptID)
	}
	check, ok := focusedTestCheck(plan, selection.CheckID)
	if !ok {
		return FocusedTestRun{}, fmt.Errorf("focused-test selection check_id %q is not declared by the active plan", selection.CheckID)
	}
	worktree, err := resolveFocusedTestWorktree(attempt.Workspace)
	if err != nil {
		return FocusedTestRun{}, err
	}
	workingDirectory, err := resolveFocusedTestWorktreeDirectory(worktree, check.WorkingDirectory)
	if err != nil {
		return FocusedTestRun{}, err
	}
	digest, err := plan.Digest()
	if err != nil {
		return FocusedTestRun{}, fmt.Errorf("digest verification plan: %w", err)
	}
	return FocusedTestRun{
		TaskID:              workflow.TaskID(meta.ID),
		WorkItemID:          attempt.WorkItemID,
		AttemptID:           attempt.ID,
		PlanDigest:          digest,
		CheckID:             check.CheckID,
		Argv:                append([]string(nil), check.Argv...),
		Worktree:            worktree,
		WorkingDirectory:    workingDirectory,
		Timeout:             time.Duration(check.TimeoutSeconds) * time.Second,
		ExpectedExitCode:    check.ExpectedExitCode,
		EvidenceDestination: check.EvidenceDestination,
		NetworkPolicy:       check.NetworkPolicy,
		AllowedEnvironment:  append([]string(nil), check.AllowedEnvironment...),
	}, nil
}

// EnsureFocusedTestNetworkEnforcement fail-closes the controlled path before
// process creation. A missing or failing enforcer opens a Core-owned
// authorization Gate and returns executable=false; callers must project that
// state and stop rather than invoking a process runner.
func EnsureFocusedTestNetworkEnforcement(store *workflow.Store, run FocusedTestRun, enforcer FocusedTestNetworkEnforcer) (state workflow.RuntimeState, executable bool, err error) {
	if store == nil {
		return workflow.RuntimeState{}, false, fmt.Errorf("focused-test workflow store is required")
	}
	if run.NetworkPolicy != workflow.NetworkPolicyDeny {
		return workflow.RuntimeState{}, false, fmt.Errorf("focused-test check %q does not require network: deny", run.CheckID)
	}
	if enforcer == nil {
		state, err = store.OpenFocusedTestNetworkEnforcementGate(run.TaskID, run.PlanDigest, run.WorkItemID, "the selected runtime has no no-network enforcer")
		return state, false, err
	}
	if enforceErr := enforcer.EnforceNoNetwork(run); enforceErr != nil {
		state, err = store.OpenFocusedTestNetworkEnforcementGate(run.TaskID, run.PlanDigest, run.WorkItemID, enforceErr.Error())
		return state, false, err
	}
	return workflow.RuntimeState{}, true, nil
}

// ExecuteFocusedTest runs one already-resolved immutable Plan entry. It
// activates and verifies the exact authorization, establishes the no-network
// boundary, records pending intent, and then persists a bounded result before
// finalizing command Evidence through Workflow Core.
func ExecuteFocusedTest(store *workflow.Store, run FocusedTestRun, enforcer FocusedTestNetworkEnforcer, process FocusedTestProcessRunner) (FocusedTestExecution, error) {
	if store == nil {
		return FocusedTestExecution{}, fmt.Errorf("focused-test workflow store is required")
	}
	if process == nil {
		return FocusedTestExecution{}, fmt.Errorf("focused-test process runner is required")
	}
	worktreeRelativeDirectory, err := filepath.Rel(run.Worktree, run.WorkingDirectory)
	if err != nil || worktreeRelativeDirectory == ".." || strings.HasPrefix(worktreeRelativeDirectory, ".."+string(filepath.Separator)) || filepath.IsAbs(worktreeRelativeDirectory) {
		return FocusedTestExecution{}, fmt.Errorf("focused-test working directory escapes Attempt worktree")
	}
	state, err := store.ActivateFocusedTestPlan(run.TaskID, run.PlanDigest)
	if err != nil {
		return FocusedTestExecution{}, err
	}
	if state.FocusedTestAuthorizationState(run.PlanDigest) != workflow.FocusedTestAuthorizationAuthorized {
		return FocusedTestExecution{RuntimeState: state}, fmt.Errorf("focused-test requires authorization for the active verification plan")
	}
	state, executable, err := EnsureFocusedTestNetworkEnforcement(store, run, enforcer)
	if err != nil {
		return FocusedTestExecution{}, err
	}
	if !executable {
		return FocusedTestExecution{RuntimeState: state}, fmt.Errorf("focused-test runtime cannot enforce network: deny")
	}

	evidenceID := focusedTestEvidenceID(run)
	pending := workflow.Evidence{ID: evidenceID, WorkItemID: run.WorkItemID, Kind: workflow.EvidenceCommand, State: workflow.EvidencePending, Reference: "focused-test pending: " + run.CheckID}
	if _, err := store.RecordEvidence(run.TaskID, pending); err != nil {
		return FocusedTestExecution{}, fmt.Errorf("record focused-test pending evidence: %w", err)
	}

	startedAt := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), run.Timeout)
	output, exitCode, runErr := process.Run(ctx, run)
	timedOut := ctx.Err() == context.DeadlineExceeded
	cancel()
	finishedAt := time.Now().UTC()
	result := workflow.FocusedTestResult{
		SchemaVersion: workflow.VerificationPlanSchemaVersion,
		PlanDigest: run.PlanDigest, CheckID: run.CheckID, Argv: append([]string(nil), run.Argv...),
		WorkingDirectory: filepath.ToSlash(worktreeRelativeDirectory), StartedAt: startedAt.Format(time.RFC3339), FinishedAt: finishedAt.Format(time.RFC3339), TimedOut: timedOut,
	}
	if exitCode >= 0 {
		result.ExitCode = &exitCode
	}
	if runErr != nil {
		result.RunnerError = runErr.Error()
	}
	if err := writeFocusedTestResult(string(run.TaskID), &result, output); err != nil {
		return FocusedTestExecution{}, fmt.Errorf("persist focused-test result: %w", err)
	}

	evidenceState := workflow.EvidenceFailed
	if !timedOut && runErr == nil && exitCode == run.ExpectedExitCode {
		evidenceState = workflow.EvidencePassed
	}
	state, err = store.FinalizeFocusedTestEvidence(run.TaskID, evidenceID, evidenceState, taskx.FocusedTestResultPath(string(run.TaskID)))
	if err != nil {
		return FocusedTestExecution{}, fmt.Errorf("finalize focused-test evidence: %w", err)
	}
	execution := FocusedTestExecution{Result: result, EvidenceID: evidenceID, EvidenceState: evidenceState, RuntimeState: state}
	if evidenceState == workflow.EvidenceFailed {
		return execution, fmt.Errorf("focused-test check %q failed", run.CheckID)
	}
	return execution, nil
}

func focusedTestEnvironment(allowed []string) []string {
	environment := make([]string, 0, len(allowed))
	for _, name := range allowed {
		if value, ok := os.LookupEnv(name); ok {
			environment = append(environment, name+"="+value)
		}
	}
	return environment
}

func focusedTestEvidenceID(run FocusedTestRun) workflow.EvidenceID {
	sum := sha256.Sum256([]byte(string(run.AttemptID) + "\x00" + run.PlanDigest + "\x00" + run.CheckID))
	return workflow.EvidenceID("focused-test-" + hex.EncodeToString(sum[:8]))
}

func writeFocusedTestResult(taskID string, result *workflow.FocusedTestResult, output []byte) error {
	outputPath := taskx.FocusedTestOutputPath(taskID)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create focused-test artifact directory: %w", err)
	}
	sum := sha256.Sum256(output)
	truncated := len(output) > maxFocusedTestOutputBytes
	storedOutput := output
	if truncated {
		storedOutput = output[:maxFocusedTestOutputBytes]
	}
	if err := os.WriteFile(outputPath, storedOutput, 0o644); err != nil {
		return fmt.Errorf("write focused-test output: %w", err)
	}
	result.Output = workflow.BoundedOutputReference{Path: outputPath, ByteCount: len(storedOutput), Truncated: truncated, SHA256: hex.EncodeToString(sum[:])}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode focused-test result: %w", err)
	}
	if err := os.WriteFile(taskx.FocusedTestResultPath(taskID), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write focused-test result: %w", err)
	}
	return nil
}

func focusedTestAttempt(attempts []workflow.Attempt, id workflow.AttemptID) (workflow.Attempt, bool) {
	for _, attempt := range attempts {
		if attempt.ID == id {
			return attempt, true
		}
	}
	return workflow.Attempt{}, false
}

func focusedTestCheck(plan workflow.VerificationPlan, checkID string) (workflow.VerificationCheck, bool) {
	for _, check := range plan.Checks {
		if check.CheckID == checkID {
			return check, true
		}
	}
	return workflow.VerificationCheck{}, false
}

func resolveFocusedTestWorktreeDirectory(worktree, relativeDirectory string) (string, error) {
	worktreePath, err := resolveFocusedTestWorktree(worktree)
	if err != nil {
		return "", err
	}
	directoryPath, err := filepath.EvalSymlinks(filepath.Join(worktreePath, relativeDirectory))
	if err != nil {
		return "", fmt.Errorf("resolve focused-test working directory: %w", err)
	}
	relative, err := filepath.Rel(worktreePath, directoryPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("focused-test working directory escapes Attempt worktree")
	}
	return directoryPath, nil
}

func resolveFocusedTestWorktree(worktree string) (string, error) {
	worktreePath, err := filepath.Abs(worktree)
	if err != nil {
		return "", fmt.Errorf("resolve focused-test Attempt worktree: %w", err)
	}
	worktreePath, err = filepath.EvalSymlinks(worktreePath)
	if err != nil {
		return "", fmt.Errorf("resolve focused-test Attempt worktree: %w", err)
	}
	return worktreePath, nil
}
