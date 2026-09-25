package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"aiw/internal/session"
	"aiw/internal/workflow"
)

// VerificationBoundary is supplied by a host which can enforce filesystem,
// process, memory, timeout and network restrictions for the whole process tree.
// Check must reject incomplete indirect toolchain identities and unstable
// target access. Within must pin/enforce the checked resources until done.
// There is deliberately no unsandboxed or network-waived default implementation.
type VerificationBoundary interface {
	Check(workflow.RuntimeState, workflow.StageRequest, *workflow.TestManifest) error
	// The boolean proves that all invoked children have stopped. False means
	// unknown even if the callback returned; Core must retain its write lease.
	Within(context.Context, workflow.StageRequest, *workflow.TestManifest, func(context.Context) error) (bool, error)
}

// JournaledVerificationHost executes Tester and Runner, using the existing
// Session and Core artifacts. The optional Other adapter owns Coder/Compiler;
// it cannot be used as a fallback when Tester/Runner isolation is unavailable.
type JournaledVerificationHost struct {
	ID string
	Store *workflow.Store
	Sessions *session.Store
	Boundary VerificationBoundary
	Assertions workflow.AssertionReviewAuthority
	Other ControlledVerificationHost
}

func (h *JournaledVerificationHost) Identity() string { return h.ID }

func (h *JournaledVerificationHost) VerifyAssertions(state workflow.RuntimeState, request workflow.StageRequest, receipt workflow.VerificationReceipt) error {
	if h.Assertions == nil { return errors.New("host-unavailable: independent requirement-based assertion review is missing") }
	return h.Assertions.VerifyAssertions(state, request, receipt)
}

func (h *JournaledVerificationHost) VerifyObservation(state workflow.RuntimeState, request workflow.StageRequest, result workflow.StageResult) error {
	var observer workflow.VerificationObservationHost
	if request.Phase == workflow.PhaseTester || request.Phase == workflow.PhaseTest {
		observer, _ = h.Boundary.(workflow.VerificationObservationHost)
	} else { observer, _ = h.Other.(workflow.VerificationObservationHost) }
	if observer == nil { return errors.New("host-unavailable: observation authority cannot prove this request state") }
	return observer.VerifyObservation(state, request, result)
}

func (h *JournaledVerificationHost) Check(state workflow.RuntimeState, request workflow.StageRequest, manifest *workflow.TestManifest) error {
	if h.Store == nil || h.ID == "" || h.Boundary == nil { return errors.New("host-unavailable: stable controlled execution boundary is missing") }
	if request.Phase != workflow.PhaseTester && request.Phase != workflow.PhaseTest {
		if h.Other == nil || h.Other.Identity() != h.ID { return errors.New("host-unavailable: original stage adapter is not installed") }
		return h.Other.Check(state, request, manifest)
	}
	if request.Phase == workflow.PhaseTester {
		if h.Sessions == nil { return errors.New("host-unavailable: independent Tester Session is missing") }
		status, err := h.Sessions.Load(request.SessionID)
		if err != nil { return err }
		if status.Task == nil || status.Task.TaskID != string(request.TaskID) || status.Task.WorkItemID != string(request.WorkItemID) || status.Task.AttemptID != string(request.AttemptID) || status.Workspace.Path != request.Workspace || status.Session.LastTurn+1 != request.Turn { return errors.New("Tester Session identity or next turn does not match the frozen request") }
	}
	return h.Boundary.Check(state, request, manifest)
}

func (h *JournaledVerificationHost) VerifyReceipt(request workflow.StageRequest, receipt workflow.VerificationReceipt) error {
	if request.Phase != workflow.PhaseTester && request.Phase != workflow.PhaseTest {
		if h.Other == nil { return errors.New("original stage host is unavailable") }
		return h.Other.VerifyReceipt(request, receipt)
	}
	saved, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID)
	if err != nil { return err }
	a, err := json.Marshal(saved)
	if err != nil { return err }
	b, err := json.Marshal(receipt)
	if err != nil { return err }
	if !bytes.Equal(a, b) || receipt.Result.Executor != h.ID { return errors.New("receipt differs from the trusted host journal") }
	return nil
}

func (h *JournaledVerificationHost) Observe(request workflow.StageRequest) (*workflow.DispatchObservation, *workflow.VerificationReceipt, error) {
	if request.Phase != workflow.PhaseTester && request.Phase != workflow.PhaseTest {
		if h.Other == nil { return nil, nil, errors.New("original stage host is unavailable") }
		return h.Other.Observe(request)
	}
	receipt, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID)
	if err != nil {
		// A missing receipt does not prove that the process never started.
		return nil, nil, fmt.Errorf("verification result is unknown; observe the original host without replay: %w", err)
	}
	return nil, &receipt, nil
}

