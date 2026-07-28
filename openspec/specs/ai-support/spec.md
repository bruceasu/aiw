# ai-support Specification

## Purpose
Define AI support behavior for aiw-flow, interactive sessions, handoff artifacts, grill workflows, and task-bound session lineage.

## Requirements
### Requirement: Enter an interactive Session loop
The system SHALL provide `loop SESSION_ID` to repeatedly accept terminal input for an existing Session within one aiw-flow process.

#### Scenario: Resume a Session with a Thread
- **WHEN** a user starts the loop for a Session with a saved Codex Thread ID and enters a message
- **THEN** the system executes the message against the saved Thread and waits for the next input

#### Scenario: Start a Session without a Thread
- **WHEN** a user starts the loop for a newly created Session without a Thread ID and enters the first message
- **THEN** the system executes a first turn, saves the returned Thread ID, and waits for the next input

### Requirement: Create a normal Session directly in loop mode
The system SHALL allow `new --loop` with an optional phase to create a normal Session and immediately wait for its first interactive message.

#### Scenario: Create and enter loop
- **WHEN** a user supplies valid `new` arguments with `--loop`
- **THEN** the system creates the Session and enters the loop without requiring a separate `run` command

#### Scenario: Preserve one-shot creation
- **WHEN** a user runs `new` without `--loop`
- **THEN** the system creates the Session and exits with the existing behavior

### Requirement: Reuse normal turn persistence
Every ordinary loop message MUST use the existing turn execution path and persist Prompt, Output, Event, status, timeout, and Thread changes exactly as a one-shot turn.

#### Scenario: Submit two messages
- **WHEN** a user submits two ordinary messages in one loop
- **THEN** the system records two sequential turns and the second turn resumes the Thread saved by the first

### Requirement: Resolve the loop phase
The system SHALL use the explicitly supplied phase, otherwise the Session current phase, otherwise `interactive`.

#### Scenario: Resume without explicit phase
- **WHEN** a Session current phase is `analyze` and the user runs `loop SESSION_ID` without `--phase`
- **THEN** each submitted turn uses phase `analyze`

#### Scenario: New Session without explicit phase
- **WHEN** a newly created Session enters loop mode without an explicit phase
- **THEN** each submitted turn uses phase `interactive`

### Requirement: Support local commands
The loop SHALL process `/help`, `/status`, `/memory`, `/handoff`, `/done`, and `/exit` locally according to their documented behavior.

#### Scenario: Inspect status without a model turn
- **WHEN** the user enters `/status`
- **THEN** the system displays current Session status and waits for input without incrementing the turn number

#### Scenario: Create a handoff without a model turn
- **WHEN** the user enters `/handoff`
- **THEN** the system creates the deterministic Session handoff and waits for input without incrementing the turn number

#### Scenario: Finish Grill discovery
- **WHEN** the loop phase is `grill` and the user enters `/done`
- **THEN** the system sends `Grill Done` as one final turn, displays the response, and exits the loop

#### Scenario: Reject done outside Grill
- **WHEN** the loop phase is not `grill` and the user enters `/done`
- **THEN** the system displays a local error and waits for input without calling Codex

### Requirement: Exit cleanly while idle
The loop SHALL exit successfully without changing Session state when it receives `/exit`, EOF, or `Ctrl+C` while waiting for input.

#### Scenario: Exit command
- **WHEN** the user enters `/exit`
- **THEN** the loop exits without executing another turn

#### Scenario: Terminal EOF
- **WHEN** terminal input raises EOF while the loop is idle
- **THEN** the loop exits successfully and preserves the current Session state

### Requirement: Ignore non-messages safely
The loop SHALL ignore empty input and SHALL report unknown slash commands without calling Codex.

#### Scenario: Empty input
- **WHEN** the user submits only whitespace
- **THEN** the loop waits for another input without executing a turn

