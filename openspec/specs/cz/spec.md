# CZ specification

## Requirements

### Requirement: CZ owns its LLM configuration

`cz` MUST resolve its LLM settings from `[cz]`, provider-specific `[cz.copilot]`,
`[cz.codex]`, and `[cz.openai]`, plus explicit CZ CLI and environment overrides.
It MUST NOT inherit the global `[ai]` provider, model, or fast profile.
At each configuration location, `cz.toml` MUST take precedence over `.cz.toml`,
and `aiw.toml` MUST take precedence over `.aiw.toml`; the pair MUST NOT be
merged when both files exist.

#### Scenario: independent provider models

- **WHEN** each CZ provider has its own model configured
- **THEN** each provider invocation uses only the selected provider's model and credentials

#### Scenario: CLI override

- **WHEN** `--provider` or `--model` is given
- **THEN** it overrides the corresponding CZ configuration for that invocation

### Requirement: provider fallback order

Without an explicit provider, `cz --llm` MUST use Codex CLI, then Copilot CLI,
then the OpenAI Responses HTTP API. A missing executable, unsupported mode,
non-zero exit, timeout, HTTP error, or invalid candidate MUST advance to the
next provider. After all providers fail, `cz` MUST explain the failure briefly
and enter the interactive flow. Copilot MUST only be invoked when
`CZ_COPILOT_COMMAND` explicitly names an installed CLI. An unconfigured Copilot
MUST be skipped without an installation prompt. CLI probes and invocations
MUST NOT read interactive input. Codex MUST send the prompt through stdin with
`exec --sandbox read-only --json -` and decode the final assistant message from
the last-message file or JSONL events. Copilot MUST request text output with
`--prompt`; both adapters MUST pass the selected provider model when configured.
Arbitrary `CZ_CODEX_ARGS` and `CZ_COPILOT_ARGS` MUST NOT replace the enforced
read-only and structured-output invocation.

#### Scenario: all providers fail

- **WHEN** no provider returns a valid candidate
- **THEN** CZ opens its manual commit message wizard

#### Scenario: explicit provider fails

- **WHEN** `--provider` selects one SDK and that SDK fails
- **THEN** CZ opens the manual wizard without calling another SDK

### Requirement: structured candidate contract

Each provider adapter MUST normalize its output to a JSON object containing a
`candidates` array. Each candidate MUST contain string fields `type`, `scope`,
`subject`, `body`, `breaking`, and `footer`. The shared validator MUST reject
unknown types, empty subjects, overlong subjects, and unrelated issue references.

### Requirement: CZ remains an independent Python plugin

`aiw cz` MUST run the Python implementation through the plugin entry point.
Its runtime requires a target-machine Python interpreter and uses only the
standard library by default. The Go AI provider workflow MUST NOT be changed as
a side effect of CZ provider selection.

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
