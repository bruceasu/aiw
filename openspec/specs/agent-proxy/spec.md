# Agent Proxy Specification

## Purpose

Define the Agent Proxy contract for one-shot model requests.

## Requirements

### Requirement: Explicit remote bind

The Agent Proxy MUST bind to `127.0.0.1` by default and reject requests from
non-loopback peers. An explicitly configured `AIW_AGENT_PROXY_HOST` MAY bind
to an IPv4 address or `0.0.0.0` and accept remote HTTP requests addressed to
an IPv4 literal on the configured port. The WebSocket endpoint MUST remain
loopback-only. Remote access has no built-in authentication or TLS.

#### Scenario: Remote request is rejected by default

- **WHEN** the host is unset and a request reaches the service from a
  non-loopback peer
- **THEN** the service rejects it without calling a Provider

#### Scenario: Explicit remote HTTP request

- **WHEN** the host is configured as `0.0.0.0` and a remote HTTP client uses
  the service machine's IPv4 address and configured port
- **THEN** the service accepts the request and applies normal request validation

### Requirement: Synchronous HTTP request

The service MUST expose `POST /v1/requests` with a single request containing a
self-asserted `client_id`, optional `request_id`, `provider`, `model`, optional
`effort`, `output_format`, and `prompt`. It MUST return one result or a
structured error. It MUST validate bounds and reject unsupported parameters.
The request MAY include `system_prompt` and `include_reasoning_summary`.
`system_prompt` MUST remain a separate high-priority instruction where
supported; the service MUST reject it for Providers without a verified
mapping rather than prepend it to `prompt`. `include_reasoning_summary` MAY
return a published summary or `null`, but MUST NOT expose raw hidden reasoning.
Both fields are optional so existing callers remain compatible.

#### Scenario: Valid request

- **WHEN** a valid request uses an enabled Provider and supported model options
- **THEN** the service returns the Provider result and available usage metadata

#### Scenario: OpenAI system instruction and published reasoning summary

- **WHEN** an OpenAI request includes `system_prompt` and requests a reasoning
  summary
- **THEN** the service sends the separate `instructions` field to Responses,
  requests an opt-in summary, and returns a published summary when supplied
- **AND** the audit record excludes both the instruction and summary text

#### Scenario: Unsupported system instruction

- **WHEN** `system_prompt` targets a Provider without a verified high-priority
  mapping
- **THEN** the service rejects the option before invoking that Provider

#### Scenario: Disabled or unsupported Provider

- **WHEN** a Provider cannot enforce the required no-shell and no-file-access
  policy, or does not support a requested option
- **THEN** the service fails closed with a structured error

#### Scenario: Codex request with restricted local access

- **WHEN** a Codex request is received
- **THEN** the service disables the default Shell tool, uses read-only sandboxing
  with no approval requests, and instructs the agent not to use tools or read
  local files; the documented residual file-read risk remains unverified

#### Scenario: Copilot CLI receives an empty tool set

- **WHEN** a Copilot request invokes the CLI with `--available-tools=` and
  explicit tool denials, without granting interactive permission requests
- **THEN** the service may invoke Copilot using those declared restrictions
  without requiring a separate negative runtime check

### Requirement: Acknowledged WebSocket result delivery

The service MUST expose a WebSocket endpoint at `/v1/events`. Each connection
MUST identify itself with a self-asserted `client_id`, may submit one request at
a time, receives a result event with `request_id` and `event_id`, and confirms
delivery with an ACK event. The service MUST retain the full result outside the
audit log until ACK, and MUST replay unacknowledged results to a reconnecting
connection using the same `client_id`.

#### Scenario: Result is acknowledged

- **WHEN** the client ACKs a pending `event_id`
- **THEN** the corresponding result file is deleted immediately and repeated
  ACKs are safe

#### Scenario: Client disconnects before ACK

- **WHEN** the client reconnects with its `client_id` before the one-hour TTL
- **THEN** the service resends each unacknowledged result event

#### Scenario: Result expires

- **WHEN** an unacknowledged result reaches its one-hour TTL
- **THEN** the service removes its result file and does not replay it

### Requirement: Minimal audit data

The service MUST record request/event identifiers, client ID, Provider, model,
status, duration, and usage fields when provided. It MUST NOT write complete
prompts, results, API keys, or complete environment variables to the audit log.
Unknown token or cost values MUST remain unknown; cost MUST NOT be fabricated
from token counts or provider-specific multipliers.

For OpenAI Responses, the service MUST retain the returned response ID, model,
service tier, status, total tokens, cached input tokens, cache-write input
tokens, and reasoning output tokens when available. These are optional usage
metadata; absent fields remain unknown. The service MUST NOT persist the whole
Provider response or calculate a monetary amount from mutable price tables.

#### Scenario: OpenAI reports detailed usage

- **WHEN** an OpenAI response supplies token breakdowns and billing context
- **THEN** the service returns those fields and writes the same allowed fields
  to the audit record, without storing prompt or response text

#### Scenario: Provider usage is unavailable

- **WHEN** the Provider does not report token or monetary usage
- **THEN** the audit record stores `null` for the unavailable values

### Requirement: Client ID is not authentication

The service MUST treat `client_id` only as a self-asserted routing label. It
MUST NOT use that value as proof of identity or authorization.

#### Scenario: Another local process claims an ID

- **WHEN** a local process supplies an existing client's `client_id`
- **THEN** the service treats it as a routing collision, not an authenticated
  identity

