# FD-009 Worker Implementation and Repair Report

- FD: `FD-009`, Revision 4, `In Progress` at Worker handoff.
- Source handoff: `FD-009-000002-changes-requested`, claimed by
  `codex-root-fd009-repair-20261001` before the repair.
- Reviewed finding: `docs/features/archive/FD-009/reviews/FD-009-review.md`.
- Source context: current uncommitted working tree based on
  `24c848a74fde49a6a6e92a80cc651352ebaa219f`. The scoped files are the
  client `ai.mjs`, server `config.ts`, `main.ts`, `service.ts`, FD-009, the
  Agent Proxy and client READMEs, and both related stable specs. FD-008 and
  concurrent FD-010 changes in this shared tree are outside this report.

## Implemented behavior

- `ai --url` accepts an HTTP(S) service origin, rejects credentials and extra
  URL components, overrides the default loopback address/port, and sends one
  request to `/v1/requests`.
- `AIW_AGENT_PROXY_HOST` explicitly selects an IPv4 bind address, including
  `0.0.0.0`; omission keeps loopback-only HTTP. WebSocket upgrades retain
  their loopback peer and local Host checks.
- Remote HTTP Host validation now matches the raw authority as an IPv4
  literal with an optional numeric port before any URL parser can normalize
  it. The raw forms `user@192.0.2.10:43127` and
  `192.0.2.10:43127/path` fail that match. The literal is then checked with
  `isIP`, the port is compared to the service port, and a specific bind address
  must match the Host address. This resolves Reviewer finding 1.

## Verification provenance

After the Host validation repair and before writing this report, the Worker
ran these compile-only checks on the current source in the primary workspace:

- `node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json` from
  `plugins/aiw-agent-proxy`: passed, exit 0, no emitted artifact.
- `node --check ai.mjs` from `plugins/aiw-agent-proxy-client`: passed, exit 0.

This report and the `implementation-ready` handoff provide the missing Worker
provenance requested in finding 2. The prior independent review did not run
either command. No tests, final build, live remote HTTP request, Provider
request, or network call was run. Direct remote access remains intentionally
unauthenticated and plain HTTP unless an external TLS proxy is used.

Work Item 1.4 was removed from the Worker checklist because its Reviewer pass
is a separate workflow stage. This cancellation does not waive review or the
closure gate.
