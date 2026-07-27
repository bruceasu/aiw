## ADDED Requirements

### Requirement: Select a workflow backend
The system SHALL support `auto`, `openspec`, and `native` backend modes for
task workflow commands.

#### Scenario: Native mode
- **WHEN** a user selects `--backend native`
- **THEN** the command uses the existing AIW implementation without probing or
  invoking OpenSpec

#### Scenario: Auto mode without OpenSpec
- **WHEN** `--backend auto` is selected and no verified OpenSpec executable is
  available
- **THEN** the command uses the native implementation and reports that
  fallback was selected

### Requirement: Verify an OpenSpec executable
The system SHALL verify an OpenSpec executable before delegating work to it.

#### Scenario: Explicit OpenSpec unavailable
- **WHEN** `--backend openspec` is selected and the executable is missing or
  `--version` fails
- **THEN** the command exits before writing task artifacts and reports how to
  configure or install OpenSpec

#### Scenario: Configured executable
- **WHEN** `AIW_OPENSPEC_BIN` points to an executable that returns successfully
  for `--version`
- **THEN** the adapter uses that executable for delegation

### Requirement: Preserve native compatibility
The system SHALL preserve existing native task behavior when OpenSpec is
unavailable or when `native` is selected.

#### Scenario: Existing script invocation without OpenSpec
- **WHEN** a script invokes a task workflow without a backend option and
  OpenSpec is unavailable
- **THEN** the command produces the existing native artifacts and exit status

#### Scenario: Automatic delegation
- **WHEN** a task workflow is invoked without a backend option and a verified
  OpenSpec executable is available
- **THEN** the command delegates to OpenSpec

### Requirement: Report backend choice
The system SHALL report the selected backend and whether delegation or fallback
was used.

#### Scenario: Auto fallback diagnostic
- **WHEN** auto mode selects the native backend
- **THEN** a concise diagnostic identifies native fallback and the reason