#### Scenario: Unknown command
- **WHEN** the user enters an unsupported slash command
- **THEN** the loop displays an error and waits for another input without executing a turn

### Requirement: Reject unavailable Session states
The system SHALL refuse to enter a loop for a Session that is running, completed, archived, or deleted.

#### Scenario: Completed Session
- **WHEN** a user starts a loop for a completed Session
- **THEN** the command exits with a clear state error before reading terminal input

### Requirement: Create a deterministic handoff
The system SHALL create a deterministic Markdown handoff from stored Session status, memory, artifact references, workspace context, and the latest final output excerpt without invoking a model.

#### Scenario: Create handoff for an existing Session
- **WHEN** a user runs `handoff create` for an existing Session
- **THEN** the system writes `artifacts/handoff.md` with goal, state, findings, decisions, files, validation, open issues, next action, suggested skills, and artifact references

#### Scenario: Add a focus
- **WHEN** a user supplies a handoff focus
- **THEN** the handoff records that focus as the recommended continuation focus

### Requirement: Store handoff safely
The system MUST write the handoff atomically under the existing per-Session lock.

#### Scenario: Concurrent Session mutation
- **WHEN** the handoff is created while another process is mutating the same Session
- **THEN** the handoff writer waits for the Session lock or exits with the existing lock timeout behavior

### Requirement: Display the stored handoff
The system SHALL provide a cross-platform `handoff show` command that prints the stored handoff.

#### Scenario: Show existing handoff
- **WHEN** `artifacts/handoff.md` exists
- **THEN** the command prints its UTF-8 content

#### Scenario: Show missing handoff
- **WHEN** no handoff has been created for the Session
- **THEN** the command exits with a clear error that recommends running `handoff create`

### Requirement: Reference rather than duplicate large artifacts
The handoff SHALL reference saved prompts, outputs, events, patches, and context by Session-relative path and SHALL include only a bounded latest-output excerpt.

#### Scenario: Session has large outputs
- **WHEN** the latest final output exceeds the excerpt limit
- **THEN** the handoff includes a truncated excerpt and the path to the complete output

### Requirement: Fork a fresh Thread from the interactive Loop
The Loop SHALL support `/fork` to create a persistent handoff, execute one fresh Thread whose business prompt is the handoff content, and then exit.

#### Scenario: Fork while idle
- **WHEN** an active Loop receives `/fork` while waiting for input
- **THEN** the system writes `artifacts/handoff.md`, starts one fresh Thread with that handoff as its prompt, and exits the Loop after the turn

#### Scenario: Fork does not reuse the current Thread
- **WHEN** the Session already has a Codex Thread ID
- **THEN** the forked turn starts without resuming that Thread

### Requirement: Bind a Session to Task handoff lineage
The Session state SHALL expose the optional Task binding and the latest fresh-Thread handoff lineage without changing ordinary loop persistence.

#### Scenario: Session status has lineage
- **WHEN** a Task agent transition has completed
- **THEN** Session status includes the bound Task ID, parent and child Thread IDs, handoff path, and handoff hash

#### Scenario: Existing Session without a Task
- **WHEN** a Session is used by existing `run`, `continue`, or `loop` commands without a Task binding
- **THEN** those commands retain their current behavior and no Task metadata is required

### Requirement: Start Grill sessions
The system SHALL provide a Grill command that creates a normal aiw-flow Session from a Session ID, title, workspace, and requirement, then starts the first Codex turn using the existing exec backend.

#### Scenario: Start requirement discovery
- **WHEN** a user supplies a valid new Session ID, existing workspace, and non-empty requirement
- **THEN** the system creates the Session with built-in Grill instructions and executes the first turn in that workspace

### Requirement: Ask one decision question per turn
The built-in Grill instructions SHALL direct the agent to ask at most one user decision question per turn, provide a recommended answer with rationale, and inspect the workspace before asking for discoverable facts.

