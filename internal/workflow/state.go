package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

type TransitionError struct {
	Entity string
	ID     string
	From   string
	To     string
}

func (e TransitionError) Error() string {
	return fmt.Sprintf("invalid %s transition for %s: %s -> %s", e.Entity, e.ID, e.From, e.To)
}

func ValidateWorkItemTransition(id WorkItemID, from, to WorkItemState) error {
	if from == to {
		return nil
	}
	if workItemTransitions[from][to] {
		return nil
	}
	return TransitionError{Entity: "work item", ID: string(id), From: string(from), To: string(to)}
}

func ValidateAttemptTransition(id AttemptID, from, to AttemptState) error {
	if from == to {
		return nil
	}
	if attemptTransitions[from][to] {
		return nil
	}
	return TransitionError{Entity: "attempt", ID: string(id), From: string(from), To: string(to)}
}

var workItemTransitions = map[WorkItemState]map[WorkItemState]bool{
	WorkItemPlanned: {WorkItemReady: true, WorkItemCancelled: true},
	WorkItemReady: {WorkItemLeased: true, WorkItemCompleted: true, WorkItemBlocked: true, WorkItemCancelled: true},
	WorkItemLeased: {WorkItemRunning: true, WorkItemReady: true, WorkItemBlocked: true, WorkItemCancelled: true},
	WorkItemRunning: {WorkItemCompleted: true, WorkItemBlocked: true, WorkItemReady: true, WorkItemCancelled: true},
	WorkItemBlocked: {WorkItemReady: true, WorkItemCancelled: true},
}

var attemptTransitions = map[AttemptState]map[AttemptState]bool{
	AttemptCreated: {AttemptRunning: true, AttemptCancelled: true},
	AttemptRunning: {AttemptPaused: true, AttemptFailed: true, AttemptCompleted: true, AttemptCancelled: true},
	AttemptPaused: {AttemptRunning: true, AttemptCancelled: true},
	AttemptFailed: {AttemptCancelled: true},
}

func ValidateRuntimeState(state RuntimeState) error {
	if state.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported workflow schema version: %d", state.SchemaVersion)
	}
	workItems := make(map[WorkItemID]struct{}, len(state.WorkItems))
	for _, item := range state.WorkItems {
		if item.ID == "" {
			return fmt.Errorf("work item id is required")
		}
		if err := validateRetryPolicy(item.RetryPolicy); err != nil {
			return fmt.Errorf("work item %s: %w", item.ID, err)
		}
		if item.NoProgressCount < 0 {
			return fmt.Errorf("work item %s has negative no-progress count", item.ID)
		}
		if _, exists := workItems[item.ID]; exists {
			return fmt.Errorf("duplicate work item id: %s", item.ID)
		}
		workItems[item.ID] = struct{}{}
	}
	for _, item := range state.WorkItems {
		for _, dependency := range item.Dependencies {
			if _, exists := workItems[dependency]; !exists {
				return fmt.Errorf("work item %s depends on unknown work item %s", item.ID, dependency)
			}
		}
	}
	if err := validateAttemptReferences(state.Attempts, workItems); err != nil {
		return err
	}
	if err := validateGateReferences(state.Gates, workItems); err != nil {
		return err
	}
	if err := validateEvidenceReferences(state.Evidence, workItems); err != nil {
		return err
	}
	if err := validateFocusedTestAuthorization(state.FocusedTestAuthorization); err != nil {
		return err
	}
	if state.FocusedTestPlanDigest != "" {
		if err := validateVerificationPlanDigest(state.FocusedTestPlanDigest); err != nil {
			return fmt.Errorf("active focused-test plan: %w", err)
		}
	}
	if state.Cancellation != nil {
		if strings.TrimSpace(state.Cancellation.Reason) == "" {
			return errors.New("cancellation reason is required")
		}
		if state.Cancellation.Delivery != DeliveryMerged && state.Cancellation.Delivery != DeliveryDiscarded {
			return fmt.Errorf("unsupported cancellation delivery: %s", state.Cancellation.Delivery)
		}
		if state.Delivery != DeliveryPending && state.Delivery != state.Cancellation.Delivery {
			return errors.New("cancellation delivery does not match the requested terminal delivery")
		}
	}
	return validateWriteLease(state.WriteLease, state.Attempts)
}

