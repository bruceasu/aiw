# OpenSpec-lite TOML Mapping for Engineering Review

> Explains how to place the output of `eng-review-finance` inside a host
> repository that uses the OpenSpec-lite TOML workflow (see `AGENTS.md` in the
> tangram-trade-mt5 family of repos).

## Two Layers

1. **Logical profile** ── what this skill emits, always the same:
   - `design.md`
   - `permissions.md` (only when running standalone; inside `autoplan-finance` this is folded into `PLAN.md` section 9)
   - `audit.md` (same conditional as above)
   - `release.md` (only when the caller also wants a release stub; usually `release-review` owns this)

2. **On-disk location** ── decided by the host repo's OpenSpec variant.

## OpenSpec-lite TOML Layout

```text
openspec/
  changes/
    <change-id>/
      task.toml          # required, machine-readable intake
      tasks.md            # required, human-readable intake
      tasks.md           # required, ordered task checklist
      design.md          # produced here (sections 1..5, 8..11 of Output Format)
      release.md         # produced here only if caller requests a release stub
  specs/
    <capability>/
      permissions.md     # produced here (Output Format section 6)
      audit.md           # produced here (Output Format section 7)
      metrics.md         # owned by metrics-review, referenced here
```

## Section-to-File Mapping

| Output Format section | Target file | Notes |
|---|---|---|
| `## Status` | `design.md` head | Also written to `task.toml` `[status]` if the host uses TOML status tracking. |
| `## 1. Context` | `design.md` | 1:1. |
| `## 2. System Boundary` | `design.md` | 1:1. |
| `## 3. Module Responsibilities` | `design.md` | 1:1. |
| `## 4. Data Contracts` | `design.md` | 1:1. |
| `## 5. Data Flow` | `design.md` | 1:1; keep sub-sections 5.1 / 5.2 / 5.3. |
| `## 6. Permissions` | `permissions.md` under `openspec/specs/<capability>/` | Long-lived spec, not change-scoped. |
| `## 7. Audit Requirements` | `audit.md` under `openspec/specs/<capability>/` | Long-lived spec, not change-scoped. |
| `## 8. Failure Modes` | `design.md` | 1:1. |
| `## 9. Observability` | `design.md` | 1:1. |
| `## 10. Testing Strategy` | `design.md` | 1:1; individual test tickets flow into `tasks.md`. |
| `## 11. Risks` | `design.md` | 1:1. |
| `## 12. Release Readiness Impact` | `release.md` under `openspec/changes/<change-id>/` | Owned jointly with `release-review`; this skill only fills the "impact" surface. |
| `## 13. Required Decisions Before Implementation` | `tasks.md` head + `design.md` tail | Each open decision becomes both an entry in the checklist and a bullet in `design.md` so it stays visible in review. |

## Change-ID Naming

Follow the host repo convention. In the tangram-trade-mt5 family this is:

```text
<yyyy-mm-dd>-<kebab-topic>
```

Example: `2026-07-07-daily-withdraw-summary`.

## Language Rule

- File names, section headings, table headers, action enums: **English**.
- Body text, bullets, table cells: match user language (中文正文 default in this repo family).

## Validator Note

The included `scripts/validate_eng_review.py` validates a **single**
`ENG_REVIEW.md` (or `design.md`) file against the Output Format headings. It
does not walk the whole `openspec/` tree. If the host repo splits section 6
into `permissions.md` and section 7 into `audit.md`, run the validator against
the pre-split `ENG_REVIEW.md` draft, then split.

## Backward Compatibility

- Never delete a section heading from `design.md` even when the section is
  intentionally empty; leave `TODO` in place so downstream reviewers and the
  validator both see the gap.
- When updating an existing `design.md`, keep the original section numbering
  and add new content as sub-sections (e.g., `## 5.4 New Reconciliation
  Stream`) rather than renumbering, to preserve links from `tasks.md` and PRs.