package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (s *Store) BeginExecution(id TaskID, revision uint64, attempt Attempt) (RuntimeState, error) {
	if err := s.requireExecutionServices(); err != nil { return RuntimeState{}, err }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.attempt.started", AttemptID: attempt.ID, WorkItemID: attempt.WorkItemID}, func(state *RuntimeState) error {
		if state.Protocol == nil || state.Protocol.Stop != nil || !state.Protocol.BudgetKnown { return errors.New("execution is disabled, stopped or has unknown legacy budget") }
		if usageBudgetAuthorizationOpen(*state) { return errors.New("Task usage budget authorization is pending") }
		if err := requireNoStageInFlight(*state); err != nil { return err }
		if attempt.ID == "" || attempt.WorkItemID == "" || attempt.Workspace != executionWorkspace(*state) { return errors.New("Attempt must bind the current Task workspace") }
		if _, exists := findAttempt(state.Attempts, attempt.ID); exists { return errors.New("Attempt identity already exists") }
		for _, cursor := range state.Protocol.Items { if cursor.Phase != PhaseAccepted { return errors.New("resume the existing execution Attempt") } }
		for i := range state.WorkItems {
			item := &state.WorkItems[i]
			if item.ID != attempt.WorkItemID { continue }
			if item.State != WorkItemReady { return errors.New("Work Item is not ready") }
			for _, dependency := range item.Dependencies {
				accepted := false
				for _, prior := range state.WorkItems {
					if prior.ID == dependency && prior.State == WorkItemCompleted && prior.AcceptedReference != nil {
						if _, err := s.ReadAcceptedExecution(id, prior, *prior.AcceptedReference, executionWorkspace(*state)); err != nil { return err }
						accepted = true
					}
				}
				if !accepted { return errors.New("dependency requires applicable Core acceptance") }
			}
			for _, gate := range state.Gates { if gate.State == GateOpen && (gate.WorkItemID == "" || gate.WorkItemID == item.ID) { return errors.New("Work Item has an open Gate") } }
			attempt.State, attempt.StartedAt = AttemptRunning, time.Now().UTC().Format(time.RFC3339)
			item.State = WorkItemRunning
			state.Attempts = append(state.Attempts, attempt)
			state.Protocol.Items = append(state.Protocol.Items, ItemExecution{WorkItemID: item.ID, AttemptID: attempt.ID, Phase: PhaseCoder})
			return nil
		}
		return errors.New("unknown Work Item")
	})
}

