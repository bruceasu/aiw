# FD-109: AIW-managed OpenAI Codex authentication for internal LLM callers and gateway

**Status:** Complete
**Revision:** 5
**Priority:** High
**Review policy:** Independent
**Evidence policy:** Markdown

## Problem

AIW LLM callers currently have no shared AIW-managed OpenAI Codex login. The Go
provider layer can use an OpenAI API key or invoke a provider CLI. The Agent
Gateway's `codex` backend invokes Codex App Server and requires a Codex
installation, while `http-openai-proxy` requires an upstream API key. This
prevents an installation from choosing a direct Codex account login and
reusing that authentication across AIW's internal LLM callers.

The intended users are AIW operators who want AIW to import an existing local
Codex login when available, or start an AIW-managed login when it is absent or
no longer refreshable. The user explicitly asked that the solution include
AIW's internal LLM calls, the Agent Gateway, and `http-openai-proxy`.

## Solution

Add an AIW-owned OpenAI Codex provider and authentication lifecycle at the
shared Go LLM runtime (`src/internal/ai`). Use the local PI
`openai-codex` OAuth and Responses implementation as the v1 compatibility
reference, as the user explicitly approved. This is a compatibility target,
not a claim that the endpoint is a documented public API. Support explicit
credential import from the local Codex installation and an explicit AIW login
command; requests may refresh valid refreshable credentials, but must not
launch an interactive login as a side effect of an ordinary model call. Keep
imported Codex credentials separate from the user's Codex files and
configuration.

Expose the direct provider through AIW's existing internal LLM configuration
and the Agent Gateway as a separate backend mode. Preserve the existing Codex
App Server mode and OpenAI API-key proxy mode so current installations can
continue using their existing execution and credential paths. The new direct
mode must preserve Gateway authentication, model allowlists, per-principal
limits, request limits, and the current function-call contract where supported.
It must not claim to provide the Codex App Server's local sandbox guarantees.

Let `http-openai-proxy` use the new mode by targeting the Agent Gateway through
its existing configurable OpenAI-compatible upstream. Do not implement a
second OAuth/token store in the Node proxy. Document the required Gateway URL
and Gateway client key configuration. Document equivalent Gateway routing for
AIW clients already using OpenAI-compatible URLs, including `aiw say` and
`aiw-cz`; these clients should continue to authenticate to the Gateway with a
Gateway key rather than manage Codex credentials themselves.

The local Codex CLI package is version 0.165.0. Its wrapper and README confirm
the installation and ChatGPT sign-in context, but do not expose its credential
schema. PI's local `openai-codex` implementation supplies the v1 compatibility
reference for OAuth, refresh, account-ID extraction, request endpoint and
headers. Do not use the separate `openai-chatgpt` direct-API OAuth flow in v1.
Keep the runtime endpoint configurable only if the existing provider patterns
require it; never log tokens or authorization responses.

## Scope

- Shared OpenAI Codex provider/auth support for Go-based internal AIW LLM calls.
- Explicit import of locally available Codex credentials without modifying
  the source credential files.
- Explicit login and logout/status operations for AIW-managed credentials.
- Refresh and expiry handling, including a clear re-login path when refresh
  is rejected or unavailable.
- A separate direct Codex backend mode in `agent-gateway`, reusing the shared
  provider and retaining Gateway client authentication and limits.
- `http-openai-proxy` documentation/configuration guidance for using that
  Gateway mode as its upstream.
- AIW-owned credential storage under the current user's config directory.
  Windows uses CurrentUser DPAPI; Unix-like platforms use a private config
  directory and a mode-0600 credential file. Refresh and writes are serialized
  across callers so rotated refresh tokens are not overwritten concurrently.

### Non-goals

- Replacing or removing the existing Codex App Server backend or its sandbox.
- Replacing OpenAI API-key authentication or changing other providers.
- Duplicating Codex OAuth implementation and token storage in Node.js.
- Changing OpenAI-compatible public request/response formats beyond the
  explicitly supported direct Gateway backend behavior.
