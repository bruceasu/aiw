package task

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"aiw/internal/session"
	"aiw/internal/taskx"
	"aiw/internal/workflow"
)

// compileSupervisedOutcome runs the frozen plan only after the supervisor has
// checked the bound Session outcome. Repairs return to the normal dispatch path
// so they retain scoped Git access, AI selection, and the owning Attempt.
func compileSupervisedOutcome(id string, store *workflow.Store, request *workflow.PreparedAgentRequest, outcome workflow.SupervisedOutcome, runner workflow.CompileCommandRunner) (workflow.SupervisedOutcome, bool, error) {
	result := request.CompilerResult
	if request.Compile != nil {
		result = request.Compile.Result
	}
	if result == nil {
		if err := prepareSupervisedCompilerRequest(id, store, request, outcome); err != nil {
			return outcome, false, err
		}
		compiled, _, err := store.RunCompiler(context.Background(), request.TaskID, *request.Compile.Request, runner)
		if compiled.RequestID == "" {
			if err == nil {
				err = fmt.Errorf("compiler returned no result")
			}
			return outcome, false, err
		}
		result = &compiled
	}
	if result.Status == workflow.ActorResultAccepted {
		return outcome, false, nil
	}
	if result.Status == workflow.ActorResultBlocked {
		return workflow.SupervisedOutcome{Kind: workflow.SupervisedOutcomeBlocked, BlockedCategory: workflow.BlockedOutcomeValidation, Detail: result.Summary, EvidenceReference: result.Diagnostics.Path}, false, nil
	}
	state, err := store.Load(request.TaskID)
	if err != nil {
		return outcome, false, err
	}
	current := state.Automation.PreparedRequest
	if current == nil || current.AttemptID != request.AttemptID || current.WorkItemID != request.WorkItemID || state.WriteLease == nil || state.WriteLease.AttemptID != request.AttemptID {
		return outcome, false, fmt.Errorf("compiler repair lost its owning Attempt")
	}
	failures := 0
	for _, item := range state.WorkItems {
		if item.ID == request.WorkItemID {
			failures = item.CompileFailureCount
		}
	}
	if failures >= workflow.MaxConsecutiveCompileFailures || (current.Compile != nil && current.Compile.Failures >= workflow.CompilerFailureLimit) {
		return workflow.SupervisedOutcome{Kind: workflow.SupervisedOutcomeBlocked, BlockedCategory: workflow.BlockedOutcomeValidation, Detail: "three consecutive compile failures; inspect diagnostics and repair before reopening this Work Item", EvidenceReference: result.Diagnostics.Path}, false, nil
	}
	if err := ensureSupervisedCompilePlan(store, current); err != nil {
		return outcome, false, err
	}
	handoff, err := writeCompilerRepairHandoff(id, current, result, failures)
	if err != nil {
		return outcome, false, err
	}
	status, err := session.NewStore("").Load(request.SessionID)
	if err != nil {
		return outcome, false, err
	}
	// Clear both recovery views together. The repaired Session must produce a
	// fresh compiler request, while the frozen plan and counters stay intact.
	current.Compile.Request, current.Compile.Result = nil, nil
	current.Compile.RepairPending = true
	current.CompilerResult = nil
	current.DispatchedAt = ""
	current.ExpectedSessionTurn = status.Session.LastTurn + 1
	current.Handoff = handoff
	_, err = store.RecordAutomation(request.TaskID, state.Automation.PlanFingerprint, workflow.AutomationCursor{Result: string(workflow.RunnerPrepared), Detail: "compiler repair"}, current)
	return outcome, err == nil, err
}

func ensureSupervisedCompilePlan(store *workflow.Store, request *workflow.PreparedAgentRequest) error {
	if request.Compile == nil {
		request.Compile = &workflow.SupervisedCompileState{}
		if err := saveSupervisedCompileRequest(store, request); err != nil {
			return err
		}
	}
	if request.Compile.Plan != nil {
		return nil
	}
	err := errors.New("compile-plan-missing: regenerate routing and prepare a fresh supervised request")
	_, gateErr := store.OpenCompilerGate(request.TaskID, request.WorkItemID, "compile-plan-missing", err.Error())
	return errors.Join(err, gateErr)
}