#### Scenario: Agent needs a user decision
- **WHEN** the agent cannot resolve the next decision from the workspace or confirmed Session facts
- **THEN** the response contains one decision question and a recommended answer with rationale

### Requirement: Finish with a confirmed specification
The built-in Grill instructions SHALL direct the agent to emit `SUCCESS: Ready to execute.` and a structured final specification only after the user explicitly confirms that discovery is complete.

#### Scenario: User confirms discovery is complete
- **WHEN** the user explicitly states that Grill discovery is done
- **THEN** the agent returns the success marker and the final specification instead of another interview question

### Requirement: Resume through the existing Session
The Grill workflow SHALL persist a normal Codex Thread ID so later answers can use the existing `continue` command.

#### Scenario: Continue a Grill interview
- **WHEN** the first Grill turn completed with a Thread ID and the user runs `continue` with phase `grill`
- **THEN** the system resumes the same Codex Thread with the saved instructions and memory

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

<!-- archived spec: grill-workflow -->

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

<!-- archived spec: interactive-session-loop -->

## ADDED Requirements

### Requirement: Enter an interactive Session loop
The system SHALL provide `loop SESSION_ID` to repeatedly accept terminal input for an existing Session within one aiw-flow process.

#### Scenario: Resume a Session with a Thread
- **WHEN** a user starts the loop for a Session with a saved Codex Thread ID and enters a message
- **THEN** the system executes the message against the saved Thread and waits for the next input

#### Scenario: Start a Session without a Thread
- **WHEN** a user starts the loop for a newly created Session without a Thread ID and enters the first message
- **THEN** the system executes a first turn, saves the returned Thread ID, and waits for the next input

### Requirement: Create a normal Session directly in loop mode
The system SHALL allow `new --loop` with an optional phase to create a normal Session and immediately wait for its first interactive message.

#### Scenario: Create and enter loop
- **WHEN** a user supplies valid `new` arguments with `--loop`
- **THEN** the system creates the Session and enters the loop without requiring a separate `run` command

#### Scenario: Preserve one-shot creation
- **WHEN** a user runs `new` without `--loop`
- **THEN** the system creates the Session and exits with the existing behavior

### Requirement: Reuse normal turn persistence
Every ordinary loop message MUST use the existing turn execution path and persist Prompt, Output, Event, status, timeout, and Thread changes exactly as a one-shot turn.

#### Scenario: Submit two messages
- **WHEN** a user submits two ordinary messages in one loop
- **THEN** the system records two sequential turns and the second turn resumes the Thread saved by the first

### Requirement: Resolve the loop phase
The system SHALL use the explicitly supplied phase, otherwise the Session current phase, otherwise `interactive`.

#### Scenario: Resume without explicit phase
- **WHEN** a Session current phase is `analyze` and the user runs `loop SESSION_ID` without `--phase`
- **THEN** each submitted turn uses phase `analyze`

#### Scenario: New Session without explicit phase
- **WHEN** a newly created Session enters loop mode without an explicit phase
- **THEN** each submitted turn uses phase `interactive`

### Requirement: Support local commands
The loop SHALL process `/help`, `/status`, `/memory`, `/handoff`, `/done`, and `/exit` locally according to their documented behavior.

#### Scenario: Inspect status without a model turn
- **WHEN** the user enters `/status`
- **THEN** the system displays current Session status and waits for input without incrementing the turn number

#### Scenario: Create a handoff without a model turn
- **WHEN** the user enters `/handoff`
- **THEN** the system creates the deterministic Session handoff and waits for input without incrementing the turn number

#### Scenario: Finish Grill discovery
- **WHEN** the loop phase is `grill` and the user enters `/done`
- **THEN** the system sends `Grill Done` as one final turn, displays the response, and exits the loop

#### Scenario: Reject done outside Grill
- **WHEN** the loop phase is not `grill` and the user enters `/done`
- **THEN** the system displays a local error and waits for input without calling Codex

