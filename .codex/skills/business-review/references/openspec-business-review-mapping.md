# OpenSpec-lite TOML Mapping for Business Review

> Explains how to place the output of `business-review` inside a host
> repository that uses the OpenSpec-lite TOML workflow (see `AGENTS.md` in
> the tangram-trade-mt5 family of repos).

## Two Layers

1. **Logical profile** ── what this skill emits:
   - Appended `## Business Review` section on `task.md`
   - Updated `[business]` block in `task.toml`
   - Follow-up rows appended to `tasks.md` for each Required Condition marked `Blocks Decision? yes`

2. **On-disk location** ── decided by the host repo's OpenSpec variant.

## Business Reviews Are Change-Scoped

A business decision is tied to a specific change. It lives under
`openspec/changes/<change-id>/`, **not** under `openspec/specs/`.

Long-lived rules that survive many decisions (e.g., "all customer-money
edits require four-eyes approval") belong in `openspec/specs/<capability>/`
and are owned by `eng-review-finance` and `metrics-review`.

```text
openspec/
  changes/
    <change-id>/
      task.toml          # [business] decision + named owner
      task.md            # append Business Review section
      tasks.md           # each Required Condition with Blocks Decision = yes becomes a task
      design.md          # produced later by eng-review-finance
      release.md         # produced later by release-review
  specs/
    <capability>/
      metrics.md         # referenced by Validation Path
      permissions.md     # referenced by Required Conditions
      audit.md           # referenced by Required Conditions
```

## Section-to-File Mapping

| Output Format section | Target location | Notes |
|---|---|---|
| `## Decision` | `task.toml` `[business].decision` + head of appended section in `task.md` | Enum: `APPROVE` / `REDUCE` / `HOLD`. |
| `## 1. Context` | `task.md` (Business Review section) | Named business owner also goes to `task.toml` `[business].owner`. |
| `## 2. Value Matrix` | `task.md` | 1:1. Keep the 6-row table intact. |
| `## 3. Cost vs Benefit` | `task.md` | 1:1. Payback horizon also goes to `task.toml` `[business].payback_months`. |
| `## 4. Scope Challenge` | `task.md` | Recommendations here may modify the upstream `office-hours-finance` Scope section ── update by appending a `## 4.1 Post-Business-Review Scope Adjustment` note; do not silently edit the original. |
| `## 5. Smaller Alternatives` | `task.md` | If any row is `reuse` and covers ≥ 80%, the decision must be `REDUCE`. |
| `## 6. Validation Path` | `task.md` + `tasks.md` | Validation metric must be defined in `openspec/specs/<capability>/metrics.md`; if missing, that becomes a task and a Required Condition. |
| `## 7. Required Conditions` | `task.md` + `tasks.md` | Every `Blocks Decision? yes` row becomes a task in `tasks.md`. |
| `## 8. Final Recommendation` | `task.toml` `[business]` block | Includes decision, next-step pointer, named owner, next review date. |

## `task.toml` Update Example

```toml
[business]
skill = "business-review"
decision = "APPROVE"                # APPROVE | REDUCE | HOLD
next_step = "metrics-review"        # metrics-review | eng-review-finance | release-review | manual-validation | back-to-office-hours
owner = "customer-service-lead-zhang-x"
payback_months = 3
next_review_date = ""               # required when decision = REDUCE or HOLD
```

## Change-ID Reference

When `business-review` runs alone on a `<change-id>` produced by
`office-hours-finance`, the change-id already exists. Do not create a new
one; append to the existing files.

If the upstream Problem Brief did not produce a change-id (e.g., ad-hoc
office hours), pick one using the host convention:

```text
<yyyy-mm-dd>-<kebab-topic>
```

Example: `2026-07-07-deposit-reconciliation-dashboard`.

## Cross-Reference Discipline

- `task.md` **references** but does not **redefine**:
  - Metric formulas (owned by `metrics.md`).
  - Permission matrix (owned by `permissions.md`).
  - Audit fields (owned by `audit.md`).
- If a business review needs a metric / permission / audit rule that does
  not yet exist, that becomes a Required Condition ── not an assumption.

## Language Rule

- File names, section headings, table headers, decision enums, strength
  enums: **English**.
- Body content: match user language (中文正文 default in this repo family).

## Validator Note

The included `scripts/validate_business_review.py` validates a **single**
`BUSINESS_REVIEW.md` (or a `task.md` extract) file against the Output Format
headings. It does not walk the whole `openspec/` tree.

## Backward Compatibility

- Never delete a heading from the Business Review section even when empty;
  leave `TODO` in place so the validator and downstream reviewers both see
  the gap.
- If a `HOLD` is later upgraded to `APPROVE`, append a
  `## 8.1 Re-review <yyyy-mm-dd>` sub-section explaining what new
  information changed the decision; do not rewrite the original section 8.
- Non-Goals inherited from `office-hours-finance` are contractual and may
  only be reversed by a new office-hours intake round, not by business
  review alone.