func prepareSupervisedCompilerRequest(id string, store *workflow.Store, request *workflow.PreparedAgentRequest, outcome workflow.SupervisedOutcome) error {
	if err := ensureSupervisedCompilePlan(store, request); err != nil {
		return err
	}
	compile := request.Compile
	if compile.Request != nil && !compile.RepairPending {
		return nil
	}
	root := request.Workspace
	if !filepath.IsAbs(root) {
		root = filepath.Join(taskx.RuntimeRoot(), filepath.FromSlash(root))
	}
	meta, err := taskx.ReadTaskMeta(taskx.ResolveTaskMetaPath(id))
	if err != nil {
		return err
	}
	if meta.Worktree != request.Workspace {
		return fmt.Errorf("compiler workspace differs from the recorded Task worktree")
	}
	environment, err := preflightSupervisedGitWorkspace(meta)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(filepath.Join(taskx.RuntimeRoot(), filepath.FromSlash(outcome.EvidenceReference)))
	if err != nil {
		return err
	}
	digest := sha256.Sum256(content)
	// Include all dirty paths: another edit to an already modified file must
	// still select its target. Use the same scoped Git trust as Agent dispatch.
	snapshot, err := (workflow.GitChangedPathChecker{Environment: environment}).Snapshot(root)
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(snapshot))
	for path := range snapshot {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	compile.Request = &workflow.CompilerRequest{
		ActorRequest: workflow.ActorRequest{SchemaVersion: workflow.ActorContractSchemaVersion, ID: fmt.Sprintf("compiler-%s-%d", request.AttemptID, request.ExpectedSessionTurn), Actor: workflow.ActorCompiler, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, Workspace: request.Workspace, PreparedAt: time.Now().UTC().Format(time.RFC3339Nano)},
		ImplementationReport: workflow.ActorReference{Kind: "implementation-report", Path: outcome.EvidenceReference, SHA256: hex.EncodeToString(digest[:])},
		Plan: compile.Plan, PlanReference: compile.PlanReference, ChangedPaths: paths, CompileRoot: root,
	}
	compile.Result, compile.RepairPending = nil, false
	request.CompilerResult = nil
	return saveSupervisedCompileRequest(store, request)
}

func saveSupervisedCompileRequest(store *workflow.Store, request *workflow.PreparedAgentRequest) error {
	state, err := store.Load(request.TaskID)
	if err != nil {
		return err
	}
	if state.WriteLease == nil || state.WriteLease.AttemptID != request.AttemptID {
		return fmt.Errorf("supervised compiler lost its owning Attempt")
	}
	current := state.Automation.PreparedRequest
	if current == nil || current.AttemptID != request.AttemptID || current.WorkItemID != request.WorkItemID {
		return fmt.Errorf("supervised compiler lost its prepared request")
	}
	current.Compile, current.CompilerResult = request.Compile, request.CompilerResult
	_, err = store.RecordAutomation(request.TaskID, state.Automation.PlanFingerprint, state.Automation.Cursor, current)
	return err
}

func writeCompilerRepairHandoff(id string, request *workflow.PreparedAgentRequest, result *workflow.CompilerResult, failures int) (string, error) {
	base := filepath.Join(taskx.RuntimeTaskDir(id), "artifacts", "handoff.md")
	content, err := os.ReadFile(base)
	if err != nil {
		return "", err
	}
	diagnostics := filepath.Join(taskx.RuntimeTaskDir(id), filepath.FromSlash(result.Diagnostics.Path))
	content = append(content, []byte(fmt.Sprintf("\n## Compiler repair\n\nStay in Attempt %s and Work Item %s. Read diagnostics at %s. Fix only these compile failures. Do not start another Attempt or run compilers or tests; the supervisor runs the frozen Compile Plan. Return a structured outcome. Consecutive compile failures: %d of %d.\n", request.AttemptID, request.WorkItemID, diagnostics, failures, workflow.MaxConsecutiveCompileFailures))...)
	path := filepath.Join(filepath.Dir(base), "compiler-repair.md")
	return path, os.WriteFile(path, content, 0o644)
}

func hasPendingSupervisedCompile(request *workflow.PreparedAgentRequest) bool {
	if request.CompilerResult != nil {
		return true
	}
	compile := request.Compile
	return compile != nil && (compile.Plan == nil || compile.Request != nil || compile.Result != nil || compile.RepairPending)
}