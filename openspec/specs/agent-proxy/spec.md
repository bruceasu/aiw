# Agent Proxy Specification

## Purpose

Define the local-only Agent Proxy contract for one-shot model requests.

## Requirements

### Requirement: Loopback-only service

The Agent Proxy MUST bind only to `127.0.0.1`. It MUST reject requests from
non-loopback peers and MUST NOT offer a configurable remote bind address.

#### Scenario: Remote request is rejected

- **WHEN** a request reaches the service from a non-loopback peer
- **THEN** the service rejects it without calling a Provider

### Requirement: Synchronous HTTP request

The service MUST expose `POST /v1/requests` with a single request containing a
self-asserted `client_id`, optional `request_id`, `provider`, `model`, optional
`effort`, `output_format`, and `prompt`. It MUST return one result or a
structured error. It MUST validate bounds and reject unsupported parameters.

#### Scenario: Valid request

- **WHEN** a valid request uses an enabled Provider and supported model options
- **THEN** the service returns the Provider result and available usage metadata

#### Scenario: Disabled or unsupported Provider

- **WHEN** a Provider cannot enforce the required no-shell and no-file-access
  policy, or does not support a requested option
- **THEN** the service fails closed with a structured error

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