func (h *JournaledVerificationHost) Start(ctx context.Context, request workflow.StageRequest, input *workflow.ExecutionInput, manifest *workflow.TestManifest) (*workflow.DispatchObservation, *workflow.VerificationReceipt, error) {
	if request.Phase != workflow.PhaseTester && request.Phase != workflow.PhaseTest {
		if h.Other == nil { return nil, nil, errors.New("original stage host is unavailable") }
		return h.Other.Start(ctx, request, input, manifest)
	}
	if _, err := h.Store.ReadVerificationReceipt(request.TaskID, request.ID); err == nil { return h.Observe(request)
	} else if !errors.Is(err, os.ErrNotExist) { return nil, nil, err }
	state, err := h.Store.Load(request.TaskID)
	if err != nil { return nil, nil, err }
	if state.Protocol == nil || state.Protocol.Stop != nil { return nil, nil, errors.New("verification is stopped") }
	if err := h.Check(state, request, manifest); err != nil { return nil, nil, err }
	receipt := workflow.VerificationReceipt{SchemaVersion: 1, Request: request, Checks: []workflow.TestCheckReceipt{}, Result: workflow.StageResult{RequestID: request.ID, TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, SessionID: request.SessionID, Turn: request.Turn, LeaseGeneration: request.LeaseGeneration, InputDigest: request.InputDigest, Executor: h.ID, Terminal: true, Status: "passed", Evidence: []workflow.ActorReference{}}}
	if request.Phase == workflow.PhaseTest {
		if manifest == nil { return nil, nil, errors.New("test manifest is missing") }
		receipt.Inputs = manifest.Inputs
	} else {
		if input == nil || input.WorkspaceInputs == nil || request.Model == nil { return nil, nil, errors.New("independent Tester context is missing") }
		receipt.Inputs = *input.WorkspaceInputs
	}
	var phaseErr error
	started := false
	terminal, boundaryErr := h.Boundary.Within(ctx, request, manifest, func(runContext context.Context) error {
		// A Stop before the first actual effect prevents this invocation.
		current, err := h.Store.Load(request.TaskID)
		if err != nil { return err }
		if current.Protocol == nil || current.Protocol.Stop != nil { return errors.New("verification is stopped before execution") }
		started = true
		if request.Phase == workflow.PhaseTest { phaseErr = h.runChecks(runContext, request, *manifest, &receipt)
		} else { phaseErr = h.runTester(runContext, request, *input, &receipt) }
		return phaseErr
	})
	// Boundary failures may leave a child process alive. Never synthesize a
	// terminal receipt or release a writer without its completion guarantee.
	if !terminal || !started { return nil, nil, errors.Join(boundaryErr, errors.New("host completion is unknown; retain the original request")) }
	if boundaryErr != nil && phaseErr == nil { phaseErr = boundaryErr }
	if phaseErr != nil {
		receipt.Result.Status, receipt.Result.FailureClass = "blocked", "infrastructure"
		if err := h.diagnostic(request, &receipt, phaseErr.Error()); err != nil { return nil, nil, err }
	}
	toolchain, err := workflow.ValidationToolchainIdentity()
	if err != nil { return nil, nil, err }
	receipt.After, err = workflow.CaptureValidationInputs(request.Workspace, receipt.Inputs.Scope, receipt.Inputs.Bindings, toolchain)
	if err != nil { return nil, nil, err }
	if err := h.Store.PersistVerificationReceipt(request.TaskID, receipt); err != nil { return nil, nil, err }
	return nil, &receipt, nil
}

type limitedTestOutput struct {
	mu sync.Mutex
	data []byte
	limit int
	truncated bool
}

func (b *limitedTestOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	remaining := b.limit-len(b.data)
	if len(p) > remaining { b.truncated = true; p = p[:remaining] }
	b.data = append(b.data, p...)
	return n, nil
}

func (h *JournaledVerificationHost) diagnostic(request workflow.StageRequest, receipt *workflow.VerificationReceipt, detail string) error {
	ref, err := h.Store.PersistVerificationArtifact(request.TaskID, "verification-diagnostic", struct { RequestID string `json:"request_id"`; Detail string `json:"detail"` }{request.ID, detail})
	if err == nil { receipt.Result.Evidence = append(receipt.Result.Evidence, ref) }
	return err
}

