package workflow

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
)

type DeliveryPath struct {
	Path string `json:"path"`
	SHA256 string `json:"sha256"`
}

type DeliveryAction struct {
	ID string `json:"id"`
	Kind string `json:"kind"`
	// Target is a frozen branch or worktree identity, never shell text.
	Target string `json:"target"`
	Before ActorReference `json:"before"`
	After ActorReference `json:"after"`
	Grant ActorReference `json:"grant"`
}

type ManagedDeliveryPlan struct {
	ID string `json:"id"`
	TaskID TaskID `json:"task_id"`
	Binding WorkspaceBinding `json:"binding"`
	TaskCommit string `json:"task_commit"`
	ParentCommit string `json:"parent_commit"`
	Message string `json:"message"`
	Accepted []ActorReference `json:"accepted"`
	Paths []DeliveryPath `json:"paths"`
	Actions []DeliveryAction `json:"actions"`
	Policy ActorReference `json:"policy"`
	Sources *ActorReference `json:"sealed_sources,omitempty"`
}

type ManagedDelivery struct {
	Plan ManagedDeliveryPlan `json:"plan"`
	PlanReference ActorReference `json:"plan_reference"`
	Actions []DeliveryActionRecord `json:"actions"`
}

type DeliveryActionRecord struct {
	ID string `json:"id"`
	State string `json:"state"`
	Executor string `json:"executor"`
	Observation *ActorReference `json:"observation,omitempty"`
	History []ActorReference `json:"history,omitempty"`
	Commit string `json:"commit,omitempty"`
}

type DeliveryObservation struct {
	ActionID string `json:"action_id"`
	Executor string `json:"executor"`
	State string `json:"state"`
	Commit string `json:"commit,omitempty"`
	Evidence ActorReference `json:"evidence"`
}

// DeliveryServices are supplied by the managed Git/AIW adapter. Check must
// inspect the real target, current grant, hooks, index, Stop, parent commits
// and sealed sources. Verify uses exact commit/ancestry facts, never messages
// or missing paths alone. Neither function performs a Git write.
type DeliveryServices struct {
	Check func(RuntimeState, ManagedDeliveryPlan, DeliveryAction) error
	Verify func(RuntimeState, ManagedDeliveryPlan, DeliveryAction, DeliveryObservation) error
	Account func(*RuntimeState, ManagedDeliveryPlan, DeliveryAction, string) error
}

func (s *Store) PrepareManagedDelivery(id TaskID, revision uint64, plan ManagedDeliveryPlan) (RuntimeState, error) {
	if s.DeliveryServices == nil || s.DeliveryServices.Check == nil || s.DeliveryServices.Verify == nil || s.DeliveryServices.Account == nil { return RuntimeState{}, errors.New("managed delivery authorization, budget and host adapters are unavailable") }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.delivery.prepared", Detail: plan.ID}, func(state *RuntimeState) error {
		if state.Protocol == nil || state.Protocol.Stop != nil { return errors.New("execution is disabled or stopped") }
		if err := requireNoStageInFlight(*state); err != nil { return err }
		if previous := state.Protocol.Delivery; previous != nil {
			if previous.Plan.ID == plan.ID { return errors.New("a changed delivery plan requires a new immutable identity") }
			for _, action := range previous.Actions { if action.State == "unknown" { return errors.New("reconcile the existing delivery action before changing its plan") } }
		}
		binding, _ := json.Marshal(state.Workspace)
		plannedBinding, _ := json.Marshal(plan.Binding)
		if state.Workspace == nil || string(binding) != string(plannedBinding) || state.Workspace.WorktreePath == "" || state.Delivery == DeliveryDiscarded { return errors.New("delivery requires a preserved isolated workspace binding") }
		if plan.ID == "" || plan.TaskID != id || plan.TaskCommit == "" || plan.ParentCommit == "" || !validProtocolReference(plan.Policy) || len(plan.Actions) == 0 { return errors.New("delivery plan is incomplete") }
		for _, item := range state.WorkItems {
			if item.State != WorkItemCompleted || item.AcceptedReference == nil { return errors.New("delivery requires current Core acceptance for every Work Item") }
			found := false
			for _, ref := range plan.Accepted { if ref == *item.AcceptedReference { found = true } }
			if !found { return errors.New("delivery plan omitted an accepted Work Item") }
		}
		for _, gate := range state.Gates { if gate.State == GateOpen { return errors.New("delivery has an open Gate") } }
		if len(plan.Paths) == 0 { return errors.New("delivery needs explicit content-bound paths") }
		paths := map[string]bool{}
		for _, path := range plan.Paths {
			clean := filepath.ToSlash(filepath.Clean(path.Path))
			if clean != path.Path || filepath.IsAbs(path.Path) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, ".git/") || clean == ".git" || len(path.SHA256) != 64 || paths[clean] { return errors.New("delivery paths must be unique, relative and content-bound") }
			paths[clean] = true
		}
		seen := map[string]bool{}
		lastOrder := 0
		hasMerge := false
		for _, action := range plan.Actions {
			order := map[string]int{"create-branch": 1, "commit": 2, "merge": 3, "remove-worktree": 4, "unbind-workspace": 5, "remove-branch": 6}[action.Kind]
			if order == 0 || order <= lastOrder || action.ID == "" || seen[action.ID] || action.Target == "" || !validProtocolReference(action.Before) || !validProtocolReference(action.After) || !validProtocolReference(action.Grant) { return errors.New("unsupported, repeated or unordered delivery action") }
			if state.Delivery == DeliveryMerged && order < 4 { return errors.New("a merged Task can only resume authorized cleanup") }
			if action.Kind == "merge" && action.Target != plan.Binding.ParentBranch { return errors.New("merge target differs from the recorded parent") }
			if action.Kind == "merge" { hasMerge = true }
			if (action.Kind == "commit" || action.Kind == "merge") && strings.TrimSpace(plan.Message) == "" { return errors.New("commit and merge require a frozen message") }
			if (action.Kind == "commit" || action.Kind == "create-branch" || action.Kind == "remove-branch") && action.Target != plan.Binding.TaskBranch { return errors.New("branch action differs from the Task binding") }
			if (action.Kind == "remove-worktree" || action.Kind == "unbind-workspace") && action.Target != plan.Binding.WorktreePath { return errors.New("cleanup target differs from the Task binding") }
			if order >= 4 && (plan.Sources == nil || !validProtocolReference(*plan.Sources)) { return errors.New("cleanup waits for a complete E05 sealed-source manifest") }
			seen[action.ID], lastOrder = true, order
		}
		if state.Delivery != DeliveryMerged && !hasMerge { return errors.New("local delivery must include the recorded parent merge") }
		// Later actions are checked immediately before their own side effects;
		// their expected preconditions may depend on earlier produced commits.
		if err := s.DeliveryServices.Check(*state, plan, plan.Actions[0]); err != nil { return err }
		ref, err := s.persistProtocolArtifactLocked(id, "delivery-plan", plan)
		if err != nil { return err }
		if state.Protocol.Delivery != nil { state.Protocol.DeliveryHistory = append(state.Protocol.DeliveryHistory, *state.Protocol.Delivery) }
		state.Protocol.Delivery = &ManagedDelivery{Plan: plan, PlanReference: ref}
		return nil
	})
}

