## ADDED Requirements

### Requirement: Start Grill in interactive mode
The system SHALL allow a Grill command with `--loop` to enter the interactive Session loop after the first Grill response.

#### Scenario: Continue immediately after the first question
- **WHEN** a user starts a valid Grill Session with `--loop` and the first turn succeeds
- **THEN** the system waits for the user's answer in the same aiw-flow process using phase `grill`

#### Scenario: Preserve one-shot Grill
- **WHEN** a user starts a valid Grill Session without `--loop`
- **THEN** the system prints the first response and exits with the existing behavior

#### Scenario: First Grill turn fails
- **WHEN** the first Grill turn returns a non-zero exit code
- **THEN** the system returns that code without entering the interactive loop
