package workflow

import (
	"fmt"
	"strings"
	"time"
)

// ChecklistCandidate is a human-authored checklist reference prepared by an
// OpenSpec adapter. The Core assigns and retains its stable WorkItemID.
type ChecklistCandidate struct {
	Item      string
	Title     string
	Completed bool
}

// ReconcileChecklist maps new checklist references to Work Items while
// preserving existing IDs, state, dependencies, Attempts, Gates, and Evidence.
// References no longer present are retained for auditability and diagnosed by
// adapters rather than silently deleted.
func (s *Store) ReconcileChecklist(id TaskID, candidates []ChecklistCandidate) (RuntimeState, error) {
	return s.syncChecklist(id, candidates, "")
}

// RepairChecklist imports authored completion markers for a Task created
// before Workflow Core became the sole runtime owner. It is deliberately
// one-time: later synchronization follows the ordinary fingerprinted path.
// The operation never reads or writes the OpenSpec artifact itself.
func (s *Store) RepairChecklist(id TaskID, candidates []ChecklistCandidate) (RuntimeState, error) {
	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	if state.Automation.ChecklistRepairedAt != "" {
		return state, nil
	}
	return s.syncChecklistWithEvent(id, candidates, "", "checklist.repaired", func(state *RuntimeState) {
		state.Automation.ChecklistRepairedAt = time.Now().UTC().Format(time.RFC3339)
	})
}

// SyncChecklist incrementally reconciles a parsed plan. The fingerprint is
// supplied by the OpenSpec adapter so Workflow Core remains independent of
// Markdown parsing. An unchanged non-empty fingerprint produces no write.
func (s *Store) SyncChecklist(id TaskID, candidates []ChecklistCandidate, fingerprint string) (RuntimeState, error) {
	if strings.TrimSpace(fingerprint) == "" {
		return RuntimeState{}, fmt.Errorf("plan fingerprint is required")
	}
	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	if state.Automation.PlanFingerprint == fingerprint {
		return state, nil
	}
	return s.syncChecklist(id, candidates, fingerprint)
}

func (s *Store) syncChecklist(id TaskID, candidates []ChecklistCandidate, fingerprint string) (RuntimeState, error) {
	return s.syncChecklistWithEvent(id, candidates, fingerprint, "checklist.reconciled", nil)
}

func (s *Store) syncChecklistWithEvent(id TaskID, candidates []ChecklistCandidate, fingerprint, eventType string, after func(*RuntimeState)) (RuntimeState, error) {
	if err := validateChecklistCandidates(candidates); err != nil {
		return RuntimeState{}, err
	}
	return s.UpdateWithEvent(id, Event{Type: eventType, Detail: fmt.Sprintf("%d checklist items", len(candidates))}, func(state *RuntimeState) error {
		byReference := make(map[string]int, len(state.WorkItems))
		usedIDs := make(map[WorkItemID]bool, len(state.WorkItems))
		for index, item := range state.WorkItems {
			if item.Checklist.Item == "" {
				continue
			}
			if _, exists := byReference[item.Checklist.Item]; exists {
				return fmt.Errorf("incompatible existing checklist mapping: %s", item.Checklist.Item)
			}
			byReference[item.Checklist.Item] = index
			usedIDs[item.ID] = true
		}
		for _, candidate := range candidates {
			if index, exists := byReference[candidate.Item]; exists {
				state.WorkItems[index].Title = candidate.Title
				continue
			}
			id := nextAvailableWorkItemID(usedIDs)
			usedIDs[id] = true
			state.WorkItems = append(state.WorkItems, WorkItem{
				ID: id, Checklist: ChecklistReference{Item: candidate.Item}, Title: candidate.Title, State: WorkItemReady,
				RetryPolicy: RetryPolicy{MaxAttempts: DefaultRetryLimit},
			})
			byReference[candidate.Item] = len(state.WorkItems) - 1
		}
		for _, candidate := range candidates {
			if !candidate.Completed {
				continue
			}
			index := byReference[candidate.Item]
			if state.WorkItems[index].State == WorkItemCompleted {
				continue
			}
			closeChecklistAttempt(state, state.WorkItems[index].ID)
			if err := completeWorkItem(state, state.WorkItems[index].ID); err != nil {
				return fmt.Errorf("synchronize completed checklist item %s: %w", candidate.Item, err)
			}
		}
		present := make(map[string]bool, len(candidates))
		for _, candidate := range candidates {
			present[candidate.Item] = true
		}
		for _, item := range state.WorkItems {
			if item.Checklist.Item == "" || present[item.Checklist.Item] {
				continue
			}
			gateID := GateID("plan-mapping-review-" + string(item.ID))
			found := false
			for _, gate := range state.Gates {
				if gate.ID == gateID {
					found = true
					break
				}
			}
			if !found {
				state.Gates = append(state.Gates, Gate{ID: gateID, WorkItemID: item.ID, Kind: GateDecision, State: GateOpen, Reason: "mapped checklist item is absent from the current plan"})
			}
		}
		if fingerprint != "" {
			state.Automation.PlanFingerprint = fingerprint
		}
		if after != nil {
			after(state)
		}
		return nil
	})
}

