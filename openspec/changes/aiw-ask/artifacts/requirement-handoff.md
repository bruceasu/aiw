# Requirement Handoff

## Source
- Requirement ID: aiw-ask
- Requirement revision: 7
- Approved by: user
- Approved at: 2026-09-10T14:26:26+09:00

## Referenced Artifacts
| Artifact | Path | Digest |
|---|---|---|
| business-case | requirements/aiw-ask/business-case.md | 35bb8e141e33dff7307f03c8a2fb4623175f78a1961d1304065a16ef5ee73fe2 |
| engineering-options | requirements/aiw-ask/engineering-options.md | 352d6305b15636d0b23eab14efa3093c148703155ea59991076a44726bfd2425 |
| problem-brief | requirements/aiw-ask/problem-brief.md | 9033b1cf2ebe412d7470d5d9a983a412e08c4053351ddf020fa71c25c3fe80b0 |
| requirement-plan | requirements/aiw-ask/requirement-plan.md | 95b875e2aebe17d2048421d6b9d6eadec49c1b5d40250932a9d0cad91f0416ed |

## Approved Scope
- Use the approved Requirement Plan and referenced artifacts as the authoritative scope.

## Non-Goals
- Automatic question analysis, clustering, plugin generation, and tool calls.
- Complex Chat editing and a new LLM provider abstraction.
- Project-file, Git, Task, or OpenSpec mutation by `aiw ask`.

## Accepted Risks
- A malformed LLM JSON response is surfaced as `error` without an automatic
  repair retry, trading resilience for predictable cost and behavior.
- General guidance intentionally omits automatic workspace context.

## Open Decisions Carried Into Engineering
%% DEFERRED: Verify Codex version-specific read-scope behavior and document the
limitation; do not treat `--add-dir` as a strict read allowlist.

## Suggested Next Workflow Action
Begin implementation from the approved OpenSpec checklist. Codex filesystem
context is supported with an explicit read-scope limitation; Copilot uses its
documented allowed-paths behavior.
