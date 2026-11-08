# FD-113: Unify shared LLM provider configuration across commands

**Status:** Complete
**Revision:** 8
**Priority:** Medium  
**Review policy:** Independent
**Evidence policy:** Markdown

## Problem

AIW commands select and configure LLM providers through separate paths. `aiw ask`
uses the shared Go provider configuration, `aiw say` only accepts OpenAI,
`aiw git guide` directly invokes Codex CLI, and Python CZ consumers use an
independent `[cz]` provider configuration. Users therefore cannot reliably set
one default provider for common AIW work, and provider-specific environment
variables are repeated or behave differently between commands.

## Solution

Use `[ai]` and the existing provider environment variables as the shared
default contract. Explicit command-line options and command-specific TOML
settings remain overrides. Preserve existing command defaults when no shared
provider is configured. API credentials continue to come from standard
provider environment variables when available; do not add per-command secret
variables or copy TOML secrets into plugin metadata.

Route Go `say` through the shared `internal/ai` provider interface. Keep
`aiw ask` on that interface. Update Python CZ resolution so explicit `[cz]`
settings remain overrides and `[ai]` supplies the provider default when CZ has
none. Add an OpenAI Python SDK Responses adapter to the Python Git guide for
OpenAI and DeepSeek, while retaining its Codex CLI adapter. Each command must
report an unsupported selected provider clearly rather than silently switching
to a different model.

## Scope

- Unify provider/model/base URL/API-key precedence for `aiw ask`, `aiw say`,
  `aiw git guide`, `aiw cz`, and Git Python commands already using CZ provider
  configuration (`git ai`, `git aic`, and `git air`).
- Preserve command-specific prompts, structured output validation, Git guide
  search/save behavior, CZ candidate/TUI behavior, and say translation options.
- Add and document the Python SDK dependency needed by the Git guide Responses
  adapter.
- Update stable CZ and AI routing specifications, configuration examples, and
  usage documentation to match the settled behavior.
- Exclude unrelated commands and gateway/app-server changes. Do not add a new
  environment-variable family for each command.

## Work items

- [x] 1.1 Define and implement the shared provider resolution contract, including
  CLI, command-specific TOML, global `[ai]`, standard provider environment,
  legacy defaults, and explicit unsupported-provider errors.
- [x] 1.2 Route `aiw say` through the shared Go provider interface while
  preserving existing say profiles, translation behavior, and CLI overrides.
- [x] 1.3 Add the Python SDK Responses adapter and packaged dependency for
  OpenAI/DeepSeek in `aiw git guide`; retain Codex CLI support and its structured
  output, validation, save, and failure behavior.
- [x] 1.4 Make Python CZ and its existing Git AI consumers inherit `[ai]` only
  when no explicit CZ provider is configured; preserve CZ-specific settings as
  overrides and reconcile the stable CZ specification.
- [x] 1.5 Align `aiw.toml.example`, `.env.example`, provider documentation,
  command help, and affected stable specifications with the shared contract.

## Acceptance

- Setting `[ai].provider` selects the same provider for each in-scope command
  that supports it; an explicit command-line or command-specific setting wins.
- With no `[ai].provider`, each command preserves its existing provider
  default or automatic fallback behavior.
- OpenAI and DeepSeek API credentials use `OPENAI_API_KEY` and
  `DEEPSEEK_API_KEY`; their existing model/base URL environment variables work
  consistently where those providers are supported.
- CZ retains its explicit `[cz]`, `CZ_LLM_PROVIDER`, and CLI overrides. If
  none selects a provider, it inherits the shared `[ai]` provider before its
  legacy automatic fallback.
- Git guide supports OpenAI/DeepSeek Responses through the Python SDK and
  Codex CLI through the existing adapter. It preserves the structured answer
  contract and saves only validated answers.
- If an explicitly selected provider is unsupported or misconfigured, the
  command explains the problem and does not silently call another provider.
- No API key is added to tracked configuration examples or copied into new
  plugin metadata variables.

## Verification

- Static inspection: reviewed shared Go provider parsing and context calls,
  Say's provider invocation, Python CZ precedence and explicit-provider
  handling, Git guide provider dispatch and structured response validation,
  dependency packaging, configuration examples, and stable specs.
- Corrected Python provider environment precedence after review identified that
  shared `[ai]` TOML could mask `OPENAI_*` / `DEEPSEEK_*`; CZ-specific fields
  remain higher priority and `AIW_LLM_*` applies after provider variables.
- Compile-only: `go build -o NUL ./cmd/aiw-say` from `src/` passed with
  `GOPROXY=off` and `GOSUMDB=off`.
- Static command: `git diff --check` passed; the stale-configuration wording
  search returned no matches. Git emitted only CRLF normalization notices.
- Final static diff inspection after the precedence fix covered the Python CZ
  resolution path, affected Go/Say paths, provider dispatch, examples and specs.
- Independent review passed for revision 5; report:
  [FD-113-review-r5.md](reviews/FD-113-review-r5.md).
- Archive gate check: `aiw fd close FD-113 Complete --reason ...` rejected the
  Reviewer receipt because its `fd_sha256` predates the Verification text added
  after that event. No implementation changes were made after review r5; a new
  receipt-bound review is requested to resolve this evidence mismatch.
- Receipt refresh review for FD revision 7 passed with no implementation drift;
  report: [FD-113-review-r6.md](reviews/FD-113-review-r6.md). The matching
  `verification-passed` receipt was emitted and acknowledged for
  `FD-113-000007-review-requested`.
- Tests, Python SDK calls, provider API requests, and final builds were not run;
  runtime provider compatibility remains unverified.

## TODO

- [x] Commit the previously authorized CZ configuration examples and guidance as
  `2075cb4` before creating this FD.
- [x] Obtain implementation review r5; no blocking findings.
- [x] Complete the receipt-bound r6 verification review and report.
- [x] Archive after the delivery gate; archive commit `3672e32`.

## Sources

- Issue: none.
- [CZ stable specification](../../../openspec/specs/cz/spec.md)
- [AI routing stable specification](../../../openspec/specs/ai-routing/spec.md)
- [Go shared AI configuration](../../../src/internal/ai/config.go)
- [Go provider calls](../../../src/internal/ai/llm.go)
- [Say configuration and provider](../../../src/internal/say/config.go)
- [Python Git guide](../../../src/plugins/aiw-git/git-guide.py)
- [Python CZ configuration](../../../src/plugins/aiw-cz/cz_config.py)

**Completed:** 2026-11-08
**Disposition reason:** Revision 8 has a current matching independent verification pass; implementation commit bb430e8 is delivered, and runtime provider checks remain explicitly unrun.
