// Package execution coordinates managed Workflow execution and its adapters.
// Workflow Core owns state transitions; task owns Task artifacts and bindings.
package execution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"aiw/internal/task"
	taskworkflow "aiw/internal/task/workflowadapter"
	"aiw/internal/workflow"
)

// Supervisor advances one Task through Agent outcomes, compile repairs, and
// delivery. RunStep and Merge connect the existing command execution adapters
// during migration; neither adapter owns the supervisor loop or its lease.
// Report and Delivered preserve CLI presentation without importing terminal UI.
type Supervisor struct {
	Store     *workflow.Store
	RunStep   func(id string, execute bool, provider, model string) error
	Merge     func(id string, meta task.TaskMeta, store *workflow.Store, message string) error
	Report    func(id string, state workflow.RuntimeState) error
	Delivered func(id, parentBranch string)
	Analyzer  RemediationAnalyzer
}

// Start owns one foreground supervisor lease and preserves the ordering of
// dispatch, Session validation, compile repair, outcome recording, and delivery.
func (s Supervisor) Start(id, provider, model string, state workflow.RuntimeState) error {
	store := s.Store
	if state.Delivery == workflow.DeliveryDiscarded ||
		(state.Delivery == workflow.DeliveryMerged && workflow.DeriveSummary(state).Execution == workflow.ExecutionCompleted) {
		return nil
	}
	leaseID := fmt.Sprintf("supervisor-%d", time.Now().UTC().UnixNano())
	if _, err := store.StartSupervisor(workflow.TaskID(id), leaseID); err != nil {
		return err
	}
	current, err := store.Load(workflow.TaskID(id))
	if err != nil {
		return err
	}
	repairedProjection := false
	for _, repair := range current.Automation.ProjectionRepairs {
		if repair.ResolvedAt != "" {
			continue
		}
		if err := s.repair(id, repair); err != nil {
			_, _ = store.PauseSupervisor(workflow.TaskID(id), leaseID, "repair-paused", err.Error(), time.Now().UTC().Add(30*time.Second))
			return err
		}
		current, err = store.Load(workflow.TaskID(id))
		if err != nil {
			return err
		}
		if _, err = store.RecordSupervisorObservation(workflow.TaskID(id), leaseID, current.LastEventSequence, "repair-resolved", repair.Target); err != nil {
			return err
		}
		repairedProjection = true
	}
	for {
		if err := syncSupervisorChecklist(id, store); err != nil {
			// A transient source read is not an Agent outcome. Preserve the
			// current Attempt and lease instead of consuming no-progress.
			if _, pauseErr := store.PauseSupervisor(workflow.TaskID(id), leaseID, "checklist-paused", err.Error(), time.Now().UTC().Add(30*time.Second)); pauseErr != nil {
				return fmt.Errorf("checklist sync: %w; pause supervisor: %v", err, pauseErr)
			}
			return nil
		}
		updated, err := store.Load(workflow.TaskID(id))
		if err != nil || updated.Automation.Supervisor.LeaseID != leaseID {
			return err
		}
		if repairedProjection || workflow.SupervisorDue(updated, time.Now().UTC()) ||
			updated.Automation.Supervisor.ObservedEvent == 0 {
			repairedProjection = false
			if _, err := store.RenewSupervisorLease(workflow.TaskID(id), leaseID); err != nil {
				return err
			}
			stepErr := s.RunStep(id, true, provider, model)
			if stepErr != nil {
				failed, loadErr := store.Load(workflow.TaskID(id))
				if loadErr != nil {
					return fmt.Errorf("runner failed: %v; reload runtime: %w", stepErr, loadErr)
				}
				if !observeDispatchedAfterRunnerError(failed) && failed.SchemaVersion < 10 {
					for _, gate := range failed.Gates {
						if gate.State == workflow.GateOpen {
							if decisionErr := s.awaitGateDecision(id, leaseID); decisionErr != nil {
								return fmt.Errorf("supervised step failed: %w; prepare Gate decision: %v", stepErr, decisionErr)
							}
							return nil
						}
					}
				}
				if !observeDispatchedAfterRunnerError(failed) {
					_, _ = store.PauseSupervisor(workflow.TaskID(id), leaseID, "runner-paused", stepErr.Error(), time.Now().UTC().Add(30*time.Second))
					return stepErr
				}
				// The request crossed the dispatch boundary. Only its bound Session
				// result can establish no-progress; a runner error cannot do so.
			}
			updated, err = store.Load(workflow.TaskID(id))
			if err != nil {
				return err
			}
			if request := updated.Automation.PreparedRequest; request != nil {
				// A model turn may outlive the one-minute supervisor lease. Renew
				// before committing its result so the completion and the next
				// outcome remain owned by this supervisor invocation.
				if _, err = store.RenewSupervisorLease(workflow.TaskID(id), leaseID); err != nil {
					return err
				}
				outcome, outcomeErr := recordSupervisorSessionOutcome(store, request)
				if outcomeErr != nil && request.DispatchedAt != "" {
					var reportErr *reportValidationError
					if !errors.As(outcomeErr, &reportErr) {
						for retry := 0; retry < 2 && outcomeErr != nil; retry++ {
							time.Sleep(time.Second)
							outcome, outcomeErr = recordSupervisorSessionOutcome(store, request)
						}
					}
				}
				if outcomeErr != nil {
					var reportErr *reportValidationError
					if errors.As(outcomeErr, &reportErr) {
						if request.ReportOrigin == nil {
							if _, err := store.PrepareReportSupplement(request.TaskID, *request, reportErr.Error()); err == nil { continue }
						}
						if _, err := store.OpenCompilerGate(request.TaskID, request.WorkItemID, "report-manual-review", reportErr.Error()); err != nil { return err }
						return s.awaitGateDecision(id, leaseID)
					}
					if request.DispatchedAt != "" {
						if stepErr != nil { outcomeErr = fmt.Errorf("runner error: %v; observe original Session: %w", stepErr, outcomeErr) }
						return s.awaitUnknownSession(id, leaseID, *request, outcomeErr)
					}
					_, _ = store.PauseSupervisor(workflow.TaskID(id), leaseID, "session-result-unknown", outcomeErr.Error(), time.Now().UTC().Add(30*time.Second))
					return outcomeErr
				}
				if outcome.Kind == workflow.SupervisedOutcomeCompleted {
					var repairPending bool
					outcome, repairPending, err = compileSupervisedOutcome(id, store, request, outcome, workflow.ExecCompileCommandRunner{})
					if err != nil {
						_, _ = store.PauseSupervisor(workflow.TaskID(id), leaseID, "compiler-paused", err.Error(), time.Now().UTC().Add(30*time.Second))
						return err
					}
					if _, err = store.RenewSupervisorLease(workflow.TaskID(id), leaseID); err != nil {
						return err
					}
					if repairPending {
						continue
					}
				}
				updated, err = store.RecordSupervisedOutcome(workflow.TaskID(id), request.AttemptID, outcome)
				if err != nil {
					return err
				}
				if updated.SchemaVersion < 10 && outcome.Kind == workflow.SupervisedOutcomeNoProgress {
					handled, remediationErr := s.remediateRunnerFailure(id, updated, request.AttemptID, outcome)
					if remediationErr != nil {
						_, _ = store.PauseSupervisor(workflow.TaskID(id), leaseID, "remediation-paused", remediationErr.Error(), time.Now().UTC().Add(30*time.Second))
						return remediationErr
					}
					if handled { continue }
					if pending, pendingErr := store.Load(workflow.TaskID(id)); pendingErr == nil && pending.Automation.Cursor.Result == "awaiting-human" {
						if err := s.Report(id, pending); err != nil { return err }
						_, err = store.PauseSupervisor(workflow.TaskID(id), leaseID, "awaiting-human", pending.Automation.Cursor.Detail, time.Now().UTC().Add(24*time.Hour))
						return err
					}
				}
				if err = s.Report(id, updated); err != nil {
					return err
				}
			}
			// The execution path leaves the old automation cursor in place.
			// Re-evaluate the durable state after closing the Attempt so the
			// Supervisor records the actual next outcome instead of repeating
			// agent-request-prepared forever.
			if err = s.RunStep(id, false, "", ""); err != nil {
				return err
			}
				updated, err = store.Load(workflow.TaskID(id))
			if err != nil {
				return err
			}
			// A fresh prepared request is external work for the next
			// supervisor iteration. Do not advance the observation cursor over
			// it, or the request would remain prepared forever.
			if updated.Automation.Cursor.Result == string(workflow.RunnerPrepared) {
				continue
			}
			if _, err = store.RecordSupervisorObservation(workflow.TaskID(id), leaseID, updated.LastEventSequence, updated.Automation.Cursor.Result, updated.Automation.Cursor.Detail); err != nil {
				return err
			}
			switch updated.Automation.Cursor.Result {
			case string(workflow.RunnerNoWork):
				delivered, deliveryErr := s.DeliverCompleted(id)
				if deliveryErr != nil {
					_, _ = store.PauseSupervisor(workflow.TaskID(id), leaseID, "delivery-paused", deliveryErr.Error(), time.Now().UTC().Add(30*time.Second))
					return deliveryErr
				}
				if delivered {
					_, err = store.StopSupervisor(workflow.TaskID(id), leaseID)
					return err
				}
				_, err = store.PauseSupervisor(workflow.TaskID(id), leaseID, "supervisor-paused", updated.Automation.Cursor.Result, time.Now().UTC().Add(30*time.Second))
				return err
			case string(workflow.RunnerGate):
				if err := s.awaitGateDecision(id, leaseID); err != nil {
					return err
				}
				return nil
			case "repair-required", "blocked":
				_, err = store.PauseSupervisor(workflow.TaskID(id), leaseID, "supervisor-paused", updated.Automation.Cursor.Result, time.Now().UTC().Add(30*time.Second))
				return err
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func (s Supervisor) remediateRunnerFailure(id string, state workflow.RuntimeState, attemptID workflow.AttemptID, outcome workflow.SupervisedOutcome) (bool, error) {
	if s.Analyzer == nil {
		return false, s.pauseForHumanRemediation(workflow.TaskID(id), state, attemptID, outcome, nil)
	}
	report, err := workflow.BuildRemediationReportForOutcome(state, attemptID, outcome, state.Automation.Supervisor.RemediationRounds)
	if err != nil { return false, err }
	diagnosis, err := s.Analyzer.Analyze(context.Background(), report.Problem, report.Options)
	if err != nil {
		return false, s.pauseForHumanRemediation(workflow.TaskID(id), state, attemptID, outcome, nil)
	}
	report.Diagnosis = &diagnosis
	retryAllowed := false
	for _, option := range report.Options { if option.Action == workflow.RemediationActionRetryWorkItem { retryAllowed = true; break } }
	if report.Round < workflow.MaxAutomaticRemediationRounds && retryAllowed && (diagnosis.RecommendedAction == workflow.RemediationActionRetryWorkItem || diagnosis.RecommendedAction == workflow.RemediationActionResumeSupervisor) && outcome.Kind == workflow.SupervisedOutcomeNoProgress {
		report.Status = workflow.RemediationAutoResolved
		report.Attempts = []workflow.RemediationAttempt{{Action: diagnosis.RecommendedAction, Outcome: "continuing", Detail: diagnosis.Reason, RecordedAt: time.Now().UTC().Format(time.RFC3339)}}
		if _, err := s.Store.PersistRemediationReport(state, report); err != nil { return false, err }
		_, err = s.Store.RecordRemediationAutoRetry(workflow.TaskID(id), report.ProblemID, diagnosis.Reason)
		return err == nil, err
	}
	return false, s.pauseForHumanRemediation(workflow.TaskID(id), state, attemptID, outcome, &diagnosis)
}

func (s Supervisor) pauseForHumanRemediation(id workflow.TaskID, state workflow.RuntimeState, attemptID workflow.AttemptID, outcome workflow.SupervisedOutcome, diagnosis *workflow.RemediationDiagnosis) error {
	report, err := workflow.BuildRemediationReportForOutcome(state, attemptID, outcome, state.Automation.Supervisor.RemediationRounds)
	if err != nil { return err }
	report.Status = workflow.RemediationAwaitingHuman
	report.Diagnosis = diagnosis
	_, err = s.Store.AwaitRemediation(id, report)
	return err
}

// repair retries the accepted Work Item projection that created this repair.
func (s Supervisor) repair(id string, repair workflow.ProjectionRepair) error {
	state, err := taskworkflow.RetryChecklistProjection(id, s.Store, repair)
	if err != nil {
		return err
	}
	return s.Report(id, state)
}

func observeDispatchedAfterRunnerError(state workflow.RuntimeState) bool {
	request := state.Automation.PreparedRequest
	return state.SchemaVersion < 10 && request != nil && request.DispatchedAt != "" && request.TaskID == state.Task.ID && state.WriteLease != nil && state.WriteLease.AttemptID == request.AttemptID
}

func syncSupervisorChecklist(id string, store *workflow.Store) error {
	for attempt := 0; attempt < 3; attempt++ {
		_, err := taskworkflow.SyncWorkflowChecklist(id, store)
		if err == nil { return nil }
		var projectionErr *task.ChecklistProjectionError
		if errors.As(err, &projectionErr) || errors.Is(err, os.ErrPermission) || attempt == 2 { return err }
		time.Sleep(time.Second)
	}
	return nil
}

func (s Supervisor) awaitGateDecision(id, leaseID string) error {
	state, err := s.Store.Load(workflow.TaskID(id))
	if err != nil { return err }
	for _, gate := range state.Gates {
		if gate.State != workflow.GateOpen { continue }
		report, err := workflow.BuildRemediationReportForGate(state, gate)
		if err != nil { return err }
		pending, err := s.Store.AwaitRemediation(workflow.TaskID(id), report)
		if err != nil { return err }
		if err := s.Report(id, pending); err != nil { return err }
		_, err = s.Store.PauseSupervisor(workflow.TaskID(id), leaseID, "awaiting-human", string(gate.ID), time.Now().UTC().Add(24*time.Hour))
		return err
	}
	return fmt.Errorf("Runner reported a Gate, but no open Gate remains")
}

func (s Supervisor) awaitUnknownSession(id, leaseID string, request workflow.PreparedAgentRequest, cause error) error {
	state, err := s.Store.Load(workflow.TaskID(id))
	if err != nil { return err }
	current := state.Automation.PreparedRequest
	if current == nil || current.TaskID != request.TaskID || current.WorkItemID != request.WorkItemID || current.AttemptID != request.AttemptID || current.SessionID != request.SessionID || current.ExpectedSessionTurn != request.ExpectedSessionTurn || current.DispatchedAt != request.DispatchedAt {
		return fmt.Errorf("dispatched Session binding changed before remediation: %w", cause)
	}
	report, err := workflow.BuildRemediationReportForUnknownSession(state, request, cause.Error())
	if err != nil { return err }
	pending, err := s.Store.AwaitRemediation(workflow.TaskID(id), report)
	if err != nil { return err }
	if err := s.Report(id, pending); err != nil { return err }
	_, err = s.Store.PauseSupervisor(workflow.TaskID(id), leaseID, "awaiting-human", report.ProblemID, time.Now().UTC().Add(24*time.Hour))
	return err
}
