# task-agent-handoff Specification

## Purpose

Define automatic Task resolution and handoff-driven creation for the sequential
`aiw task agent next` workflow.

## ADDED Requirements

### Requirement: Resolve an existing or new Task

The system SHALL resolve whether the supplied Task ID exists before mutating
Task, Session, branch, or worktree state.

#### Scenario: Existing Task

- **WHEN** the supplied Task ID resolves to an existing Task
- **THEN** the command SHALL reuse that Task and follow the existing-Task path

#### Scenario: New Task

- **WHEN** the supplied Task ID does not resolve to an existing Task
- **THEN** the command SHALL follow the new-Task creation path

### Requirement: Create lifecycle resources for a new Task

The system SHALL create a Task, Session, branch, worktree, and fresh Thread for
a new Task only after resolving a valid handoff and passing all preflight checks.

#### Scenario: Create from handoff

- **WHEN** a new Task has a valid handoff and no conflicting resources
- **THEN** the system SHALL create and bind the lifecycle resources and start a fresh Thread

#### Scenario: New Task without handoff

- **WHEN** no handoff source can be resolved for a new Task
- **THEN** the command SHALL refuse before creating lifecycle resources

### Requirement: Reuse resources for an existing Task

The system SHALL preserve an existing Task's bound Session, branch, and
worktree and SHALL create only a fresh Thread for the next agent.

#### Scenario: Existing complete Task

- **WHEN** an existing Task has valid Session, branch, and worktree bindings
- **THEN** the command SHALL reuse them and start a fresh Thread

#### Scenario: Existing partial Task

- **WHEN** an existing Task is missing one or more lifecycle resources
- **THEN** the command SHALL preserve the Task and create only the missing resources

### Requirement: Resolve handoff deterministically

The system SHALL resolve handoff sources in the order explicit option, existing
Task artifact, and current Session artifact.

#### Scenario: Explicit source wins

- **WHEN** `--handoff` is supplied
- **THEN** the command SHALL use that source before any implicit source

#### Scenario: No source

- **WHEN** no source exists after applying the precedence order
- **THEN** the command SHALL refuse new-Task creation

### Requirement: Preserve and audit a new Task handoff

The system SHALL copy the selected handoff into the new Task and SHALL preserve
the original source.

#### Scenario: Copy source

- **WHEN** a new Task is created from a handoff
- **THEN** the new Task SHALL contain a copy and record source path, hash, and timestamp

#### Scenario: Pending consumption

- **WHEN** the new Thread has not successfully started and consumed the copy
- **THEN** the new Task SHALL retain `handoff_status = pending`

#### Scenario: Successful consumption

- **WHEN** the new Thread starts successfully and consumes the copied handoff
- **THEN** the new Task SHALL record consumption time, consumer Thread, and hash

### Requirement: Confirm normalized Task identity

The system SHALL require explicit confirmation before using a generated valid
name for an invalid Task ID.

#### Scenario: Invalid identity

- **WHEN** the supplied Task ID violates naming rules
- **THEN** the command SHALL present a deterministic candidate mapping and refuse by default

#### Scenario: Confirmed identity

- **WHEN** the user confirms the candidate mapping
- **THEN** the command SHALL continue using the confirmed identity

### Requirement: Refuse unsafe Session reuse and conflicts

The system SHALL refuse a running Session or conflicting unbound resource by
default and SHALL NOT overwrite existing resources.

#### Scenario: Running Session

- **WHEN** the resolved Session is running
- **THEN** the command SHALL refuse unless explicit takeover is requested

#### Scenario: Resource conflict

- **WHEN** a branch, worktree, Session, or other resource name conflicts with an unrelated resource
- **THEN** the command SHALL stop without overwriting it

### Requirement: Create resources transactionally

The system SHALL clean up only resources created by the current invocation when
new-Task creation or Thread startup fails.

#### Scenario: Creation failure

- **WHEN** a later creation step fails
- **THEN** the command SHALL remove resources created by this invocation, preserve existing resources, and record the failure

#### Scenario: Safe retry

- **WHEN** a prior creation attempt failed and cleanup completed
- **THEN** a retry SHALL be able to resolve the original handoff without stale partial resources

### Requirement: Record parent and child lineage

The system SHALL record parent Task, Session, and Thread identifiers, child Task
and Session identifiers when applicable, handoff path/hash, timestamps, and
transition status.

#### Scenario: New Task lineage

- **WHEN** a new Task is created from an existing handoff
- **THEN** the child Task SHALL record the available source Task, Session, and Thread lineage

#### Scenario: Existing Task transition

- **WHEN** a fresh Thread is started for an existing Task
- **THEN** the transition SHALL record parent and child Thread identifiers

### Requirement: Transition Session and Task states

The system SHALL mark the source Session as handed-off or completed and SHALL
mark the new Task as running after the new Thread starts.

#### Scenario: Successful handoff

- **WHEN** the new Thread starts successfully
- **THEN** the source Session SHALL leave the active state and the child Task SHALL be running

### Requirement: Preserve existing workflows

The new behavior SHALL NOT change ordinary `run`, `continue`, `loop`, or
same-Thread continuation semantics.

#### Scenario: Existing continuation

- **WHEN** a user invokes existing same-Thread continuation
- **THEN** the system SHALL resume the current Thread without creating a new Task or handoff transition
