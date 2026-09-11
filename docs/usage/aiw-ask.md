# AIW Ask

`aiw ask` provides read-only AIW usage guidance. It does not change project
files, Git state, Tasks, or OpenSpec artifacts.

## Usage

```text
aiw ask "How do I start a Task?"
aiw ask --system-prompt "Answer for a Go service team." "How do I use a handoff?"
aiw ask --system-prompt-file .aiw/ask-prompt.txt "Explain aiw wt status."
aiw ask --allow-path C:\docs "What should I check in this guide?"
aiw ask --chat
aiw ask --resume
```

The command accepts these options:

- `--chat` starts an interactive, multi-turn session.
- `--resume` continues the most recently updated Ask session. If no session is
  available, it starts a new Chat session and says that nothing was resumed.
- `--system-prompt TEXT` provides a system-prompt override.
- `--system-prompt-file FILE` reads a system-prompt override from `FILE`.
- `--allow-path PATH` can be repeated to authorize additional directories for a
  filesystem-aware provider.

For system prompts, precedence is command-line text, command-line file,
`$HOME/.aiw/ask/config.toml` `[ask].system_prompt`, then
`[ask].system_prompt_file`. If none is set, Ask uses no override. A
command-line prompt file is explicitly authorized. A configured prompt file
outside the current workspace or `$HOME/.aiw/ask/` must be covered by
`--allow-path`; noninteractive usage rejects it otherwise.

## Response format

Ask requests one JSON response with `schema_version: "1.0"`. Every valid
response contains `schema_version`, `status`, `summary`, `capability`, and
`safety`. Status is one of `ok`, `needs_clarification`, `unsupported`,
`error`, or `interrupted`.

An `ok` response can also include ordered steps, alternatives, warnings,
references, and follow-up questions. An `unsupported` response explains the
gap and offers alternatives when available. If the provider returns malformed
JSON or an invalid schema, Ask reports `error`; it does not make a retry and it
does not save an answer payload.

## Chat controls

In Chat, enter one line at a time. Empty lines remain part of the pending
question. Use `/send` to submit the accumulated lines and `/exit` to finish the
session. `Ctrl+C` cancels only the current line; it does not corrupt the
session.

## Privacy and local memory

General guidance does not read or attach workspace files. The current directory
is an authorization ceiling, not automatic context. Use `--allow-path` only
when you intentionally authorize an additional directory. In Chat, Ask requests
one confirmation before reading a configured prompt file outside the workspace,
private Ask directory, and allowed paths.

Ask stores private session Markdown files outside the repository at
`$HOME/.aiw/ask/<YYYY-MM-DD>/`. Successful turns contain the question, rendered
answer, and canonical JSON. Failed and interrupted turns keep only the
question, timestamp, status, and error message. These files are not added to
Git.

## Provider limitations

For Codex CLI, Ask uses the current directory, a read-only sandbox, no approval
requests, and one `--add-dir` value for each explicitly authorized path. Codex
`--add-dir` grants an additional directory; it is not a strict read allowlist,
and exact sandbox behavior depends on the installed Codex version.

For Copilot CLI, Ask passes each explicitly authorized path as `--add-dir`.
Ask remains read-only and does not enable broad tool permissions.
