# Agent Proxy CLI client specification

## Purpose

Provide a small one-shot `ai` command for the existing Agent Proxy
without changing the separate `aiw ask` guidance command.

## Requirements

### Requirement: Explicit input selection

The client MUST accept one positional question, `-` for complete stdin input,
or one interactive terminal line when no positional input is supplied. It
MUST NOT call Agent Proxy for blank/EOF input. Without a terminal and without
`-`, it MUST fail promptly and explain how to read stdin.

#### Scenario: Empty interactive question

- **WHEN** the user presses Enter or sends EOF before entering text
- **THEN** the command exits without sending a request

### Requirement: Separate system instruction

`--system <content>` MUST send a separate system instruction. If the value
begins with `@`, the client MUST read the remainder as a local UTF-8 file path
before requesting; relative paths resolve from the current directory.
Missing, unreadable, invalid UTF-8, empty, or oversized files MUST fail before
any request and MUST NOT expose their contents in errors. The client MUST NOT
silently prepend system instructions to the ordinary question.

#### Scenario: System prompt file

- **WHEN** `--system @instructions.txt` is provided
- **THEN** the file contents become `system_prompt` in the one Agent Proxy
  request, or a local error occurs before any request

### Requirement: One bounded request and honest output

The client MUST call only the configured Agent Proxy HTTP endpoint and
MUST NOT call Providers directly, retry a paid request automatically, or
persist the question. `--json` MUST request JSON output and keep stdout a
single JSON value. `--verbose` MUST show only a published reasoning summary
and reported usage/cost; missing summary or cost MUST be labelled unavailable
or unknown, not invented. Diagnostics MUST remain separate from stdout JSON.

`--url` MUST accept an HTTP(S) service origin and append `/v1/requests`.
It MUST reject credentials, a path other than `/`, query, fragment, or an
unsupported scheme before sending a request. When omitted, the client MUST
use the existing loopback address and `AIW_AGENT_PROXY_PORT` behavior.

#### Scenario: URL override

- **WHEN** `--url http://192.0.2.10:43127` is supplied
- **THEN** the client sends one request to `http://192.0.2.10:43127/v1/requests`
  regardless of `AIW_AGENT_PROXY_PORT`

#### Scenario: Proxy omits cost and reasoning summary

- **WHEN** a successful response has token counts but `cost: null` and no
  published reasoning summary
- **THEN** verbose output reports the counts, unknown cost, and unavailable
  summary without calculating a charge or displaying hidden reasoning
