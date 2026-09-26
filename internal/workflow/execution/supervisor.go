// Package execution coordinates managed Workflow execution and its adapters.
// Workflow Core owns state transitions; task owns Task artifacts and bindings.
package execution

import (
	"context"
	"errors"
	"fmt"
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
	if state.Delivery == workflow.DeliveryMerged ||
		state.Delivery == workflow.DeliveryDiscarded {
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
	for _, repair := range current.Automation.ProjectionRepairs {
		if repair.ResolvedAt != "" {
			continue
		}
		if err := s.repair(id); err != nil {
			_, _ = store.RecordSupervisorObservation(workflow.TaskID(id), leaseID, current.LastEventSequence, "repair-paused", repair.Target)
			return err
		}
		current, err = store.Load(workflow.TaskID(id))
		if err != nil {
			return err
		}
		_, err = store.RecordSupervisorObservation(workflow.TaskID(id), leaseID, current.LastEventSequence, "repair-resolved", repair.Target)
		return err
	}
	for {
		if _, err := taskworkflow.SyncWorkflowChecklist(id, store); err != nil {
			return err
		}
		updated, err := store.Load(workflow.TaskID(id))
		if err != nil || updated.Automation.Supervisor.LeaseID != leaseID {
			return err
		}
		if workflow.SupervisorDue(updated, time.Now().UTC()) ||
			updated.Automation.Supervisor.ObservedEvent == 0 {
			if _, err := store.RenewSupervisorLease(workflow.TaskID(id), leaseID); err != nil {
				return err
			}
			if err := s.RunStep(id, true, provider, model); err != nil {
				failed, loadErr := store.Load(workflow.TaskID(id))
				// Compiler preparation and repair errors retain their Attempt;
				// only an ordinary Agent runner failure consumes no-progress.
				if loadErr == nil && failed.SchemaVersion < 10 && failed.Automation.PreparedRequest != nil && !hasPendingSupervisedCompile(failed.Automation.PreparedRequest) {
					attemptID := failed.Automation.PreparedRequest.AttemptID
					outcome := workflow.SupervisedOutcome{Kind: workflow.SupervisedOutcomeNoProgress, Detail: err.Error(), EvidenceReference: "supervisor runner error"}
					if failed, loadErr = store.RecordSupervisedOutcome(workflow.TaskID(id), attemptID, outcome); loadErr == nil {
						handled, remediationErr := s.remediateRunnerFailure(id, failed, attemptID, outcome)
						if remediationErr != nil {
							_, _ = store.PauseSupervisor(workflow.TaskID(id), leaseID, "remediation-paused", remediationErr.Error(), time.Now().UTC().Add(30*time.Second))
							return remediationErr
						}
						if handled {
							continue
						}
						if pending, pendingErr := store.Load(workflow.TaskID(id)); pendingErr == nil && pending.Automation.Cursor.Result == "awaiting-human" {
							_ = s.Report(id, pending)
							_, pauseErr := store.PauseSupervisor(workflow.TaskID(id), leaseID, "awaiting-human", pending.Automation.Cursor.Detail, time.Now().UTC().Add(24*time.Hour))
							return pauseErr
						}
						_ = s.Report(id, failed)
					}
				}
				_, _ = store.PauseSupervisor(workflow.TaskID(id), leaseID, "runner-paused", err.Error(), time.Now().UTC().Add(30*time.Second))
				return err
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
				if outcomeErr != nil {
					var reportErr *reportValidationError
					if errors.As(outcomeErr, &reportErr) {
						if request.ReportOrigin == nil {
							if _, err := store.PrepareReportSupplement(request.TaskID, *request, reportErr.Error()); err == nil { continue }
						}
						_, _ = store.OpenCompilerGate(request.TaskID, request.WorkItemID, "report-manual-review", reportErr.Error())
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
			case string(workflow.RunnerGate), "repair-required", "blocked":
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

// repair synchronizes Task artifacts before reporting the repaired state.
func (s Supervisor) repair(id string) error {
	state, err := taskworkflow.RepairWorkflowChecklist(id)
	if err != nil {
		return err
	}
	return s.Report(id, state)
}
