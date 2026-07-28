# Capability Discovery Specification

### Requirement: Stable template discovery guidance

Codex templates SHALL point AI agents to the capability catalog and runtime discovery commands without embedding absolute plugin paths.

#### Scenario: Load repository instructions

- **WHEN** an AI agent loads the Codex template
- **THEN** it learns to inspect the capability catalog and runtime help before using unfamiliar AIW tools

### Requirement: Machine-readable capability records

The system SHALL provide normalized machine-readable capability records for available commands and plugins.

#### Scenario: Discover installed capabilities

- **WHEN** a user invokes runtime capability discovery
- **THEN** the system returns each capability's name, description, invocation, metadata source, side-effect classification, confirmation requirement, and output format

#### Scenario: Legacy plugin metadata

- **WHEN** a plugin exposes only an existing META dictionary
- **THEN** the system includes it with conservative defaults for fields that are absent

### Requirement: Runtime-resolved metadata paths

The system SHALL return metadata paths resolved for the current installation or worktree.

#### Scenario: Different worktree

- **WHEN** the same command is run from another worktree
- **THEN** capability records use paths valid in that worktree and do not require a hard-coded absolute path

### Requirement: Safe AI selection

The discovery output SHALL distinguish read-only capabilities from capabilities that modify files, the index, external systems, or other state.

#### Scenario: Mutating capability

- **WHEN** a capability can modify state
- **THEN** the record marks the side effect and whether explicit confirmation is required