### Requirement: Exit cleanly while idle
The loop SHALL exit successfully without changing Session state when it receives `/exit`, EOF, or `Ctrl+C` while waiting for input.

#### Scenario: Exit command
- **WHEN** the user enters `/exit`
- **THEN** the loop exits without executing another turn

#### Scenario: Terminal EOF
- **WHEN** terminal input raises EOF while the loop is idle
- **THEN** the loop exits successfully and preserves the current Session state

### Requirement: Ignore non-messages safely
The loop SHALL ignore empty input and SHALL report unknown slash commands without calling Codex.

#### Scenario: Empty input
- **WHEN** the user submits only whitespace
- **THEN** the loop waits for another input without executing a turn

#### Scenario: Unknown command
- **WHEN** the user enters an unsupported slash command
- **THEN** the loop displays an error and waits for another input without executing a turn

### Requirement: Reject unavailable Session states
The system SHALL refuse to enter a loop for a Session that is running, completed, archived, or deleted.

#### Scenario: Completed Session
- **WHEN** a user starts a loop for a completed Session
- **THEN** the command exits with a clear state error before reading terminal input

<!-- archived spec: aiw-flow-cli-help -->

## ADDED Requirements

### Requirement: Task-oriented top-level HELP
The system SHALL explain what aiw-flow does, show the common Session workflow, summarize every top-level command, provide quick-start examples, and direct users to command-specific HELP.

#### Scenario: First-time user requests HELP
- **WHEN** a user runs `aiw-flow --help`
- **THEN** the output explains the Session workflow and shows at least one creation example and one interactive example

#### Scenario: User scans available commands
- **WHEN** top-level HELP is displayed
- **THEN** every supported top-level command has a plain-language summary

### Requirement: Actionable command HELP
Every top-level command SHALL provide its purpose, descriptions for all positional and optional arguments, relevant usage constraints, and at least one valid example.

#### Scenario: User requests creation HELP
- **WHEN** a user runs `aiw-flow new --help`
- **THEN** the output explains each required input, optional loop behavior, and shows a complete creation command

#### Scenario: User requests execution HELP
- **WHEN** a user runs `aiw-flow run --help`
- **THEN** the output explains phase, prompt sources, timeout, thread replacement risk, and valid examples

### Requirement: Discoverable nested HELP
The `memory`, `handoff`, and `daemon` command groups SHALL summarize their child actions, and every child action SHALL provide argument descriptions and examples.

#### Scenario: User explores Memory commands
- **WHEN** a user runs `aiw-flow memory --help`
- **THEN** the output explains `show`, `append`, and `replace` and how to request their detailed HELP

#### Scenario: User explores one nested action
- **WHEN** a user runs `aiw-flow handoff create --help`
- **THEN** the output explains the Session ID and focus arguments and shows a valid example

### Requirement: HELP terminology and layout
The system SHALL use Easy English, consistent Session terminology, meaningful metavariables, and preserved multiline formatting for workflows and examples.

#### Scenario: HELP renders examples
- **WHEN** argparse renders any command HELP
- **THEN** multiline examples remain readable and use the actual command and option names

### Requirement: Parser compatibility
HELP improvements MUST NOT change command names, option names, required arguments, defaults, parsed destination names, or dispatch behavior.

#### Scenario: Existing command invocation
- **WHEN** an existing valid aiw-flow argument list is parsed
- **THEN** it produces the same command and option values as before the HELP change

#### Scenario: Existing invalid invocation
- **WHEN** a required argument is omitted
- **THEN** argparse rejects the invocation as before

<!-- archived spec: session-handoff -->

## ADDED Requirements

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

<!-- archived spec: workspace-context -->

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

<!-- archived spec: session-skill-invocation -->

## ADDED Requirements

