package execution

import (
	"errors"
	"fmt"
	"os"

	"aiw/internal/workflow"
)

// StageExecutor owns the controlled host boundary. Start must recheck real
// targets and host constraints; its result must come from the exact request.
// Reconcile is read-only and cannot start, retry or cancel an unknown writer.
type StageExecutor interface {
	Identity() string
	Check(workflow.RuntimeState, workflow.StageRequest) error
	Start(workflow.StageRequest) (*workflow.DispatchObservation, *workflow.StageResult, error)
	Reconcile(workflow.StageRequest) (*workflow.DispatchObservation, *workflow.StageResult, error)
}

// RunStage performs one bounded stage step. It never discovers a command or
// creates a fresh request to work around an unknown result or exhausted budget.
func RunStage(store *workflow.Store, id workflow.TaskID, requestID string, executor StageExecutor) error {
	if executor == nil { return errors.New("controlled stage executor is unavailable") }
	state, err := store.Load(id)
	if err != nil { return err }
	if state.Protocol == nil { return errors.New("durable execution is not enabled") }
	var record *workflow.StageRecord
	for i := range state.Protocol.Requests { if state.Protocol.Requests[i].Request.ID == requestID { record = &state.Protocol.Requests[i]; break } }
	if record == nil { return errors.New("stage request is not prepared") }
	if record.Consumed { return nil }
	if saved, err := store.ReadStageResult(id, requestID); err == nil {
		_, err = store.ConsumeStageResult(id, state.StateRevision, saved)
		return err
	} else if !errors.Is(err, os.ErrNotExist) { return err }
	var observation *workflow.DispatchObservation
	var result *workflow.StageResult
	if record.Dispatch == "intent" {
		if state.Protocol.Stop != nil { return errors.New("execution is stopped") }
		if err := executor.Check(state, record.Request); err != nil { return err }
		claimed, claimErr := store.ClaimStageDispatch(id, state.StateRevision, requestID, executor.Identity())
		if claimErr != nil { return claimErr }
		// A Stop arriving before the host call prevents this new side effect.
		current, loadErr := store.Load(id)
		if loadErr != nil { return loadErr }
		if current.StateRevision != claimed.StateRevision || current.Protocol.Stop != nil { return errors.New("dispatch state changed; reconcile the claimed request") }
		if err := executor.Check(current, record.Request); err != nil { return err }
		observation, result, err = executor.Start(record.Request)
	} else if record.Dispatch == "unknown" || record.Dispatch == "dispatched" {
		if executor.Identity() != record.Executor { return errors.New("reconciliation executor identity changed") }
		observation, result, err = executor.Reconcile(record.Request)
	} else {
		return fmt.Errorf("stage is %s; a new request requires Core preparation", record.Dispatch)
	}
	// Preserve an available terminal result even if the executor also reports
	// a projection error. Do not turn a host error into a model failure.
	if result != nil {
		if _, saveErr := store.PersistStageResult(id, *result); saveErr != nil { return errors.Join(err, saveErr) }
		current, loadErr := store.Load(id)
		if loadErr != nil { return errors.Join(err, loadErr) }
		_, consumeErr := store.ConsumeStageResult(id, current.StateRevision, *result)
		return errors.Join(err, consumeErr)
	}
	if observation != nil {
		current, loadErr := store.Load(id)
		if loadErr != nil { return errors.Join(err, loadErr) }
		_, observeErr := store.ObserveStage(id, current.StateRevision, *observation)
		return errors.Join(err, observeErr)
	}
	if err != nil { return err }
	return errors.New("executor returned no durable observation; retain writer and reconcile")
}

// ManagedDeliveryExecutor is an AIW host adapter, never an Agent shell. It may
// perform only the frozen local action. Cleanup belongs to the AIW host and
// requires its own grant and complete sealed-source evidence.
type ManagedDeliveryExecutor interface {
	Identity() string
	Check(workflow.RuntimeState, workflow.ManagedDeliveryPlan, workflow.DeliveryAction) error
	Execute(workflow.ManagedDeliveryPlan, workflow.DeliveryAction) (workflow.DeliveryObservation, error)
	Reconcile(workflow.ManagedDeliveryPlan, workflow.DeliveryAction) (workflow.DeliveryObservation, error)
}

func RunDeliveryAction(store *workflow.Store, id workflow.TaskID, executor ManagedDeliveryExecutor) error {
	if executor == nil { return errors.New("managed AIW delivery host is unavailable") }
	state, err := store.Load(id)
	if err != nil { return err }
	if state.Protocol == nil || state.Protocol.Delivery == nil { return errors.New("managed delivery plan is unavailable") }
	delivery := state.Protocol.Delivery
	index := 0
	for index < len(delivery.Actions) && delivery.Actions[index].State == "completed" { index++ }
	if index == len(delivery.Plan.Actions) { return nil }
	action := delivery.Plan.Actions[index]
	var observation workflow.DeliveryObservation
	if index < len(delivery.Actions) && delivery.Actions[index].State != "not-dispatched" {
		if delivery.Actions[index].Executor != executor.Identity() { return errors.New("delivery reconciliation executor changed") }
		observation, err = executor.Reconcile(delivery.Plan, action)
	} else {
		if state.Protocol.Stop != nil { return errors.New("delivery is stopped") }
		if err := executor.Check(state, delivery.Plan, action); err != nil { return err }
		claimed, claimErr := store.ClaimDeliveryAction(id, state.StateRevision, action.ID, executor.Identity())
		if claimErr != nil { return claimErr }
		current, loadErr := store.Load(id)
		if loadErr != nil { return loadErr }
		if current.StateRevision != claimed.StateRevision || current.Protocol.Stop != nil { return errors.New("delivery state changed; reconcile the claimed action") }
		if err := executor.Check(current, delivery.Plan, action); err != nil { return err }
		observation, err = executor.Execute(delivery.Plan, action)
	}
	if observation.ActionID == "" { return errors.Join(err, errors.New("delivery result is unknown; preserve the workspace")) }
	current, loadErr := store.Load(id)
	if loadErr != nil { return errors.Join(err, loadErr) }
	_, observeErr := store.ObserveDeliveryAction(id, current.StateRevision, observation)
	return errors.Join(err, observeErr)
}
