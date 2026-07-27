# OpenSpec-lite TOML Mapping for Office Hours Finance

> Explains how to place the output of `office-hours-finance` inside a host
> repository that uses the OpenSpec-lite TOML workflow (see `AGENTS.md` in
> the tangram-trade-mt5 family of repos).

## Two Layers

1. **Logical profile** ── what this skill emits:
   - `task.md` (always, when running standalone)
   - Optionally seed `task.toml` and initial `tasks.md` rows

2. **On-disk location** ── decided by the host repo's OpenSpec variant.

## Problem Briefs Are Change-Scoped

A problem brief is the *intake* for a specific change. It belongs under
`openspec/changes/<change-id>/`, not under `openspec/specs/`.

```text
openspec/
  changes/
    <change-id>/
      task.toml          # machine-readable intake header
      task.md            # produced here by office-hours-finance
      tasks.md           # seeded here from Unknowns rows with Blocks Next Step = yes
      design.md          # produced later by eng-review-finance
      release.md         # produced later by release-review
  specs/
    <capability>/
      metrics.md         # produced by metrics-review, not here
      permissions.md     # produced by eng-review-finance
      audit.md           # produced by eng-review-finance
```

## Section-to-File Mapping

| Output Format section | Target location | Notes |
|---|---|---|
| `## Recommendation` | `task.md` head + `task.toml` `[intake].recommendation` | Enum: `PROCEED` / `HOLD` / `REDUCE` / `NEEDS_VALIDATION`. |
| `Next:` line | `task.toml` `[intake].next_step` | Enum: `business-review` / `metrics-review` / `eng-review-finance` / `release-review` / `manual-validation`. |
| `## 1. Problem` | `task.md` | 1:1. Keep sub-sections 1.1 / 1.2 / 1.3 / 1.4. |
| `## 2. Decision Flow` | `task.md` | 1:1. Every column must be filled. |
| `## 3. Stakeholders` | `task.md` | 1:1. Every row that says "Operational Owner? yes" becomes a required approver on `task.toml`. |
| `## 4. Scope` | `task.md` | 1:1. Sub-sections 4.1 / 4.2 / 4.3 / 4.4. Non-Goals must be preserved verbatim through downstream skills. |
| `## 5. Unknowns` | `task.md` + `tasks.md` | Every row with `Blocks Next Step? yes` becomes a task in `tasks.md`; others become follow-ups. |
| `## 6. Next Review` | `task.toml` `[intake].next_review_date` | ISO date. |

## Change-ID Naming

Follow the host repo convention. In the tangram-trade-mt5 family this is:

```text
<yyyy-mm-dd>-<kebab-topic>
```

Example: `2026-07-07-customer-freeze-workflow-intake`.

For intake-only changes that may not proceed, add an `-intake` suffix so it
is clear the change may be closed as `HOLD` without a code change.

## `task.toml` Seed Example

```toml
[intake]
skill = "office-hours-finance"
recommendation = "PROCEED"           # PROCEED | HOLD | REDUCE | NEEDS_VALIDATION
next_step = "business-review"
next_review_date = "2026-07-14"

[intake.stakeholders]
operational_owner = "risk-desk-lead"  # required person, not team

[intake.scope]
must_have_count = 3
non_goals_recorded = true
```

## Language Rule

- File names, section headings, table headers, recommendation and next-step
  enums: **English**.
- Body content: match user language (中文正文 default in this repo family).

## Validator Note

The included `scripts/validate_office_hours.py` validates a **single**
`task.md` (or `Problem Brief` draft) file against the Output Format
headings. It does not walk the whole `openspec/` tree.

## Backward Compatibility

- Never delete a heading from `task.md` even when the section is
  intentionally empty; leave `TODO` in place so the validator and downstream
  reviewers both see the gap.
- Non-Goals are treated as **contractual** ── they may only be removed by a
  new intake round, not by silently dropping them from a later `design.md`.
- If the recommendation changes (`HOLD` → `PROCEED`), append a
  `## 6.1 Re-intake <yyyy-mm-dd>` sub-section explaining what new information
  changed the decision; do not rewrite the original brief.