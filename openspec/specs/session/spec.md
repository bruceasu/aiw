# session Specification

## Purpose

Define interactive Session behavior and deterministic handoff artifacts for
continuing work safely across turns and processes.

## Requirements

### Requirement: Enter an interactive Session loop
The system SHALL provide `loop SESSION_ID` to repeatedly accept terminal input
for an existing Session within one aiw-flow process.

#### Scenario: Resume a Session with a Thread
- **WHEN** a user starts the loop for a Session with a saved Codex Thread ID and
  enters a message
- **THEN** the system executes the message against the saved Thread and waits
  for the next input

#### Scenario: Start a Session without a Thread
- **WHEN** a user starts the loop for a newly created Session without a Thread
  ID and enters the first message
- **THEN** the system executes a first turn, saves the returned Thread ID, and
  waits for the next input

### Requirement: Create a normal Session directly in loop mode
The system SHALL allow `new --loop` with an optional phase to create a normal
Session and immediately wait for its first interactive message.

#### Scenario: Create and enter loop
- **WHEN** a user supplies valid `new` arguments with `--loop`
- **THEN** the system creates the Session and enters the loop without requiring
  a separate `run` command

#### Scenario: Preserve one-shot creation
- **WHEN** a user runs `new` without `--loop`
- **THEN** the system creates the Session and exits with the existing behavior

### Requirement: Reuse normal turn persistence
Every ordinary loop message MUST use the existing turn execution path and
persist Prompt, Output, Event, status, timeout, and Thread changes exactly as a
one-shot turn.

#### Scenario: Submit two messages
- **WHEN** a user submits two ordinary messages in one loop
- **THEN** the system records two sequential turns and the second turn resumes
  the Thread saved by the first

### Requirement: Resolve the loop phase
The system SHALL use the explicitly supplied phase, otherwise the Session
current phase, otherwise `interactive`.

#### Scenario: Resume without explicit phase
- **WHEN** a Session current phase is `analyze` and the user runs
  `loop SESSION_ID` without `--phase`
- **THEN** each submitted turn uses phase `analyze`

#### Scenario: New Session without explicit phase
- **WHEN** a newly created Session enters loop mode without an explicit phase
- **THEN** each submitted turn uses phase `interactive`

### Requirement: Support local commands
The loop SHALL process `/help`, `/status`, `/memory`, `/handoff`, `/done`, and
`/exit` locally according to their documented behavior.

#### Scenario: Inspect status without a model turn
- **WHEN** the user enters `/status`
- **THEN** the system displays current Session status and waits for input
  without incrementing the turn number

#### Scenario: Create a handoff without a model turn
- **WHEN** the user enters `/handoff`
- **THEN** the system creates the deterministic Session handoff and waits for
  input without incrementing the turn number

#### Scenario: Finish Grill discovery
- **WHEN** the loop phase is `grill` and the user enters `/done`
- **THEN** the system sends `Grill Done` as one final turn, displays the
  response, and exits the loop

#### Scenario: Reject done outside Grill
- **WHEN** the loop phase is not `grill` and the user enters `/done`
- **THEN** the system displays a local error and waits for input without calling
  Codex

### Requirement: Exit cleanly while idle
The loop SHALL exit successfully without changing Session state when it
receives `/exit`, EOF, or `Ctrl+C` while waiting for input.

#### Scenario: Exit command
- **WHEN** the user enters `/exit`
- **THEN** the loop exits without executing another turn

#### Scenario: Terminal EOF
- **WHEN** terminal input raises EOF while the loop is idle
- **THEN** the loop exits successfully and preserves the current Session state

### Requirement: Ignore non-messages safely
The loop SHALL ignore empty input and SHALL report unknown slash commands
without calling Codex.

#### Scenario: Empty input
- **WHEN** the user submits only whitespace
- **THEN** the loop waits for another input without executing a turn

#### Scenario: Unknown command
- **WHEN** the user enters an unsupported slash command
- **THEN** the loop displays an error and waits for another input without
  executing a turn

### Requirement: Reject unavailable Session states
The system SHALL refuse to enter a loop for a Session that is running,
completed, archived, or deleted.

#### Scenario: Completed Session
- **WHEN** a user starts a loop for a completed Session
- **THEN** the command exits with a clear state error before reading terminal
  input

### Requirement: Create a deterministic handoff
The system SHALL create a deterministic Markdown handoff from stored Session
status, memory, artifact references, workspace context, and the latest final
output excerpt without invoking a model.

#### Scenario: Create handoff for an existing Session
- **WHEN** a user runs `handoff create` for an existing Session
- **THEN** the system writes `artifacts/handoff.md` with goal, state, findings,
  decisions, files, validation, open issues, next action, suggested skills, and
  artifact references

#### Scenario: Add a focus
- **WHEN** a user supplies a handoff focus
- **THEN** the handoff records that focus as the recommended continuation focus

### Requirement: Store handoff safely
The system MUST write the handoff atomically under the existing per-Session
lock.

#### Scenario: Concurrent Session mutation
- **WHEN** the handoff is created while another process is mutating the same
  Session
- **THEN** the handoff writer waits for the Session lock or exits with the
  existing lock timeout behavior

### Requirement: Display the stored handoff
The system SHALL provide a cross-platform `handoff show` command that prints the
stored handoff.

#### Scenario: Show existing handoff
- **WHEN** `artifacts/handoff.md` exists
- **THEN** the command prints its UTF-8 content

#### Scenario: Show missing handoff
- **WHEN** no handoff has been created for the Session
- **THEN** the command exits with a clear error that recommends running
  `handoff create`

### Requirement: Reference rather than duplicate large artifacts
The handoff SHALL reference saved prompts, outputs, events, patches, and context
by Session-relative path and SHALL include only a bounded latest-output excerpt.

#### Scenario: Session has large outputs
- **WHEN** the latest final output exceeds the excerpt limit
- **THEN** the handoff includes a truncated excerpt and the path to the complete
  output


### Requirement: Fork a fresh Thread from the interactive Loop
The Loop SHALL support `/fork` to create a persistent handoff, execute one
fresh Thread whose business prompt is the handoff content, and then exit.

#### Scenario: Fork while idle
- **WHEN** an active Loop receives `/fork` while waiting for input
- **THEN** the system writes `artifacts/handoff.md`, starts one fresh Thread
  with that handoff as its prompt, and exits the Loop after the turn

#### Scenario: Fork does not reuse the current Thread
- **WHEN** the Session already has a Codex Thread ID
- **THEN** the forked turn starts without resuming that Thread

### Requirement: Bind a Session to Task handoff lineage
The Session state SHALL expose the optional Task binding and the latest
fresh-Thread handoff lineage without changing ordinary loop persistence.

#### Scenario: Session status has lineage
- **WHEN** a Task agent transition has completed
- **THEN** Session status includes the bound Task ID, parent and child Thread
  IDs, handoff path, and handoff hash

#### Scenario: Existing Session without a Task
- **WHEN** a Session is used by existing `run`, `continue`, or `loop` commands
  without a Task binding
- **THEN** those commands retain their current behavior and no Task metadata is
  required