// PrepareStage commits intent, the reserved budget and the writer generation
// together. Only ClaimStageDispatch may hand this request to an executor.
func (s *Store) PrepareStage(id TaskID, revision uint64, request StageRequest) (RuntimeState, error) {
	if err := s.requireExecutionServices(); err != nil { return RuntimeState{}, err }
	if err := validateStageRequest(request); err != nil { return RuntimeState{}, err }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.stage.prepared", AttemptID: request.AttemptID, WorkItemID: request.WorkItemID}, func(state *RuntimeState) error {
		item, err := executionItem(state, request.WorkItemID)
		if err != nil { return err }
		if state.Protocol.Stop != nil || !state.Protocol.BudgetKnown { return errors.New("execution is stopped or its budget is unknown") }
		if isGeneration(request) && usageBudgetAuthorizationOpen(*state) { return errors.New("Task usage budget authorization is pending before model generation") }
		if item.Phase != request.Phase || item.AttemptID != request.AttemptID || request.TaskID != id || request.Workspace != executionWorkspace(*state) { return errors.New("stage request does not match the current execution") }
		if err := requireNoStageInFlight(*state); err != nil { return err }
		if item.CurrentRequest != "" {
			prior, err := stageRecord(state, item.CurrentRequest)
			if err != nil { return err }
			if prior.Result != nil {
				var result StageResult
				if err := s.ReadExecutionArtifact(id, *prior.Result, &result); err != nil { return err }
				if result.Status != "passed" {
					if prior.Attribution != nil {
						var attribution FailureAttribution
						if err := s.ReadExecutionArtifact(id, *prior.Attribution, &attribution); err != nil { return err }
						if attribution.RequestID != prior.Request.ID || prior.Result == nil || attribution.Result != *prior.Result { return errors.New("failure attribution does not match the original result") }
						result.FailureClass = attribution.FailureClass
					}
					switch result.FailureClass {
					case "infrastructure":
						reserved := false
						for _, account := range state.Protocol.Recoveries { for _, key := range account.Requests { if key == prior.Request.ID { reserved = true } } }
						if !reserved { return errors.New("infrastructure retry requires the original stage recovery reservation") }
					case "implementation": if request.Phase != PhaseCoder { return errors.New("implementation defects return to Coder") }
					case "test": if request.Phase != PhaseTester { return errors.New("test defects return to Tester") }
					default: return errors.New("failure requires report repair, diagnosis or manual resolution before dispatch")
					}
				}
			}
		}
		if _, err := stageRecord(state, request.ID); err == nil { return errors.New("stage request identity cannot be reused") }
		// Freeze the generators of these input bytes before a delayed compiler
		// or test diagnosis can arrive. A caller cannot redirect attribution.
		sources := budgetGenerationSources(*state, request)
		if len(request.GenerationSources) != 0 && !equalJSON(request.GenerationSources, sources) { return errors.New("generation provenance differs from the current input producers") }
		request.GenerationSources = sources
		if request.Phase == PhaseReport {
			prior, err := stageRecord(state, item.CurrentRequest)
			if err != nil || prior.Request.Phase != PhaseCoder || !prior.Consumed { return errors.New("only the original terminal Coder generation may request a report supplement") }
			for _, key := range state.Automation.ReportSupplements { if key == prior.Request.ID { return errors.New("report supplement allowance is exhausted; manual review is required") } }
			state.Automation.ReportSupplements = append(state.Automation.ReportSupplements, prior.Request.ID)
		}
		var input json.RawMessage
		if err := s.ReadExecutionArtifact(id, request.Input, &input); err != nil { return err }
		if request.Phase == PhaseCoder || request.Phase == PhaseTester || request.Phase == PhaseReport {
			var frozen ExecutionInput
			if err := json.Unmarshal(input, &frozen); err != nil { return err }
			if frozen.RequestID != request.ID || frozen.TaskID != id || frozen.WorkItemID != request.WorkItemID || frozen.AttemptID != request.AttemptID || frozen.SessionID != request.SessionID || frozen.Turn != request.Turn || frozen.Actor != request.Actor || frozen.Workspace != request.Workspace || !equalJSON(frozen.AISelection, request.Model) { return errors.New("frozen Actor input differs from the requested generation") }
		}
		state.Protocol.LeaseGeneration++
		request.LeaseGeneration = state.Protocol.LeaseGeneration
		// Authorize the candidate cursor and writer lease. updateWithEvent does
		// not commit either when authorization or budget reservation fails.
		if request.Phase == PhaseCoder || request.Phase == PhaseTester {
			state.WriteLease = &WriteLease{AttemptID: request.AttemptID, Workspace: request.Workspace, AcquiredAt: request.PreparedAt, RequestID: request.ID, Generation: request.LeaseGeneration}
		}
		item.CurrentRequest = request.ID
		if err := s.ExecutionServices.Authorize(*state, request); err != nil { return err }
		if err := s.ExecutionServices.Budget(state, request, "reserve"); err != nil { return err }
		state.Protocol.Requests = append(state.Protocol.Requests, StageRecord{Request: request, Dispatch: "intent"})
		return nil
	})
}

// ClaimStageDispatch marks the one external invocation window as unknown
// before crossing it. A competing or restarted caller cannot claim it again.
// The executor rechecks real paths, Stop and host constraints before effects.
func (s *Store) ClaimStageDispatch(id TaskID, revision uint64, requestID, executor string) (RuntimeState, error) {
	if err := s.requireExecutionServices(); err != nil { return RuntimeState{}, err }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.stage.dispatch-intent", Detail: requestID}, func(state *RuntimeState) error {
		record, err := stageRecord(state, requestID)
		if err != nil { return err }
		if executor == "" || state.Protocol.Stop != nil || record.Dispatch != "intent" { return errors.New("dispatch is stopped or already claimed; reconcile the same request") }
		if isGeneration(record.Request) && usageBudgetAuthorizationOpen(*state) { return errors.New("Task usage budget authorization is pending before model generation") }
		if err := s.ExecutionServices.Authorize(*state, record.Request); err != nil { return err }
		record.Dispatch, record.Executor = "unknown", executor
		return nil
	})
}

type DispatchObservation struct {
	RequestID string `json:"request_id"`
	Executor string `json:"executor"`
	Generation uint64 `json:"generation"`
	Status string `json:"status"`
	Evidence ActorReference `json:"evidence"`
}

