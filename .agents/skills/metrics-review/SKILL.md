---
name: metrics-review
version: 0.2.0
stability: beta
last_reviewed: 2026-07-07
owner: platform-finance
related_skills:
  - office-hours-finance
  - business-review
  - eng-review-finance
  - release-review
  - autoplan-finance
description: metrics governance and data definition workflow for financial systems, reporting dashboards, operations analytics, risk metrics, and finance reports. use when a user needs to define, review, reconcile, or validate financial metrics, formulas, data sources, owners, refresh frequency, time dimensions, currency handling, precision, rounding, cut-off, settlement, snapshot-vs-transaction semantics, and consistency with finance or risk definitions. read-only, do not write code, do not modify files, do not create pull requests.
---

# Metrics Review

Use this skill to define or review financial metrics **before** they are wired into dashboards, reports, APIs, admin tools, data marts, or decision workflows. This skill is a **read-only metrics governance reviewer**: it emits a structured `METRICS_SPEC.md` (metric-registry-first) and never writes code or SQL beyond illustrative snippets inside the spec.

## When To Use

- The user asks for a new metric, dashboard, report, KPI, or risk indicator on a financial admin / operations / analytics / risk / finance system.
- The user asks to reconcile / validate / align an existing metric with finance, risk, or regulatory definitions.
- `eng-review-finance` or `autoplan-finance` invokes this skill because the target design depends on undefined metrics.

## When NOT To Use

- The user wants a SQL query, ETL patch, dashboard code, or migration ── produce the metric definition first, then hand off to engineering.
- The metric is purely presentational (label, color, sort order) with no aggregation, source mapping, or business semantics.
- The requirement is non-financial and has no currency / cut-off / precision / regulatory concern.

## Hard Rules

- Do not only ask how to query the data; first define what the number means in business terms.
- Do not approve a metric without a business definition, formula, unit, time dimension, refresh frequency, source mapping, and owner.
- Treat financial amount, currency, precision, rounding, time zone, settlement period, and snapshot-vs-transaction ambiguity as high-risk.
- Check whether the metric duplicates or conflicts with existing finance, risk, or management reporting definitions.
- If the source tables or fields are unknown, mark the metric `INCOMPLETE`.
- If finance / risk / regulatory / management-reporting definitions disagree and no owner has resolved it, mark the metric `CONFLICT`.
- Do not upgrade an `INCOMPLETE` or `CONFLICT` metric to `READY` without new information from the user or a named resolution owner.
- Do not invent business owners, technical owners, source systems, or regulatory drivers.

## Inputs

Required from caller:

- **Metric name(s) or business question** ── e.g., "客户日累计提款金额" or "为什么昨日 T+0 净流入与结算表对不上".
- **Consuming surface** ── which dashboard / report / API / decision workflow will use the metric.
- **Business context one-liner** ── who acts on this number and how.

Optional but improves quality:

- Existing metric registry entries or dashboard screenshots.
- Known source systems (payment / trading / settlement / risk / KYC ledgers).
- Finance / risk / compliance definitions to reconcile against.
- Currency, timezone, and cut-off conventions used by the requesting team.

If required inputs are missing, ask **once**, then proceed with `Status: INCOMPLETE` and list the gaps under `## 8. Open Issues`.

## Outputs

