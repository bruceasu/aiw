# FD-008: ai agent-proxy-client

**Status:** Complete
**Revision:** 7
**Priority:** Medium

## Problem

Agent Proxy already provides local `POST /v1/requests`, but sending a simple
question currently requires crafting JSON and an HTTP request. A small
terminal client should let the user ask one question with optional system
instructions, JSON output, and readable usage information. The existing
`aiw ask` command serves a different purpose: read-only AIW usage guidance.

## Options and decision

1. **Separate thin client for Agent Proxy (chosen).** Keep the existing
   `aiw ask` guidance contract intact. A small client can call the loopback
   HTTP endpoint without importing the service's internals or a Provider SDK.
   The executable is `ai`, following the user's clarification that `ask`
   already exists.
2. Extend the existing `aiw ask`. This would change its documented response
   schema, provider behavior, and session semantics; it has a wider
   compatibility impact than the requested one-shot client.
3. Call Provider CLIs/APIs directly. This duplicates Agent Proxy's routing,
   safety settings, and usage collection rather than using the requested
   service.

The revised executable contract is `ai [options] [input]`. The user also
specified `--system <content>`: when `<content>` begins with `@`, the
remainder is a local filename whose UTF-8 contents become the system prompt;
otherwise the value is the literal system prompt. This later human decision
supersedes the earlier `ask` naming assumption.
`@` alone, a missing/unreadable file, invalid UTF-8, or an empty file is an
argument error before any proxy request. Resolve relative paths from the
current working directory, never as URLs; do not print the file contents in
errors or audit data. Quoted paths with spaces remain one option value.
`--system` means a real separate high-priority instruction, not text prepended
to the ordinary question. OpenAI Responses supports the separate
`instructions` parameter; other Providers will reject this option until a
verified equivalent mapping exists. `--verbose` reports an available reasoning
summary, never raw hidden reasoning, and reports monetary cost as unknown when
the proxy has no amount. These choices preserve the stated behavior without
silently inventing data. The user's correction fixes the command and file
semantics; the previously stated summary/cost behavior remains the design
decision unless later revised.

## Solution

- Add a one-shot CLI client under `plugins/aiw-agent-proxy-client/`, with a
  simple `ai [options] [input]` entry.
  Use Node.js ESM and built-in `fetch`; keep Agent Proxy as the sole Provider
  integration and preserve `aiw ask` unless the user selects its replacement.
- Parse `--system <content>`, `--json`, and `--verbose`; reject unknown options,
  missing option values, and multiple positional questions with a clear
  usage error. For `--system @<path>`, read a bounded UTF-8 file before
  constructing the request (at most 64 KiB); for other values use the literal
  content. Do not
  invoke a shell or expand arbitrary variables while resolving the path.
  `--json` sets
  Agent Proxy `output_format: "json"` and writes only the returned JSON value
  to stdout. Without it, request `markdown` and write the response text.
- Input precedence: a non-`-` positional argument is the question; `-`
  reads the entire standard input to EOF; no positional argument opens a
  one-line interactive prompt on a terminal. Empty/whitespace-only input,
  Enter before typing, or EOF before text exits without calling the service.
  If no terminal is available and no input argument was supplied, report how
  to use `-` rather than waiting indefinitely. Piped content is read only
  when `-` is explicit.
- Send one request to `http://127.0.0.1:<port>/v1/requests` using the current
  request fields and a generated request ID. Default port 43127, aligned with
  Agent Proxy; accept `AIW_AGENT_PROXY_PORT` for local configuration. The
  client must not start the service, call a Provider directly, or persist the
  prompt/response. Select provider/model from a documented local environment
  configuration, defaulting to the user's verified OpenAI `gpt-6-luna` only
  when no override is supplied. Do not add an arbitrary remote URL option.
- Add optional `system_prompt` and `include_reasoning_summary` request fields
  to Agent Proxy. Validate both without changing existing requests. For
  OpenAI, map `system_prompt` to Responses `instructions` and, only when
  requested, ask for `reasoning.summary: "auto"`; return any published
  summary as an optional response field. Reject `system_prompt` for Codex and
  Copilot until a separately validated high-priority mapping exists.
  `include_reasoning_summary` may return unknown for unsupported models or
  Providers. Never put a summary in the metadata-only audit log or expose raw
  opaque reasoning tokens. Preserve existing no-tool Provider restrictions.
- `--verbose` formats only fields actually returned by Agent Proxy: provider,
  model, request ID, input/output and detailed OpenAI Token counts, and cost
  when known. Include a clearly labelled reasoning summary when provided;
  otherwise say it is unavailable. Keep diagnostics separate from the answer
  (stderr when `--json`). Never calculate a monetary charge from mutable
  prices or label reasoning Token counts as readable thinking text. Show
  unknown monetary cost as unknown, not zero.
- Keep HTTP errors bounded and actionable: show service unavailable,
  structured error code, and safe message without printing credentials or
  whole response bodies. Use a finite timeout slightly beyond the proxy's
  60-second request timeout. No retry that could duplicate a paid request.

## Scope

In scope: a simple one-shot command, three requested options, positional,
interactive, and explicit stdin input; HTTP Agent Proxy integration; readable
usage reporting; backward-compatible optional proxy request/response fields;
and documentation/stable spec updates.

Out of scope: changing the existing AIW guidance tool by default, chat
history, streaming/WebSocket, direct Provider credentials, automatic service
startup, cost estimation from price tables, and displaying hidden reasoning
that the Provider or proxy does not return.

## Work items

