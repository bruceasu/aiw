# FD-110: Add DeepSeek provider support across AIW LLM callers

**Status:** In Progress
**Revision:** 3
**Priority:** Medium
**Review policy:** Independent
**Evidence policy:** Markdown

## Problem

AIW's Go LLM runtime, `aiw-cz`, and several existing callers support OpenAI
formats, but DeepSeek is not a named provider with its own model, endpoint, and
API-key settings. Operators can point some clients at a compatible endpoint by
overriding OpenAI settings, but that reuses OpenAI credentials and does not
give AIW's internal provider selection a DeepSeek option.

DeepSeek's current official API supports the Responses API at
`https://api.deepseek.com` with `deepseek-flash` and `deepseek-v4-pro`. Its API
is stateless and does not support `previous_response_id`; the DeepSeek provider
must not send or rely on response chaining.

## Solution

Add a named `deepseek` provider to the shared Go LLM runtime, using the existing
OpenAI Responses transport with DeepSeek-specific config/environment settings
and defaults. Disable stored-response and previous-response chaining for this
provider. Add the same provider choice to `aiw-cz`, which already uses Responses
and supports JSON Schema output. Preserve generic OpenAI base-URL support.

The Agent Gateway's `openai_proxy` backend, `http-openai-proxy`, and `aiw say`
already accept compatible base URLs and API keys. Keep their runtime contracts
unchanged and document the direct DeepSeek configuration for each. Gateway
callers continue to use Gateway keys; Gateway resolves the upstream DeepSeek
key from its configured environment variable.

## Scope

- Named DeepSeek provider selection, model/URL/API-key config and environment
  variables for Go LLM callers and `aiw-cz`.
- Documentation examples for the supported OpenAI-compatible Gateway, proxy,
  `aiw say`, and `aiw-cz` routes.
- Preserve current OpenAI provider defaults and all existing provider names.
- No direct DeepSeek-specific backend in Agent Gateway and no changes to its
  transparent `openai_proxy` forwarding contract.
- No DeepSeek CLI, Anthropic API, billing/account management, or automatic
  migration of existing OpenAI credentials.

## Work items

- [x] 1.1 Add `deepseek` to the shared Go LLM provider/configuration path with
  DeepSeek-specific environment/config keys, documented defaults, Responses
  API transport, and stateless response handling.
- [x] 1.2 Add explicit DeepSeek configuration and provider selection/fallback
  behavior to `aiw-cz`; retain existing OpenAI behavior and errors.
- [x] 1.3 Document direct DeepSeek configuration for `aiw say`,
  `http-openai-proxy`, and Gateway `openai_proxy`, including model and key
  separation.
- [x] 1.4 Review the relevant contracts and update stable specs only if the
  implementation changes an existing Gateway capability requirement.

## Acceptance

- A caller can select `provider=deepseek` and use DeepSeek-specific settings
  without reading or overwriting OpenAI API-key settings.
- When no model or base URL is configured, the provider uses the documented
  DeepSeek defaults; a configured model/key/base URL overrides those defaults.
- Requests do not send `previous_response_id` or request server-side storage;
  Responses ID values are not treated as resumable DeepSeek conversation state.
- `aiw-cz` can select DeepSeek explicitly and its auto/fallback path can use
  DeepSeek when its credentials are configured; existing OpenAI routing remains
  available.
- Existing compatible clients can reach DeepSeek directly or through the
  Gateway's existing proxy mode using the correct upstream URL/key.
- Missing credentials and upstream errors remain bounded and do not expose
  API keys. No default network requests or test execution are required for
  implementation completion.

## Verification

- Static review confirmed Go provider selection resolves DeepSeek model, base
  URL, and key from DeepSeek-specific settings. Automatic Go detection uses
  `DEEPSEEK_API_KEY`; Python `aiw-cz` only adds DeepSeek to fallback when its
  DeepSeek key is configured. Neither path falls back to `OPENAI_API_KEY`.
- The DeepSeek Responses request omits `store` and `previous_response_id`, and
  does not expose the response ID as a resumable thread ID.
- Compile-only command passed:
  `GOMAXPROCS=2 go build -p 1 ./internal/ai ./internal/commands/ask ./cmd/aiw`
  (run from `src/`). The first attempt found an unused import; it was removed
  and the same command passed on the single permitted retry.
- No tests, final builds, live API calls, or dependency downloads were run.
- Reviewed the existing Agent Gateway `openai_proxy` contract; it already
  accepts an upstream base URL and environment-selected upstream key, so no
  stable spec change was needed.
- Independent review is required because this adds a public provider/config
  option across Go and Python and must preserve existing provider behavior.
- Independent review requested changes: see
  `docs/features/FD-110/reviews/reviewer-12b3920.md` (source event
  `FD-110-000002-review-requested`). The CLI provider choices have now been
  extended with `deepseek`; `--model` already writes to the selected provider's
  model slot, so `aiw-cz --provider deepseek --model <id>` uses the DeepSeek
  model setting. This correction awaits focused independent re-review.
- Test support needs an explicit request and its command/revision authorization.

## TODO

- Address the first review finding and obtain focused independent re-review;
  then archive this FD.

## Sources

- DeepSeek official API overview: https://api-docs.deepseek.com/api/responses
- DeepSeek Responses compatibility guide:
  https://api-docs.deepseek.com/guides/responses_api/
- `src/internal/ai/config.go`, `provider.go`, `openai.go`,
  `openai_compatible.go`, and `llm.go`
- `src/plugins/aiw-cz/cz_config.py`, `cz_llm.py`, and `cz_openai.py`
- `src/programs/agent-gateway/config.go` and `README.md`
- `src/programs/http-openai-proxy/src/providers.ts` and `README.md`
- `src/programs/aiw-say/README.md`

Link relevant specs and evidence used for decisions.
