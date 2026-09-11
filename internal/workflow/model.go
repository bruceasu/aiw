// Package workflow defines the AIW-managed execution model.
//
// It intentionally has no dependency on Task, Session, OpenSpec, or command
// packages. Adapters own those integrations; this package owns the typed
// concepts used to coordinate their execution.
package workflow

const (
	SchemaVersion     = 4
	MinRetryLimit     = 1
	MaxRetryLimit     = 5
	DefaultRetryLimit = 3
)

type TaskID string
type WorkItemID string
type AttemptID string
type GateID string
type EvidenceID string

type PlanningState string

const (
	PlanningDraft         PlanningState = "draft"
	PlanningReady         PlanningState = "ready"
	PlanningNeedsDecision PlanningState = "needs-decision"
)

type ExecutionState string

const (
	ExecutionQueued    ExecutionState = "queued"
	ExecutionLeased    ExecutionState = "leased"
	ExecutionRunning   ExecutionState = "running"
	ExecutionPaused    ExecutionState = "paused"
	ExecutionBlocked   ExecutionState = "blocked"
	ExecutionCompleted ExecutionState = "completed"
	ExecutionCancelled ExecutionState = "cancelled"
)

type ValidationState string

const (
	ValidationNotRequired           ValidationState = "not-required"
	ValidationPending               ValidationState = "pending"
	ValidationAwaitingAuthorization ValidationState = "awaiting-authorization"
	ValidationPassed                ValidationState = "passed"
	ValidationFailed                ValidationState = "failed"
	ValidationWaived                ValidationState = "waived"
)

// FocusedTestAuthorizationState describes whether the focused-test profile is
// usable for a particular normalized Verification Plan digest. The profile is
// disabled unless a human authorization record exists.
type FocusedTestAuthorizationState string

const (
	FocusedTestAuthorizationDisabled FocusedTestAuthorizationState = "disabled"
	FocusedTestAuthorizationAuthorized FocusedTestAuthorizationState = "authorized"
	FocusedTestAuthorizationStale    FocusedTestAuthorizationState = "stale"
)

type DeliveryState string

const (
	DeliveryUnmanaged DeliveryState = "unmanaged"
	DeliveryPending   DeliveryState = "pending"
	DeliveryMerged    DeliveryState = "merged"
	DeliveryDiscarded DeliveryState = "discarded"
)

type WorkspaceState string

const (
	WorkspacePrimary    WorkspaceState = "primary"
	WorkspaceIsolated   WorkspaceState = "isolated"
	WorkspaceUnassigned WorkspaceState = "unassigned"
	WorkspaceUnknown    WorkspaceState = "unknown"
)

type WorkItemState string

const (
	WorkItemPlanned   WorkItemState = "planned"
	WorkItemReady     WorkItemState = "ready"
	WorkItemLeased    WorkItemState = "leased"
	WorkItemRunning   WorkItemState = "running"
	WorkItemBlocked   WorkItemState = "blocked"
	WorkItemCompleted WorkItemState = "completed"
	WorkItemCancelled WorkItemState = "cancelled"
)

type AttemptState string

const (
	AttemptCreated   AttemptState = "created"
	AttemptRunning   AttemptState = "running"
	AttemptPaused    AttemptState = "paused"
	AttemptFailed    AttemptState = "failed"
	AttemptCompleted AttemptState = "completed"
	AttemptCancelled AttemptState = "cancelled"
)

type GateKind string

const (
	GateDependency    GateKind = "dependency"
	GateDecision      GateKind = "decision"
	GateAuthorization GateKind = "authorization"
	GateValidation    GateKind = "validation"
	GateDelivery      GateKind = "delivery"
)

type GateState string

const (
	GateOpen     GateState = "open"
	GateResolved GateState = "resolved"
	GateWaived   GateState = "waived"
)

type EvidenceKind string

const (
	EvidenceStaticReview EvidenceKind = "static-review"
	EvidenceCommand      EvidenceKind = "command"
	EvidenceApproval     EvidenceKind = "approval"
	EvidenceManual       EvidenceKind = "manual"
)

type EvidenceState string

const (
	EvidencePending EvidenceState = "pending"
	EvidencePassed  EvidenceState = "passed"
	EvidenceFailed  EvidenceState = "failed"
	EvidenceWaived  EvidenceState = "waived"
)

type TaskDisplayState string

