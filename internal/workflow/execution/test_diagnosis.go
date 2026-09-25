package execution

import (
	"context"
	"errors"

	"aiw/internal/workflow"
)

type VerificationDiagnostician interface {
	// Diagnose journals the same request before the one read-only invocation.
	Diagnose(context.Context, workflow.StageRequest, workflow.ActorReference) (workflow.FailureAttribution, error)
	Observe(workflow.StageRequest) (workflow.FailureAttribution, error)
	Verify(workflow.RuntimeState, workflow.FailureAttribution) error
}

func DiagnoseVerificationFailure(ctx context.Context, store *workflow.Store, id workflow.TaskID, requestID string, host VerificationDiagnostician) error {
	if host == nil { return errors.New("read-only failure diagnosis host is unavailable") }
	state, err := store.Load(id)
	if err != nil { return err }
	if state.Protocol == nil || state.Protocol.Stop != nil { return errors.New("failure diagnosis is not active") }
	var record *workflow.StageRecord
	for i := range state.Protocol.Requests { if state.Protocol.Requests[i].Request.ID == requestID { record = &state.Protocol.Requests[i]; break } }
	if record == nil || record.Result == nil { return errors.New("failed controlled request is missing") }
	if record.Attribution != nil { return nil }
	reserved := false
	for _, prior := range state.Protocol.Diagnoses { if prior == requestID { reserved = true } }
	var attribution workflow.FailureAttribution
	if reserved {
		attribution, err = host.Observe(record.Request)
	} else {
		if _, err = store.DiagnoseStage(id, state.StateRevision, requestID); err != nil { return err }
		current, loadErr := store.Load(id)
		if loadErr != nil { return loadErr }
		if current.Protocol.Stop != nil { return errors.New("Stop prevents a new diagnosis") }
		attribution, err = host.Diagnose(ctx, record.Request, *record.Result)
	}
	if err != nil { return err }
	state, err = store.Load(id)
	if err != nil { return err }
	_, err = store.AttributeVerificationFailure(id, state.StateRevision, attribution, host.Verify)
	return err
}