// FocusedTestAuthorizationState returns the authorization status for exactly
// one normalized Verification Plan digest. A changed digest is stale rather
// than implicitly covered by the prior approval.
func (state RuntimeState) FocusedTestAuthorizationState(planDigest string) FocusedTestAuthorizationState {
	authorization := state.FocusedTestAuthorization
	if authorization == nil {
		return FocusedTestAuthorizationDisabled
	}
	if authorization.PlanDigest != planDigest {
		return FocusedTestAuthorizationStale
	}
	return FocusedTestAuthorizationAuthorized
}

func validateFocusedTestAuthorization(authorization *FocusedTestAuthorization) error {
	if authorization == nil {
		return nil
	}
	if authorization.Profile != FocusedTestProfile {
		return fmt.Errorf("focused-test authorization profile must be %q", FocusedTestProfile)
	}
	if err := validateVerificationPlanDigest(authorization.PlanDigest); err != nil {
		return fmt.Errorf("focused-test authorization: %w", err)
	}
	if strings.TrimSpace(authorization.Approver) == "" {
		return errors.New("focused-test authorization approver is required")
	}
	if _, err := time.Parse(time.RFC3339, authorization.ApprovedAt); err != nil {
		return fmt.Errorf("focused-test authorization approved_at must be RFC3339: %w", err)
	}
	return nil
}

func validateVerificationPlanDigest(digest string) error {
	if len(digest) != sha256.Size*2 || strings.ToLower(digest) != digest {
		return errors.New("verification plan digest must be a lowercase SHA-256 hex digest")
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return fmt.Errorf("verification plan digest must be hexadecimal: %w", err)
	}
	return nil
}

func validateRetryPolicy(policy RetryPolicy) error {
	if policy.MaxAttempts < MinRetryLimit || policy.MaxAttempts > MaxRetryLimit {
		return fmt.Errorf("retry limit must be between %d and %d", MinRetryLimit, MaxRetryLimit)
	}
	return nil
}

func validateAttemptReferences(attempts []Attempt, workItems map[WorkItemID]struct{}) error {
	seen := make(map[AttemptID]struct{}, len(attempts))
	for _, attempt := range attempts {
		if attempt.ID == "" {
			return fmt.Errorf("attempt id is required")
		}
		if _, exists := seen[attempt.ID]; exists {
			return fmt.Errorf("duplicate attempt id: %s", attempt.ID)
		}
		seen[attempt.ID] = struct{}{}
		if _, exists := workItems[attempt.WorkItemID]; !exists {
			return fmt.Errorf("attempt %s references unknown work item %s", attempt.ID, attempt.WorkItemID)
		}
	}
	return nil
}

func validateGateReferences(gates []Gate, workItems map[WorkItemID]struct{}) error {
	seen := make(map[GateID]struct{}, len(gates))
	for _, gate := range gates {
		if gate.ID == "" {
			return fmt.Errorf("gate id is required")
		}
		if _, exists := seen[gate.ID]; exists {
			return fmt.Errorf("duplicate gate id: %s", gate.ID)
		}
		seen[gate.ID] = struct{}{}
		if gate.WorkItemID != "" {
			if _, exists := workItems[gate.WorkItemID]; !exists {
				return fmt.Errorf("gate %s references unknown work item %s", gate.ID, gate.WorkItemID)
			}
		}
	}
	return nil
}

func validateEvidenceReferences(evidence []Evidence, workItems map[WorkItemID]struct{}) error {
	seen := make(map[EvidenceID]struct{}, len(evidence))
	for _, record := range evidence {
		if record.ID == "" {
			return fmt.Errorf("evidence id is required")
		}
		if _, exists := seen[record.ID]; exists {
			return fmt.Errorf("duplicate evidence id: %s", record.ID)
		}
		seen[record.ID] = struct{}{}
		if record.WorkItemID != "" {
			if _, exists := workItems[record.WorkItemID]; !exists {
				return fmt.Errorf("evidence %s references unknown work item %s", record.ID, record.WorkItemID)
			}
		}
	}
	return nil
}

func validateWriteLease(lease *WriteLease, attempts []Attempt) error {
	if lease == nil {
		return nil
	}
	if lease.AttemptID == "" || lease.Workspace == "" {
		return errors.New("write lease requires attempt id and workspace")
	}
	for _, attempt := range attempts {
		if attempt.ID == lease.AttemptID {
			if attempt.Workspace != lease.Workspace {
				return fmt.Errorf("write lease workspace does not match attempt %s", lease.AttemptID)
			}
			return nil
		}
	}
	return fmt.Errorf("write lease references unknown attempt %s", lease.AttemptID)
}

