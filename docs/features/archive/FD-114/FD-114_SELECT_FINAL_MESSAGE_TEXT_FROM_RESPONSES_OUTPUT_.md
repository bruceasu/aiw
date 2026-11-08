# FD-114: Select final message text from Responses output in CZ

**Status:** Complete
**Revision:** 2
**Priority:** Medium
**Review policy:** Self
**Evidence policy:** Markdown

## Problem

`aiw cz --llm` with DeepSeek reports `LLM response does not contain a JSON
object` and falls through to the manual wizard. The error is raised when CZ's
candidate parser receives text without a JSON object. The raw API response was
not captured, so the exact response for this invocation is unknown.

Static inspection found that the shared Responses extractor returns the first
text field from any output item. DeepSeek Responses may return distinct
`reasoning` and `message` output items. This can make CZ parse reasoning text
instead of the final candidate message.

## Solution

Extract user-facing text only from final `message` output items and their
`output_text` content. Continue using the SDK-provided `output_text` when
available. Keep the existing structured candidate validator and provider
fallback behavior unchanged.

## Scope

Change the Python CZ Responses text extractor and update its focused
documentation/spec only if needed. Exclude request replay, API calls, model
prompt changes, and unrelated provider behavior.

## Work items

- [x] 1.1 Extract final Responses message text without treating reasoning or
  other non-message items as the CZ candidate response.

## Acceptance

- Responses containing a reasoning item followed by a JSON message yield the
  message text to candidate validation.
- Non-message text is never accepted as a commit candidate.
- Missing final message text still reports a clear provider response error and
  follows the existing CZ fallback/wizard behavior.
- Existing `output_text` responses and candidate validation remain supported.

## Verification

- Static inspection: traced `cz_llm.candidates` through `cz_openai._generate`,
  `_text`, and `cz_core.parse_candidates`. The extractor previously returned
  the first text from any output item; it now takes only `output_text` from
  `message` items (or the SDK `output_text` value).
- Compile-only: `python -c "from pathlib import Path; p=Path('src/plugins/aiw-cz/cz_openai.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec')"` passed.
- Static command: `git diff --check` passed; Git emitted only a CRLF
  normalization notice.
- Runtime requests and tests are not authorized and will remain unrun.
- The original response body is unavailable, so diagnosis remains a supported
  hypothesis; DeepSeek documents separate reasoning and message output items
  and exposes final response text through `output_text`.

## TODO

- [x] Implement Work Item 1.1 and record static/compile evidence.
- [ ] Archive after implementation checks pass.

## Sources

- Issue: none

- [CZ stable specification](../../../openspec/specs/cz/spec.md)
- [CZ Responses extraction](../../../src/plugins/aiw-cz/cz_openai.py)
- [Candidate parser](../../../src/plugins/aiw-cz/cz_core.py)
- [DeepSeek Responses API](https://api-docs.deepseek.com/guides/responses_api/)

**Completed:** 2026-11-08
**Disposition reason:** The Responses text extractor now selects final message output; syntax-only compile and static diff check passed. Runtime response capture remains unverified.
