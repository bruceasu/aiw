## ADDED Requirements

### Requirement: Session working directory metadata
The system SHALL extract and expose the original working directory from each
Codex session when that metadata is available, and SHALL preserve sessions with
unknown working directories without guessing a value.

#### Scenario: Session contains working directory metadata
- **WHEN** the scanner reads a supported Codex session metadata record
- **THEN** the resulting session metadata contains the normalized original
  working directory

#### Scenario: Session has no recognized working directory
- **WHEN** no supported metadata record contains a working directory
- **THEN** the session remains discoverable with an unknown working directory

### Requirement: Workspace-scoped session listing
The GUI and `resume-ext` skill SHALL show only sessions belonging to the current
workspace by default and SHALL provide an explicit option to show sessions from
all workspaces. The existing CLI list SHALL remain global by default.

#### Scenario: Open the default GUI or skill session view
- **WHEN** a user opens the GUI or invokes `resume-ext` without selecting
  all-workspaces mode
- **THEN** the result includes sessions whose original directory is the current
  workspace or one of its descendants

#### Scenario: Session workspace is unknown
- **WHEN** a session has no recoverable original working directory
- **THEN** the default workspace view excludes it

#### Scenario: Show all workspaces
- **WHEN** a user enables all-workspaces mode
- **THEN** the result includes global sessions and marks sessions with unknown
  working directories

#### Scenario: Use the existing list command
- **WHEN** a user invokes `aiw cxs list` without a workspace filter
- **THEN** the command continues to list global sessions as before

### Requirement: Backward-compatible metadata caching
The system MUST refresh legacy cached session metadata that lacks original
working-directory information without modifying session JSONL files or existing
alias mappings.

#### Scenario: Read a legacy cache record
- **WHEN** an otherwise valid cache record predates working-directory metadata
- **THEN** the scanner reinspects that session file and writes an updated cache
  record

#### Scenario: Open the new session view
- **WHEN** the user opens a list or GUI without changing an alias
- **THEN** the existing alias index remains unchanged

### Requirement: Desktop session browser
The system SHALL provide an `aiw cxs gui` entry point that displays session
identity, aliases, title, update time, turn count, and original working
directory.

#### Scenario: Open the GUI in a workspace
- **WHEN** a user runs `aiw cxs gui`
- **THEN** the GUI opens with the workspace-scoped session list selected by
  default

#### Scenario: GUI toolkit is unavailable
- **WHEN** the configured Python runtime cannot load the standard GUI toolkit
- **THEN** the command exits with an actionable error and existing CLI commands
  remain usable

### Requirement: Safe conversation preview
The GUI SHALL render a bounded readable preview of the selected session's user
and assistant content and MUST NOT modify the session file.

#### Scenario: Select a session
- **WHEN** a user selects a session in the GUI
- **THEN** the preview shows bounded readable user and assistant entries for
  that session

#### Scenario: Session contains tool or system events
- **WHEN** the preview encounters raw tool or system events
- **THEN** those events are omitted from the default preview

### Requirement: Alias maintenance
The GUI SHALL allow a user to create, rename, and remove workspace-scoped
aliases while preserving existing alias validation and conflict protection.

#### Scenario: Rename an alias
- **WHEN** a user supplies a valid unused replacement name
- **THEN** the alias index is updated atomically to retain the same session
  binding under the new name

#### Scenario: Replacement name conflicts
- **WHEN** the requested alias name already exists
- **THEN** the GUI refuses replacement unless the user explicitly confirms it

#### Scenario: Remove an alias
- **WHEN** a user confirms alias removal
- **THEN** only that alias mapping is removed and the session file is unchanged

### Requirement: Interactive session resume
The GUI SHALL continue the selected session with interactive
`codex resume <session-id>` using the session's original working directory.

#### Scenario: Resume a valid session
- **WHEN** the selected session has an existing original working directory and
  the invoking environment supports an interactive terminal
- **THEN** the GUI closes and starts interactive Codex resume with that
  directory as the child process working directory

#### Scenario: Original directory is unavailable
- **WHEN** the selected session's original working directory is unknown or no
  longer exists
- **THEN** the GUI does not launch Codex and displays a recovery message and the
  intended command

#### Scenario: No interactive terminal is available
- **WHEN** the GUI cannot safely hand control to an interactive terminal
- **THEN** it displays a copyable resume command instead of starting a nested or
  detached Codex process

### Requirement: Resume selection skill
The repository SHALL provide a concise `resume-ext` skill that presents
workspace-scoped sessions as numbered options and resolves a selected number or
alias to an interactive resume handoff.

#### Scenario: Invoke the skill
- **WHEN** a user invokes `resume-ext` through the supported skill surface
- **THEN** the skill lists compact workspace-scoped session options containing
  alias, title, update time, and original directory

#### Scenario: Select a session option
- **WHEN** the user selects a displayed number or alias
- **THEN** the skill displays the resolved session and exact interactive resume
  command

#### Scenario: Skill runs inside an active Codex session
- **WHEN** native thread handoff is unavailable
- **THEN** the skill provides a copyable command and MUST NOT launch a nested
  Codex process

### Requirement: Existing CLI compatibility
The change MUST preserve existing `aiw cxs` command names, arguments, defaults,
alias file compatibility, and `codex exec resume` behavior.

#### Scenario: Use an existing command
- **WHEN** a caller invokes an existing valid `aiw cxs` command
- **THEN** it parses and behaves compatibly with the previous command surface
