package workflow

import (
	"fmt"
	"strings"
	"time"
)

// ActorContractSchemaVersion versions the persisted cross-Actor handoff
// envelope independently from the Task runtime schema.
const ActorContractSchemaVersion = 1

type ActorKind string

const (
	ActorAnalysis  ActorKind = "analysis"
	ActorCoder     ActorKind = "coder"
	ActorCompiler  ActorKind = "compiler"
	ActorTester    ActorKind = "tester"
	ActorVerifier  ActorKind = "verifier"
)

type ActorResultStatus string

const (
	ActorResultAccepted ActorResultStatus = "accepted"
	ActorResultBlocked  ActorResultStatus = "blocked"
	ActorResultFailed   ActorResultStatus = "failed"
)

// ActorReference identifies a persisted input or output without embedding
// mutable files or free-form commands in runtime state.
type ActorReference struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
}

// ActorRequest is the common, durable envelope for every specialized Actor.
// Role-specific requests below embed it so adapters can persist and validate a
// stable identity before invoking an Actor.
type ActorRequest struct {
	SchemaVersion int              `json:"schema_version"`
	ID            string           `json:"id"`
	Actor         ActorKind        `json:"actor"`
	TaskID        TaskID           `json:"task_id"`
	WorkItemID    WorkItemID       `json:"work_item_id"`
	AttemptID     AttemptID        `json:"attempt_id"`
	Workspace     string           `json:"workspace"`
	Inputs        []ActorReference `json:"inputs,omitempty"`
	PreparedAt    string           `json:"prepared_at"`
}

// ActorResult is the common durable response envelope. Outputs must be
// persisted artifacts; callers cannot use a result to imply an unrecorded
// state transition.
type ActorResult struct {
	SchemaVersion int               `json:"schema_version"`
	RequestID     string            `json:"request_id"`
	Actor         ActorKind         `json:"actor"`
	TaskID        TaskID            `json:"task_id"`
	WorkItemID    WorkItemID        `json:"work_item_id"`
	AttemptID     AttemptID         `json:"attempt_id"`
	Status        ActorResultStatus `json:"status"`
	Outputs       []ActorReference  `json:"outputs,omitempty"`
	Summary       string            `json:"summary"`
	CompletedAt   string            `json:"completed_at"`
}

// Specialized contracts keep each role's permitted context explicit. They
// deliberately contain references only; later work items own contract content,
// changed-path policy, and executable plan selection.
type AnalysisRequest struct { ActorRequest; RequirementSources []ActorReference `json:"requirement_sources"` }
type AnalysisResult struct { ActorResult; ImplementationContract ActorReference `json:"implementation_contract"` }
type CoderRequest struct { ActorRequest; ImplementationContract ActorReference `json:"implementation_contract"`; PriorEvidence []ActorReference `json:"prior_evidence,omitempty"` }
type CoderResult struct { ActorResult; ImplementationReport ActorReference `json:"implementation_report"` }
type CompilerRequest struct {
	ActorRequest
	ImplementationReport ActorReference `json:"implementation_report"`
	Plan *CompilePlan `json:"compile_plan,omitempty"`
	PlanReference ActorReference `json:"plan_reference"`
	ChangedPaths []string `json:"changed_paths"`
	CompileRoot string `json:"compile_root,omitempty"`
}
type CompilerResult struct { ActorResult; Diagnostics ActorReference `json:"diagnostics,omitempty"` }
type TesterRequest struct { ActorRequest; InterfaceInventory ActorReference `json:"interface_inventory"` }
type TesterResult struct { ActorResult; TestCaseInventory ActorReference `json:"test_case_inventory"` }
type VerifierRequest struct { ActorRequest; EvidenceReferences []ActorReference `json:"evidence_references"` }
type VerifierResult struct { ActorResult; Report ActorReference `json:"report"` }

// ActorHandoff is the persisted, validated link between an Actor request and
// its response. It is append-only by ID and survives process interruption.
type ActorHandoff struct {
	ID        string      `json:"id"`
	Request   ActorRequest `json:"request"`
	Result    ActorResult  `json:"result"`
	RecordedAt string      `json:"recorded_at"`
}

func (r ActorRequest) Validate() error {
	if r.SchemaVersion != ActorContractSchemaVersion || strings.TrimSpace(r.ID) == "" || !validActor(r.Actor) || r.TaskID == "" || r.WorkItemID == "" || r.AttemptID == "" || strings.TrimSpace(r.Workspace) == "" || strings.TrimSpace(r.PreparedAt) == "" {
		return fmt.Errorf("actor request requires schema, id, supported actor, Task, Work Item, Attempt, workspace, and prepared time")
	}
	return validateActorReferences(r.Inputs)
}

func (r ActorResult) Validate() error {
	if r.SchemaVersion != ActorContractSchemaVersion || strings.TrimSpace(r.RequestID) == "" || !validActor(r.Actor) || r.TaskID == "" || r.WorkItemID == "" || r.AttemptID == "" || !validActorResultStatus(r.Status) || strings.TrimSpace(r.Summary) == "" || strings.TrimSpace(r.CompletedAt) == "" {
		return fmt.Errorf("actor result requires schema, request, supported actor, Task, Work Item, Attempt, status, summary, and completion time")
	}
	return validateActorReferences(r.Outputs)
}

// Validate rejects a Coder request unless it is bound to a persisted Analysis
// contract.  Coder is intentionally unable to substitute a prose prompt for
// that operational handoff.
func (r CoderRequest) Validate() error {
	if err := r.ActorRequest.Validate(); err != nil { return err }
	if r.Actor != ActorCoder { return fmt.Errorf("coder request actor must be coder") }
	if err := validateActorReferences([]ActorReference{r.ImplementationContract}); err != nil { return fmt.Errorf("implementation contract: %w", err) }
	return validateActorReferences(r.PriorEvidence)
}