### Requirement: Discover Skills from standard locations
The system SHALL discover Skill directories from project and user `.agents/skills` locations and from compatible project and user `.codex/skills` locations without requiring configurable search paths.

#### Scenario: Discover project Skills
- **WHEN** a Session workspace or its repository ancestry contains a valid Skill under `.agents/skills`
- **THEN** the system includes that Skill in the available project Skills

#### Scenario: Discover compatible project Skills
- **WHEN** the Session project root contains a valid Skill under `.codex/skills`
- **THEN** the system includes that Skill in the available project Skills

#### Scenario: Discover user Skills
- **WHEN** the user home contains a valid Skill under `.agents/skills` or the effective Codex home contains a valid Skill under `skills`
- **THEN** the system includes that Skill in the available user Skills

#### Scenario: Resolve a non-Git workspace
- **WHEN** the Session workspace is not inside a Git repository
- **THEN** the system treats the workspace as the project root when resolving project Skill locations

### Requirement: List Skills without executing a turn
The interactive loop SHALL process `/skills` locally and display discovered Skills with their scope and source without calling Codex.

#### Scenario: List valid Skills
- **WHEN** the user enters `/skills` and valid project or user Skills are discoverable
- **THEN** the system displays their names, descriptions, scopes, and source paths without incrementing the Session turn number

#### Scenario: List an empty catalog
- **WHEN** the user enters `/skills` and no valid Skills are discoverable
- **THEN** the system reports that no Skills were found and waits for another input without calling Codex

#### Scenario: Report malformed candidates
- **WHEN** the user enters `/skills` and a candidate has missing or invalid required frontmatter
- **THEN** the system reports a warning containing the candidate path while continuing to display other valid Skills

### Requirement: Invoke a discovered Skill
The interactive loop SHALL accept `/skill <name> <message>` and execute a valid request through the normal turn path using Codex's native `$<name>` invocation syntax.

#### Scenario: Invoke a project Skill
- **WHEN** the user enters `/skill metrics-review Review the revenue metrics` and exactly one discovered Skill is named `metrics-review`
- **THEN** the system executes `$metrics-review Review the revenue metrics` as one normal Session turn

#### Scenario: Preserve normal turn behavior
- **WHEN** a valid `/skill` request executes
- **THEN** the system persists its Prompt, Output, Event, status, timeout result, and Thread changes exactly as an ordinary loop message

#### Scenario: Preserve direct invocation
- **WHEN** the user enters an ordinary message beginning with `$metrics-review`
- **THEN** the system sends that message unchanged through the normal turn path

### Requirement: Reject invalid Skill commands safely
The interactive loop SHALL reject malformed, missing, or ambiguous Skill invocations without calling Codex.

#### Scenario: Missing Skill message
- **WHEN** the user enters `/skill metrics-review` without a task message
- **THEN** the system displays the required syntax and waits for another input without incrementing the turn number

#### Scenario: Unknown Skill name
- **WHEN** the user invokes a name that is not present in the current discovery result
- **THEN** the system reports that the Skill was not found and waits for another input without calling Codex

#### Scenario: Duplicate Skill name
- **WHEN** more than one discovered Skill declares the requested name
- **THEN** the system reports every conflicting source path and refuses to execute the turn

#### Scenario: Skill changes before invocation
- **WHEN** the discovery contents change after an earlier `/skills` command and before `/skill` is entered
- **THEN** the system uses a fresh discovery result to validate the invocation

### Requirement: Preserve existing loop command behavior
The new Skill commands MUST coexist with all existing loop messages, escapes, and local commands without changing their behavior.

#### Scenario: Escape a Skill-like slash message
- **WHEN** the user enters `//skill metrics-review Review this text`
- **THEN** the system sends `/skill metrics-review Review this text` as an ordinary message instead of processing a local Skill command

#### Scenario: Use an existing command
- **WHEN** the user enters an existing local command such as `/status`
- **THEN** the system processes that command with its existing behavior

