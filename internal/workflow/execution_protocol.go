package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Schema 9 remains the default. Schema 10 is entered only by the managed
// migration after E01-E04 services and platform evidence are available.
const DurableSchemaVersion = 10

var errProtocolNoChange = errors.New("protocol fact is already committed")

type ExecutionPhase string

const (
	PhaseCoder ExecutionPhase = "coder"
	PhaseReport ExecutionPhase = "report-validation"
	PhaseCompile ExecutionPhase = "compile"
	PhaseTester ExecutionPhase = "tester"
	PhaseTest ExecutionPhase = "test-run"
	PhaseAccept ExecutionPhase = "acceptance"
	PhaseAccepted ExecutionPhase = "accepted"
)

type ExecutionProtocol struct {
	Auxiliary *TaskAuxiliary `json:"auxiliary,omitempty"`
	Grants GrantAnchor `json:"grants"`
	Version int `json:"version"`
	Migration ActorReference `json:"migration"`
	Activation []ActorReference `json:"activation"`
	Stop *ExecutionStop `json:"stop,omitempty"`
	LeaseGeneration uint64 `json:"lease_generation"`
	Items []ItemExecution `json:"items"`
	Requests []StageRecord `json:"requests"`
	Recoveries []StageRecovery `json:"recoveries"`
	Diagnoses []string `json:"diagnoses"`
	BudgetKnown bool `json:"budget_known"`
	// E04 owns this versioned ledger; E02 preserves it in the same commit.
	Budget json.RawMessage `json:"budget"`
	// Usage is the optional Task-wide AI usage ledger. Its absence in an older
	// Schema 10 state means historical usage is unknown; readers must not create
	// an empty projection as a read side effect.
	Usage *TaskUsageLedger `json:"usage_ledger,omitempty"`
	Delivery *ManagedDelivery `json:"delivery,omitempty"`
	DeliveryHistory []ManagedDelivery `json:"delivery_history,omitempty"`
}

type ExecutionStop struct {
	Kind string `json:"kind,omitempty"`
	Reason string `json:"reason"`
	Reference ActorReference `json:"reference"`
}

type ItemExecution struct {
	VerifierSnapshot *ActorReference `json:"verifier_snapshot,omitempty"`
	VerifierGap string `json:"verifier_gap,omitempty"`
	ValidatedReport *ActorReference `json:"validated_report,omitempty"`
	TestAuthoringRequest string `json:"test_authoring_request,omitempty"`
	WorkItemID WorkItemID `json:"work_item_id"`
	AttemptID AttemptID `json:"attempt_id"`
	Phase ExecutionPhase `json:"phase"`
	CurrentRequest string `json:"current_request,omitempty"`
	Accepted *ActorReference `json:"accepted,omitempty"`
	AcceptanceCandidate *ActorReference `json:"acceptance_candidate,omitempty"`
}

type StageRequest struct {
	ActorRequest
	Route *GenerationRoute `json:"route,omitempty"`
	GenerationSources map[ActorKind]string `json:"generation_sources,omitempty"`
	Phase ExecutionPhase `json:"phase"`
	SessionID string `json:"session_id,omitempty"`
	Turn int `json:"turn,omitempty"`
	Model *AISelection `json:"model,omitempty"`
	AllowedPaths []string `json:"allowed_paths"`
	Input ActorReference `json:"input"`
	InputDigest string `json:"input_digest"`
	Plan ActorReference `json:"plan"`
	Grant ActorReference `json:"grant"`
	Policy ActorReference `json:"policy"`
	LeaseGeneration uint64 `json:"lease_generation"`
}

type StageRecord struct {
	Attribution *ActorReference `json:"attribution,omitempty"`
	Request StageRequest `json:"request"`
	// intent is not evidence that the executor started.
	Dispatch string `json:"dispatch"`
	Executor string `json:"executor,omitempty"`
	Observation *ActorReference `json:"observation,omitempty"`
	Observations []ActorReference `json:"observations,omitempty"`
	Result *ActorReference `json:"result,omitempty"`
	Consumed bool `json:"consumed"`
}

type StageResult struct {
	RequestID string `json:"request_id"`
	TaskID TaskID `json:"task_id"`
	WorkItemID WorkItemID `json:"work_item_id"`
	AttemptID AttemptID `json:"attempt_id"`
	SessionID string `json:"session_id,omitempty"`
	Turn int `json:"turn,omitempty"`
	LeaseGeneration uint64 `json:"lease_generation"`
	InputDigest string `json:"input_digest"`
	Executor string `json:"executor"`
	Terminal bool `json:"terminal"`
	Status string `json:"status"`
	FailureClass string `json:"failure_class,omitempty"`
	Evidence []ActorReference `json:"evidence"`
}

