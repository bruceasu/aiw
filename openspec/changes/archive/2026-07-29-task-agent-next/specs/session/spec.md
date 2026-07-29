# session Specification Delta

## ADDED Requirements

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
- **THEN** those commands retain their current behavior and no Task metadata
  is required
