# CZ Python runtime and provider specification

## Requirements

### Requirement: provider fallback order

Without an explicit provider, `cz --llm` MUST use Codex CLI, then Copilot CLI,
then the OpenAI Responses HTTP API. A missing executable, unsupported mode,
non-zero exit, timeout, HTTP error, or invalid candidate MUST advance to the
next provider. After all providers fail, `cz` MUST enter the interactive flow.
Copilot MUST only be invoked when `CZ_COPILOT_COMMAND` explicitly names an
installed CLI. An unconfigured Copilot MUST be skipped without an installation
prompt. CLI probes and invocations MUST NOT read interactive input.
Codex MUST send the prompt through stdin with `exec --sandbox read-only --json -`
and decode the final assistant message from the last-message file or JSONL
events. Copilot MUST request text output with `--prompt`; both adapters MUST
pass the selected provider model when configured.
Arbitrary `CZ_CODEX_ARGS` and `CZ_COPILOT_ARGS` MUST NOT replace the enforced
read-only and structured-output invocation.

### Requirement: structured candidate contract

Each provider adapter MUST normalize its output to a JSON object containing a
`candidates` array. Each candidate MUST contain string fields `type`, `scope`,
`subject`, `body`, `breaking`, and `footer`. The shared validator MUST reject
unknown types, empty subjects, overlong subjects, and unrelated issue references.

### Requirement: TUI interaction

An interactive invocation MUST use the TUI. It MUST NOT require tkinter or any
third-party GUI framework.

### Requirement: locale fallback

An explicit language MUST take precedence. Without one, the plugin MUST use the
operating-system locale and MUST fall back to English when no matching locale
exists.

### Requirement: human review

Every provider and fallback path MUST display the candidate in TUI and offer
edit and cancel before committing. Accepting the preview MUST create a local
Git commit without another prompt. The plugin MUST NOT push.

### Requirement: retry uses the previous commit

`--retry` MUST load the previous commit message as the draft and MUST skip all
LLM provider calls, even when LLM mode is configured.