type StageRecovery struct {
	WorkItemID WorkItemID `json:"work_item_id"`
	Phase ExecutionPhase `json:"phase"`
	Requests []string `json:"requests"`
}

// ExecutionServices are deterministic, read-only validators except for Budget
// and MigrateBudget, which mutate only the supplied in-memory ledger. They must
// not invoke models or executors inside a Store transition. Nil fails closed.
// E03 provides authorization/result/acceptance; E04 provides budget accounting.
type ExecutionServices struct {
	Authorize func(RuntimeState, StageRequest) error
	ValidateResult func(RuntimeState, StageRequest, StageResult) error
	ValidateAcceptance func(RuntimeState, AcceptanceCandidate) error
	Budget func(*RuntimeState, StageRequest, string) error
	MigrateBudget func(*RuntimeState) error
	VerifyActivation func(RuntimeState, []ActorReference) error
}

func (s *Store) requireExecutionServices() error {
	x := s.ExecutionServices
	if x == nil || x.Authorize == nil || x.ValidateResult == nil || x.ValidateAcceptance == nil || x.Budget == nil || x.MigrateBudget == nil || x.VerifyActivation == nil {
		return errors.New("E01-E04 execution services are not connected; durable execution remains disabled")
	}
	return nil
}

func validateExecutionProtocol(state RuntimeState) error {
	p := state.Protocol
	if p == nil || p.Version != 1 || state.StateRevision == 0 || state.CommitID == "" || !validProtocolReference(p.Migration) || len(p.Activation) == 0 {
		return errors.New("durable execution has incomplete migration, activation or commit provenance")
	}
	if p.Stop != nil && (strings.TrimSpace(p.Stop.Reason) == "" || !validProtocolReference(p.Stop.Reference)) { return errors.New("durable Stop has no provenance") }
	if p.Grants.Revision < 0 || (p.Grants.Revision == 0 && p.Grants.Digest != "") || (p.Grants.Revision > 0 && len(p.Grants.Digest) != 64) { return errors.New("invalid grant history anchor") }
	seen := map[string]bool{}
	for _, r := range p.Requests {
		if err := validateStageRequest(r.Request); err != nil { return err }
		if r.Request.TaskID != state.Task.ID || seen[r.Request.ID] { return errors.New("duplicate or cross-Task stage request") }
		seen[r.Request.ID] = true
		switch r.Dispatch { case "intent", "unknown", "dispatched", "not-dispatched", "terminal": default: return errors.New("unknown dispatch observation") }
		if r.Consumed && (r.Dispatch != "terminal" || r.Result == nil) { return errors.New("consumed request lacks a terminal result") }
		if r.Attribution != nil && (!r.Consumed || !validProtocolReference(*r.Attribution)) { return errors.New("failure attribution lacks a consumed request or evidence") }
	}
	items := map[WorkItemID]bool{}
	for _, item := range p.Items {
		if items[item.WorkItemID] { return errors.New("duplicate execution item") }
		items[item.WorkItemID] = true
		a, ok := findAttempt(state.Attempts, item.AttemptID)
		if !ok || a.WorkItemID != item.WorkItemID { return errors.New("execution item has no matching Attempt") }
		switch item.Phase { case PhaseCoder, PhaseReport, PhaseCompile, PhaseTester, PhaseTest, PhaseAccept, PhaseAccepted: default: return errors.New("unknown execution phase") }
		if item.CurrentRequest != "" && !seen[item.CurrentRequest] { return errors.New("execution cursor references an unknown request") }
		if item.TestAuthoringRequest != "" {
			authored, err := stageRecord(&state, item.TestAuthoringRequest)
			if err != nil || authored.Request.Phase != PhaseTester || authored.Request.WorkItemID != item.WorkItemID || authored.Request.AttemptID != item.AttemptID || !authored.Consumed { return errors.New("test authoring cursor has no matching terminal Tester") }
		}
		if item.ValidatedReport != nil && !validProtocolReference(*item.ValidatedReport) { return errors.New("validated report reference is incomplete") }
		if item.Phase == PhaseAccepted && (item.Accepted == nil || !validProtocolReference(*item.Accepted)) { return errors.New("accepted execution has no immutable evidence") }
	}
	if state.WriteLease != nil {
		lease := state.WriteLease
		if lease.RequestID == "" || lease.Generation == 0 || lease.Generation > p.LeaseGeneration || !seen[lease.RequestID] { return errors.New("durable writer has no fenced request") }
	}
	for _, account := range p.Recoveries {
		if len(account.Requests) > 2 { return errors.New("infrastructure recovery budget exceeds two additional attempts") }
		keys := map[string]bool{}
		for _, id := range account.Requests { if !seen[id] || keys[id] { return errors.New("recovery account has an unknown or duplicate request") }; keys[id] = true }
	}
	diagnoses := map[string]bool{}
	for _, id := range p.Diagnoses { if !seen[id] || diagnoses[id] { return errors.New("diagnostic account has an unknown or duplicate request") }; diagnoses[id] = true }
	if d := p.Delivery; d != nil {
		if d.Plan.TaskID != state.Task.ID || !validProtocolReference(d.PlanReference) || len(d.Actions) > len(d.Plan.Actions) { return errors.New("invalid managed delivery binding") }
		for i, action := range d.Actions {
			if action.ID != d.Plan.Actions[i].ID || action.Executor == "" { return errors.New("delivery action identity mismatch") }
			switch action.State { case "unknown", "not-dispatched", "failed", "completed": default: return errors.New("unknown delivery action state") }
			if action.State == "completed" && action.Observation == nil { return errors.New("completed delivery action has no evidence") }
		}
	}
	if err := validateTaskAuxiliary(state); err != nil { return err }
	if err := validateTaskUsageLedger(p.Usage); err != nil { return err }
	if state.PendingEvent != nil && (state.PendingEvent.CommitID != state.CommitID || state.PendingEvent.StateRevision != state.StateRevision) { return errors.New("pending event does not match its state commit") }
	return nil
}

