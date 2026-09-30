# CZ Python runtime and provider specification

## Requirements

### Requirement: provider fallback order

Without an explicit provider, `cz --llm` MUST use Codex CLI, then Copilot CLI,
then the OpenAI Responses HTTP API. A missing executable, unsupported mode,
non-zero exit, timeout, HTTP error, or invalid candidate MUST advance to the
next provider. After all providers fail, `cz` MUST enter the interactive flow.

### Requirement: structured candidate contract

Each provider adapter MUST normalize its output to a JSON object containing a
`candidates` array. Each candidate MUST contain string fields `type`, `scope`,
`subject`, `body`, `breaking`, and `footer`. The shared validator MUST reject
unknown types, empty subjects, overlong subjects, and unrelated issue references.

### Requirement: GUI and TUI fallback

An interactive invocation MUST use GUI only when a usable tkinter runtime and
display environment are detected. Otherwise it MUST use TUI. GUI failure MUST
fall back to TUI without bypassing human review.

### Requirement: locale fallback

An explicit language MUST take precedence. Without one, the plugin MUST use the
operating-system locale and MUST fall back to English when no matching locale
exists.

### Requirement: human review

No provider or fallback path MAY commit without the interactive human review.