// closeChecklistAttempt releases a prepared or running Attempt when its
// authoritative OpenSpec checklist item has already been marked complete.
// This can happen when an isolated Agent updates tasks.md before the
// Supervisor records the turn outcome.
func closeChecklistAttempt(state *RuntimeState, workItemID WorkItemID) {
	for index := range state.Attempts {
		attempt := &state.Attempts[index]
		if attempt.WorkItemID != workItemID || attempt.State != AttemptRunning {
			continue
		}
		attempt.State = AttemptCompleted
		attempt.EndedAt = time.Now().UTC().Format(time.RFC3339)
		if state.WriteLease != nil && state.WriteLease.AttemptID == attempt.ID {
			state.WriteLease = nil
		}
		if request := state.Automation.PreparedRequest; request != nil && request.AttemptID == attempt.ID {
			state.Automation.PreparedRequest = nil
		}
	}
}

func validateChecklistCandidates(candidates []ChecklistCandidate) error {
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		if candidate.Item == "" || candidate.Title == "" {
			return fmt.Errorf("checklist item and title are required")
		}
		if seen[candidate.Item] {
			return fmt.Errorf("duplicate checklist item: %s", candidate.Item)
		}
		seen[candidate.Item] = true
	}
	return nil
}

func nextAvailableWorkItemID(used map[WorkItemID]bool) WorkItemID {
	for sequence := uint(1); ; sequence++ {
		id := NewWorkItemID(sequence)
		if !used[id] {
			return id
		}
	}
}

// PlanWork registers a ready Work Item. Callers provide the stable ID and
// human-readable checklist reference; the Core verifies dependency ownership.
func (s *Store) PlanWork(id TaskID, item WorkItem) (RuntimeState, error) {
	if item.ID == "" || item.Title == "" {
		return RuntimeState{}, fmt.Errorf("work item id and title are required")
	}
	if item.RetryPolicy.MaxAttempts == 0 {
		item.RetryPolicy.MaxAttempts = DefaultRetryLimit
	}
	if err := validateRetryPolicy(item.RetryPolicy); err != nil {
		return RuntimeState{}, err
	}
	return s.UpdateWithEvent(id, Event{Type: "work-item.planned", WorkItemID: item.ID, Detail: item.Title}, func(state *RuntimeState) error {
		for _, existing := range state.WorkItems {
			if existing.ID == item.ID {
				return fmt.Errorf("work item already exists: %s", item.ID)
			}
		}
		item.State = WorkItemReady
		state.WorkItems = append(state.WorkItems, item)
		return nil
	})
}

// EnsureLegacyManagedWorkItem retains the unmapped Task compatibility path
// while making its one-time bootstrap visible in ordered event history.
func (s *Store) EnsureLegacyManagedWorkItem(id TaskID) (RuntimeState, WorkItemID, error) {
	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, "", err
	}
	for _, item := range state.WorkItems {
		if item.ID == WorkItemID("wi-0001") {
			return state, item.ID, nil
		}
	}
	item := WorkItem{ID: WorkItemID("wi-0001"), Title: "Managed agent execution", State: WorkItemReady, RetryPolicy: RetryPolicy{MaxAttempts: DefaultRetryLimit}}
	state, err = s.UpdateWithEvent(id, Event{Type: "work-item.compatibility-created", WorkItemID: item.ID, Detail: item.Title}, func(current *RuntimeState) error {
		current.WorkItems = append(current.WorkItems, item)
		return nil
	})
	if err != nil {
		return RuntimeState{}, "", err
	}
	return state, item.ID, nil
}

