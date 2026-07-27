# aiw-flow Specification

## Purpose

Define workflow behavior provided by the aiw-flow plugin.

## Requirements

### Requirement: Start a Grill Session
The system SHALL provide a Grill command that creates a normal aiw-flow Session
from a Session ID, title, workspace, and requirement, then starts the first
Codex turn using the existing exec backend.

#### Scenario: Start requirement discovery
- **WHEN** a user supplies a valid new Session ID, existing workspace, and
  non-empty requirement
- **THEN** the system creates the Session with built-in Grill instructions and
  executes the first turn in that workspace

#### Scenario: Reject missing requirement
- **WHEN** the requirement is empty or no supported requirement input is
  provided
- **THEN** the system exits with a clear error without starting a Codex turn

### Requirement: Ask one decision question per turn
The built-in Grill instructions SHALL direct the agent to ask at most one user
decision question per turn, provide a recommended answer with rationale, and
inspect the workspace before asking for discoverable facts.

#### Scenario: Agent needs a user decision
- **WHEN** the agent cannot resolve the next decision from the workspace or
  confirmed Session facts
- **THEN** the response contains one decision question and a recommended answer
  with rationale

### Requirement: Finish with a confirmed specification
The built-in Grill instructions SHALL direct the agent to emit
`SUCCESS: Ready to execute.` and a structured final specification only after
the user explicitly confirms that discovery is complete.

#### Scenario: User confirms discovery is complete
- **WHEN** the user explicitly states that Grill discovery is done
- **THEN** the agent returns the success marker and the final specification
  instead of another interview question

### Requirement: Resume through the existing Session
The Grill workflow SHALL persist a normal Codex Thread ID so later answers can
use the existing `continue` command.

#### Scenario: Continue a Grill interview
- **WHEN** the first Grill turn completed with a Thread ID and the user runs
  `continue` with phase `grill`
- **THEN** the system resumes the same Codex Thread with the saved instructions
  and memory

### Requirement: Start Grill in interactive mode
The system SHALL allow a Grill command with `--loop` to enter the interactive
Session loop after the first Grill response.

#### Scenario: Continue immediately after the first question
- **WHEN** a user starts a valid Grill Session with `--loop` and the first turn
  succeeds
- **THEN** the system waits for the user's answer in the same aiw-flow process
  using phase `grill`

#### Scenario: Preserve one-shot Grill
- **WHEN** a user starts a valid Grill Session without `--loop`
- **THEN** the system prints the first response and exits with the existing
  behavior

#### Scenario: First Grill turn fails
- **WHEN** the first Grill turn returns a non-zero exit code
- **THEN** the system returns that code without entering the interactive loop
