# workflow Specification

## Purpose
Define the core workflow model for AIW, including task lifecycle, OpenSpec-lite structure, worktrees, context, decisions, specs, registry, and backend routing.

## Requirements
### Requirement: Select a workflow backend
The system SHALL support `auto`, `openspec`, and `native` backend modes for task workflow commands.

#### Scenario: Native mode
- **WHEN** a user selects `--backend native`
- **THEN** the command uses the existing AIW implementation without probing or invoking OpenSpec

#### Scenario: Auto mode without OpenSpec
- **WHEN** `--backend auto` is selected and no verified OpenSpec executable is available
- **THEN** the command uses the native implementation and reports that fallback was selected

### Requirement: Verify an OpenSpec executable
The system SHALL verify an OpenSpec executable before delegating work to it.

#### Scenario: Explicit OpenSpec unavailable
- **WHEN** `--backend openspec` is selected and the executable is missing or `--version` fails
- **THEN** the command exits before writing task artifacts and reports how to configure or install OpenSpec

#### Scenario: Configured executable
- **WHEN** `AIW_OPENSPEC_BIN` points to an executable that returns successfully for `--version`
- **THEN** the adapter uses that executable for delegation

### Requirement: Preserve native compatibility
The system SHALL preserve existing native task behavior when OpenSpec is unavailable or when `native` is selected.

#### Scenario: Existing script invocation without OpenSpec
- **WHEN** a script invokes a task workflow without a backend option and OpenSpec is unavailable
- **THEN** the command produces the existing native artifacts and exit status

#### Scenario: Automatic delegation
- **WHEN** a task workflow is invoked without a backend option and a verified OpenSpec executable is available
- **THEN** the command delegates to OpenSpec

### Requirement: Report backend choice
The system SHALL report the selected backend and whether delegation or fallback was used.

#### Scenario: Auto fallback diagnostic
- **WHEN** auto mode selects the native backend
- **THEN** a concise diagnostic identifies native fallback and the reason

### Requirement: Bind new Tasks to the primary workspace
The system SHALL bind newly created native and delegated Tasks to the AIW
project's primary checkout and current branch with `workspace_kind = "primary"`
and `delivery = "unmanaged"`.

#### Scenario: Create an ordinary Task
- **WHEN** a user creates a Task from a clean primary checkout
- **THEN** AIW records `worktree = "."` without creating or predicting a linked worktree

#### Scenario: Dirty or linked checkout
- **WHEN** creation occurs from a linked worktree or from dirty state without
  `--allow-dirty`
- **THEN** creation fails without changing Git state

### Requirement: Protect explicit workspace lifecycle
The system MUST use verified workspace state and Git registration for isolation,
cleanup, discard, repair, and archive decisions.

#### Scenario: Explicit isolation
- **WHEN** committed Task artifacts and an available branch and path permit
  `aiw wt add`
- **THEN** the Task becomes isolated with pending delivery without an automatic fetch

#### Scenario: Unsafe destructive target
- **WHEN** a cleanup target is primary, unassigned, unknown, or not registered
  by Git
- **THEN** the operation fails without deleting files or branches

### Requirement: Separate Task completion from Git delivery
The system SHALL treat DONE as Task completion and SHALL track isolated Git
delivery separately as pending, merged, or discarded.

#### Scenario: Primary archive
- **WHEN** a DONE primary Task is archived
- **THEN** it archives with unmanaged delivery and does not perform Git delivery

#### Scenario: Isolated archive
- **WHEN** a DONE isolated Task is archived
- **THEN** Git ancestry and managed-resource cleanup must be verified first
### Requirement: Sync archived spec context
The system SHALL sync referenced change specs into the stable `openspec/specs/` tree when archiving a change that references specs.

#### Scenario: Change is archived with linked specs
- **WHEN** a change with `specs` references is archived
- **THEN** the matching `openspec/specs/<spec-id>/spec.md` file is updated from the change spec content when that source spec exists

