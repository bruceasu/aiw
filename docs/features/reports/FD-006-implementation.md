# FD-006 Worker Implementation Report

- FD: `FD-006`
- Source handoff: `FD-006-000003-design-ready`
- Reviewed plan: `docs/features/FD-006_TYPESCRIPT_AGENT_PROXY.md`, Revision 4
- Workspace: primary workspace

## Implemented

- Added the independent TypeScript plugin package at `plugins/aiw-agent-proxy/`.
- Added loopback-only HTTP and WebSocket service code, bounded request/response sizes, concurrency/connection limits, self-asserted client routing, pending-result file storage, ACK deletion, reconnect replay, TTL cleanup, and metadata-only JSONL audit records.
- Added the OpenAI Responses API adapter using HTTPS `fetch`; token counts are retained when supplied, monetary cost remains `null` because the response does not provide a monetary amount.
- Codex calls fail closed because the inspected SDK thread options do not expose a tool allowlist. Copilot calls also remain disabled behind `isolationVerified = false`; the SDK declarations expose empty mode and an explicit empty tool allowlist, but no negative runtime verification was authorized or run.
- Added package usage documentation and the stable `openspec/specs/agent-proxy/spec.md`.
- Corrected the README startup command and prepared `scripts/smoke-local.mjs` for an isolated loopback-only runtime check; the script has not been run.

## Work Item status

- `1.1` pending: provider isolation has not had the required runtime evidence; both CLI Providers stay disabled.
- `1.2` pending: service and OpenAI path are implemented and package dependencies are installed, but the Provider runtime has not been executed.
- `1.3` pending: ACK, replay and expiration are implemented but not runtime-verified.
- `1.4` pending: audit structure and OpenAI token extraction are implemented but the emitted log has not been runtime-checked.
- `1.5` complete: usage documentation and stable specification are present.

## Verification

- Following the user's confirmation, `npm install --ignore-scripts` completed and generated `package-lock.json`. npm reported one high severity dependency advisory, not yet triaged.
- Dependency-backed compile-only command: `node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json` (run from `plugins/aiw-agent-proxy`). The first run failed on two Copilot SDK declaration mismatches; after adjusting effort validation and the event handler, the one permitted rerun passed.
- No tests, final build, service/provider execution, external Provider call, or runtime security probe was run.

## Required follow-up

Runtime checks are still needed for the service, WebSocket ACK/replay/expiry, audit records, and the Copilot deny-all safety gate. The current authorization covered dependency installation and compile-only checking, not runtime validation. Keep Codex and Copilot disabled unless the no-shell/no-file-access gate is verified. Triage the npm high severity advisory before using the package in a less trusted environment.

The next bounded check is `npm run build`, followed by `node scripts/smoke-local.mjs` from `plugins/aiw-agent-proxy`. The latter uses a synthetic OpenAI response and only accesses loopback; it does not contact a real Provider. Both commands await separate authorization.
