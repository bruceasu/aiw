## ADDED Requirements

### Requirement: Collect context from the declared workspace
The system SHALL collect workspace context only from the resolved workspace stored for the session.

#### Scenario: Grill session starts
- **WHEN** a Grill session is created
- **THEN** the system collects context from that session's workspace and saves it as `artifacts/workspace-context.md`

### Requirement: Bound context collection
The collector MUST enforce maximum directory depth, directory entry count, per-file bytes, and total content bytes.

#### Scenario: Workspace exceeds limits
- **WHEN** the workspace has more entries or metadata content than the configured limits
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
