# FD-009: Remote Agent Proxy URL

**Status:** Complete
**Revision:** 7
**Priority:** Medium

## Problem

The `ai` client targets only a loopback Agent Proxy. The user needs to invoke
an Agent Proxy running on another machine with `--url`.

## Options and decision

The service currently binds to loopback and rejects remote peers. The user
chose direct remote access rather than a tunnel, and explicitly deferred
authentication. Remote access therefore requires an opt-in server bind as well
as the client option. The default remains local-only. This decision supersedes
FD-008's exclusion of an arbitrary remote URL.

## Solution

- `ai --url <http(s)://host:port>` selects an Agent Proxy origin for one
  request. It overrides `AIW_AGENT_PROXY_PORT`, appends `/v1/requests`, and
  rejects credentials, path, query, fragment, or unsupported schemes. Existing
  default behavior remains unchanged.
- `AIW_AGENT_PROXY_HOST` can bind the service to an IPv4 address or
  `0.0.0.0`. Without it, the service stays on `127.0.0.1`. In remote mode,
  HTTP requests must use an IPv4 Host header for the configured port. WebSocket
  access stays loopback-only.
- Remote HTTP has no service authentication or transport encryption. The
  operator must restrict network access to trusted clients and use a TLS
  terminating proxy when encryption is required.

## Scope

In scope: client URL selection, explicit server HTTP bind, stable specs, and
operator documentation. Out of scope: authentication, service-native TLS,
remote WebSocket, Provider routing changes, and automatic retries.

## Work items

- [x] 1.1 Add strict `ai --url` parsing and endpoint selection.
- [x] 1.2 Add explicit remote HTTP bind while preserving the default local
  access policy and local WebSocket access policy.
- [x] 1.3 Update client/server documentation and stable specs.
- [-] 1.4 Reviewer pass is a separate handoff, not a Worker Work Item; the
  independent review remains required before closure.

## Acceptance

- No `--url`: client targets `127.0.0.1` and uses `AIW_AGENT_PROXY_PORT`.
- Valid `--url`: client targets exactly that origin's `/v1/requests`.
- Invalid URL: the client fails before sending a request.
- Default server: remote peers remain rejected. Explicit IPv4 bind: remote
  HTTP requests using an IPv4 Host header can reach normal validation.
- Remote WebSocket requests remain rejected.

## Verification

- Static call-path review traced URL parsing through `fetch` and server bind,
  HTTP Host validation, and the unchanged WebSocket peer check.
- `node --check ai.mjs` passed after the final URL validation edit.
- `node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json` passed.
- Independent review: `docs/features/archive/FD-009/reviews/FD-009-review.md` found that
  remote Host validation accepts non-IPv4-authority syntax and that the Worker
  report/compile-check provenance is missing. Outcome: `CHANGES_REQUESTED`;
  Reviewer acceptance remains open.
- Worker repair: raw remote Host authority is now matched before normalization;
  `docs/features/archive/FD-009/reports/FD-009-implementation.md` records the source context
  and the two compile-only commands rerun after this repair. A fresh independent
  review passed: `docs/features/archive/FD-009/reviews/FD-009-review-r2.md` found no remaining
  material acceptance gap. Outcome: `VERIFICATION_PASSED`.
- Tests, final build, and live remote requests were not run.

%% RISK: Unauthenticated plain HTTP can expose prompts/results and consume
%% Provider quota when the configured port is reachable. Restrict network
%% access and use a trusted network or TLS terminating proxy.

## Sources

- User request and follow-up choice in this conversation (2026-10-01).
- [FD-008](archive/FD-008/FD-008_AI_AGENT_PROXY_CLIENT.md).
- `openspec/specs/agent-proxy/spec.md` and
  `openspec/specs/agent-proxy-client/spec.md`.

**Completed:** 2026-10-01