<!-- archived spec: workflow-backend-routing -->

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

<!-- archived spec: session -->

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

<!-- archived spec: task-agent-handoff -->

# task-agent-handoff Specification

## Purpose

Define a sequential Task handoff that starts a fresh Codex Thread while
preserving the Task's worktree and aiw-flow Session.

## ADDED Requirements

### Requirement: Start the next agent Thread for a Task
The system SHALL provide `aiw task agent next TASK_ID` for an existing Task
with a valid worktree and aiw-flow Session binding.

#### Scenario: Start a child Thread
- **WHEN** the Task, worktree, and Session are valid and no execution lease is
  held
- **THEN** the system creates or refreshes the persistent handoff, starts one
  fresh Codex Thread in the Task worktree, and records the child Thread ID

#### Scenario: Reject an unresolved Task
- **WHEN** the Task does not exist, has no valid worktree, or has no valid
  Session binding
- **THEN** the command exits before invoking Codex and reports the missing
  binding

### Requirement: Preserve context in the child Thread
The child Thread's initial prompt SHALL include the Task identifier and goal,
Session identifier and phase, Session Memory, and the Session-relative handoff
artifact reference.

#### Scenario: Continue from a handoff
- **WHEN** a child Thread is started successfully
- **THEN** its initial prompt tells the agent to read the handoff and referenced
  artifacts before taking action

### Requirement: Record handoff lineage
The system SHALL persist parent Thread ID, child Thread ID, Task ID, Session
ID, handoff path and content hash, and transition timestamps.

#### Scenario: Inspect lineage after transition
- **WHEN** the child Thread has started
- **THEN** status and diagnostic output show the parent-to-child transition and
  the exact handoff consumed

### Requirement: Prevent concurrent writers
The system SHALL refuse a second `task agent next` transition while the same
Task worktree has an active execution lease.

#### Scenario: Lease conflict
- **WHEN** another agent currently holds the Task execution lease
- **THEN** the command exits with a conflict and does not create a new Thread

### Requirement: Keep the transition atomic on failure
The system SHALL create the handoff and lineage intent before launching Codex
and SHALL mark the transition failed if process startup fails.

#### Scenario: Codex startup failure
- **WHEN** handoff creation succeeds but the Codex process cannot start
- **THEN** no successful child Thread is recorded, the failure is persisted,
  and the execution lease is released

### Requirement: Preserve existing workflows
The new command SHALL NOT change the behavior of one-shot runs, interactive
loops, or same-Thread continuation.

#### Scenario: Existing continuation
- **WHEN** a user runs the existing `aiw-flow continue SESSION_ID`
- **THEN** it resumes the current Thread without creating a task-agent
  handoff transition

<!-- archived spec: file-operations -->

# File Operations Specification

### Requirement: Detect supported text encodings

The system SHALL detect UTF-8, UTF-16, GB18030, and Windows-31J and SHALL expose encoding and confidence metadata.

#### Scenario: Deterministic encoding

- **WHEN** a file has a supported BOM or valid UTF-8 content
- **THEN** the system reports the matching encoding with high confidence

#### Scenario: Ambiguous legacy encoding

- **WHEN** bytes can be decoded as more than one legacy encoding
- **THEN** the system reports ambiguity and requires an explicit encoding for writes

### Requirement: Preserve file format

The system SHALL preserve an existing file's BOM and newline style by default when writing text.

#### Scenario: Preserve an existing file

- **WHEN** an AI writes a detected text file without overriding format options
- **THEN** the system writes using the existing encoding, BOM, and newline style

### Requirement: Atomic text writes

The system SHALL write through a temporary file and atomically replace the destination only after encoding succeeds.

#### Scenario: Encoding failure

- **WHEN** content cannot be represented in the selected encoding
- **THEN** the original file remains unchanged and the command returns a non-zero status

### Requirement: Skill integration

