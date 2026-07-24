## ADDED Requirements

### Requirement: Start a Grill session
The system SHALL provide a Grill command that creates a normal aiw-flow session from a session ID, title, workspace, and requirement, then starts the first Codex turn using the existing exec backend.

#### Scenario: Start requirement discovery
- **WHEN** a user supplies a valid new session ID, existing workspace, and non-empty requirement
- **THEN** the system creates the session with built-in Grill instructions and executes the first turn in that workspace

#### Scenario: Reject missing requirement
- **WHEN** the requirement is empty or no supported requirement input is provided
- **THEN** the system exits with a clear error without starting a Codex turn

### Requirement: Ask one decision question per turn
The built-in Grill instructions SHALL direct the agent to ask at most one user decision question per turn, provide a recommended answer with rationale, and inspect the workspace before asking for discoverable facts.

#### Scenario: Agent needs a user decision
- **WHEN** the agent cannot resolve the next decision from the workspace or confirmed session facts
- **THEN** the response contains one decision question and a recommended answer with rationale

### Requirement: Finish with a confirmed specification
The built-in Grill instructions SHALL direct the agent to emit `SUCCESS: Ready to execute.` and a structured final specification only after the user explicitly confirms that discovery is complete.

#### Scenario: User confirms discovery is complete
- **WHEN** the user explicitly states that Grill discovery is done
- **THEN** the agent returns the success marker and the final specification instead of another interview question

### Requirement: Resume through the existing session
The Grill workflow SHALL persist a normal Codex thread ID so later answers can use the existing `continue` command.

#### Scenario: Continue a Grill interview
- **WHEN** the first Grill turn completed with a thread ID and the user runs `continue` with phase `grill`
- **THEN** the system resumes the same Codex thread with the saved instructions and memory
