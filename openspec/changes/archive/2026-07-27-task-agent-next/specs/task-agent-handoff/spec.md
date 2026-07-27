# task-agent-handoff Specification

## Purpose

Define a sequential Task handoff that starts a fresh Codex Thread while
preserving the Task's worktree and aiw-flow Session.

## ADDED Requirements

### Requirement: Start the next agent Thread for a Task
The system SHALL provide `aiw task agent next TASK_ID` for an existing Task
with a valid worktree and aiw-flow Session binding.

#### Scenario: Start a child Thread
- **WHEN** the Task, worktree, and Session are valid and no execution lease is
  held
- **THEN** the system creates or refreshes the persistent handoff, starts one
  fresh Codex Thread in the Task worktree, and records the child Thread ID

#### Scenario: Reject an unresolved Task
- **WHEN** the Task does not exist, has no valid worktree, or has no valid
  Session binding
- **THEN** the command exits before invoking Codex and reports the missing
  binding

### Requirement: Preserve context in the child Thread
The child Thread's initial prompt SHALL include the Task identifier and goal,
Session identifier and phase, Session Memory, and the Session-relative handoff
artifact reference.

#### Scenario: Continue from a handoff
- **WHEN** a child Thread is started successfully
- **THEN** its initial prompt tells the agent to read the handoff and referenced
  artifacts before taking action

### Requirement: Record handoff lineage
The system SHALL persist parent Thread ID, child Thread ID, Task ID, Session
ID, handoff path and content hash, and transition timestamps.

#### Scenario: Inspect lineage after transition
- **WHEN** the child Thread has started
- **THEN** status and diagnostic output show the parent-to-child transition and
  the exact handoff consumed

### Requirement: Prevent concurrent writers
The system SHALL refuse a second `task agent next` transition while the same
Task worktree has an active execution lease.

#### Scenario: Lease conflict
- **WHEN** another agent currently holds the Task execution lease
- **THEN** the command exits with a conflict and does not create a new Thread

### Requirement: Keep the transition atomic on failure
The system SHALL create the handoff and lineage intent before launching Codex
and SHALL mark the transition failed if process startup fails.

#### Scenario: Codex startup failure
- **WHEN** handoff creation succeeds but the Codex process cannot start
- **THEN** no successful child Thread is recorded, the failure is persisted,
  and the execution lease is released

### Requirement: Preserve existing workflows
The new command SHALL NOT change the behavior of one-shot runs, interactive
loops, or same-Thread continuation.

#### Scenario: Existing continuation
- **WHEN** a user runs the existing `aiw-flow continue SESSION_ID`
- **THEN** it resumes the current Thread without creating a task-agent
  handoff transition
