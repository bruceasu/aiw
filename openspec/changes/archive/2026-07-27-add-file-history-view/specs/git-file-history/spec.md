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