// CheckpointAttempt changes an active Attempt only when it owns the current
// write lease. Checkpoints are intentionally limited to running/paused states.
func (s *Store) CheckpointAttempt(id TaskID, attemptID AttemptID, target AttemptState) (RuntimeState, error) {
	if target != AttemptRunning && target != AttemptPaused {
		return RuntimeState{}, fmt.Errorf("unsupported attempt checkpoint: %s", target)
	}
	return s.UpdateWithEvent(id, Event{Type: "attempt.checkpointed", AttemptID: attemptID, Detail: string(target)}, func(state *RuntimeState) error {
		if state.WriteLease == nil || state.WriteLease.AttemptID != attemptID {
			return fmt.Errorf("attempt %s does not own the write lease", attemptID)
		}
		for index := range state.Attempts {
			attempt := &state.Attempts[index]
			if attempt.ID != attemptID {
				continue
			}
			if err := ValidateAttemptTransition(attempt.ID, attempt.State, target); err != nil {
				return err
			}
			attempt.State = target
			return nil
		}
		return fmt.Errorf("unknown attempt %s", attemptID)
	})
}

func (s *Store) RecordEvidence(id TaskID, evidence Evidence) (RuntimeState, error) {
	if evidence.ID == "" || evidence.Kind == "" || evidence.State == "" {
		return RuntimeState{}, fmt.Errorf("evidence id, kind, and state are required")
	}
	if evidence.Kind == EvidenceCommand {
		state, err := s.Load(id)
		if err != nil {
			return RuntimeState{}, err
		}
		if state.FocusedTestPlanDigest != "" && state.FocusedTestAuthorizationState(state.FocusedTestPlanDigest) != FocusedTestAuthorizationAuthorized {
			return RuntimeState{}, fmt.Errorf("focused-test command evidence requires authorization for the active verification plan")
		}
		if state.FocusedTestPlanDigest == "" && !hasResolvedAuthorizationGateFor(evidence.WorkItemID, state) {
			return RuntimeState{}, fmt.Errorf("runtime validation requires a resolved authorization gate")
		}
	}
	if evidence.RecordedAt == "" {
		evidence.RecordedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return s.UpdateWithEvent(id, Event{Type: "evidence.recorded", WorkItemID: evidence.WorkItemID, Detail: string(evidence.ID)}, func(state *RuntimeState) error {
		for _, existing := range state.Evidence {
			if existing.ID == evidence.ID {
				return fmt.Errorf("evidence already exists: %s", evidence.ID)
			}
		}
		state.Evidence = append(state.Evidence, evidence)
		return nil
	})
}

// FinalizeFocusedTestEvidence changes the one pending command Evidence created
// by the controlled focused-test runner into its terminal state. Keeping this
// transition in Workflow Core prevents adapters from editing Evidence directly.
func (s *Store) FinalizeFocusedTestEvidence(id TaskID, evidenceID EvidenceID, state EvidenceState, reference string) (RuntimeState, error) {
	if evidenceID == "" {
		return RuntimeState{}, fmt.Errorf("focused-test evidence ID is required")
	}
	if state != EvidencePassed && state != EvidenceFailed {
		return RuntimeState{}, fmt.Errorf("focused-test evidence state must be passed or failed")
	}
	if strings.TrimSpace(reference) == "" {
		return RuntimeState{}, fmt.Errorf("focused-test evidence result reference is required")
	}
	return s.UpdateWithEvent(id, Event{Type: "focused-test.evidence-finalized", Detail: string(evidenceID)}, func(runtime *RuntimeState) error {
		for index := range runtime.Evidence {
			evidence := &runtime.Evidence[index]
			if evidence.ID != evidenceID {
				continue
			}
			if evidence.Kind != EvidenceCommand || evidence.State != EvidencePending {
				return fmt.Errorf("focused-test evidence %s is not pending command evidence", evidenceID)
			}
			evidence.State = state
			evidence.Reference = reference
			return nil
		}
		return fmt.Errorf("focused-test pending evidence not found: %s", evidenceID)
	})
}

// ActivateFocusedTestPlan records the normalized digest of the authored Plan
// currently offered for focused validation. Missing or stale approval opens a
// dedicated authorization Gate; this transition never starts a process.
func (s *Store) ActivateFocusedTestPlan(id TaskID, planDigest string) (RuntimeState, error) {
	if err := validateVerificationPlanDigest(planDigest); err != nil {
		return RuntimeState{}, err
	}
	return s.UpdateWithEvent(id, Event{Type: "focused-test.plan-activated", Detail: planDigest}, func(state *RuntimeState) error {
		state.FocusedTestPlanDigest = planDigest
		if state.FocusedTestAuthorizationState(planDigest) == FocusedTestAuthorizationAuthorized {
			return nil
		}
		reason := "focused-test requires explicit human authorization for verification plan digest " + planDigest
		if state.FocusedTestAuthorization != nil {
			reason = "focused-test authorization applies to verification plan digest " + state.FocusedTestAuthorization.PlanDigest + "; authorize active digest " + planDigest
		}
		for index := range state.Gates {
			if state.Gates[index].ID == FocusedTestAuthorizationGateID {
				state.Gates[index].Kind = GateAuthorization
				state.Gates[index].State = GateOpen
				state.Gates[index].Reason = reason
				return nil
			}
		}
		state.Gates = append(state.Gates, Gate{ID: FocusedTestAuthorizationGateID, Kind: GateAuthorization, State: GateOpen, Reason: reason})
		return nil
	})
}

// AuthorizeFocusedTest records an explicit human approval for one exact
// normalized Verification Plan digest. It does not resolve Gates or start a
// process; adapters must compare the current digest before execution.
func (s *Store) AuthorizeFocusedTest(id TaskID, planDigest, approver string) (RuntimeState, error) {
	if err := validateVerificationPlanDigest(planDigest); err != nil {
		return RuntimeState{}, err
	}
	if strings.TrimSpace(approver) == "" {
		return RuntimeState{}, fmt.Errorf("focused-test authorization approver is required")
	}
	authorization := &FocusedTestAuthorization{
		Profile:    FocusedTestProfile,
		PlanDigest: planDigest,
		Approver:   approver,
		ApprovedAt: time.Now().UTC().Format(time.RFC3339),
	}
	return s.UpdateWithEvent(id, Event{Type: "focused-test.authorized", Detail: planDigest}, func(state *RuntimeState) error {
		state.FocusedTestAuthorization = authorization
		if state.FocusedTestPlanDigest == planDigest {
			for index := range state.Gates {
				if state.Gates[index].ID == FocusedTestAuthorizationGateID && state.Gates[index].State == GateOpen {
					state.Gates[index].State = GateResolved
				}
			}
		}
		return nil
	})
}

// OpenFocusedTestNetworkEnforcementGate records that an otherwise selected
// focused check cannot run because its runtime cannot enforce network: deny.
// It deliberately does not change authorization or record command Evidence.
func (s *Store) OpenFocusedTestNetworkEnforcementGate(id TaskID, planDigest string, workItemID WorkItemID, detail string) (RuntimeState, error) {
	if err := validateVerificationPlanDigest(planDigest); err != nil {
		return RuntimeState{}, err
	}
	if strings.TrimSpace(string(workItemID)) == "" {
		return RuntimeState{}, fmt.Errorf("focused-test network enforcement work item ID is required")
	}
	detail = strings.TrimSpace(detail)
	if detail == "" {
		detail = "the selected runtime does not provide enforced network isolation"
	}
	reason := fmt.Sprintf("focused-test check for verification plan digest %s requires network: deny, but %s; use a runtime with enforceable network isolation", planDigest, detail)
	return s.UpdateWithEvent(id, Event{Type: "focused-test.network-enforcement-unavailable", WorkItemID: workItemID, Detail: detail}, func(state *RuntimeState) error {
		for index := range state.Gates {
			if state.Gates[index].ID != FocusedTestNetworkEnforcementGateID {
				continue
			}
			state.Gates[index].WorkItemID = workItemID
			state.Gates[index].Kind = GateAuthorization
			state.Gates[index].State = GateOpen
			state.Gates[index].Reason = reason
			return nil
		}
		state.Gates = append(state.Gates, Gate{
			ID:         FocusedTestNetworkEnforcementGateID,
			WorkItemID: workItemID,
			Kind:       GateAuthorization,
			State:      GateOpen,
			Reason:     reason,
		})
		return nil
	})
}

// SetDelivery records a delivery result only after explicit delivery
// authorization. It does not invoke Git; existing Task adapters retain their
// primary/isolated workspace policy and call this only after their own checks.
func (s *Store) SetDelivery(id TaskID, delivery DeliveryState) (RuntimeState, error) {
	if delivery != DeliveryMerged && delivery != DeliveryDiscarded {
		return RuntimeState{}, fmt.Errorf("unsupported delivery transition: %s", delivery)
	}
	state, err := s.Load(id)
	if err != nil {
		return RuntimeState{}, err
	}
	if hasGateKind(state.Gates, GateDelivery) && !hasResolvedGateKind(state.Gates, GateDelivery) {
		return RuntimeState{}, fmt.Errorf("delivery transition requires a resolved delivery gate")
	}
	return s.UpdateWithEvent(id, Event{Type: "delivery.recorded", Detail: string(delivery)}, func(current *RuntimeState) error {
		if current.Cancellation != nil && current.Cancellation.Delivery != delivery {
			return fmt.Errorf("force-close requested %s delivery, cannot record %s", current.Cancellation.Delivery, delivery)
		}
		current.Delivery = delivery
		return nil
	})
}

// RecordDeliveryFailure preserves a failed delivery attempt as runtime audit
// evidence without changing delivery, cancellation, leases, or work items.
// Callers use it after a preflight or Git merge failure so recovery can start
// from the intact Task resources rather than an inferred terminal result.
func (s *Store) RecordDeliveryFailure(id TaskID, stage, detail string) (RuntimeState, error) {
	stage = strings.TrimSpace(stage)
	detail = strings.TrimSpace(detail)
	if stage == "" || detail == "" {
		return RuntimeState{}, fmt.Errorf("delivery failure stage and detail are required")
	}
	return s.UpdateWithEvent(id, Event{
		Type:   "delivery.failed",
		Detail: fmt.Sprintf("stage=%s; detail=%s", stage, detail),
	}, func(*RuntimeState) error {
		return nil
	})
}

// ForceClose terminally stops managed execution in one recoverable Core
// transition. Delivery records the requested terminal disposition, while the
// runtime delivery state remains pending until an adapter has durably finished
// the corresponding Git delivery operation and calls SetDelivery.
func (s *Store) ForceClose(id TaskID, reason string, delivery DeliveryState) (RuntimeState, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return RuntimeState{}, fmt.Errorf("force-close reason is required")
	}
	if delivery != DeliveryMerged && delivery != DeliveryDiscarded {
		return RuntimeState{}, fmt.Errorf("unsupported force-close delivery: %s", delivery)
	}
	return s.UpdateWithEvent(id, Event{Type: "task.force-closed", Detail: fmt.Sprintf("delivery=%s; reason=%s", delivery, reason)}, func(state *RuntimeState) error {
		if state.Cancellation != nil {
			return fmt.Errorf("Task is already force-closed")
		}
		for index := range state.Attempts {
			attempt := &state.Attempts[index]
			switch attempt.State {
			case AttemptCreated, AttemptRunning, AttemptPaused:
				if err := ValidateAttemptTransition(attempt.ID, attempt.State, AttemptCancelled); err != nil {
					return err
				}
				attempt.State = AttemptCancelled
				attempt.EndedAt = time.Now().UTC().Format(time.RFC3339)
			}
		}
		for index := range state.WorkItems {
			item := &state.WorkItems[index]
			if item.State == WorkItemCompleted || item.State == WorkItemCancelled {
				continue
			}
			if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemCancelled); err != nil {
				return err
			}
			item.State = WorkItemCancelled
		}
		state.WriteLease = nil
		state.Automation.PreparedRequest = nil
		state.Automation.Supervisor.LeaseID = ""
		state.Automation.Supervisor.LeaseExpiresAt = ""
		state.Automation.Supervisor.RetryAfter = ""
		state.Automation.Supervisor.StoppedAt = time.Now().UTC().Format(time.RFC3339)
		state.Automation.Supervisor.Result = "cancelled"
		state.Automation.Supervisor.Detail = reason
		state.Delivery = DeliveryPending
		state.Cancellation = &Cancellation{Reason: reason, Delivery: delivery, RecordedAt: time.Now().UTC().Format(time.RFC3339)}
		return nil
	})
}