// ObserveStage is also the recovery entry point. It never invokes an executor
// and only a proved not-dispatched observation refunds a reservation/lease.
func (s *Store) ObserveStage(id TaskID, revision uint64, observation DispatchObservation) (RuntimeState, error) {
	if err := s.requireExecutionServices(); err != nil { return RuntimeState{}, err }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.stage.observed", Detail: observation.RequestID}, func(state *RuntimeState) error {
		record, err := stageRecord(state, observation.RequestID)
		if err != nil { return err }
		if observation.Executor == "" || observation.Executor != record.Executor || observation.Generation != record.Request.LeaseGeneration { return errors.New("executor observation does not match dispatch ownership") }
		if observation.Status != "unknown" && observation.Status != "dispatched" && observation.Status != "not-dispatched" { return errors.New("unsupported dispatch observation") }
		var evidence json.RawMessage
		if err := s.ReadExecutionArtifact(id, observation.Evidence, &evidence); err != nil { return err }
		if record.Dispatch == "terminal" || record.Dispatch == "not-dispatched" || (record.Dispatch == "dispatched" && observation.Status != "dispatched") { return errors.New("observation cannot reverse an established execution fact") }
		// Result validation also authenticates executor/reconciliation evidence;
		// a caller-provided status alone cannot prove that no process started.
		proof := StageResult{RequestID: record.Request.ID, TaskID: id, WorkItemID: record.Request.WorkItemID, AttemptID: record.Request.AttemptID, SessionID: record.Request.SessionID, Turn: record.Request.Turn, LeaseGeneration: observation.Generation, InputDigest: record.Request.InputDigest, Executor: observation.Executor, Status: observation.Status, Evidence: []ActorReference{observation.Evidence}}
		if err := s.ExecutionServices.ValidateResult(*state, record.Request, proof); err != nil { return err }
		ref, err := s.persistProtocolArtifactLocked(id, "dispatch-observation", observation)
		if err != nil { return err }
		if record.Observation != nil && *record.Observation == ref { return errProtocolNoChange }
		if record.Dispatch != observation.Status {
			if err := s.ExecutionServices.Budget(state, record.Request, observation.Status); err != nil { return err }
		}
		record.Dispatch, record.Observation = observation.Status, &ref
		record.Observations = append(record.Observations, ref)
		if observation.Status == "not-dispatched" { releaseStageWriter(state, record.Request) }
		return nil
	})
}

// ConsumeStageResult saves result bytes before the conditional state commit.
// Recovery supplies the same result; duplicate consumption never changes the
// phase, budget or writer. Coder success cannot close the enclosing Attempt.
func (s *Store) ConsumeStageResult(id TaskID, revision uint64, result StageResult) (RuntimeState, error) {
	if err := s.requireExecutionServices(); err != nil { return RuntimeState{}, err }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.stage.result", Detail: result.RequestID}, func(state *RuntimeState) error {
		record, err := stageRecord(state, result.RequestID)
		if err != nil { return err }
		r := record.Request
		if result.TaskID != id || result.WorkItemID != r.WorkItemID || result.AttemptID != r.AttemptID || result.SessionID != r.SessionID || result.Turn != r.Turn || result.InputDigest != r.InputDigest || result.LeaseGeneration != r.LeaseGeneration || result.Executor == "" || result.Executor != record.Executor || !result.Terminal {
			return errors.New("terminal result does not match the exact stage request")
		}
		if record.Dispatch != "unknown" && record.Dispatch != "dispatched" && record.Dispatch != "terminal" { return errors.New("request has no possible dispatch") }
		if result.Status != "passed" && result.Status != "failed" && result.Status != "blocked" { return errors.New("unsupported terminal result") }
		if err := s.ExecutionServices.ValidateResult(*state, r, result); err != nil { return err }
		ref, err := s.persistStageResultLocked(id, result)
		if err != nil { return err }
		if record.Consumed {
			if record.Result == nil || *record.Result != ref { return errors.New("conflicting result for a consumed request") }
			return errProtocolNoChange
		}
		if err := s.ExecutionServices.Budget(state, r, "consume:"+result.Status+":"+result.FailureClass); err != nil { return err }
		record.Result, record.Consumed, record.Dispatch = &ref, true, "terminal"
		releaseStageWriter(state, r)
		item, err := executionItem(state, r.WorkItemID)
		if err != nil { return err }
		if item.CurrentRequest != r.ID { return errors.New("late result cannot advance a different stage") }
		if result.Status == "passed" {
			switch r.Phase {
			case PhaseCoder: item.Phase, item.TestAuthoringRequest, item.ValidatedReport = PhaseReport, "", nil
			case PhaseReport: item.Phase = PhaseReport // deterministic validation still owns advancement
			case PhaseCompile:
				item.Phase = PhaseTester
				if item.TestAuthoringRequest != "" { item.Phase = PhaseTest }
			case PhaseTester:
				// Newly authored or repaired tests invalidate the earlier compile.
				// Recompile once, then continue to Runner without another Tester.
				item.Phase, item.TestAuthoringRequest = PhaseCompile, r.ID
			case PhaseTest: item.Phase = PhaseAccept
			}
		} else {
			switch result.FailureClass {
			case "implementation": item.Phase = PhaseCoder
			case "test": item.Phase = PhaseTester
			}
		}
		return nil
	})
}

