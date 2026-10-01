# Agent Proxy client (`ai`)

`ai` sends one question to the running local Agent Proxy. It uses Node.js
22.12 or newer and no npm dependencies. It is separate from `aiw ask`, which
provides AIW usage guidance.

From this directory, run `node ./ai.mjs "Your question"`. To expose the
`ai` executable on your PATH, link this package locally with `npm link`.
Start the Agent Proxy service separately before making requests.

```text
ai "What changed?"
ai --json "Return a JSON object with one answer field"
ai --verbose --system "Answer briefly." "What changed?"
ai --system @instructions.txt "What changed?"
ai --url http://192.0.2.10:43127 "What changed?"
ai -
```

`--system <content>` uses literal text unless it begins with `@`, in which
case the rest is a UTF-8 filename. Relative paths use the current directory;
quote paths that contain spaces. The file is limited to 64 KiB. A missing,
unreadable, empty, or invalid file fails before any HTTP request. OpenAI maps
this value to separate Responses `instructions`; other Providers reject it.

Without a positional question, `ai` prompts for one line on a terminal. A
blank line or EOF exits without a request. `ai -` reads all of stdin to EOF;
an empty stream also exits without a request. Noninteractive use without `-`
fails promptly. There is no chat history or automatic retry.

By default the client calls OpenAI model `gpt-6-luna` through
`127.0.0.1:43127`. Set `AIW_AGENT_PROXY_PORT` for another local proxy port.
`--url <http(s)://host:port>` overrides the address and port for one request.
Use the service origin without `/v1/requests`, credentials, query, or fragment;
the client appends `/v1/requests`. For direct remote access, start Agent Proxy
on the other machine with `AIW_AGENT_PROXY_HOST` set to its IPv4 address or
`0.0.0.0`, and use that machine's IPv4 address in `--url`. The service has no
remote authentication; restrict the listening port to trusted clients. Plain
HTTP does not encrypt prompts or results in transit; use a trusted private
network or an HTTPS terminating proxy where confidentiality is required.
Set `AIW_AI_PROVIDER` to `openai`, `codex`, or `copilot` and set
`AIW_AI_MODEL` for another model. A non-OpenAI Provider requires an explicit
`AIW_AI_MODEL`; the client never reads a Provider API key.

Normal stdout contains only the answer. `--json` requests JSON output and
prints one formatted JSON value to stdout. `--verbose` writes request and
usage details to stderr so stdout remains usable in a pipe. It asks OpenAI
for an optional published reasoning summary; the API does not expose raw
hidden reasoning. If no summary or monetary amount is returned, the client
prints `unavailable` or `unknown`. It does not estimate prices from tokens.

The client makes one bounded HTTP request. A service error, invalid option,
timeout, or unreadable system file exits nonzero without printing the prompt,
file contents, credentials, or a raw response body.