- **Primary**: a single `METRICS_SPEC.md` document following the [Output Format](#output-format) below. Return as a message; do not write to disk unless the caller explicitly asks.
- **Secondary (when OpenSpec Profile is requested)**: contents targeted at `metrics.md`. Return as a fenced code block preceded by the intended relative path. See [OpenSpec Handoff](#openspec-handoff).

## Handoff

- **Upstream**: `office-hours-finance` (business question framing), `business-review` (business decision that motivates the metric).
- **Downstream**: `eng-review-finance` (consumes the metric registry to design data flow, cut-off, precision), `release-review` (uses `## 5. Consistency Review` to gate reporting-impact risk).
- **Aggregator**: `autoplan-finance` embeds this skill's output as `## 6. Metrics` of the master `PLAN.md`.

## Workflow

Run these passes in order. A failed earlier gate blocks all later ones.

1. Restate each metric as a **business definition** in one sentence.
2. Fill the **Metric Registry** (name, definition, formula, unit, time dimension, refresh, currency, precision, cut-off, owners, status).
3. Fill the **Source Mapping** (system.table.field, transformation, filters).
4. Fill **Financial Correctness** (currency, FX source + timestamp, precision, rounding, timezone, cut-off, settlement / value date, snapshot-vs-transaction, dedup, inclusion / exclusion of test / frozen / closed / internal / abnormal accounts).
5. Run **Consistency Review** against finance / risk / regulatory / management reporting and existing dashboard aliases.
6. Confirm **Ownership** (business owner + technical owner, both required).
7. List **Known Limitations** and **Open Issues**.
8. Set **Status** per each metric using the Status Model below.

## Review Gates

Ordered; a failed earlier gate blocks all later ones.

| Gate | Trigger | Status Effect |
|---|---|---|
| Definition Gate | Missing business definition, formula, unit, time dimension, or refresh frequency | `INCOMPLETE: definition missing` |
| Source Gate | Source system, table, or field mapping unknown for any registry row | `INCOMPLETE: source mapping missing` |
| Financial Correctness Gate | Currency, FX source, precision, rounding, timezone, cut-off, or settlement semantics undefined for a monetary metric | `INCOMPLETE: financial correctness undefined` |
| Ownership Gate | Business owner or technical owner missing | `INCOMPLETE: owner missing` |
| Consistency Gate | Existing finance / risk / regulatory / management-reporting definition disagrees and no resolution owner | `CONFLICT` |

## Status Model

Per-metric status:

- `READY` ── all gates passed, all owners named, all financial-correctness fields set, no unresolved conflict.
- `INCOMPLETE` ── one or more gates failed; every gap must appear in `## 8. Open Issues`.
- `CONFLICT` ── an authoritative source disagrees; the assigned resolution owner and target decision date must appear in `## 5. Consistency Review`.

Alignment with sibling skills:

| Metrics Review Status | Maps to `autoplan-finance` Plan Status | Effect on `release-review` Decision |
|---|---|---|
| `READY`      | `APPROVE`                | May contribute to `GO` or `GO WITH RISK` |
| `INCOMPLETE` | `HOLD`, `BLOCKED`        | Forces `NO GO` on any release that surfaces the metric |
| `CONFLICT`   | `HOLD`, `BLOCKED`        | Forces `NO GO` until reconciliation |

## Output Format

```markdown
# Metrics Spec

## Status
READY / INCOMPLETE / CONFLICT

<!-- If this document covers multiple metrics, Status here reflects the WORST
     status across the registry. Per-metric status lives in the Metric Registry
     table below. -->

## 1. Context

## 2. Metric Registry
| Metric | Business Definition | Formula | Unit | Time Dim | Refresh | Currency | Precision | Rounding | Cut-off | Business Owner | Technical Owner | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|

## 3. Source Mapping
| Metric | System | Table | Field | Transformation | Filter (test / frozen / closed / internal / abnormal excluded?) | Notes |
|---|---|---|---|---|---|---|

## 4. Financial Correctness
| Metric | Currency + Base Reporting Currency | FX Source | FX Conversion Timestamp | Precision (decimals) | Rounding Mode | Timezone | Business Day Cut-off | Settlement / Value Date | Snapshot vs Transaction | Dedup Rule |
|---|---|---|---|---|---|---|---|---|---|---|

## 5. Consistency Review
| Metric | Existing Similar Metric | Finance Definition | Risk Definition | Regulatory Definition | Management Reporting | Conflict? | Resolution Owner | Target Decision Date |
|---|---|---|---|---|---|---|---|---|

## 6. Ownership
| Metric | Business Owner | Technical Owner | Escalation Path |
|---|---|---|---|

## 7. Known Limitations

## 8. Open Issues
| Issue | Impact | Owner | Required Decision | Target Date |
|---|---|---|---|---|
```

Use `references/metrics-spec-template.md` when you need to hand the raw template to the user.

## References

- `references/metrics-spec-template.md` ── the full raw template mirroring [Output Format](#output-format).
- `references/metrics-governance-rules.md` ── detailed rules for definition, financial correctness, consistency, and blockers.
- `references/openspec-metrics-mapping.md` ── how to place metric spec output into an OpenSpec-lite TOML repo.

## OpenSpec Handoff

When this skill runs **inside** `autoplan-finance`, its output populates `## 6. Metrics` of `PLAN.md`. No file emission is needed.

When this skill runs **standalone** and the caller asks for OpenSpec output, emit `metrics.md` under `openspec/specs/<capability>/` (long-lived spec, not change-scoped):

```text
openspec/
  specs/
    <capability>/
      metrics.md
```

Full rules in `references/openspec-metrics-mapping.md`.

## Validator

Structural coverage of an emitted `METRICS_SPEC.md` can be verified with:

```bash
python scripts/validate_metrics_review.py <path-to-METRICS_SPEC.md>
python scripts/validate_metrics_review.py <path-to-METRICS_SPEC.md> --strict
python scripts/validate_metrics_review.py <path-to-METRICS_SPEC.md> --json
```

The validator checks that all required (heading-level, heading-title) pairs are present as real Markdown headings (skipping fenced code blocks) and, with `--strict`, that they appear in the required order. It tolerates a leading UTF-8 BOM.

## Examples

### Good invocation ── produces `READY`

> User: 请帮我定义「客户日累计提款金额」这个指标，用于风控每日早会看板。

The skill:

1. Restates in one sentence: 单个客户在自然日（UTC+8）内所有已成功结算的提款金额之和，以客户账户币种记账，跨币种折算按 T-1 收盘中行牌价。
2. Fills Metric Registry: unit=金额、Time Dim=natural day、Refresh=每 15 分钟、Currency=账户币种、Precision=2、Rounding=half-up、Cut-off=T+0 03:00 UTC+8。
3. Source Mapping: `settlement.withdraw_daily.amount_ccy`，filter 已排除内部账户与冻结账户。
4. Financial Correctness: FX source=中行牌价 T-1 close；snapshot-vs-transaction=transaction；dedup=按 `withdraw_id`。
5. Consistency Review: 与财务表 `finance.withdraw_daily.amount` 一致；与风控现有「提款金额」定义一致；无 conflict。
6. Ownership: 业务=风控数据组王 X；技术=数据平台组李 Y。
7. Status: `READY`.

### Anti-pattern ── must be blocked

> User: 我们看板上「净流入」和结算系统对不上，帮我改一下。

The skill must **not** propose a SQL fix. Correct behavior:

- Return `Status: CONFLICT`.
- Fill `## 5. Consistency Review` with both definitions (dashboard side vs settlement side), tag the discrepancy (dedup / cut-off / snapshot-vs-transaction / currency), and assign a resolution owner + target date.
- Route back to `office-hours-finance` if the business owner is unclear.

## Language & Style

- Section headings, table headers, and status enums: **English** (stable, machine-parseable, validator-friendly).
- Metric names, definitions, cell content: match the user's language (this repo family defaults to 中文正文).
- Never conflate `READY` (metric governance) with `GO` (release) ── they live on different tracks.
- Use `TODO` markers for uncertainties instead of guessing.
- Never delete required section headings, even when empty; leave `TODO` in place so gaps stay visible to the validator and reviewers.
- Prefer tables over prose for registry, source mapping, financial correctness, consistency review, and open issues.