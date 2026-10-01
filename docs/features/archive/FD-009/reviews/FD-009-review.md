# FD-009 Independent Review

- Outcome: `CHANGES_REQUESTED`
- Source event: none; direct user request, with no FD-009 handoff event recorded
- Reviewed base: `24c848a74fde49a6a6e92a80cc651352ebaa219f` plus the current working tree
- Scope: `docs/features/FD-009_REMOTE_AGENT_PROXY_URL.md`, the Agent Proxy client URL path, server host/bind path, related stable specs, and package READMEs. Concurrent FD-008 and FD-010 work was not reviewed as part of FD-009.

## Findings

1. **Host validation accepts values that are not an IPv4 Host authority.** `plugins/aiw-agent-proxy/src/service.ts` passes the raw Host header to `new URL()` in `remoteHostHeader` and checks only the parsed hostname and port. For example, `user@192.0.2.10:43127` and `192.0.2.10:43127/path` parse to an IPv4 hostname and the configured port, so the function accepts them when bound to `0.0.0.0` (and when the address matches a specific bind). FD-009 requires remote HTTP requests to use an IPv4 Host header for the configured port. Validate the raw Host authority before URL normalization and reject userinfo, path, query, and fragment syntax.
2. **Worker verification evidence is incomplete.** No `docs/features/archive/FD-009/reports/FD-009-implementation.md` or pre-review `implementation-ready` handoff event was present. The FD says `node --check ai.mjs` and TypeScript `tsc --noEmit` passed, but this review cannot verify when or against which source revision those commands ran. Record the Worker report and exact verification context, or mark those claims unverified.

The static call path otherwise supports the default loopback endpoint, `--url` endpoint override before `fetch`, explicit IPv4 bind, ordinary HTTP request validation, and the unchanged loopback-only WebSocket upgrade check. The two findings prevent verification from passing.

## Commands and skipped checks

- Read-only commands: `aiw fd --help`, `aiw fd show FD-009`, `aiw fd claim --help`, `aiw fd emit --help`, `git status --short`, `git branch --show-current`, `git diff --stat`, scoped `git diff`, `git log -1`, `git ls-files`, `rg`, `Get-Content`, and file existence checks.
- No test, build, type check, syntax check, or live remote request was run in this review.
- Residual risk: remote HTTP is intentionally unauthenticated and may carry prompts and results over plain HTTP. Live remote reachability and WebSocket rejection have not been exercised.