// Validate binds Compiler to the implementation report produced by Coder. A
// compiler adapter never accepts a free-form command as part of this handoff.
func (r CompilerRequest) Validate() error {
	if err := r.ActorRequest.Validate(); err != nil { return err }
	if r.Actor != ActorCompiler { return fmt.Errorf("compiler request actor must be compiler") }
	if err := validateActorReferences([]ActorReference{r.ImplementationReport}); err != nil { return fmt.Errorf("implementation report: %w", err) }
	return nil
}

// Validate makes failed compiler outcomes actionable: they must point Coder
// at persisted structured diagnostics instead of relying on transient output.
func (r CompilerResult) Validate() error {
	if err := r.ActorResult.Validate(); err != nil { return err }
	if r.Actor != ActorCompiler { return fmt.Errorf("compiler result actor must be compiler") }
	if r.Status == ActorResultFailed {
		if err := validateActorReferences([]ActorReference{r.Diagnostics}); err != nil { return fmt.Errorf("failed compiler result diagnostics: %w", err) }
	}
	return nil
}

// Validate keeps Tester bound to the compiler-produced inventory. It cannot
// receive implementation details or an unrestricted test command instead.
func (r TesterRequest) Validate() error {
	if err := r.ActorRequest.Validate(); err != nil { return err }
	if r.Actor != ActorTester { return fmt.Errorf("tester request actor must be tester") }
	if err := validateActorReferences([]ActorReference{r.InterfaceInventory}); err != nil { return fmt.Errorf("interface inventory: %w", err) }
	return nil
}

func (r TesterResult) Validate() error {
	if err := r.ActorResult.Validate(); err != nil { return err }
	if r.Actor != ActorTester { return fmt.Errorf("tester result actor must be tester") }
	if r.Status == ActorResultAccepted {
		if err := validateActorReferences([]ActorReference{r.TestCaseInventory}); err != nil { return fmt.Errorf("test-case inventory: %w", err) }
	}
	return nil
}

// Validate keeps the optional Verifier read-only and bound to evidence
// already produced by the required pipeline stages.
func (r VerifierRequest) Validate() error {
	if err := r.ActorRequest.Validate(); err != nil { return err }
	if r.Actor != ActorVerifier { return fmt.Errorf("verifier request actor must be verifier") }
	if len(r.EvidenceReferences) == 0 { return fmt.Errorf("verifier request requires prior evidence") }
	return validateActorReferences(r.EvidenceReferences)
}

// Validate requires every completed optional verification to retain its
// report. A failed report is still valid: Verifier findings are report-only
// and do not, by themselves, change delivery eligibility.
func (r VerifierResult) Validate() error {
	if err := r.ActorResult.Validate(); err != nil { return err }
	if r.Actor != ActorVerifier { return fmt.Errorf("verifier result actor must be verifier") }
	return validateActorReferences([]ActorReference{r.Report})
}

func (h ActorHandoff) Validate() error {
	if strings.TrimSpace(h.ID) == "" || strings.TrimSpace(h.RecordedAt) == "" {
		return fmt.Errorf("actor handoff id and recorded time are required")
	}
	if err := h.Request.Validate(); err != nil { return fmt.Errorf("handoff request: %w", err) }
	if err := h.Result.Validate(); err != nil { return fmt.Errorf("handoff result: %w", err) }
	if h.Result.RequestID != h.Request.ID || h.Result.Actor != h.Request.Actor || h.Result.TaskID != h.Request.TaskID || h.Result.WorkItemID != h.Request.WorkItemID || h.Result.AttemptID != h.Request.AttemptID {
		return fmt.Errorf("actor handoff result does not match its request")
	}
	return nil
}

// RecordActorHandoff persists one completed handoff through the normal event
// protocol. Repeating the exact ID is rejected rather than silently replacing
// evidence from a prior Actor invocation.
func (s *Store) RecordActorHandoff(id TaskID, handoff ActorHandoff) (RuntimeState, error) {
	if handoff.RecordedAt == "" { handoff.RecordedAt = time.Now().UTC().Format(time.RFC3339) }
	if err := handoff.Validate(); err != nil { return RuntimeState{}, err }
	if handoff.Request.TaskID != id { return RuntimeState{}, fmt.Errorf("actor handoff Task does not match runtime Task") }
	return s.UpdateWithEvent(id, Event{Type: "actor.handoff.recorded", WorkItemID: handoff.Request.WorkItemID, AttemptID: handoff.Request.AttemptID, Detail: handoff.ID}, func(state *RuntimeState) error {
		if _, ok := findAttempt(state.Attempts, handoff.Request.AttemptID); !ok { return fmt.Errorf("actor handoff references unknown Attempt %s", handoff.Request.AttemptID) }
		for _, existing := range state.ActorHandoffs { if existing.ID == handoff.ID { return fmt.Errorf("actor handoff %s already exists", handoff.ID) } }
		state.ActorHandoffs = append(state.ActorHandoffs, handoff)
		return nil
	})
}

func validActor(actor ActorKind) bool { switch actor { case ActorAnalysis, ActorCoder, ActorCompiler, ActorTester, ActorVerifier: return true }; return false }
func validActorResultStatus(status ActorResultStatus) bool { switch status { case ActorResultAccepted, ActorResultBlocked, ActorResultFailed: return true }; return false }
func validateActorReferences(references []ActorReference) error { for index, reference := range references { if strings.TrimSpace(reference.Kind) == "" || strings.TrimSpace(reference.Path) == "" { return fmt.Errorf("reference %d requires kind and path", index) } }; return nil }
