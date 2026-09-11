## Context

`aiw ask` is a core, read-only guidance command. It answers how to use AIW,
persists private memory, and must be safe to run in an arbitrary project. The
repository already exposes `ai.RunLLMWithSystemPrompt`, which accepts an output
schema and a system prompt; this command reuses that path rather than creating
a provider abstraction.

## Goals / Non-Goals

**Goals:**

- Provide one-shot and lightweight multi-turn guidance.
- Produce a stable JSON response that can be rendered for humans and analyzed
  later from Markdown files.
- Preserve private, append-only local session history.
- Ensure the command never mutates the current project or AIW Task state.

**Non-Goals:**

- Automatic question analysis, clustering, plugin generation, or tool calls.
- Complex text editing in Chat.
- A new provider abstraction or LLM rearchitecture.
- Claiming sandboxed filesystem access for providers that cannot enforce it.

## Decisions

### Command and session behavior

- `aiw ask "<prompt>"` performs one turn and creates a new session file.
- `aiw ask --chat` starts a new Chat session. `aiw ask --resume` resumes the
  most recently updated session; if none exists, it starts a new one.
- Chat uses `promptui` line prompts: `/send` submits accumulated lines,
  `/exit` exits, and `Ctrl+C` cancels the current input. Empty lines are kept
  in the pending message. `Ctrl+D` is not required on Windows.
- A session file is named using the UTC-safe timestamp
  `YYYYMMDDTHHMMSSZ-<sha256(initial-prompt)>.md`, avoiding `:` in Windows file
  names. It can accumulate turns across date directories.

### Structured answer contract

- The canonical LLM result is JSON with `schema_version: "1.0"` and required
  `status`, `summary`, `capability`, and `safety` fields.
- Valid statuses are `ok`, `needs_clarification`, `unsupported`, `error`, and
  `interrupted`.
- `ok` may include ordered `steps`, alternatives, warnings, references, and
  follow-up questions. Each step declares command, working directory,
  read-only expectation, and expected result.
- `unsupported` MUST explain the gap and offer alternatives when possible.
- The command parses and validates JSON once. Invalid JSON/schema is an
  `error`: it does not make a second LLM call and does not persist an answer.
  This keeps costs and behavior predictable.
- Terminal output is rendered from validated JSON. Markdown persists the user
  question, rendered answer, and a fenced canonical JSON block.

### System prompt resolution

- Precedence is `--system-prompt` text, then `--system-prompt-file`, then
  `$HOME/.aiw/ask/config.toml` `[ask] system_prompt` or
  `system_prompt_file`, then an empty prompt.
- A system-prompt file argument is explicit authorization to read that file.
- A configured prompt-file path outside `$HOME/.aiw/ask/` or the current
  workspace requires `--allow-path`; otherwise it is rejected without reading.

### Filesystem and provider boundary

- `ask` itself writes only its private memory under `$HOME/.aiw/ask/`; it does
  not write project files, Git state, Tasks, or OpenSpec artifacts.
- Current-directory access is an authorization ceiling, not implicit context:
  a general AIW usage question supplies no project-file content.
- `--allow-path <path>` is repeatable and authorizes additional read roots.
  Noninteractive execution rejects unapproved external paths. Chat asks once
  using `promptui` before the first external-path read.
- For Codex CLI, AIW may pass `--cd <cwd>`, `--sandbox read-only`,
  `--ask-for-approval never`, and one `--add-dir <path>` per explicitly
  authorized path. `--add-dir` is documented as an additional writable
  directory and MUST NOT be presented as a strict read allowlist; the active
  Codex version's sandbox semantics must be reported as a limitation.
- For Copilot CLI, AIW may pass repeated `--add-dir <path>` values because its
  documented meaning is an allowed-paths list. Tool permissions remain read-only
  for `ask`.

### Persistence and failures

- Files live at `$HOME/.aiw/ask/<YYYY-MM-DD>/` and are never added to Git.
- A session begins with `# session` metadata. Each turn uses `# <datetime>`,
  `## question`, `## answer`, and `## data` (canonical JSON).
- Failed or interrupted turns persist the question, timestamp, and status/error
  message only; they have no `## answer` or `## data` answer payload.

## Design Readiness

- Status: FD_APPLIED
- FD_APPLIED: LLM entry point, Chat protocol, persistence format, JSON contract,
  prompt precedence, and failure behavior are now decided.
- %% DEFERRED: Verify Codex version-specific read-scope behavior and document
  the limitation. Do not substitute a system prompt for enforceable
  authorization.

## Risks / Trade-offs

- Refusing malformed JSON instead of auto-repairing is predictable but can make
  a transient model formatting error visible to users.
- No automatic workspace context is safer and cheaper, but users must make
  file relevance explicit in a later, authorization-aware extension.
- Codex CLI integration is constrained because `--add-dir` is not a proven
  read-scope allowlist; AIW must surface that limitation.