- Introducing remote account management, multi-user OAuth consent, or a cloud
  credential broker.
- Supporting the separate PI `openai-chatgpt` dynamic-client/direct-API OAuth
  flow or claiming that Codex backend endpoints are stable public API.
- Updating stable OpenSpec requirements before the behavior is confirmed.

## Work items

- [x] 1.1 Inspect the locally installed Codex CLI package and the local PI
  `openai-codex` OAuth/Responses implementation; use the latter as the
  user-approved v1 compatibility reference. Keep the separate PI
  `openai-chatgpt` flow out of scope and record that this is not a public API
  support claim.
- [x] 1.2 Design and implement AIW-managed credentials in shared
  `src/pkg/openai-codex`:
  explicit import/login/logout/status, CurrentUser DPAPI on Windows, a private
  0600 file on Unix-like platforms, redaction, refresh, expiry/error behavior
  and serialized refresh/write operations.
- [x] 1.3 Add a direct Codex Responses provider to the shared AI runtime and
  expose it to current Go-based LLM callers without changing API-key or CLI
  provider defaults. Preserve context cancellation, timeout and usage
  reporting behavior supported by the provider.
- [x] 1.4 Add a separate direct Codex backend mode to `agent-gateway`. Reuse
  the shared credential/provider path; preserve client authentication,
  per-principal authorization/limits, structured outputs and supported
  function-call behavior. Keep the existing App Server backend unchanged.
- [x] 1.5 Document and configure `http-openai-proxy` to target the new Gateway
  backend with its normal Gateway client key. Document the same Gateway route
  for `aiw say` and `aiw-cz`; make only the smallest required compatibility
  change if an existing client cannot express the Gateway contract.
- [x] 1.6 Update relevant stable specs after the behavior and compatibility
  rules are settled; add no OpenSpec change unless separately requested.

## Acceptance

- An operator can explicitly import a locally available Codex credential into
  AIW without changing the source credential file, or explicitly start the
  AIW-managed login flow when import is unavailable.
- A normal model request never opens a browser or starts an interactive login.
- Valid stored credentials are refreshed when required. If refresh cannot
  recover the session, the request fails with a safe actionable re-login
  message and does not expose tokens in errors or logs.
- AIW Go LLM callers can select the direct Codex provider through existing
  configuration, while existing API-key and CLI-backed provider selections
  retain their behavior.
- The Agent Gateway can use the direct provider without requiring a Codex
  executable in that backend mode. Its existing App Server mode remains
  available and retains its current sandbox behavior.
- Gateway callers still authenticate with Gateway-issued client keys, and
  model access and configured per-principal rate/concurrency/usage limits
  remain enforced for the new mode.
- `http-openai-proxy` can send its supported requests through the Gateway's
  direct Codex mode using its configured Gateway URL and client key; it does
  not store or refresh Codex OAuth tokens itself.
- Existing OpenAI-compatible AIW clients (`aiw say`, `aiw-cz`) can use the
  Gateway's `/v1/chat/completions` route with a Gateway client key, without
  adding their own Codex credential handling.
- Unsupported API features, account/usage limits, expired or revoked sessions,
  and upstream errors produce bounded, redacted errors with no automatic
  retry that could duplicate a request.
- Documentation clearly distinguishes the direct API-backed mode from the
  existing App Server mode and does not claim equivalent local sandboxing.

## Verification

- Static review covered provider/configuration call paths, credential storage
  and redaction boundaries, refresh/cancellation behavior, Gateway principal
  enforcement, proxy compatibility and updated documentation/specs.
- No tests, final builds, network checks or credentialed requests were run.
- Independent review is required because this work changes authentication,
  credential persistence, public Gateway behavior and multiple modules.
- 2026-11-08: user approved using local PI source as an implementation
  reference. Inspected PI's `openai-codex.ts`, Codex Responses provider,
  request URL/header construction, OAuth refresh/login code and tests, plus
  the local `@openai/codex` package metadata/README. No network requests or
  credential reads were made.