func validProtocolReference(ref ActorReference) bool {
	return ref.Kind != "" && ref.Path != "" && len(ref.SHA256) == 64
}

func validateStageRequest(r StageRequest) error {
	if r.SchemaVersion != ActorContractSchemaVersion || r.ID == "" || r.TaskID == "" || r.WorkItemID == "" || r.AttemptID == "" || r.Workspace == "" || r.PreparedAt == "" || !validProtocolReference(r.Input) || len(r.InputDigest) != 64 || !validProtocolReference(r.Plan) || !validProtocolReference(r.Policy) || len(r.AllowedPaths) == 0 {
		return errors.New("stage request lacks frozen identity, inputs, plan, policy or scope")
	}
	expected := map[ExecutionPhase]ActorKind{PhaseCoder: ActorCoder, PhaseCompile: ActorCompiler, PhaseTester: ActorTester, PhaseTest: ActorTester, PhaseReport: ActorAnalysis}
	if expected[r.Phase] == "" || expected[r.Phase] != r.Actor { return errors.New("stage and Actor do not match") }
	if r.Phase == PhaseCoder || r.Phase == PhaseTester || r.Phase == PhaseReport {
		if r.SessionID == "" || r.Turn <= 0 || r.Model == nil || r.Model.Model == "" || r.Model.Provider == "" || r.Model.Digest == "" { return errors.New("generation requires exact Session, turn and model") }
		identity := PreparedAgentRequest{TaskID: r.TaskID, WorkItemID: r.WorkItemID, AttemptID: r.AttemptID, SessionID: r.SessionID, ExpectedSessionTurn: r.Turn}
		if r.ID != ExecutionRequestID(identity) { return errors.New("generation request ID must match the E01 execution identity") }
	}
	if r.Phase == PhaseTest && !validProtocolReference(r.Grant) { return errors.New("test execution requires an exact grant") }
	return nil
}

func executionItem(state *RuntimeState, id WorkItemID) (*ItemExecution, error) {
	if state.SchemaVersion != DurableSchemaVersion || state.Protocol == nil { return nil, errors.New("durable execution is not enabled") }
	for i := range state.Protocol.Items { if state.Protocol.Items[i].WorkItemID == id { return &state.Protocol.Items[i], nil } }
	return nil, fmt.Errorf("no execution cursor for work item %s", id)
}

func stageRecord(state *RuntimeState, id string) (*StageRecord, error) {
	if state.Protocol == nil { return nil, errors.New("durable execution is not enabled") }
	for i := range state.Protocol.Requests { if state.Protocol.Requests[i].Request.ID == id { return &state.Protocol.Requests[i], nil } }
	return nil, fmt.Errorf("unknown stage request %s", id)
}

func stageInFlight(r StageRecord) bool { return r.Dispatch != "not-dispatched" && !r.Consumed }

func requireNoStageInFlight(state RuntimeState) error {
	if state.WriteLease != nil { return errors.New("workspace writer must be reconciled") }
	if state.Protocol != nil {
		for _, r := range state.Protocol.Requests { if stageInFlight(r) { return fmt.Errorf("request %s requires reconciliation", r.Request.ID) } }
	}
	return nil
}

func executionWorkspace(state RuntimeState) string {
	if state.Workspace != nil && state.Workspace.WorktreePath != "" { return state.Workspace.WorktreePath }
	return state.Task.Workspace
}
