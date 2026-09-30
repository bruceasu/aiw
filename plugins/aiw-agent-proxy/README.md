# AIW Agent Proxy

The TypeScript service binds only to `127.0.0.1`. Start it from this directory
after installing its package dependencies and building:

```powershell
npm install
npm run build
node ./aiw-agent-proxy.js start --port 43127
```

`AIW_AGENT_PROXY_PORT` sets the default port. `AIW_AGENT_PROXY_STATE_DIR` may
set a private local temporary state directory; it stores ACK-pending results
and `audit.jsonl`. Results expire one hour after creation. The service removes
each result file after ACK. Do not put the state directory in a synced or
shared folder. Stop it with Ctrl+C or SIGTERM.

## HTTP

`POST /v1/requests` accepts JSON:

```json
{
  "client_id": "editor-1",
  "request_id": "optional-id",
  "provider": "openai",
  "model": "gpt-5",
  "effort": "medium",
  "output_format": "json",
  "prompt": "Summarize the requested change."
}
```

`provider` is `codex`, `copilot`, or `openai`; `output_format` is `json` or
`markdown`. Provider/model combinations and unsupported effort values are
rejected by the provider. OpenAI reads `OPENAI_API_KEY` and optionally
`OPENAI_BASE_URL` from the service environment. Cost is reported as `null`
unless a Provider supplies a documented monetary amount; token totals are
reported when supplied by the Provider.

## WebSocket

Connect to `ws://127.0.0.1:43127/v1/events?client_id=<self-asserted-id>`.
Send one request event at a time:

```json
{"type":"request","request":{"client_id":"editor-1","provider":"openai","model":"gpt-5","output_format":"markdown","prompt":"Write a short summary."}}
```

The service sends a `result` event containing `event_id`, `request_id`, result
text and usage. Send `{"type":"ack","event_id":"<event_id>"}` after
receiving it. If the connection drops before ACK, reconnect using the same
`client_id`; pending result events are sent again. Duplicate ACKs are safe.

The client ID is only a self-asserted routing label. It is not authentication
or an authorization boundary; another local process can claim the same value.
The service does not expose CORS access and rejects non-loopback network peers.

## Provider safety

OpenAI uses its HTTPS API. Copilot stays disabled until an isolated negative
runtime check proves that its deny-all tool configuration cannot execute Shell
or access files. Codex stays disabled because the SDK available during design
exposes read-only sandboxing but no tool allowlist; read-only alone does not
meet the no-Shell rule. Disabled Providers fail closed.

Audit JSONL records request/event IDs, self-asserted client ID, Provider/model,
status, duration and available usage metadata. It does not record the prompt,
result body, API key or complete environment. The result spool is separate
from the audit log and temporarily contains the result until ACK or expiry.
