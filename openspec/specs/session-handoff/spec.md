# session-handoff Specification

## Purpose
TBD - created by archiving change integrate-grill-handoff. Update Purpose after archive.
## Requirements
### Requirement: Create a deterministic handoff
The system SHALL create a deterministic Markdown handoff from stored session status, memory, artifact references, workspace context, and the latest final output excerpt without invoking a model.

#### Scenario: Create handoff for an existing session
- **WHEN** a user runs `handoff create` for an existing session
- **THEN** the system writes `artifacts/handoff.md` with goal, state, findings, decisions, files, validation, open issues, next action, suggested skills, and artifact references

#### Scenario: Add a focus
- **WHEN** a user supplies a handoff focus
- **THEN** the handoff records that focus as the recommended continuation focus

### Requirement: Store handoff safely
The system MUST write the handoff atomically under the existing per-session lock.

#### Scenario: Concurrent session mutation
- **WHEN** the handoff is created while another process is mutating the same session
- **THEN** the handoff writer waits for the session lock or exits with the existing lock timeout behavior

### Requirement: Display the stored handoff
The system SHALL provide a cross-platform `handoff show` command that prints the stored handoff.

#### Scenario: Show existing handoff
- **WHEN** `artifacts/handoff.md` exists
- **THEN** the command prints its UTF-8 content

#### Scenario: Show missing handoff
- **WHEN** no handoff has been created for the session
- **THEN** the command exits with a clear error that recommends running `handoff create`

### Requirement: Reference rather than duplicate large artifacts
The handoff SHALL reference saved prompts, outputs, events, patches, and context by session-relative path and SHALL include only a bounded latest-output excerpt.

#### Scenario: Session has large outputs
- **WHEN** the latest final output exceeds the excerpt limit
- **THEN** the handoff includes a truncated excerpt and the path to the complete output
