# skills Specification

## Purpose

Define how AIW discovers, lists, and invokes Skills from an interactive Session.

## Requirements

### Requirement: Discover Skills from standard locations
The system SHALL discover Skill directories from project and user
`.agents/skills` locations and from compatible project and user `.codex/skills`
locations without requiring configurable search paths.

#### Scenario: Discover project Skills
- **WHEN** a Session workspace or its repository ancestry contains a valid Skill
  under `.agents/skills`
- **THEN** the system includes that Skill in the available project Skills

#### Scenario: Discover compatible project Skills
- **WHEN** the Session project root contains a valid Skill under `.codex/skills`
- **THEN** the system includes that Skill in the available project Skills

#### Scenario: Discover user Skills
- **WHEN** the user home contains a valid Skill under `.agents/skills` or the
  effective Codex home contains a valid Skill under `skills`
- **THEN** the system includes that Skill in the available user Skills

#### Scenario: Resolve a non-Git workspace
- **WHEN** the Session workspace is not inside a Git repository
- **THEN** the system treats the workspace as the project root when resolving
  project Skill locations

### Requirement: List Skills without executing a turn
The interactive loop SHALL process `/skills` locally and display discovered
Skills with their scope and source without calling Codex.

#### Scenario: List valid Skills
- **WHEN** the user enters `/skills` and valid project or user Skills are
  discoverable
- **THEN** the system displays their names, descriptions, scopes, and source
  paths without incrementing the Session turn number

#### Scenario: List an empty catalog
- **WHEN** the user enters `/skills` and no valid Skills are discoverable
- **THEN** the system reports that no Skills were found and waits for another
  input without calling Codex

#### Scenario: Report malformed candidates
- **WHEN** the user enters `/skills` and a candidate has missing or invalid
  required frontmatter
- **THEN** the system reports a warning containing the candidate path while
  continuing to display other valid Skills

### Requirement: Invoke a discovered Skill
The interactive loop SHALL accept `/skill <name> <message>` and execute a valid
request through the normal turn path using Codex's native `$<name>` invocation
syntax.

#### Scenario: Invoke a project Skill
- **WHEN** the user enters
  `/skill metrics-review Review the revenue metrics` and exactly one discovered
  Skill is named `metrics-review`
- **THEN** the system executes
  `$metrics-review Review the revenue metrics` as one normal Session turn

#### Scenario: Preserve normal turn behavior
- **WHEN** a valid `/skill` request executes
- **THEN** the system persists its Prompt, Output, Event, status, timeout result,
  and Thread changes exactly as an ordinary loop message

#### Scenario: Preserve direct invocation
- **WHEN** the user enters an ordinary message beginning with
  `$metrics-review`
- **THEN** the system sends that message unchanged through the normal turn path

### Requirement: Reject invalid Skill commands safely
The interactive loop SHALL reject malformed, missing, or ambiguous Skill
invocations without calling Codex.

#### Scenario: Missing Skill message
- **WHEN** the user enters `/skill metrics-review` without a task message
- **THEN** the system displays the required syntax and waits for another input
  without incrementing the turn number

#### Scenario: Unknown Skill name
- **WHEN** the user invokes a name that is not present in the current discovery
  result
- **THEN** the system reports that the Skill was not found and waits for another
  input without calling Codex

#### Scenario: Duplicate Skill name
- **WHEN** more than one discovered Skill declares the requested name
- **THEN** the system reports every conflicting source path and refuses to
  execute the turn

#### Scenario: Skill changes before invocation
- **WHEN** the discovery contents change after an earlier `/skills` command and
  before `/skill` is entered
- **THEN** the system uses a fresh discovery result to validate the invocation

### Requirement: Preserve existing loop command behavior
The Skill commands MUST coexist with all existing loop messages, escapes, and
local commands without changing their behavior.

#### Scenario: Escape a Skill-like slash message
- **WHEN** the user enters
  `//skill metrics-review Review this text`
- **THEN** the system sends
  `/skill metrics-review Review this text` as an ordinary message instead of
  processing a local Skill command

#### Scenario: Use an existing command
- **WHEN** the user enters an existing local command such as `/status`
- **THEN** the system processes that command with its existing behavior