func (s *Store) ResolveGate(id TaskID, gateID GateID, target GateState) (RuntimeState, error) {
	if target != GateResolved && target != GateWaived {
		return RuntimeState{}, fmt.Errorf("unsupported gate resolution: %s", target)
	}
	return s.UpdateWithEvent(id, Event{Type: "gate.resolved", Detail: string(gateID)}, func(state *RuntimeState) error {
		for index := range state.Gates {
			if state.Gates[index].ID == gateID {
				if state.Gates[index].State != GateOpen {
					return fmt.Errorf("gate %s is already %s", gateID, state.Gates[index].State)
				}
				state.Gates[index].State = target
				return nil
			}
		}
		return fmt.Errorf("unknown gate %s", gateID)
	})
}

func (s *Store) CompleteWorkItem(id TaskID, workItemID WorkItemID) (RuntimeState, error) {
	return s.UpdateWithEvent(id, Event{Type: "work-item.completed", WorkItemID: workItemID}, func(state *RuntimeState) error {
		return completeWorkItem(state, workItemID)
	})
}

func completeWorkItem(state *RuntimeState, workItemID WorkItemID) error {
	for index := range state.WorkItems {
		item := &state.WorkItems[index]
		if item.ID != workItemID {
			continue
		}
		if err := completionPreconditions(*state, *item); err != nil {
			return err
		}
		if err := ValidateWorkItemTransition(item.ID, item.State, WorkItemCompleted); err != nil {
			return err
		}
		item.State = WorkItemCompleted
		return nil
	}
	return fmt.Errorf("unknown work item %s", workItemID)
}