func DeriveSummary(state RuntimeState) TaskSummary {
	summary := TaskSummary{
		Planning:  state.Planning,
		Execution: deriveExecution(state),
		Validation: deriveValidation(state),
		Delivery:   state.Delivery,
		Workspace:  state.Task.Kind,
		BlockedBy:  openBlockingGates(state.Gates),
		Active:     activeAttempt(state.Attempts),
	}
	summary.Status = deriveDisplayState(summary)
	return summary
}

func deriveExecution(state RuntimeState) ExecutionState {
	if state.Cancellation != nil {
		return ExecutionCancelled
	}
	if hasBlockingExecutionGate(state.Gates) || hasWorkItemState(state.WorkItems, WorkItemBlocked) {
		return ExecutionBlocked
	}
	if hasAttemptState(state.Attempts, AttemptRunning) || hasWorkItemState(state.WorkItems, WorkItemRunning) {
		return ExecutionRunning
	}
	if hasWorkItemState(state.WorkItems, WorkItemLeased) {
		return ExecutionLeased
	}
	if len(state.WorkItems) > 0 && allWorkItemsCompleted(state.WorkItems) {
		return ExecutionCompleted
	}
	return ExecutionQueued
}

func deriveValidation(state RuntimeState) ValidationState {
	if state.FocusedTestPlanDigest != "" && state.FocusedTestAuthorizationState(state.FocusedTestPlanDigest) != FocusedTestAuthorizationAuthorized {
		return ValidationAwaitingAuthorization
	}
	if hasOpenGate(state.Gates, GateAuthorization) {
		return ValidationAwaitingAuthorization
	}
	if hasEvidenceState(state.Evidence, EvidencePending) || hasOpenGate(state.Gates, GateValidation) {
		return ValidationPending
	}
	if hasEvidenceState(state.Evidence, EvidenceFailed) {
		return ValidationFailed
	}
	if hasEvidenceState(state.Evidence, EvidencePassed) {
		return ValidationPassed
	}
	if state.FocusedTestPlanDigest != "" {
		return ValidationPending
	}
	if hasEvidenceState(state.Evidence, EvidenceWaived) {
		return ValidationWaived
	}
	return ValidationNotRequired
}

func deriveDisplayState(summary TaskSummary) TaskDisplayState {
	if summary.Execution == ExecutionCancelled {
		return TaskCancelled
	}
	if summary.Planning == PlanningNeedsDecision {
		return TaskNeedsDecision
	}
	if summary.Execution == ExecutionBlocked {
		return TaskBlocked
	}
	if summary.Validation == ValidationAwaitingAuthorization {
		return TaskAwaitingAuthorization
	}
	if summary.Execution == ExecutionCompleted {
		if summary.Validation == ValidationPending || summary.Validation == ValidationFailed {
			return TaskAwaitingVerification
		}
		return TaskDone
	}
	if summary.Execution == ExecutionRunning || summary.Execution == ExecutionLeased {
		return TaskInProgress
	}
	if summary.Planning == PlanningDraft || summary.Planning == "" {
		return TaskDraft
	}
	return TaskReady
}

func hasWorkItemState(items []WorkItem, wanted WorkItemState) bool {
	for _, item := range items {
		if item.State == wanted {
			return true
		}
	}
	return false
}

func allWorkItemsCompleted(items []WorkItem) bool {
	for _, item := range items {
		if item.State != WorkItemCompleted {
			return false
		}
	}
	return true
}

func hasAttemptState(attempts []Attempt, wanted AttemptState) bool {
	for _, attempt := range attempts {
		if attempt.State == wanted {
			return true
		}
	}
	return false
}

func activeAttempt(attempts []Attempt) AttemptID {
	for _, attempt := range attempts {
		if attempt.State == AttemptRunning {
			return attempt.ID
		}
	}
	return ""
}

func hasEvidenceState(records []Evidence, wanted EvidenceState) bool {
	for _, record := range records {
		if record.State == wanted {
			return true
		}
	}
	return false
}

func hasOpenGate(gates []Gate, kind GateKind) bool {
	for _, gate := range gates {
		if gate.Kind == kind && gate.State == GateOpen {
			return true
		}
	}
	return false
}

func hasBlockingExecutionGate(gates []Gate) bool {
	return hasOpenGate(gates, GateDependency) || hasOpenGate(gates, GateDecision)
}

func openBlockingGates(gates []Gate) []GateID {
	var result []GateID
	for _, gate := range gates {
		if gate.State == GateOpen {
			result = append(result, gate.ID)
		}
	}
	return result
}