AI Skills SHALL use aiw file read/info/write for text file content access and SHALL use aiw patch for generated code patches.

#### Scenario: Skill reads or modifies text

- **WHEN** a Skill needs project file content
- **THEN** it uses the shared file tools unless the tool is unavailable or the file is binary

<!-- archived spec: ai-support -->

# Patch Application Specification

## Purpose

Provide reliable, Git-backed patch application across Windows terminal encodings.

### Requirement: Normalize patch input

The system SHALL accept patch input from a file or standard input and SHALL normalize supported UTF-8 and UTF-16 input before invoking Git.

#### Scenario: UTF-8 patch file

- **WHEN** a user runs aiw patch check patch.diff with a UTF-8 patch
- **THEN** the system invokes Git with equivalent UTF-8 patch text and does not modify the worktree

#### Scenario: UTF-8 BOM or UTF-16 patch

- **WHEN** a user supplies a patch encoded as UTF-8 with BOM or UTF-16
- **THEN** the system removes transport-specific markers as needed and Git receives valid patch text

### Requirement: Delegate validation and application to Git

The system SHALL use git apply --check for check and preflight validation and SHALL use git apply for normal application.

#### Scenario: Apply a valid patch

- **WHEN** a user runs aiw patch apply patch.diff
- **THEN** the system validates the patch with Git and applies it only after validation succeeds

#### Scenario: Reverse a valid patch

- **WHEN** a user runs aiw patch reverse patch.diff
- **THEN** the system applies the patch using Git reverse mode

### Requirement: Protect state-changing options

The system SHALL NOT enable index modification or three-way application by default.

#### Scenario: Explicit index update

- **WHEN** a user supplies --index
- **THEN** the system passes the index option to Git

#### Scenario: Explicit three-way application

- **WHEN** a user supplies --3way
- **THEN** the system passes the three-way option to Git

### Requirement: Report failures safely

The system SHALL preserve Git failure status, show actionable diagnostics, and remove temporary normalized patch files after completion.

#### Scenario: Invalid patch

- **WHEN** Git rejects a patch
- **THEN** the system returns a non-zero status and reports Git diagnostics

#### Scenario: Missing Git

- **WHEN** Git is unavailable on PATH
- **THEN** the system reports that Git is required and returns a non-zero status
### Requirement: AI patch integration

The system SHALL expose the patch adapter as the default application path for AI-generated code patches.

#### Scenario: AI applies a generated patch

- **WHEN** an AI coding workflow produces a patch for repository changes
- **THEN** it invokes the patch adapter, which normalizes input, runs Git preflight validation, and applies the patch through Git

#### Scenario: AI receives an application failure

- **WHEN** Git rejects an AI-generated patch
- **THEN** the adapter returns a structured failure with the Git exit status and diagnostics, and the AI workflow SHALL NOT report the change as applied

#### Scenario: AI receives a successful application

- **WHEN** Git applies an AI-generated patch successfully
- **THEN** the adapter returns a structured success result with the applied operation and affected paths when available
### Requirement: Convert AI patch syntax

The system SHALL recognize the AI patch envelope and convert supported operations into a standard unified diff before invoking Git.

#### Scenario: Convert an update operation

- **WHEN** an input contains a Begin Patch envelope with an Update File operation
- **THEN** the system produces a standard unified diff for that file and passes the converted patch to Git

#### Scenario: Convert an add or delete operation

- **WHEN** an input contains an Add File or Delete File operation
- **THEN** the system produces a standard unified diff representing the file creation or deletion

#### Scenario: Convert a move operation

- **WHEN** an input contains a Move to File operation with a supported source and target
- **THEN** the system produces a standard Git rename-style patch or an equivalent delete-and-add patch

#### Scenario: Conversion failure

- **WHEN** the AI patch syntax is malformed or contains an unsupported operation
- **THEN** the system returns a conversion error naming the operation and path, and SHALL NOT invoke Git apply
