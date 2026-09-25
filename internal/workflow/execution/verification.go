package execution

import (
	"context"
	"errors"

	"aiw/internal/workflow"
)

// ControlledVerificationHost must execute inside the boundary checked by
// VerificationHost. Tester uses its own frozen Session with only test-path
// writes; Runner uses the exact manifest with network denied and bounded
// outputs. Start journals terminal receipts before returning. Observe is
// read-only: no new process/model invocation may be hidden in reconciliation.
type ControlledVerificationHost interface {
	workflow.VerificationHost
	Start(context.Context, workflow.StageRequest, *workflow.ExecutionInput, *workflow.TestManifest) (*workflow.DispatchObservation, *workflow.VerificationReceipt, error)
	Observe(workflow.StageRequest) (*workflow.DispatchObservation, *workflow.VerificationReceipt, error)
}

type VerificationStageExecutor struct {
	Context context.Context
	Store *workflow.Store
	Host ControlledVerificationHost
}

func (e *VerificationStageExecutor) Identity() string {
	if e.Host == nil { return "" }
	return e.Host.Identity()
}

func (e *VerificationStageExecutor) Check(state workflow.RuntimeState, request workflow.StageRequest) error {
	service := workflow.VerificationService{Store: e.Store, Host: e.Host}
	return service.Authorize(state, request)
}

func (e *VerificationStageExecutor) Start(request workflow.StageRequest) (*workflow.DispatchObservation, *workflow.StageResult, error) {
	if e.Host == nil || e.Store == nil { return nil, nil, errors.New("host-unavailable: verification executor is not installed") }
	state, err := e.Store.Load(request.TaskID)
	if err != nil { return nil, nil, err }
	if state.Protocol == nil || state.Protocol.Stop != nil { return nil, nil, errors.New("verification dispatch is stopped") }
	// The host must atomically enforce these constraints when starting. Static
	// symlink checks alone do not close the check/use race.
	if err := e.Check(state, request); err != nil { return nil, nil, err }
	var input *workflow.ExecutionInput
	var manifest *workflow.TestManifest
	if request.Phase == workflow.PhaseTest {
		frozen, err := e.Store.CheckTestManifest(state, request)
		if err != nil { return nil, nil, err }
		manifest = &frozen
	} else if request.Phase == workflow.PhaseTester || request.Phase == workflow.PhaseCoder || request.Phase == workflow.PhaseReport {
		input = &workflow.ExecutionInput{}
		if err := e.Store.ReadExecutionArtifact(request.TaskID, request.Input, input); err != nil { return nil, nil, err }
		if err := e.Store.ValidateFrozenTaskSources(*input); err != nil { return nil, nil, err }
	}
	ctx := e.Context
	if ctx == nil { ctx = context.Background() }
	observation, receipt, runErr := e.Host.Start(ctx, request, input, manifest)
	return e.save(request, observation, receipt, runErr)
}

func (e *VerificationStageExecutor) Reconcile(request workflow.StageRequest) (*workflow.DispatchObservation, *workflow.StageResult, error) {
	if e.Host == nil { return nil, nil, errors.New("host-unavailable: original executor cannot be observed") }
	observation, receipt, err := e.Host.Observe(request)
	return e.save(request, observation, receipt, err)
}

func (e *VerificationStageExecutor) save(request workflow.StageRequest, observation *workflow.DispatchObservation, receipt *workflow.VerificationReceipt, runErr error) (*workflow.DispatchObservation, *workflow.StageResult, error) {
	if receipt == nil { return observation, nil, runErr }
	if err := e.Host.VerifyReceipt(request, *receipt); err != nil { return observation, nil, errors.Join(runErr, err) }
	ref, err := e.Store.PersistVerificationArtifact(request.TaskID, "verification-receipt", *receipt)
	if err != nil { return observation, nil, errors.Join(runErr, err) }
	result := receipt.Result
	result.Evidence = append([]workflow.ActorReference{ref}, receipt.Result.Evidence...)
	// RunStage persists the exact result before consumption. If saving fails,
	// the original host journal is consulted; it never invokes Start again.
	return observation, &result, runErr
}

// RunVerificationStage connects E03 to the same E02 dispatch/Stop/budget and
// reconciliation path. It does not mutate old FocusedTestAuthorization or
// bypass Core through the legacy ExecuteFocusedTest command.
func RunVerificationStage(ctx context.Context, store *workflow.Store, id workflow.TaskID, requestID string, host ControlledVerificationHost) error {
	return RunStage(store, id, requestID, &VerificationStageExecutor{Context: ctx, Store: store, Host: host})
}