const (
	TaskDraft                 TaskDisplayState = "DRAFT"
	TaskReady                 TaskDisplayState = "READY"
	TaskInProgress            TaskDisplayState = "IN_PROGRESS"
	TaskBlocked               TaskDisplayState = "BLOCKED"
	TaskAwaitingVerification  TaskDisplayState = "AWAITING_VERIFICATION"
	TaskAwaitingAuthorization TaskDisplayState = "AWAITING_AUTHORIZATION"
	TaskDone                  TaskDisplayState = "DONE"
	TaskCancelled             TaskDisplayState = "CANCELLED"
	TaskNeedsDecision         TaskDisplayState = "NEEDS_DECISION"
)

// RuntimeState is the schema-versioned local projection for one managed Task.
// Durable Task identity and OpenSpec content are supplied by adapters.
type RuntimeState struct {
	SchemaVersion     int             `json:"schema_version"`
	Task              TaskReference   `json:"task"`
	Planning          PlanningState   `json:"planning"`
	Delivery          DeliveryState   `json:"delivery"`
	WorkItems         []WorkItem      `json:"work_items"`
	Attempts          []Attempt       `json:"attempts"`
	Gates             []Gate          `json:"gates"`
	Evidence          []Evidence      `json:"evidence"`
	WriteLease        *WriteLease     `json:"write_lease,omitempty"`
	LastEventSequence uint64          `json:"last_event_sequence,omitempty"`
	PendingEvent      *Event          `json:"pending_event,omitempty"`
	Automation        AutomationState `json:"automation,omitempty"`
	FocusedTestAuthorization *FocusedTestAuthorization `json:"focused_test_authorization,omitempty"`
	FocusedTestPlanDigest    string                    `json:"focused_test_plan_digest,omitempty"`
	Cancellation      *Cancellation   `json:"cancellation,omitempty"`
	Summary           TaskSummary     `json:"summary"`
}

// FocusedTestAuthorizationGateID is reserved for the active task-local
// Verification Plan. It is resolved only by an explicit human authorization.
const FocusedTestAuthorizationGateID GateID = "focused-test-authorization"

// FocusedTestNetworkEnforcementGateID records that the selected runtime cannot
// technically enforce a Plan's required network boundary. Resolving this Gate
// alone never grants execution: the adapter must check the capability again
// before every process creation.
const FocusedTestNetworkEnforcementGateID GateID = "focused-test-network-enforcement"

// FocusedTestAuthorization is the explicit human approval required to enable
// the otherwise disabled focused-test profile for one exact Verification Plan.
// Plan content remains task-local; Workflow Core stores only its digest and
// authorization metadata.
type FocusedTestAuthorization struct {
	Profile    string `json:"profile"`
	PlanDigest string `json:"plan_digest"`
	Approver   string `json:"approver"`
	ApprovedAt string `json:"approved_at"`
}

// Cancellation records the operator decision that terminally stopped managed
// execution. Delivery names the requested terminal disposition; delivery
// preflight and external Git effects remain adapter responsibilities.
type Cancellation struct {
	Reason     string        `json:"reason"`
	Delivery   DeliveryState `json:"delivery"`
	RecordedAt string        `json:"recorded_at"`
}

// AutomationState records only local orchestration facts. Its zero value is a
// compatible unsynchronized state for runtimes created before automation.
type AutomationState struct {
	PlanFingerprint   string                `json:"plan_fingerprint,omitempty"`
	ChecklistRepairedAt string               `json:"checklist_repaired_at,omitempty"`
	Cursor            AutomationCursor      `json:"cursor,omitempty"`
	PreparedRequest   *PreparedAgentRequest `json:"prepared_request,omitempty"`
	ProjectionRepairs []ProjectionRepair    `json:"projection_repairs,omitempty"`
	Supervisor        SupervisorState       `json:"supervisor,omitempty"`
}

// SupervisorState is the durable control plane for an explicitly started
// foreground workflow supervisor. It never grants execution authority itself.
type SupervisorState struct {
	LeaseID       string `json:"lease_id,omitempty"`
	LeaseExpiresAt string `json:"lease_expires_at,omitempty"`
	StartedAt     string `json:"started_at,omitempty"`
	StoppedAt     string `json:"stopped_at,omitempty"`
	ObservedEvent uint64 `json:"observed_event,omitempty"`
	RetryAfter    string `json:"retry_after,omitempty"`
	Result        string `json:"result,omitempty"`
	Detail        string `json:"detail,omitempty"`
}

