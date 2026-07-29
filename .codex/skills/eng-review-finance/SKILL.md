---
name: eng-review-finance
version: 0.2.0
stability: beta
last_reviewed: 2026-07-07
owner: platform-finance
related_skills:
  - office-hours-finance
  - business-review
  - metrics-review
  - release-review
  - autoplan-finance
description: architecture review workflow for financial admin platforms, operations systems, analytics dashboards, reporting systems, and data pipelines. use when a user needs technical design review before implementation, including system boundaries, module responsibilities, data contracts, data flow, permissions, auditability, failure modes, observability, and testing strategy. read-only, do not write code, do not modify files, do not create pull requests.
---

# Engineering Review Finance

Use this skill to review or draft a financial-system technical design **before implementation**. This skill is a **read-only architecture reviewer**: it emits a structured `ENG_REVIEW.md` and never writes code, migrations, or PRs.

## When To Use

- The user describes a financial admin / operations / reporting / risk / analytics system and asks for a technical review, architecture review, design review, or "is this design ready to build".
- The user asks for a boundary / module / data-flow / permission / audit / failure-mode / observability / testing bundle.
- `autoplan-finance` invokes this skill as its engineering-review step.

## When NOT To Use

- The user wants code, a schema migration, a PR, or a deployment.
- Metrics are undefined ── invoke `metrics-review` first, then return here.
- Business value is unclear ── invoke `business-review` first.
- The requirement is non-financial and has no permission / audit governance concern.
- The user only needs the release-gate checklist ── invoke `release-review` directly.

## Hard Rules

- Do not write code, implementation patches, database migrations, deployment scripts, or pull requests.
- Do not modify repository files, even under `openspec/`.
- Do not invent missing data contracts, permissions, or audit requirements ── mark them as `TODO` and downgrade `Status`.
- Do not upgrade an `INCOMPLETE` or `HIGH RISK` review to `READY` without new information from the user.
- If metrics are undefined, route to `metrics-review` first; do not fabricate metric definitions.
- If view / export / edit / approval / field-level / data-scope permissions are undefined for sensitive data, mark the design `INCOMPLETE`.
- If audit for sensitive actions does not capture actor / timestamp / action / before value / after value / reason / request-id or trace-id / retention, mark the design `INCOMPLETE`.
- Prefer explicit boundaries, data contracts, and failure modes over generic architecture prose.

## Inputs

Required from caller:

- **Requirement one-liner** ── what business capability is being built.
- **Target system(s)** ── which admin platform, service, or module.
- **Upstream / downstream systems** ── at minimum the names of data producers and consumers.

Optional but improves quality:

- Existing OpenSpec artifacts (`task.md`, prior `design.md`, spec deltas).
- Metric definitions from `metrics-review` (source system.table.field, owner, cut-off).
- Draft permission matrix or role list.
- Known audit / retention / regulatory constraints.
- SLA / freshness / throughput targets.

If required inputs are missing, ask **once**, then proceed with `Status: INCOMPLETE` and list what is missing under `## 13. Required Decisions Before Implementation`.

## Outputs

