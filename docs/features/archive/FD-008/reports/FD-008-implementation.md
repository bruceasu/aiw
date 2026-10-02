# FD-008 Worker Implementation Report

- FD: `FD-008`
- Source handoff: `FD-008-000005-decision-recorded`
- Reviewed plan: `docs/features/FD-008_AI_AGENT_PROXY_CLIENT.md`, Revision 5
- Workspace: primary workspace

## Implemented

- Added the independent Node.js package `plugins/aiw-agent-proxy-client/` with
  an `ai` bin entry. It parses `--system`, `--json`, `--verbose`, one positional
  question, explicit `-` stdin, and one interactive terminal line. Blank
  input exits without a request. `--system @<path>` reads a bounded UTF-8
  file relative to the current directory before any network request.
- The client sends one bounded request to the loopback Agent Proxy, uses
  environment variables for local port, Provider, and model, prints the
  answer or JSON value on stdout, and prints verbose reported metadata on
  stderr. It does not call Provider APIs directly, persist prompts, calculate
  prices, or retry a request.
- Extended Agent Proxy's optional request fields with `system_prompt` and
  `include_reasoning_summary`. OpenAI maps system instructions to Responses
  `instructions` and requests an opt-in published summary. Codex/Copilot
  reject system instructions before invocation. Summary is returned only in
  the response and never copied to metadata-only audit records.
- Added a stable client capability spec and updated the existing Agent Proxy
  spec and package usage documentation. Existing requests without the new
  optional fields retain their previous mapping.

## Static evidence and commands

- Traced argument parsing and all three input branches to the single fetch
  call; blank/EOF, invalid files, invalid options, unsupported Provider
  system instructions, and missing configuration terminate before fetch.
- Traced optional fields through validation, Provider mapping, HTTP result,
  WebSocket completion, and audit writing. Audit still copies usage metadata
  only; it does not read the new `reasoning_summary` property.
- `node --check ai.mjs` passed from the client directory before a bounded
  file-read loop correction. `node node_modules/typescript/bin/tsc --noEmit -p
  tsconfig.json` passed from the Agent Proxy directory. Both no-output checks
  passed again after the final source corrections. A focused `git diff --check`
  on changed tracked files passed. No final artifact was retained.
- No tests, final build, npm link, live service request, real Provider call,
  dependency download, or Git write was run under the repository budget.

## Remaining risks

- The `ai` bin mapping is documented but not installed or invoked through a
  linked shell command in this run. Interactive and HTTP behavior have static
  evidence only; no runtime mock was authorized for FD-008.
- Reasoning summaries depend on model/API support. When absent, verbose
  output says unavailable. Monetary cost remains unknown when the Proxy
  reports `null`; no price estimate is presented as an actual charge.