func releaseStageWriter(state *RuntimeState, r StageRequest) {
	if lease := state.WriteLease; lease != nil && lease.RequestID == r.ID && lease.Generation == r.LeaseGeneration { state.WriteLease = nil }
}

// StopExecution preserves in-flight ownership. Resume is an explicit decision
// with evidence; neither operation resets request or recovery budgets.
func (s *Store) StopExecution(id TaskID, revision uint64, decision ExecutionStop, resume bool) (RuntimeState, error) {
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.stop", Detail: decision.Reason}, func(state *RuntimeState) error {
		if state.Protocol == nil || decision.Reason == "" || !validProtocolReference(decision.Reference) { return errors.New("Stop/resume requires a durable protocol and decision evidence") }
		var proof json.RawMessage
		if err := s.ReadExecutionArtifact(id, decision.Reference, &proof); err != nil { return err }
		if resume { state.Protocol.Stop = nil } else { state.Protocol.Stop = &decision }
		return nil
	})
}

// RecoverStage reserves one of two additional infrastructure recoveries per
// Work Item and stage. The key is independent of Attempts and process restarts.
func (s *Store) RecoverStage(id TaskID, revision uint64, requestID string, condition ActorReference) (RuntimeState, error) {
	if err := s.requireExecutionServices(); err != nil { return RuntimeState{}, err }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.stage.recovery", Detail: requestID}, func(state *RuntimeState) error {
		if err := requireNoStageInFlight(*state); err != nil { return err }
		r, err := stageRecord(state, requestID)
		if err != nil { return err }
		if r.Request.Phase == PhaseReport { return errors.New("report supplement has its own single allowance; infrastructure recovery cannot extend it") }
		if state.Protocol.Stop != nil { return errors.New("execution is stopped") }
		item, err := executionItem(state, r.Request.WorkItemID)
		if err != nil || item.CurrentRequest != requestID { return errors.New("recovery must target the current failed stage") }
		var result StageResult
		if r.Result == nil { return errors.New("infrastructure recovery requires a saved terminal failure") }
		if err := s.ReadExecutionArtifact(id, *r.Result, &result); err != nil { return err }
		if r.Attribution != nil {
			var attribution FailureAttribution
			if err := s.ReadExecutionArtifact(id, *r.Attribution, &attribution); err != nil { return err }
			if attribution.RequestID != r.Request.ID || attribution.Result != *r.Result { return errors.New("recovery attribution differs from the original result") }
			result.FailureClass = attribution.FailureClass
		}
		if result.Status == "passed" || result.FailureClass != "infrastructure" { return errors.New("failure is not an infrastructure recovery") }
		result.Status, result.Evidence = "recovery-ready", []ActorReference{condition}
		if err := s.ExecutionServices.ValidateResult(*state, r.Request, result); err != nil { return err }
		for i := range state.Protocol.Recoveries {
			account := &state.Protocol.Recoveries[i]
			if account.WorkItemID != r.Request.WorkItemID || account.Phase != r.Request.Phase { continue }
			for _, prior := range account.Requests { if prior == requestID { return errProtocolNoChange } }
			if len(account.Requests) >= 2 { return errors.New("stage infrastructure recovery budget exhausted") }
			account.Requests = append(account.Requests, requestID)
			return nil
		}
		state.Protocol.Recoveries = append(state.Protocol.Recoveries, StageRecovery{WorkItemID: r.Request.WorkItemID, Phase: r.Request.Phase, Requests: []string{requestID}})
		return nil
	})
}

// DiagnoseStage records the one read-only diagnostic allowance for a failure.
// Actual diagnosis is dispatched through the service with a read-only scope.
func (s *Store) DiagnoseStage(id TaskID, revision uint64, requestID string) (RuntimeState, error) {
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.stage.diagnosis", Detail: requestID}, func(state *RuntimeState) error {
		r, err := stageRecord(state, requestID)
		if err != nil { return err }
		if !r.Consumed || r.Result == nil || state.Protocol.Stop != nil { return errors.New("diagnosis requires a terminal failed request and no Stop") }
		var result StageResult
		if err := s.ReadExecutionArtifact(id, *r.Result, &result); err != nil { return err }
		if result.Status != "failed" || result.FailureClass != "unattributed" { return errors.New("diagnosis is only for an unattributed defect") }
		for _, prior := range state.Protocol.Diagnoses { if prior == requestID { return fmt.Errorf("diagnosis already reserved for %s; reconcile or request manual review", requestID) } }
		state.Protocol.Diagnoses = append(state.Protocol.Diagnoses, requestID)
		return nil
	})
}
