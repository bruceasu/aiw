# FD-109: AIW-managed OpenAI Codex authentication for internal LLM callers and gateway

**Status:** Design  
**Revision:** 1  
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
shared Go LLM runtime (`src/internal/ai`). It must support explicit credential
import from the local Codex installation and an explicit AIW login command;
requests may refresh valid refreshable credentials, but must not launch an
interactive login as a side effect of an ordinary model call. Keep imported
Codex credentials separate from the user's Codex files and configuration.

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
and Gateway client key configuration.

The Pi source at `D:/03_projects/AI-tools/pi` is an implementation reference
for provider-owned OAuth, token refresh, credential import patterns and
credential-store boundaries. Reuse concepts, not its TypeScript implementation
or undocumented endpoint assumptions. Verify the exact supported Codex auth
flow and protocol against suitable authoritative sources before implementing
network exchanges.

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
- Security decisions for local token storage, permissions, redaction, refresh
  concurrency, and account scope.

### Non-goals

- Replacing or removing the existing Codex App Server backend or its sandbox.
- Replacing OpenAI API-key authentication or changing other providers.
- Duplicating Codex OAuth implementation and token storage in Node.js.
- Changing OpenAI-compatible public request/response formats beyond the
  explicitly supported direct Gateway backend behavior.
- Introducing remote account management, multi-user OAuth consent, or a cloud
  credential broker.
- Updating stable OpenSpec requirements before the behavior is confirmed.

## Work items

- [ ] 1.1 Verify the supported OpenAI Codex authorization, refresh, request
  protocol, required headers/account metadata and applicable usage limits from
  authoritative sources. Record which Pi patterns are safe to reuse.
- [ ] 1.2 Design and implement AIW-managed credentials in `src/internal/ai`:
  explicit import/login/logout/status, secure storage, redaction, refresh,
  expiry/error behavior and safe handling of concurrent requests. Select the
  platform storage contract before writing credentials.
- [ ] 1.3 Add a direct Codex Responses provider to the shared AI runtime and
  expose it to current Go-based LLM callers without changing API-key or CLI
  provider defaults. Preserve context cancellation, timeout and usage
  reporting behavior supported by the provider.
- [ ] 1.4 Add a separate direct Codex backend mode to `agent-gateway`. Reuse
  the shared credential/provider path; preserve client authentication,
  per-principal authorization/limits, structured outputs and supported
  function-call behavior. Keep the existing App Server backend unchanged.
- [ ] 1.5 Document and configure `http-openai-proxy` to target the new Gateway
  backend with its normal Gateway client key. Confirm the current proxy request
  profile can express the chosen Gateway contract; make only the smallest
  required compatibility change if it cannot.
- [ ] 1.6 Update relevant stable specs after the behavior and compatibility
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
- Unsupported API features, account/usage limits, expired or revoked sessions,
  and upstream errors produce bounded, redacted errors with no automatic
  retry that could duplicate a request.
- Documentation clearly distinguishes the direct API-backed mode from the
  existing App Server mode and does not claim equivalent local sandboxing.

## Verification

- Planned: static review of provider/configuration call paths, credential
  storage and redaction boundaries, refresh/cancellation behavior, Gateway
  principal enforcement, proxy compatibility and updated documentation/specs.
- Planned after implementation: one narrow compile-only check per changed Go
  target, following repository resource limits. No tests, builds, network
  checks or credentialed requests are authorized by this FD plan alone.
- Independent review is required because this work changes authentication,
  credential persistence, public Gateway behavior and multiple modules.
- Not run: implementation checks; this FD is planning-only at this stage.

## TODO

- Resolve before implementation: verify the exact current Codex OAuth/request
  protocol and the permitted way to import/refresh credentials. Do not infer
  these contracts solely from Pi source.
- Resolve before implementation: choose an OS-appropriate credential-storage
  mechanism for supported AIW platforms and define behavior where a secure
  credential store is unavailable.
- Resolve during design: confirm whether direct Gateway mode supports every
  currently accepted function-call/schema case or documents a narrower subset.
- No implementation has started.

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
  network or protocol verification performed.
