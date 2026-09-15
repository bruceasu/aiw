// Package workflow defines the AIW-managed execution model.
//
// It intentionally has no dependency on Task, Session, OpenSpec, or command
// packages. Adapters own those integrations; this package owns the typed
// concepts used to coordinate their execution.
package workflow

import "encoding/json"

const (
	SchemaVersion     = 9
	MinRetryLimit     = 1
	MaxRetryLimit     = 5
	DefaultRetryLimit = 3
)

type TaskID string
type WorkItemID string
type AttemptID string
type GateID string
type EvidenceID string
type NotificationID string

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
	FocusedTestAuthorizationConsumed FocusedTestAuthorizationState = "consumed"
)

type DeliveryState string

const (
	DeliveryUnmanaged DeliveryState = "unmanaged"
	DeliveryPending   DeliveryState = "pending"
	DeliveryMerged    DeliveryState = "merged"
	DeliveryDiscarded DeliveryState = "discarded"
)

// NotificationState records the durable state of a notification outbox item.
// A Plugin receives the stable notification ID so a delivery backend can
// provide idempotency across a recovered dispatch.
type NotificationState string

const (
	NotificationPending     NotificationState = "pending"
	NotificationDispatching NotificationState = "dispatching"
	NotificationFailed      NotificationState = "failed"
	NotificationDelivered   NotificationState = "delivered"
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
	GateWorkspaceAccess GateKind = "workspace-access"
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
	ActorHandoffs     []ActorHandoff  `json:"actor_handoffs,omitempty"`
	WriteLease        *WriteLease     `json:"write_lease,omitempty"`
	LastEventSequence uint64          `json:"last_event_sequence,omitempty"`
	PendingEvent      *Event          `json:"pending_event,omitempty"`
	Automation        AutomationState `json:"automation,omitempty"`
	Workspace         *WorkspaceBinding `json:"workspace_binding,omitempty"`
	FocusedTestAuthorization *FocusedTestAuthorization `json:"focused_test_authorization,omitempty"`
	FocusedTestPlanDigest    string                    `json:"focused_test_plan_digest,omitempty"`
	Cancellation      *Cancellation   `json:"cancellation,omitempty"`
	Policy            *PolicySnapshot `json:"policy_snapshot,omitempty"`
	OperationalRetries OperationalRetryAccounting `json:"operational_retries,omitempty"`
	Notifications      []Notification             `json:"notifications,omitempty"`
	Summary           TaskSummary     `json:"summary"`
}

// Notification is a persisted outbox record. Payload is frozen before a
// Plugin process starts, so workflow transition completion does not depend on
// a transient Plugin invocation.
type Notification struct {
	ID               NotificationID    `json:"id"`
	Topic            string            `json:"topic"`
	Payload          json.RawMessage   `json:"payload"`
	State            NotificationState `json:"state"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
	DispatchAttempts int               `json:"dispatch_attempts"`
	LastError        string            `json:"last_error,omitempty"`
	Receipt          string            `json:"receipt,omitempty"`
}

// WorkspaceBinding is the durable boundary between a Task worktree and its
// parent checkout. It is recorded before managed execution so recovery can
// reject a repointed worktree without relying on mutable Task metadata.
type WorkspaceBinding struct {
	ParentPath   string `json:"parent_path"`
	ParentBranch string `json:"parent_branch"`
	ParentCommit string `json:"parent_commit"`
	WorktreePath string `json:"worktree_path"`
	TaskBranch   string `json:"task_branch"`
	ParentDrift  []ParentDrift `json:"parent_drift,omitempty"`
}

// ParentDrift is an audit observation only. External parent changes do not
// stop work; only ConfirmParentWriteFenceViolation opens the blocking Gate.
type ParentDrift struct {
	ObservedAt string   `json:"observed_at"`
	Commit     string   `json:"commit"`
	Paths      []string `json:"paths,omitempty"`
}

// FocusedTestAuthorizationGateID is reserved for the active task-local
// Verification Plan. It is resolved only by an explicit human authorization.
const FocusedTestAuthorizationGateID GateID = "focused-test-authorization"

// FocusedTestNetworkEnforcementGateID records that the selected runtime cannot
// technically enforce a Plan's required network boundary. Resolving this Gate
// requires the adapter to check the capability again; waiving it explicitly
// authorizes one frozen-command execution without technical isolation.
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
	ConsumedAt string `json:"consumed_at,omitempty"`
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
	// ExpectedSessionTurn binds this request to the next Session turn observed
	// at preparation time. Zero is accepted for pre-schema requests.
	ExpectedSessionTurn int `json:"expected_session_turn,omitempty"`
	// DispatchedAt is set immediately before the managed adapter is asked to
	// start the bound Session turn. A prepared request is deliberately not a
	// dispatched request: it may have been created by a dry workflow pass.
	DispatchedAt string `json:"dispatched_at,omitempty"`
	// CompilerResult is committed with the compile counter so recovery cannot
	// count the same completed compiler invocation twice.
	CompilerResult *CompilerResult `json:"compiler_result,omitempty"`
	Workspace  string     `json:"workspace"`
	Handoff    string     `json:"handoff,omitempty"`
	// SkillManifest freezes the selected Skills for this Actor request.  A
	// resumed request must use this snapshot rather than rediscovering Skills
	// from mutable filesystem roots.
	SkillManifest *SkillManifest `json:"skill_manifest,omitempty"`
	// AISelection freezes the supervised Actor's resolved routing choice. It
	// intentionally contains no credentials, so retries and recovery do not
	// depend on mutable configuration while the runtime state remains safe to
	// inspect.
	AISelection *AISelection `json:"ai_selection,omitempty"`
	Compile *SupervisedCompileState `json:"compile,omitempty"`
	PreparedAt string     `json:"prepared_at"`
}

// SupervisedCompileState is frozen with the prepared request. A nil Plan is
// a legacy request, never permission to discover another compiler at runtime.
type SupervisedCompileState struct {
	Plan *CompilePlan `json:"plan,omitempty"`
	PlanReference ActorReference `json:"plan_reference"`
	Request *CompilerRequest `json:"request,omitempty"`
	Result *CompilerResult `json:"result,omitempty"`
	Failures int `json:"failures"`
	RepairPending bool `json:"repair_pending,omitempty"`
}

// AISelection is the secret-free AI configuration provenance for one
// supervised request. Digest is computed by the command adapter from the
// resolved non-secret configuration fields.
type AISelection struct {
	Profile  string `json:"profile"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Digest   string `json:"digest"`
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
	CompileFailureCount int                `json:"compile_failure_count,omitempty"`
	LastOutputReference string             `json:"last_output_reference,omitempty"`
}