- [x] 1.1 Build the command entry and strict argument/input parser. Acceptance:
  positional text, explicit `-` stdin, and no-argument interactive behavior
  match the rules above; `ai` is the executable, and empty input makes no
  request.
- [x] 1.2 Implement one bounded loopback HTTP Agent Proxy request with
  documented provider/model configuration, `--json`, structured errors, and
  no automatic retry. Acceptance: the request/response mapping matches the
  current service contract and does not touch Provider credentials.
- [x] 1.3 Add optional proxy fields for independent OpenAI instructions and
  a requested published reasoning summary, then implement `--system` and
  `--verbose` without claiming unavailable thinking or cost. Acceptance:
  `--system` accepts literal text or `@<path>` file contents, with malformed
  or unreadable files rejected before a request; unsupported Provider system
  instructions fail explicitly; existing proxy callers remain compatible;
  summaries are opt-in and excluded from audit.
- [x] 1.4 Document installation/invocation, configuration, output streams,
  empty-input behavior, and compatibility with existing `aiw ask`; update
  stable specs only if an external contract changes.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## TODO

- [x] Inspect existing `aiw ask` and Agent Proxy contracts.
- [x] Record CLI, input, instruction-priority, summary, and cost decisions.
- [x] Reconcile the user's later `ai` and `--system @<path>` decisions.
- [x] Implement the numbered Work Items and prepare Worker evidence.
- [ ] Obtain independent Reviewer outcome and close only after a current pass.

## Acceptance

1. A one-shot question produces only the model answer on stdout by default;
   `--json` requests valid JSON and prints a JSON value without extra prose.
2. No argument prompts once on a terminal and exits without a request on
   blank/EOF; `-` consumes stdin to EOF and likewise avoids a request if empty.
3. `--system` reaches OpenAI as separate `instructions`; unsupported
   Providers fail explicitly rather than pretending ordinary prompt text has
   higher priority. An `@<path>` value is read as UTF-8 before the request;
   missing, unreadable, empty, invalid UTF-8, or oversized files fail without
   sending their contents or making an API request.
4. `--verbose` shows a published reasoning summary when returned, otherwise
   marks thinking unavailable; it reports available usage and known cost,
   clearly marks unknown cost, and never fabricates raw reasoning or charges.
5. Invalid arguments, unavailable service, timeout, and structured proxy
   errors produce a concise nonzero failure without prompt/credential leaks
   or implicit retry. Existing `aiw ask` remains compatible unless the user
   explicitly selects a replacement.

## Verification

- Design evidence: `docs/usage/aiw-ask.md`,
  `openspec/specs/agent-proxy/spec.md`, `plugins/aiw-agent-proxy/src/types.ts`,
  `validate.ts`, `service.ts`, `providers.ts`, `config.ts`, and package README.
- No implementation or runtime check has been run for this FD. Worker should
  perform one compile-only check after code edits; a focused local mock HTTP
  scenario needs authorization under repository runtime rules.
- Design readiness: `FD_APPLIED`. The command name follows the user's usage
  correction (`ai`), while absent thinking/cost is represented honestly.
  The user's `--system @<path>` rule is now part of Work Item 1.3 and
  Acceptance 3. OpenAI's
  [`instructions`](https://developers.openai.com/api/reference/cli/resources/responses/methods/create)
  field and opt-in [reasoning summaries](https://developers.openai.com/api/docs/guides/reasoning)
  are documented; no raw reasoning is available through that API.
- Human decision on 2026-10-01: `ask` already exists; use `ai`. Treat a
  `--system <content>` value beginning with `@` as a filename and read it
  before calling Agent Proxy. This supersedes the initial command-name
  assumption without changing the numbered Work Item IDs.
- Worker implementation report: `docs/features/archive/FD-008/reports/FD-008-implementation.md`.
  The client adds strict CLI/input parsing, bounded UTF-8 file/stdin reads,
  one local HTTP request, JSON stdout, and verbose stderr. Agent Proxy adds
  optional validated system/summary request fields; OpenAI uses independent
  `instructions` and opt-in `reasoning.summary`, while Codex/Copilot reject
  `system_prompt`. Summary text is excluded from JSONL audit. The new client
  spec and both package READMEs document these contracts.
- Compile-only evidence: `node --check ai.mjs` in the new client package and
  `node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json` in the
  Agent Proxy package both passed, including a final rerun after the bounded
  file-read, response-size, and timeout corrections. A focused
  `git diff --check` on changed tracked files passed. No final build,
  npm link, test, local mock HTTP request, or real Provider call was run.
- `%% RISK: No interactive or HTTP runtime check was authorized for FD-008; entry-point linking and model-specific reasoning-summary availability remain unverified. Missing summary/cost are displayed as unavailable/unknown rather than inferred.`
- Independent Reviewer: `docs/features/archive/FD-008/reviews/FD-008-review.md` reviewed the working-tree diff against `24c848a` for handoff `FD-008-000006-implementation-ready` and found no material acceptance gap. Outcome: `VERIFICATION_PASSED` by static review; the stated runtime risks remain.

## Sources

- Issue: none; direct user request on 2026-10-01.
- Agent Proxy base capability: `openspec/specs/agent-proxy/spec.md`.
- Existing Ask guidance contract: `docs/usage/aiw-ask.md`.
- OpenAI request and summary behavior:
  `https://developers.openai.com/api/reference/cli/resources/responses/methods/create`
  and `https://developers.openai.com/api/docs/guides/reasoning`.

**Completed:** 2026-10-01
