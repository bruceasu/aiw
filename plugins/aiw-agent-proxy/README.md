# AIW Agent Proxy

The TypeScript service binds to `127.0.0.1` by default. Start it from this directory
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

For direct access from another machine, set `AIW_AGENT_PROXY_HOST` to a
specific local IPv4 address or `0.0.0.0` before starting the service. Remote
HTTP clients must use an IPv4 address in the URL; the WebSocket endpoint stays
loopback-only. For example, on the service machine:

```powershell
$env:AIW_AGENT_PROXY_HOST = "0.0.0.0"
node ./aiw-agent-proxy.js start --port 43127
```

Then use `ai --url http://<service-ip>:43127 "Question"` on the client
machine. No authentication is applied to remote HTTP requests. Anyone able to
reach that port can submit Provider requests and consume the service's quota;
restrict access with a host firewall or a trusted private network. The
service itself serves plain HTTP, so use an HTTPS terminating proxy when
transport encryption is needed. Leave the host variable unset for local-only
operation.

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

Optional `system_prompt` sends a separate OpenAI Responses `instructions`
value; Codex and Copilot requests with this field fail explicitly. Optional
`include_reasoning_summary: true` asks OpenAI for a published reasoning
summary and adds `reasoning_summary` to the result (`null` when unavailable).
The summary is never written to the audit JSONL. Existing requests need
neither field. The separate [Agent Proxy client](../aiw-agent-proxy-client/README.md)
offers the one-shot `ai` command for this endpoint.

For OpenAI Responses, `usage.openai_usage` also returns the response ID, actual
model, service tier, response status, total tokens, cached input tokens,
cache-write input tokens, and reasoning output tokens. The same object is
written to the audit record. Missing values are `null`; `cost` stays `null`
because Responses does not supply a monetary amount. To calculate a charge,
use the model and tier's prices for uncached input, cached input, cache writes,
and output at the time of the request. Reasoning tokens are already included
in `output_tokens`; do not add them a second time. Pricing can also depend on
context length, processing mode, and tools, so reconcile estimates with the
provider's billing records.

## Standalone OpenAI API check

From this directory, run `node ./scripts/check-openai.mjs` after setting
`OPENAI_API_KEY` in the process environment. The script needs only Node.js;
the Agent Proxy service does not need to be started. It makes one short request
to the official Responses endpoint and prints only the HTTP status, whether
text was returned, and available token counts. The default model is
`gpt-6-luna`; set `OPENAI_MODEL` to use another model available to your key.
It does not print the key, response body, or API error body.

In PowerShell, enter the key without putting it in command history:

```powershell
$secret = Read-Host "OpenAI API key" -AsSecureString
try {
  $env:OPENAI_API_KEY = [System.Net.NetworkCredential]::new("", $secret).Password
  node ./scripts/check-openai.mjs
} finally {
  Remove-Item Env:OPENAI_API_KEY -ErrorAction SilentlyContinue
}
```

To check the OpenAI path through the built Agent Proxy, run
`node ./scripts/check-openai-proxy.mjs` with the same environment variable.
This script starts a temporary loopback service, makes one real request, checks
the audit record, and removes its temporary state. It prints no key, prompt,
response body, or audit record content. It prints the allowed OpenAI usage
metadata and whether it matches the audit. Run it yourself when ready; it is
not part of the automated build.

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

OpenAI uses its HTTPS API. Copilot invokes the CLI with an empty
`--available-tools=` list, explicit tool denials, disabled built-in MCP servers,
and no interactive questions. On Windows it prefers the npm Copilot entrypoint
on PATH; `AIW_AGENT_PROXY_COPILOT_EXECUTABLE` can select an existing executable.
Its CLI-declared tool restrictions are used without a separate negative runtime
check. Codex is invoked with `features.shell_tool=false`,
read-only sandboxing, no approval requests, disabled hooks/apps, and instructions
to use no tools or local files. `AIW_AGENT_PROXY_CODEX_EXECUTABLE` may name an
existing Codex executable; otherwise on Windows the service prefers the npm
Codex entrypoint found on PATH and falls back to `codex.exe`. Other platforms
use `codex`. If it is unavailable, the request fails with
`provider_not_configured`. The Codex restrictions reduce access but cannot prove
that no sensitive file is read; this residual risk is accepted in FD-006.

Audit JSONL records request/event IDs, self-asserted client ID, Provider/model,
status, duration and available usage metadata. It does not record the prompt,
result body, API key or complete environment. The result spool is separate
from the audit log and temporarily contains the result until ACK or expiry.
