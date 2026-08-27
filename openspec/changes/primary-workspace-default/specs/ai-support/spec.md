## MODIFIED Requirements

### Requirement: Start the next agent Thread for a Task
The system SHALL provide `aiw task agent next TASK_ID` for an existing Task
with a valid workspace and aiw-flow Session binding, and SHALL reuse that
workspace unless isolation is explicitly requested.

#### Scenario: Start a child Thread in the bound workspace
- **WHEN** the Task workspace and Session are valid and no execution lease is
  held
- **THEN** the system creates or refreshes the persistent handoff, starts one
  fresh Codex Thread in the bound workspace, and records the child Thread ID

#### Scenario: Explicit isolated handoff
- **WHEN** the user requests `--isolated` and worktree creation prerequisites
  are satisfied
- **THEN** the command delegates isolation to the shared `aiw wt add` operation
  before starting the fresh Thread

#### Scenario: Isolation fails
- **WHEN** explicit isolation cannot complete safely
- **THEN** the command preserves the Task, Session, and handoff state and exits
  without starting a Thread in the primary workspace

#### Scenario: Reject an unresolved Task
- **WHEN** the Task does not exist, has an unassigned or unknown workspace, or
  has no valid Session binding
- **THEN** the command exits before invoking Codex and reports the missing
  binding

### Requirement: Prevent concurrent writers
The system SHALL refuse a second `task agent next` transition while the same
Task workspace has an active execution lease.

#### Scenario: Lease conflict
- **WHEN** another agent currently holds the Task execution lease
- **THEN** the command exits with a conflict and does not create a new Thread
