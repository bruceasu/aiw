# OpenSpec-lite TOML Mapping for Metrics Review

> Explains how to place the output of `metrics-review` inside a host
> repository that uses the OpenSpec-lite TOML workflow (see `AGENTS.md` in
> the tangram-trade-mt5 family of repos).

## Two Layers

1. **Logical profile** ── what this skill emits:
   - `metrics.md` (always, when running standalone)

2. **On-disk location** ── decided by the host repo's OpenSpec variant.

## Metrics Are Long-Lived Specs, Not Change Docs

A metric definition survives many changes. It belongs under
`openspec/specs/<capability>/metrics.md`, **not** under
`openspec/changes/<change-id>/`.

Change docs may reference metrics by name, but should not re-define them.

```text
openspec/
  changes/
    <change-id>/
      task.toml
      task.md
      tasks.md
      design.md          # may reference metrics from specs/
  specs/
    <capability>/
      metrics.md         # produced here by metrics-review
      permissions.md     # produced by eng-review-finance
      audit.md           # produced by eng-review-finance
```

## Section-to-File Mapping

| Output Format section | Target location | Notes |
|---|---|---|
| `## Status` | `metrics.md` head | Worst status across the registry. |
| `## 1. Context` | `metrics.md` | 1:1. |
| `## 2. Metric Registry` | `metrics.md` | 1:1. Canonical registry table. |
| `## 3. Source Mapping` | `metrics.md` | 1:1. |
| `## 4. Financial Correctness` | `metrics.md` | 1:1. |
| `## 5. Consistency Review` | `metrics.md` | 1:1. Any `CONFLICT` row here must have a resolution owner + target date before the metric can leave `CONFLICT`. |
| `## 6. Ownership` | `metrics.md` | 1:1. |
| `## 7. Known Limitations` | `metrics.md` | 1:1. |
| `## 8. Open Issues` | `metrics.md` + `openspec/changes/<change-id>/tasks.md` | Each open issue becomes both an entry in the checklist (change-scoped) and a bullet in `metrics.md` so it stays visible in spec review. |

## Capability Naming

Pick the smallest stable business capability that owns the metric family. In
this repo family, examples are:

- `customer-fund-review` ── 客户资金审查后台指标（余额、流水、异常）
- `withdraw-monitoring` ── 提款监控相关指标
- `settlement-reconciliation` ── 结算对账相关指标
- `risk-exposure` ── 风控敞口相关指标

Avoid using change-ids or dashboard names as capability names ── they are too
short-lived.

## Change-ID Reference

When `metrics-review` runs as part of a change (via `autoplan-finance` or on
its own), reference the change from `metrics.md`:

```markdown
> Introduced by change: 2026-07-07-daily-withdraw-summary
> Modified by change: 2026-08-01-add-fx-conversion
```

Do **not** duplicate the metric definition inside `openspec/changes/<id>/`.

## Language Rule

- File names, section headings, table headers, status enums: **English**.
- Metric names, definitions, cell content: match user language (中文正文
  default in this repo family).

## Validator Note

The included `scripts/validate_metrics_review.py` validates a **single**
`METRICS_SPEC.md` (or `metrics.md`) file against the Output Format headings.
It does not walk the whole `openspec/` tree.

## Backward Compatibility

- Never delete a heading from `metrics.md` even when the section is
  intentionally empty; leave `TODO` in place so the validator and downstream
  reviewers both see the gap.
- When adding new metrics, append rows to the existing tables; do not
  reformat historical rows.
- When a metric definition changes, keep the old row with a `deprecated`
  flag in the Notes column and add a new row with the new formula. Never
  silently rewrite a formula ── downstream reconciliation depends on it.