- **Primary**: a single `ENG_REVIEW.md` document following the [Output Format](#output-format) below. Return as a message; do not write to disk unless the caller explicitly asks.
- **Secondary (when OpenSpec Profile is requested)**: file contents targeted at `design.md` (plus `permissions.md`, `audit.md` when this skill runs standalone). Return as multiple fenced code blocks, one per file, each preceded by the intended relative path. See [OpenSpec Handoff](#openspec-handoff).

## Handoff

- **Upstream**: `office-hours-finance` (problem framing), `business-review` (business decision), `metrics-review` (metric registry).
- **Downstream**: `release-review` (release gate). Section `## 12. Release Readiness Impact` is the handoff surface ── `release-review` consumes it directly.
- **Aggregator**: `autoplan-finance` embeds this skill's output as `## 8. Architecture` + parts of `## 9`, `## 10`, `## 12` of the master `PLAN.md`.

## Workflow

Run these passes in order. Each pass has a corresponding review gate below; a failed earlier gate blocks all later ones.

1. Confirm metrics exist (else route to `metrics-review`).
2. Review **system boundary** (in-scope / out-of-scope, external dependencies).
3. Review **module responsibilities** (module → owner → responsibility → dependencies).
4. Review **data contracts** (producer / consumer / contract / backward compatibility).
5. Review **data flow** (source → ingestion → store → API → UI, freshness, quality).
6. Review **permission model** (view / export / edit / approve / field-level / data-scope).
7. Review **auditability** (actor / timestamp / action / before / after / reason / trace / retention).
8. Review **failure modes** (impact / detection / mitigation).
9. Review **observability** (logs / metrics / alerts / tracing / freshness / quality).
10. Review **testing strategy** (unit / integration / e2e / data / permission / audit / migration + rollback).
11. Assess **release readiness impact** (schema, data, permission, audit, rollback, monitoring).
12. List **required decisions before implementation**.

## Review Gates

Ordered from earliest to latest; a failed earlier gate blocks all later ones.

| Gate | Trigger | Status Effect |
|---|---|---|
| Metrics Gate | Metric registry missing or any metric lacks definition, source, or owner | `INCOMPLETE`; route to `metrics-review` first |
| Boundary Gate | `## 2. System Boundary` empty, or in/out-of-scope unclear, or external dependencies unlisted | `INCOMPLETE: unclear system boundary` |
| Data Contract Gate | Producer, consumer, contract, or backward compatibility risk unspecified for any cross-team interface | `INCOMPLETE: missing data contract` |
| Permission Gate | View / export / edit / approve / field-level / data-scope permissions undefined for sensitive data | `INCOMPLETE: permission model undefined` |
| Audit Gate | Sensitive actions missing actor / timestamp / before / after / reason / trace / retention | `INCOMPLETE: audit coverage insufficient` |
| Failure Mode Gate | Any listed failure mode lacks detection or mitigation | `HIGH RISK: failure modes not mitigated` |
| Observability Gate | No alert on data freshness, data quality, or error rate for user-facing surfaces | `HIGH RISK: observability insufficient` |
| Testing Gate | No migration + rollback test, or no permission / audit test for sensitive actions | `HIGH RISK: testing insufficient` |
| Release Impact Gate | `## 12. Release Readiness Impact` missing schema / data / permission / audit / rollback / monitoring lines | `INCOMPLETE: release impact not assessed` |

## Status Model

Two status axes ── both must be filled:

- **Review Status** (engineering track, produced by this skill): `READY` | `INCOMPLETE` | `HIGH RISK`
- **Release Readiness** (consumed by `release-review`): `GO` | `GO WITH RISK` | `NO GO` | `NOT YET REVIEWED`

Allowed combinations:

| Review Status | Allowed Release Readiness | Alignment with `autoplan-finance` Plan Status |
|---|---|---|
| `READY`      | `GO`, `GO WITH RISK`, `NOT YET REVIEWED` | `APPROVE` |
| `INCOMPLETE` | `NOT YET REVIEWED`, `NO GO`              | `HOLD`, `BLOCKED` |
| `HIGH RISK`  | `GO WITH RISK`, `NO GO`, `NOT YET REVIEWED` | `REDUCE`, `HOLD` |

Any other combination is a bug in the review and must be fixed before returning to the caller.

## Output Format

```markdown
# Engineering Review

## Status
READY / INCOMPLETE / HIGH RISK

## 1. Context

## 2. System Boundary
| In Scope | Out of Scope | External Dependency | Owner |
|---|---|---|---|

## 3. Module Responsibilities
| Module | Responsibility | Owner | Upstream | Downstream | Backward Compatibility Risk |
|---|---|---|---|---|---|

## 4. Data Contracts
| Producer | Consumer | Contract (schema / API / topic) | Version | Compatibility Risk | Owner |
|---|---|---|---|---|---|

## 5. Data Flow
```text
Source system
  -> ingestion / ETL / stream processor
  -> warehouse / mart / operational store
  -> service API
  -> admin UI / dashboard / report
```
### 5.1 Data Flow Table
| Step | Input | Processing | Output | Owner | Failure Mode | Validation |
|---|---|---|---|---|---|---|
### 5.2 Data Freshness
| Dataset | Expected Refresh | SLA | Alert Threshold |
|---|---|---|---|
### 5.3 Data Quality
| Check | Rule | Severity | Owner |
|---|---|---|---|

## 6. Permissions
### 6.1 Roles
### 6.2 Permission Matrix
| Role | View | Export | Edit | Approve | Field-level Restriction | Data-scope Restriction |
|---|---|---|---|---|---|---|
### 6.3 Field-level Restrictions
### 6.4 Data-scope Rules

## 7. Audit Requirements
### 7.1 Audited Actions
| Action | Actor | Timestamp | Target | Before | After | Reason | Trace / Request Id | Queryable | Retention |
|---|---|---|---|---|---|---|---|---|---|
### 7.2 Retention Policy
### 7.3 Audit Query Requirements

## 8. Failure Modes
| Failure Mode | Impact | Detection | Mitigation | Owner |
|---|---|---|---|---|

## 9. Observability
| Signal | Tool | Threshold | Alert Channel | Owner |
|---|---|---|---|---|

## 10. Testing Strategy
| Layer | Scope | Tool | Coverage Target | Owner |
|---|---|---|---|---|

## 11. Risks

## 12. Release Readiness Impact
- Schema or migration risk:
- Data impact:
- Permission impact:
- Audit impact:
- Rollback considerations:
- Monitoring required before release:
- Handoff to `release-review`: GO / GO WITH RISK / NO GO / NOT YET REVIEWED

## 13. Required Decisions Before Implementation
```

Use `references/architecture-review-template.md` when you need to hand the raw template to the user.

## References

- `references/architecture-review-template.md` ── the full raw template mirroring [Output Format](#output-format).
- `references/data-flow-template.md` ── data-pipeline-specific tables (freshness, quality, backfill).
- `references/permission-audit-checklist.md` ── sensitive-action list, permission blockers, audit fields.
- `references/openspec-eng-review-mapping.md` ── how to map this review into an OpenSpec-lite TOML repo.

## OpenSpec Handoff

When this skill runs **inside** `autoplan-finance`, its output is embedded into `PLAN.md` sections 8/9/10/12. No file emission is needed.

When this skill runs **standalone** and the caller asks for OpenSpec output, emit these logical files:

```text
design.md         # sections 1..5, 8..11 from Output Format
permissions.md    # section 6
audit.md          # section 7
release.md        # section 12 only when caller also wants a release stub
```

Host-project mapping (OpenSpec-lite TOML variant, matching `AGENTS.md` conventions in this repo family):

| Logical file | On-disk location |
|---|---|
| `design.md` | `openspec/changes/<change-id>/design.md` |
| `permissions.md` | `openspec/specs/<capability>/permissions.md` |
| `audit.md` | `openspec/specs/<capability>/audit.md` |
| `release.md` | `openspec/changes/<change-id>/release.md` |

Full rules in `references/openspec-eng-review-mapping.md`.

## Validator

Structural coverage of an emitted `ENG_REVIEW.md` can be verified with:

```bash
python scripts/validate_eng_review.py <path-to-ENG_REVIEW.md>
python scripts/validate_eng_review.py <path-to-ENG_REVIEW.md> --strict
python scripts/validate_eng_review.py <path-to-ENG_REVIEW.md> --json
```

The validator checks that all required (heading-level, heading-title) pairs are present as real Markdown headings (skipping fenced code blocks) and, with `--strict`, that they appear in the required order. It tolerates a leading UTF-8 BOM.

## Examples

### Good invocation ── produces `READY / GO WITH RISK`

> User: 我需要在客户资金审查后台加一个「客户按日累计提款汇总视图」，供风控每天早会查看。

The skill:

1. Confirms metrics exist (`metrics-review` already produced 客户日累计提款金额 with source `settlement.withdraw_daily.amount_ccy`, owner 风控数据组, cut-off `T+0 03:00 UTC+8`).
2. Boundary: in-scope = 后台读视图 + 数据服务；out-of-scope = 提款流程本身。
3. Data contract: 消费 `dwd_withdraw_daily` 视图，schema owner = 数据平台组，向前兼容 (只加列)。
4. Permission matrix: 风控 = View + Export；客服 = View 且字段级屏蔽账号；运营 = 无权限。
5. Audit: 导出动作记录 actor / 客户号 / 时间 / trace-id / 保留 7 年。
6. Failure modes: T+0 03:00 数据未到 → 页面顶部红条 + 值班告警；金额币种混淆 → 单元测试 + 页面显式币种列。
7. Testing: 权限测试覆盖三角色；导出审计断言。
8. Emits `ENG_REVIEW.md` with `Status: READY`, Release Readiness `GO WITH RISK` (待补一个跨月对账用例)。

### Anti-pattern ── must be blocked

> User: 帮我做一个「客户交易异常检测面板」的架构方案。

The skill must **not** return `Status: READY`. Correct behavior:

- If "异常" 的判定规则 / 数据源 / 阈值未定义 → return `Status: INCOMPLETE: metric registry missing`, route to `metrics-review`.
- If 风控可见的字段与客服可见的字段没有说明 → return `Status: INCOMPLETE: permission model undefined`.
- List every missing decision under `## 13. Required Decisions Before Implementation` and stop.

## Language & Style

- Section headings: English (stable, machine-parseable, validator-friendly).
- Body content: match the user's language (this repo family defaults to 中文正文).
- Never conflate `READY` (engineering) and `GO` (release) semantics ── keep them on separate lines.
- Use `TODO` markers for uncertainties instead of guessing.
- Never delete required section headings, even when empty; leave `TODO` in place so gaps stay visible to the validator and reviewers.
- Prefer tables over prose for boundaries, contracts, permissions, audit, and failure modes.