// ClaimDeliveryAction publishes an unknown result before invoking the host.
// There is no blind replay: unknown/failed actions require a saved observation.
func (s *Store) ClaimDeliveryAction(id TaskID, revision uint64, actionID, executor string) (RuntimeState, error) {
	if s.DeliveryServices == nil || s.DeliveryServices.Check == nil || s.DeliveryServices.Account == nil { return RuntimeState{}, errors.New("managed delivery service is unavailable") }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.delivery.intent", Detail: actionID}, func(state *RuntimeState) error {
		if state.Protocol == nil || state.Protocol.Stop != nil || state.Protocol.Delivery == nil || executor == "" { return errors.New("delivery is stopped or unavailable") }
		if err := requireNoStageInFlight(*state); err != nil { return err }
		d := state.Protocol.Delivery
		index := len(d.Actions)
		if index > 0 && d.Actions[index-1].State == "not-dispatched" { index-- }
		if index >= len(d.Plan.Actions) || d.Plan.Actions[index].ID != actionID { return errors.New("reconcile the original action; do not repeat a side effect") }
		for _, prior := range d.Actions[:index] { if prior.State != "completed" { return errors.New("prior delivery action requires reconciliation") } }
		action := d.Plan.Actions[index]
		if (action.Kind == "remove-worktree" || action.Kind == "unbind-workspace" || action.Kind == "remove-branch") && state.Delivery != DeliveryMerged { return errors.New("cleanup requires recorded merge ancestry") }
		if err := s.DeliveryServices.Check(*state, d.Plan, action); err != nil { return err }
		if err := s.DeliveryServices.Account(state, d.Plan, action, "reserve"); err != nil { return err }
		if index < len(d.Actions) { d.Actions[index].State, d.Actions[index].Executor = "unknown", executor } else { d.Actions = append(d.Actions, DeliveryActionRecord{ID: actionID, Executor: executor, State: "unknown"}) }
		return nil
	})
}

func (s *Store) ObserveDeliveryAction(id TaskID, revision uint64, observation DeliveryObservation) (RuntimeState, error) {
	if s.DeliveryServices == nil || s.DeliveryServices.Verify == nil || s.DeliveryServices.Account == nil { return RuntimeState{}, errors.New("delivery reconciliation service is unavailable") }
	return s.updateWithEvent(id, &revision, Event{Type: "protocol.delivery.observed", Detail: observation.ActionID}, func(state *RuntimeState) error {
		if state.Protocol == nil || state.Protocol.Delivery == nil { return errors.New("no managed delivery") }
		d := state.Protocol.Delivery
		for i := range d.Actions {
			record := &d.Actions[i]
			if record.ID != observation.ActionID { continue }
			if observation.Executor != record.Executor || !validProtocolReference(observation.Evidence) { return errors.New("delivery observation ownership mismatch") }
			if observation.State != "completed" && observation.State != "unknown" && observation.State != "failed" && observation.State != "not-dispatched" { return errors.New("unsupported delivery observation") }
			if record.State == "completed" && (observation.State != "completed" || record.Commit != observation.Commit) { return errors.New("cannot reverse a completed delivery action") }
			if err := s.DeliveryServices.Verify(*state, d.Plan, d.Plan.Actions[i], observation); err != nil { return err }
			ref, err := s.persistProtocolArtifactLocked(id, "delivery-observation", observation)
			if err != nil { return err }
			if record.Observation != nil && *record.Observation == ref { return errProtocolNoChange }
			if record.State != observation.State {
				if err := s.DeliveryServices.Account(state, d.Plan, d.Plan.Actions[i], observation.State); err != nil { return err }
			}
			record.State, record.Observation, record.Commit = observation.State, &ref, observation.Commit
			record.History = append(record.History, ref)
			if observation.State == "completed" && d.Plan.Actions[i].Kind == "merge" { state.Delivery = DeliveryMerged }
			return nil
		}
		return errors.New("delivery action was never claimed")
	})
}
