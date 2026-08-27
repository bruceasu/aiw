## MODIFIED Requirements

### Requirement: Respect AIW execution workspace
An implementation Skill SHALL verify the workspace kind, project-root-relative
workspace path, and branch declared for the resolved AIW Task before modifying
implementation files. It SHALL NOT require an isolated worktree for ordinary
sequential work.

#### Scenario: Execute in the primary workspace
- **WHEN** a primary Task's project root and branch match its metadata
- **THEN** the Skill may implement the selected Task item in the current primary
  checkout

#### Scenario: Execute in an isolated workspace
- **WHEN** an isolated Task's registered linked worktree and branch match its
  metadata
- **THEN** the Skill may implement the selected Task item in that worktree

#### Scenario: Workspace does not match Task metadata
- **WHEN** the current workspace, workspace kind, or branch conflicts with the
  resolved Task metadata
- **THEN** the Skill stops before modifying implementation files and reports
  the expected Task workspace

#### Scenario: Workspace is unassigned or unknown
- **WHEN** the resolved Task has no assigned workspace or its legacy binding
  cannot be verified
- **THEN** the Skill remains read-only and requests an explicit bind or repair

### Requirement: Keep ordinary slices inside one change
A ticketing Skill SHALL represent work that shares one goal and Task lifecycle
as numbered checklist items in the change `tasks.md`, without requiring a
dedicated branch or worktree.

#### Scenario: Split one change into tracer bullets
- **WHEN** approved tracer-bullet slices can be implemented sequentially within
  the same AIW Task
- **THEN** the Skill writes them as ordered, independently verifiable checklist
  items in `tasks.md`

#### Scenario: Slice needs an independent lifecycle
- **WHEN** a slice requires parallel writes, independent isolation, status,
  archive lifecycle, or delivery boundary
- **THEN** the Skill proposes a separate OpenSpec change instead of creating a
  second worktree for the same Task

## ADDED Requirements

### Requirement: Require explicit isolation authority
Engineering Skills MUST NOT create a feature branch or linked worktree merely
because work is managed by AIW.

#### Scenario: Ordinary sequential work
- **WHEN** a resolved Task can safely proceed in its primary workspace
- **THEN** the Skill continues there without invoking `aiw wt`

#### Scenario: Isolation is beneficial
- **WHEN** work is parallel, conflicting, long-running, disposable, or the user
  explicitly requests isolation
- **THEN** the Skill explains the isolation reason and invokes `aiw wt` only
  when the request authorizes the Git mutation

### Requirement: Separate Task completion from Git delivery
Engineering Skills SHALL update Task completion artifacts without automatically
committing, merging, pushing, deleting branches, or cleaning worktrees.

#### Scenario: Complete primary workspace implementation
- **WHEN** all selected Task items and Verification records are complete
- **THEN** the Skill updates Task completion state and reports Git delivery as
  unmanaged without performing Git writes

#### Scenario: Complete isolated implementation
- **WHEN** isolated implementation is complete but Git delivery was not
  explicitly requested
- **THEN** the Skill leaves delivery pending and preserves the branch and
  worktree
