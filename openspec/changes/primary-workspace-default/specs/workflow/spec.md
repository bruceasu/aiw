## ADDED Requirements

### Requirement: Bind new Tasks to the primary workspace
The system SHALL bind a newly created Task to the AIW project's primary Git
checkout and its current branch without creating or predicting a feature branch
or linked worktree.

#### Scenario: Create an ordinary Task
- **WHEN** a user creates a Task from the primary checkout
- **THEN** its metadata records the current branch, `worktree = "."`,
  `workspace_kind = "primary"`, and `delivery = "unmanaged"`

#### Scenario: Create from a linked worktree
- **WHEN** a user attempts to create an ordinary Task from a linked worktree
- **THEN** the system refuses and reports the primary checkout path

#### Scenario: Create with unrelated dirty state
- **WHEN** the primary checkout contains uncommitted changes and the user has
  not supplied `--allow-dirty`
- **THEN** the system refuses Task creation without stashing, committing, or
  moving files

### Requirement: Normalize lifecycle metadata across backends
The system SHALL apply the same AIW-owned workspace and delivery defaults after
native or delegated Task artifact creation without overwriting OpenSpec-owned
content.

#### Scenario: Delegated Task creation
- **WHEN** an OpenSpec backend successfully creates a new change
- **THEN** AIW fills the Task lifecycle fields using the primary workspace
  defaults and preserves proposal, design, specs, and checklist content

### Requirement: Manage explicit workspace transitions
The system SHALL use `aiw wt add` as the single low-level operation that binds a
Task to an isolated linked worktree.

#### Scenario: Create an isolated workspace
- **WHEN** a primary or unassigned Task has committed artifacts on its recorded
  parent branch and the requested branch and path are available
- **THEN** `aiw wt add` creates the linked worktree and records isolated,
  pending-delivery metadata

#### Scenario: Isolation prerequisites are missing
- **WHEN** the Task artifacts are absent from the parent branch, locally
  modified, or conflict with an existing branch or path
- **THEN** `aiw wt add` fails without fetching, committing, stashing,
  overwriting, or partially rebinding the Task

#### Scenario: Bind an unassigned Task to primary
- **WHEN** a user explicitly binds an unassigned Task from the primary checkout
  and its current branch matches the requested binding
- **THEN** the system records a primary workspace binding without checking out
  another branch

### Requirement: Protect workspace cleanup and discard
The system MUST verify explicit workspace state and Git worktree registration
before removing a worktree or deleting a branch.

#### Scenario: Remove an isolated worktree
- **WHEN** a user removes a verified isolated worktree without discarding it
- **THEN** the branch remains, and the Task becomes unassigned with pending
  delivery

#### Scenario: Discard an experiment
- **WHEN** a user explicitly confirms discard of an isolated Task
- **THEN** the system removes its verified worktree and unmerged branch and
  records `status = "CANCELLED"` and `delivery = "discarded"`

#### Scenario: Unsafe cleanup target
- **WHEN** workspace state is primary, unassigned, unknown, or inconsistent
  with Git worktree registration
- **THEN** cleanup fails without deleting a workspace, branch, or untracked file

### Requirement: Separate completion, delivery, and archive
The system SHALL track Task completion independently from optional Git delivery
and SHALL NOT infer commit, merge, push, or publication from DONE status.

#### Scenario: Archive a primary Task
- **WHEN** a primary Task is DONE with unmanaged delivery
- **THEN** it may archive without Git delivery and reports any uncommitted-work
  warning without modifying Git state

#### Scenario: Archive an isolated Task
- **WHEN** an isolated Task is DONE
- **THEN** archive proceeds only after Git ancestry verifies merged delivery and
  the linked worktree and temporary branch have been cleaned up

#### Scenario: Archive a cancelled Task
- **WHEN** a Task is CANCELLED with a recorded reason
- **THEN** it may archive without pretending its checklist or delivery completed

### Requirement: Infer legacy workspace state safely
The system SHALL read Task metadata without `workspace_kind` by deriving a
compatibility state and MUST block destructive operations when the state cannot
be verified.

#### Scenario: Read known legacy bindings
- **WHEN** a legacy worktree value is empty, resolves to the primary checkout,
  or resolves to a Git-registered linked worktree
- **THEN** the system derives unassigned, primary, or isolated respectively

#### Scenario: Read an unknown legacy binding
- **WHEN** a non-empty legacy path cannot be verified as primary or as a
  registered linked worktree
- **THEN** the Task remains readable with unknown workspace state and mutation
  requiring a valid workspace is refused

#### Scenario: Persist inferred state
- **WHEN** a later authorized metadata mutation writes a legacy Task
- **THEN** the system may persist the verified inferred state without performing
  a bulk repository migration