func (h *JournaledVerificationHost) runChecks(ctx context.Context, request workflow.StageRequest, manifest workflow.TestManifest, receipt *workflow.VerificationReceipt) error {
	for _, invocation := range manifest.Invocations {
		state, err := h.Store.Load(request.TaskID)
		if err != nil { return err }
		if state.Protocol == nil || state.Protocol.Stop != nil { return errors.New("Stop prevents the next test check") }
		if _, err := h.Store.CheckTestManifest(state, request); err != nil { return err }
		if err := h.Boundary.Check(state, request, &manifest); err != nil { return err }
		started := time.Now().UTC()
		checkCtx, cancel := context.WithTimeout(ctx, time.Duration(invocation.Check.TimeoutSeconds)*time.Second)
		command := exec.CommandContext(checkCtx, invocation.Executable.RealPath, invocation.Check.Argv[1:]...)
		command.Dir, command.Env = invocation.Directory, append([]string{}, invocation.Environment...)
		command.WaitDelay = time.Second
		output := &limitedTestOutput{limit: manifest.Limits.MaxOutputBytes}
		command.Stdout, command.Stderr = output, output
		runErr := command.Run()
		timedOut := errors.Is(checkCtx.Err(), context.DeadlineExceeded)
		cancel()
		code := 0
		if runErr != nil {
			code = -1
			var exit *exec.ExitError
			if errors.As(runErr, &exit) { code = exit.ExitCode() }
		}
		ref, err := h.Store.PersistVerificationArtifact(request.TaskID, "verification-test-output", struct { Data []byte `json:"data"` }{output.data})
		if err != nil { return err }
		receipt.Checks = append(receipt.Checks, workflow.TestCheckReceipt{CheckID: invocation.Check.CheckID, ExitCode: code, TimedOut: timedOut, Output: ref, OutputBytes: len(output.data), OutputTruncated: output.truncated, StartedAt: started.Format(time.RFC3339Nano), EndedAt: time.Now().UTC().Format(time.RFC3339Nano)})
		if runErr != nil || timedOut || code != invocation.Check.ExpectedExitCode {
			receipt.Result.Status, receipt.Result.FailureClass = "failed", "unattributed"
			if timedOut || code == -1 { receipt.Result.Status, receipt.Result.FailureClass = "blocked", "infrastructure" }
			return h.diagnostic(request, receipt, "controlled test failed: "+invocation.Check.CheckID+"; preserve assertions and use the one bounded attribution step")
		}
	}
	return nil
}

func (h *JournaledVerificationHost) runTester(ctx context.Context, request workflow.StageRequest, input workflow.ExecutionInput, receipt *workflow.VerificationReceipt) error {
	_, err := session.ExecuteFrozenTurn(ctx, h.Sessions, request.SessionID, "tester", session.FrozenTurn{Prompt: input.Prompt, ExpectedTurn: request.Turn}, request.Model.Provider, request.Model.Model, []string{})
	if err != nil { return err }
	status, err := h.Sessions.Load(request.SessionID)
	if err != nil { return err }
	prepared := workflow.PreparedAgentRequest{TaskID: request.TaskID, WorkItemID: request.WorkItemID, AttemptID: request.AttemptID, SessionID: request.SessionID, ExpectedSessionTurn: request.Turn, Workspace: request.Workspace}
	if err := validateSupervisorSessionResult(status, &prepared); err != nil { return err }
	output, err := h.Sessions.ReadText(request.SessionID, status.Result.FinalOutputFile)
	if err != nil { return err }
	// Preserve even invalid output before interpreting the authored result.
	ref, err := h.Store.PersistVerificationArtifact(request.TaskID, "verification-tester-output", struct { RequestID string `json:"request_id"`; Text string `json:"text"` }{request.ID, output})
	if err != nil { return err }
	receipt.Result.Evidence = append(receipt.Result.Evidence, ref)
	authored, err := workflow.ParseTesterSubmission(output)
	if err != nil {
		receipt.Result.Status, receipt.Result.FailureClass = "failed", "test"
		return nil
	}
	planRef, err := h.Store.PersistVerificationArtifact(request.TaskID, "verification-test-plan", authored.Plan)
	if err != nil { return err }
	receipt.Inventory, receipt.TestPlan, receipt.AssertionReview = &authored.Inventory, &planRef, authored.AssertionReview
	return nil
}
