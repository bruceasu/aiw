## ADDED Requirements

### Requirement: Provide structured AIW guidance

The system MUST provide `aiw ask "<prompt>"` as a read-only guidance command
that reuses the core LLM and renders a validated structured response.

#### Scenario: Answer a supported guidance question

- **WHEN** a user invokes `aiw ask "<prompt>"` with a nonempty prompt
- **THEN** the system MUST request JSON matching schema version `1.0`
- **AND** render a friendly answer from the validated JSON
- **AND** it MUST NOT modify project files, Git state, Tasks, or OpenSpec data.

#### Scenario: Report unsupported guidance

- **WHEN** the requested work is unsupported by AIW
- **THEN** the result status MUST be `unsupported`
- **AND** the answer MUST state the limitation and include viable alternatives
  when available.

#### Scenario: Reject malformed structured output

- **WHEN** the LLM output is not valid for the answer schema
- **THEN** the result status MUST be `error`
- **AND** the system MUST NOT perform an automatic second LLM request
- **AND** it MUST NOT persist an answer payload.

### Requirement: Preserve private ask sessions

The system MUST store ask sessions outside the project and outside Git.

#### Scenario: Store a successful one-shot turn

- **WHEN** a one-shot ask response is valid
- **THEN** the system MUST create a Markdown file below
  `$HOME/.aiw/ask/<YYYY-MM-DD>/`
- **AND** its name MUST use a Windows-safe UTC datetime and the SHA-256 of the
  initial prompt
- **AND** it MUST include session metadata, question, rendered answer, and
  canonical JSON data.

#### Scenario: Store a failed or interrupted turn

- **WHEN** an LLM turn fails or is interrupted
- **THEN** the system MUST persist its timestamp, question, and status/error
- **AND** it MUST NOT persist an answer or canonical answer-data payload.

### Requirement: Support lightweight chat and resume

The system MUST support simple multi-turn Chat without complex editing or tool
calling.

#### Scenario: Chat input and exit

- **WHEN** a user invokes `aiw ask --chat`
- **THEN** the system MUST use promptui-assisted line input
- **AND** `/send` MUST submit accumulated multiline input
- **AND** `/exit` MUST end the Chat session
- **AND** `Ctrl+C` MUST cancel the current input without corrupting the session.

#### Scenario: Resume latest session

- **WHEN** a user invokes `aiw ask --resume` and a previous session exists
- **THEN** the system MUST append subsequent turns to the most recently updated
  session.

#### Scenario: Resume without a session

- **WHEN** a user invokes `aiw ask --resume` and no session exists
- **THEN** the system MUST start a new Chat session and explain that no session
  was available to resume.

### Requirement: Enforce conservative read boundaries

The system MUST keep general ask guidance independent of project-file access
and require explicit authorization for paths outside the current workspace.

#### Scenario: General guidance without file context

- **WHEN** the question does not require project-file content
- **THEN** the system MUST NOT read or attach workspace files to the LLM prompt.

#### Scenario: External path without authorization

- **WHEN** a requested context path is outside the current workspace and is not
  allowed by `--allow-path`
- **THEN** noninteractive execution MUST reject the read before it occurs
- **AND** Chat MUST request one-time confirmation before the first such read.

#### Scenario: Pass authorized directories to CLI providers

- **WHEN** a filesystem-aware ask turn uses Codex CLI
- **THEN** the system MUST pass `--cd <cwd>`, `--sandbox read-only`,
  `--ask-for-approval never`, and repeated `--add-dir` values for explicitly
  authorized paths
- **AND** it MUST report that Codex `--add-dir` is not guaranteed to be a
  strict read allowlist.

#### Scenario: Use Copilot allowed paths

- **WHEN** a filesystem-aware ask turn uses Copilot CLI
- **THEN** the system MUST pass repeated `--add-dir` values only for explicitly
  authorized paths
- **AND** it MUST keep `ask` tool permissions read-only.