func completionPreconditions(state RuntimeState, item WorkItem) error {
	for _, dependency := range item.Dependencies {
		for _, candidate := range state.WorkItems {
			if candidate.ID == dependency && candidate.State != WorkItemCompleted {
				return fmt.Errorf("work item %s is waiting for dependency %s", item.ID, dependency)
			}
		}
	}
	for _, gate := range state.Gates {
		if gate.WorkItemID == item.ID && gate.State == GateOpen {
			return fmt.Errorf("work item %s is blocked by gate %s", item.ID, gate.ID)
		}
	}
	for _, evidence := range state.Evidence {
		if evidence.WorkItemID == item.ID && (evidence.State == EvidencePending || evidence.State == EvidenceFailed) {
			return fmt.Errorf("work item %s has unresolved evidence %s", item.ID, evidence.ID)
		}
	}
	return nil
}

func hasResolvedAuthorizationGateFor(workItemID WorkItemID, state RuntimeState) bool {
	for _, gate := range state.Gates {
		if gate.Kind == GateAuthorization && gate.State == GateResolved && (gate.WorkItemID == "" || gate.WorkItemID == workItemID) {
			return true
		}
	}
	return false
}

func hasResolvedGateKind(gates []Gate, kind GateKind) bool {
	for _, gate := range gates {
		if gate.Kind == kind && gate.State == GateResolved {
			return true
		}
	}
	return false
}

func hasGateKind(gates []Gate, kind GateKind) bool {
	for _, gate := range gates {
		if gate.Kind == kind {
			return true
		}
	}
	return false
}