- 2026-11-08: workspace selected as `D:/03_projects/AI-tools/aiw/.wt/FD-109`
  on `feature/FD-109`, parent `develop`; parent plan commit `6e7d00f`.
- 2026-11-08: Implemented the shared credential package, AIW auth commands and
  provider selection, direct Gateway mode with existing Gateway authorization
  and durable quota/usage accounting, plus Gateway/proxy/client documentation
  and the Gateway stable spec. No tests, final builds, network calls or
  credential reads were run.
- 2026-11-08: Compile-only checks passed for AIW packages with
  `GOMAXPROCS=2 go build -p 1 ./internal/ai ./internal/commands/auth ./cmd/aiw`
  from `src/`, and for Gateway with `GOMAXPROCS=2 go build -p 1 -o NUL .`
  from `src/programs/agent-gateway/`. An initial parallel compile exhausted
  memory; the serialized retry passed. Gateway's local module now declares the
  existing `golang.org/x/sys v0.29.0` dependency and its existing checksums.
- 2026-11-08: Independent review of commit `f32c4e2` returned
  **changes-requested** for finding F1, recorded in
  `docs/features/FD-109/reviews/FD-109_REVIEW_20261108.md`; source event:
  `FD-109-000002-review-requested`. Review policy is Independent due to
  authentication, credential persistence, concurrency and cross-module public
  interfaces.
- 2026-11-08: Fixed F1 in the direct Gateway executor: any terminal execution
  that never records `StartedAt` is now persisted as the existing recoverable
  `spawn_failed` state, while HTTP/audit error codes and caller-facing result
  states remain intact. This covers missing credentials, cancellation before
  start and pre-start storage failures.
- 2026-11-08: Recompiled the affected Gateway module successfully with
  `GOMAXPROCS=2 go build -p 1 -o NUL .` from
  `src/programs/agent-gateway/`. No tests, gateway process, network request or
  credential read was performed.
- 2026-11-08: Focused independent recheck of F1 on `157f07e` passed. Report:
  `docs/features/FD-109/reviews/FD-109_REVIEW_20261108_R2.md`; source event:
  `FD-109-000004-implementation-ready`. No tests, builds, network requests or
  credential reads were run by Reviewer.

## TODO

- Archive after the current independent review pass and delivery checks.
- If the local Codex auth file differs from the supported import schema, return
  a redacted actionable error and keep browser login available.
- Confirm the existing App Server backend remains unchanged in review; direct
  mode intentionally has no equivalent local sandbox.

## Sources

- User request in this conversation: support direct OpenAI Codex login without
  requiring Codex installation, allow local credential import and login when
  absent/expired, and cover AIW internal LLM calls, Agent Gateway and
  `http-openai-proxy`.
- `src/internal/ai/provider.go`, `config.go`, `openai.go`, `cli.go`, and
  `llm.go`: shared Go provider/configuration/runtime paths.
- `src/programs/agent-gateway/config.go`, `appserver.go`, `app_runner.go`,
  `server.go`, and `README.md`: App Server dependency, Gateway auth/limits and
  current request contract.
- `src/programs/http-openai-proxy/src/providers.ts` and `README.md`: current
  API-key upstream, request profile and configurable base URL.
- `openspec/specs/agent-gateway/spec.md` and
  `openspec/specs/agent-gateway-client/spec.md`: stable Gateway contracts.
- PI reference: `D:/03_projects/AI-tools/pi/packages/ai/src/auth/oauth/`,
  `auth/credential-store.ts`, `providers/openai-codex.ts`, and
  `api/openai-codex-responses.ts`. Local code inspected only; no upstream
  network verification performed. User confirmed this is an acceptable v1
  compatibility reference.
- Local Codex CLI package: `C:/green/nodejs/node_modules/@openai/codex`,
  version `0.165.0`; wrapper and README inspected. No credential file was read.

**Completed:** 2026-11-07