// RetryPolicy bounds automatic Attempts for one Work Item. A zero value is
// normalized to DefaultRetryLimit while loading or creating runtime state so
// pre-policy state remains compatible.
type RetryPolicy struct {
	MaxAttempts int `json:"max_attempts"`
}

// OperationalRetryKind identifies retries that are deliberately independent
// from a Work Item's implementation Attempts.  A failed notification, for
// example, must not make an otherwise repairable Work Item unavailable.
type OperationalRetryKind string

const (
	RetryRecovery     OperationalRetryKind = "recovery"
	RetryNotification OperationalRetryKind = "notification"
	RetryDelivery     OperationalRetryKind = "delivery"
)

// OperationalRetryAccounting persists the bounded retry budget for each
// task-level operation category.  Notification outbox records can retain
// their own delivery details while consuming only the notification budget.
type OperationalRetryAccounting struct {
	Recovery     RetryCounter `json:"recovery"`
	Notification RetryCounter `json:"notification"`
	Delivery     RetryCounter `json:"delivery"`
}

// RetryCounter is a compatibility-safe counter: a zero MaxAttempts is
// normalized to DefaultRetryLimit when an older state snapshot is read.
type RetryCounter struct {
	MaxAttempts int `json:"max_attempts"`
	Used        int `json:"used"`
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
	Outcome    *SupervisedOutcome `json:"outcome,omitempty"`
}

type SupervisedOutcomeKind string

const (
	SupervisedOutcomeCompleted  SupervisedOutcomeKind = "completed"
	SupervisedOutcomeBlocked    SupervisedOutcomeKind = "blocked"
	SupervisedOutcomeNoProgress SupervisedOutcomeKind = "no-progress"
)

type BlockedOutcomeCategory string

const (
	BlockedOutcomeWorkspaceAccess BlockedOutcomeCategory = "workspace-access"
	BlockedOutcomeAuthorization   BlockedOutcomeCategory = "authorization"
	BlockedOutcomeDependency      BlockedOutcomeCategory = "dependency"
	BlockedOutcomeValidation      BlockedOutcomeCategory = "validation"
	BlockedOutcomeUnknown         BlockedOutcomeCategory = "unknown"
)

// SupervisedOutcome is the structured result of one supervised Agent stage.
// EvidenceReference points to immutable Session output; it is intentionally
// separate from validation Evidence because an Agent statement is not itself
// proof that the authored checklist has been completed.
type SupervisedOutcome struct {
	Kind              SupervisedOutcomeKind `json:"kind"`
	BlockedCategory   BlockedOutcomeCategory `json:"blocked_category,omitempty"`
	Detail            string                `json:"detail,omitempty"`
	EvidenceReference string                `json:"evidence_reference"`
	RecordedAt        string                `json:"recorded_at"`
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