// AutomationCursor describes the last bounded orchestration result. It is an
// audit cursor, not a background-work scheduler.
type AutomationCursor struct {
	Result     string `json:"result,omitempty"`
	RecordedAt string `json:"recorded_at,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

// PreparedAgentRequest is an Attempt-bound request for an existing managed
// agent adapter. Creating it never starts a model turn.
type PreparedAgentRequest struct {
	TaskID     TaskID     `json:"task_id"`
	WorkItemID WorkItemID `json:"work_item_id"`
	AttemptID  AttemptID  `json:"attempt_id"`
	SessionID  string     `json:"session_id,omitempty"`
	Workspace  string     `json:"workspace"`
	Handoff    string     `json:"handoff,omitempty"`
	PreparedAt string     `json:"prepared_at"`
}

// ProjectionRepair identifies one committed transition whose durable Task or
// OpenSpec projection needs an explicit retry. Event sequence plus target is
// the deduplication identity; a repair never replays the original transition.
type ProjectionRepair struct {
	EventSequence uint64 `json:"event_sequence"`
	Target        string `json:"target"`
	ErrorClass    string `json:"error_class"`
	Recommended   string `json:"recommended"`
	ResolvedAt    string `json:"resolved_at,omitempty"`
}

type TaskReference struct {
	ID        TaskID         `json:"id"`
	Workspace string         `json:"workspace"`
	Kind      WorkspaceState `json:"workspace_kind"`
}

type WorkItem struct {
	ID                  WorkItemID         `json:"id"`
	Checklist           ChecklistReference `json:"checklist"`
	Title               string             `json:"title"`
	Dependencies        []WorkItemID       `json:"dependencies,omitempty"`
	State               WorkItemState      `json:"state"`
	RetryPolicy         RetryPolicy        `json:"retry_policy"`
	NoProgressCount     int                `json:"no_progress_count"`
	LastOutputReference string             `json:"last_output_reference,omitempty"`
}

// RetryPolicy bounds automatic Attempts for one Work Item. A zero value is
// normalized to DefaultRetryLimit while loading or creating runtime state so
// pre-policy state remains compatible.
type RetryPolicy struct {
	MaxAttempts int `json:"max_attempts"`
}

// ChecklistReference preserves the human-readable checkbox identifier from
// tasks.md while WorkItemID remains stable if headings are reordered or renamed.
type ChecklistReference struct {
	Item string `json:"item"`
}

type Attempt struct {
	ID         AttemptID    `json:"id"`
	WorkItemID WorkItemID   `json:"work_item_id"`
	SessionID  string       `json:"session_id,omitempty"`
	Workspace  string       `json:"workspace"`
	State      AttemptState `json:"state"`
	StartedAt  string       `json:"started_at,omitempty"`
	EndedAt    string       `json:"ended_at,omitempty"`
	Handoff    Handoff      `json:"handoff,omitempty"`
}

type Handoff struct {
	SourceSessionID string `json:"source_session_id,omitempty"`
	SourceThreadID  string `json:"source_thread_id,omitempty"`
	TargetSessionID string `json:"target_session_id,omitempty"`
	TargetThreadID  string `json:"target_thread_id,omitempty"`
	ArtifactPath    string `json:"artifact_path,omitempty"`
	ArtifactHash    string `json:"artifact_hash,omitempty"`
	ConsumedAt      string `json:"consumed_at,omitempty"`
}

type Gate struct {
	ID         GateID     `json:"id"`
	WorkItemID WorkItemID `json:"work_item_id,omitempty"`
	Kind       GateKind   `json:"kind"`
	State      GateState  `json:"state"`
	Reason     string     `json:"reason"`
}

type Evidence struct {
	ID         EvidenceID    `json:"id"`
	WorkItemID WorkItemID    `json:"work_item_id,omitempty"`
	Kind       EvidenceKind  `json:"kind"`
	State      EvidenceState `json:"state"`
	Reference  string        `json:"reference,omitempty"`
	RecordedAt string        `json:"recorded_at,omitempty"`
}

// WriteLease serializes state-changing work in one Task workspace. A lease is
// always tied to an Attempt so a recovered runtime state can explain its owner.
type WriteLease struct {
	AttemptID  AttemptID `json:"attempt_id"`
	Workspace  string    `json:"workspace"`
	AcquiredAt string    `json:"acquired_at"`
}

// TaskSummary is a read model derived from RuntimeState. Its Status field is
// for display only; transition validation is based on the individual
// dimensions and records above.
type TaskSummary struct {
	Status     TaskDisplayState `json:"status"`
	Planning   PlanningState    `json:"planning"`
	Execution  ExecutionState   `json:"execution"`
	Validation ValidationState  `json:"validation"`
	Delivery   DeliveryState    `json:"delivery"`
	Workspace  WorkspaceState   `json:"workspace"`
	BlockedBy  []GateID         `json:"blocked_by,omitempty"`
	Active     AttemptID        `json:"active_attempt_id,omitempty"`
}