### Requirement: Collect workspace context
The system SHALL collect workspace context only from the resolved workspace stored for the session.

#### Scenario: Session workspace collected
- **WHEN** a session needs context for a workflow step
- **THEN** the system collects context from that session's workspace and stores it as a session-relative artifact

### Requirement: Bound context collection
The collector MUST enforce maximum directory depth, directory entry count, per-file bytes, and total content bytes.

#### Scenario: Workspace exceeds limits
- **WHEN** the workspace has more entries or metadata content than configured limits
- **THEN** the collector stops at the limits and records truncation in the context artifact

### Requirement: Read only allow-listed metadata
The collector SHALL read content only from an explicit allow-list of project metadata and instruction filenames, and SHALL skip hidden, VCS, dependency, cache, and build directories.

#### Scenario: Workspace contains an environment file
- **WHEN** the workspace contains `.env`, credential files, private keys, or other files outside the allow-list
- **THEN** the collector does not read their content

### Requirement: Redact potential credential assignments
The collector MUST replace potential password, secret, token, and API key assignment values before saving or sending collected context.

#### Scenario: Allowed metadata contains a token example
- **WHEN** an allow-listed file contains a credential-like assignment
- **THEN** the saved context contains `[REDACTED]` instead of the assignment value

### Requirement: Use cross-platform safe process execution
The collector MUST invoke Git with argument arrays and MUST NOT use `shell=True` or Unix-only `find`, `head`, or shell functions.

#### Scenario: Git metadata is unavailable
- **WHEN** Git is missing, the workspace is not a repository, or Git returns an error
- **THEN** the context artifact records Git metadata as unavailable and collection continues

<!-- archived spec: git-file-history -->

## ADDED Requirements

### Requirement: File commit history
The system SHALL provide a read-only file history view that follows renames by default and SHALL support default, concise, patch, statistics, graph, and full-evolution output modes.

#### Scenario: Default rename-aware history
- **WHEN** a user requests file history with one file path and no mode flag
- **THEN** the system invokes `git log --follow -- <path>`

#### Scenario: Full file evolution
- **WHEN** a user requests full file history
- **THEN** the system invokes a rename-aware file log with patch and statistics output

#### Scenario: History without rename tracking
- **WHEN** a user includes `--no-follow`
- **THEN** the system omits `--follow` while preserving the selected output mode

#### Scenario: Conflicting output modes
- **WHEN** a user supplies more than one output mode
- **THEN** the system reports a usage error without invoking Git

### Requirement: Line attribution
The system SHALL provide a read-only view that attributes every current file line using Git blame.

#### Scenario: Blame one file
- **WHEN** a user requests blame with one file path
- **THEN** the system invokes `git blame -- <path>`

### Requirement: Historical file content
The system SHALL provide a read-only view of a file at a specified Git revision.

#### Scenario: Show file at revision
- **WHEN** a user supplies a revision and repository-relative file path
- **THEN** the system invokes `git show <revision>:<path>`

### Requirement: Line or function evolution
The system SHALL provide a read-only view that forwards a line-range or function selector to Git's `log -L` history.

#### Scenario: Track line range
- **WHEN** a user supplies `10,30` and a file path
- **THEN** the system invokes `git log -L 10,30:<path>`

#### Scenario: Track function
- **WHEN** a user supplies `:function_name` and a file path
- **THEN** the system invokes `git log -L :function_name:<path>`

### Requirement: Discoverable and compatible command surface
The system SHALL document all file-history views in `aiw git show` help and SHALL preserve every existing view and its behavior.

#### Scenario: Show command help
- **WHEN** a user requests help for `aiw git show`
- **THEN** the output lists file history, blame, historical content, and line-history views with examples

#### Scenario: Existing view
- **WHEN** a user invokes an existing `aiw git show` view
- **THEN** the system dispatches it with unchanged behavior
