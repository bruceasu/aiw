---
name: release-review
version: 0.2.0
stability: beta
last_reviewed: 2026-07-07
owner: platform-finance
related_skills:
  - office-hours-finance
  - business-review
  - metrics-review
  - eng-review-finance
  - autoplan-finance
description: release gate review workflow for financial admin systems, operations platforms, reporting tools, risk dashboards, data pipelines, permission changes, schema migrations, and production deployments. use before launch to check scope, schema and migration risk, data impact, metrics and reporting impact, permission impact, audit impact, rollback strategy, observability and operations, open risks, and final GO / GO WITH RISK / NO GO decision. read-only, do not write code, do not modify files, do not create pull requests.
---

# Release Review

Use this skill to review whether a financial admin, operations, reporting, risk, finance, analytics, dashboard, or data-pipeline change is safe to release. This skill is a **read-only release gate**: it emits a structured `RELEASE_REVIEW.md` and never writes code, migrations, or deployment scripts.

## When To Use

- The user is preparing to release a change that touches money, limits, fees, customer data, permissions, schema, backfills, dashboards, reports, or risk indicators.
- The user asks for a launch checklist, go / no-go, or release readiness assessment.
- `eng-review-finance` has produced `## 12. Release Readiness Impact` and now needs the gate decision.
- `autoplan-finance` invokes this skill as its release-readiness step.

## When NOT To Use

- The change is a documentation-only or internal-code-only change with no schema, data, permission, audit, dashboard, or user-facing impact.
- The user wants a rollback script, migration script, or feature-flag code ── this skill only assesses whether the strategy exists; implementation belongs to engineering.
- Metrics are undefined ── route to `metrics-review` first.
- Architecture is unclear ── route to `eng-review-finance` first.

## Hard Rules

- Do not approve a release without explicit rollback coverage (application + schema + data + feature flag).
- Do not approve a release when permission impact is unknown.
- Do not approve a release when audit impact is missing for sensitive actions.
- Treat schema migrations, backfills, permission changes, export features, customer asset views, finance reports, and risk dashboards as high-risk by default.
- Use `NO GO` when critical data, permission, audit, rollback, or monitoring gaps remain unresolved.
- Do not write implementation code, migration scripts, or deployment scripts.
- Do not modify repository files, even under `openspec/`.
- Do not upgrade a `NO GO` decision to `GO` or `GO WITH RISK` without new information from the user or an explicitly named risk owner.
- Do not invent owners, monitoring, or rollback procedures ── if unknown, they are gaps.

## Inputs

Required from caller:

- **Release scope one-liner** ── what is being launched.
- **Affected systems** ── services, jobs, tables, dashboards, APIs, admin screens.
- **Target launch window** ── when the release is planned.

Optional but improves quality:

- `eng-review-finance` output (`ENG_REVIEW.md` or `design.md`, especially section 12).
- `metrics-review` output (`METRICS_SPEC.md`), particularly for any dashboard or report launch.
- Migration script summary (DDL, indexes, backfill plan).
- Rollback plan draft.
- Monitoring / alert configuration draft.
- Regulatory or compliance sign-off status.

If required inputs are missing, ask **once**, then proceed with `Decision: NO GO` and list the gaps under `## 10. Open Risks` with severity `blocker`.

## Outputs

- **Primary**: a single `RELEASE_REVIEW.md` document following the [Output Format](#output-format) below. Return as a message; do not write to disk unless the caller explicitly asks.
- **Secondary (when OpenSpec Profile is requested)**: contents targeted at `release.md` under `openspec/changes/<change-id>/`. Return as a fenced code block preceded by the intended relative path. See [OpenSpec Handoff](#openspec-handoff).

## Handoff

- **Upstream**: `eng-review-finance` (consumes its `## 12. Release Readiness Impact` verbatim), `metrics-review` (consumes any `CONFLICT` status), `business-review` (residual business risk owner).
- **Downstream**: operations / SRE for launch execution; risk / compliance for regulator-facing surfaces.
- **Aggregator**: `autoplan-finance` embeds this skill's output as `## 11. Release Readiness` of the master `PLAN.md`.

## Workflow

Run these passes in order. A failed earlier gate forces `NO GO` regardless of later sections.

1. Confirm **scope** and affected systems.
2. Review **schema and migration** risk (DDL, indexes, lock, backward/forward compat, backfill).
3. Review **data impact** (historical, real-time, recalculation, idempotency, reconciliation).
4. Review **metrics and reporting impact** (changed definitions, cut-off, currency, precision).
5. Review **permission impact** (view / export / edit / approve / field-level / data-scope, role defaults).
6. Review **audit impact** (actor / timestamp / before / after / reason / trace / retention for new sensitive actions).
7. Review **rollback plan** (application, schema, data, feature flag, manual recovery, owner, verification).
8. Review **observability and operations** (release-health dashboard, freshness / quality / permission-error / audit-write alerts, on-call).
9. List **open risks** with severity and blocker flag.
10. Produce a **final recommendation** (GO / GO WITH RISK / NO GO) with the named risk owner.

## Review Gates

Ordered; any failed gate forces `NO GO` unless the row explicitly says otherwise.

| Gate | Trigger | Decision Effect |
|---|---|---|
| Scope Gate | Affected systems / customer-facing flag / feature-flag availability undefined | `NO GO` until scope is bounded |
| Migration Gate | Schema change lacks backward compatibility, migration lock analysis, or backfill idempotency | `NO GO` (or `GO WITH RISK` if manual recovery is credible and owner is named) |
| Data Reconciliation Gate | Financial or reporting data change lacks reconciliation plan or tolerance | `NO GO` |
| Metrics Gate | Any surfaced metric has `INCOMPLETE` or `CONFLICT` status in `metrics-review` | `NO GO` |
| Permission Gate | New view / export / edit / approve permission not defined, or default access unclear | `NO GO` |
| Audit Gate | New sensitive action lacks before + after value capture, reason, trace, or retention | `NO GO` |
| Rollback Gate | Missing rollback for application, schema, data, or feature flag | `NO GO` |
| Observability Gate | Missing alert on data freshness, data quality, permission errors, or audit write failures for the changed surface | `NO GO` (or `GO WITH RISK` if a credible manual detection path is documented) |
| Ownership Gate | No named owner for release, rollback, or post-release monitoring | `NO GO` |

## Decision Model

- `GO` ── all gates passed; critical paths, schema, data, metrics, permissions, audit, rollback, and monitoring are confirmed with named owners.
- `GO WITH RISK` ── risks are known and bounded; recovery is credible; monitoring will detect failure quickly; a business owner accepts the residual risk; a named engineering owner is on standby.
- `NO GO` ── one or more gates failed; the release must not proceed until the failing gate is resolved.

Alignment with sibling skills:

| Release Review Decision | Requires from `eng-review-finance` Review Status | Consistent with `autoplan-finance` Plan Status |
|---|---|---|
| `GO`          | `READY`                     | `APPROVE` |
| `GO WITH RISK`| `READY` or `HIGH RISK`      | `APPROVE` or `REDUCE` |
| `NO GO`       | any                         | `HOLD`, `BLOCKED`, or `REDUCE` |

Any other combination is a bug in the review and must be fixed before returning to the caller.

## Output Format

```markdown
# Release Review

## Decision
GO / GO WITH RISK / NO GO

## 1. Scope
- Feature or change summary:
- Affected services / jobs / tables / dashboards / APIs / admin screens:
- Customer-facing / internal-only / regulatory impact:
- Feature flag or staged rollout availability:
- Target launch window:

## 2. Release Checklist
| Area | Status | Evidence | Risk | Required Action | Owner |
|---|---|---|---|---|---|
| Schema and migration        |  |  |  |  |  |
| Backfill                    |  |  |  |  |  |
| Data reconciliation         |  |  |  |  |  |
| Metrics and reporting       |  |  |  |  |  |
| Permissions                 |  |  |  |  |  |
| Audit                       |  |  |  |  |  |
| Rollback                    |  |  |  |  |  |
| Observability               |  |  |  |  |  |
| On-call / ownership         |  |  |  |  |  |

## 3. Schema and Migration
- DDL changes:
- Index changes:
- Migration lock risk:
- Backward compatibility (old app reads new schema):
- Forward compatibility (new app reads old schema):
- Migration verification query:

## 4. Data Impact
- Historical data impact:
- Real-time data impact:
- Recalculation / backfill need:
- Backfill idempotency:
- Data freshness expectation:
- Reconciliation plan + tolerance:

## 5. Metrics and Reporting Impact
- Changed metric definitions (link to `metrics-review` entries):
- Changed aggregation / dedup logic:
- Time-window / cut-off changes:
- Currency / precision / rounding / settlement changes:
- Finance / risk / management-reporting conflicts:

## 6. Permission Impact
- New / changed view permission:
- New / changed export permission:
- New / changed edit permission:
- New / changed approval permission:
- Field-level restriction changes:
- Data-scope restriction changes:
- Role migration / default access risk:

## 7. Audit Impact
- Sensitive actions introduced or changed:
- Actor / timestamp / action / target captured:
- Before + after values captured:
- Reason captured:
- Request ID or trace ID captured:
- Retention policy:
- Audit queryability (who queries, latency budget, filters):

## 8. Rollback Plan
- Application rollback:
- Schema rollback:
- Data rollback:
- Feature flag fallback:
- Manual recovery procedure:
- Rollback owner:
- Rollback verification (how do we prove rollback succeeded):

## 9. Observability and Operations
- Release health dashboard:
- Logs / metrics / alerts / tracing:
- Data freshness checks:
- Data quality checks:
- Permission error monitoring:
- Audit write failure monitoring:
- On-call rota + first responder:

## 10. Open Risks
| Risk | Severity | Owner | Mitigation | Release Blocker? |
|---|---|---|---|---|

## 11. Final Recommendation
- Decision: GO / GO WITH RISK / NO GO
- Rationale:
- Named risk owner (business):
- Named risk owner (engineering):
- Next review date if `GO WITH RISK` or `NO GO`:
```

Use `references/release-review-template.md` when you need to hand the raw template to the user.

## References

- `references/release-review-template.md` ── the full raw template mirroring [Output Format](#output-format).
- `references/release-gate-checklist.md` ── detailed GO / GO WITH RISK / NO GO criteria.
- `references/migration-risk-template.md` ── deep dive for schema and backfill heavy releases.
- `references/openspec-release-mapping.md` ── how to map this review into an OpenSpec-lite TOML repo.

## OpenSpec Handoff

When this skill runs **inside** `autoplan-finance`, its output populates `## 11. Release Readiness` of `PLAN.md`. No file emission is needed.

When this skill runs **standalone** and the caller asks for OpenSpec output, emit `release.md` under `openspec/changes/<change-id>/`:

```text
openspec/
  changes/
    <change-id>/
      task.toml
      task.md
      tasks.md
      design.md          # from eng-review-finance
      release.md         # produced here
```

Full rules in `references/openspec-release-mapping.md`.

## Validator

Structural coverage of an emitted `RELEASE_REVIEW.md` can be verified with:

```bash
python scripts/validate_release_review.py <path-to-RELEASE_REVIEW.md>
python scripts/validate_release_review.py <path-to-RELEASE_REVIEW.md> --strict
python scripts/validate_release_review.py <path-to-RELEASE_REVIEW.md> --json
```

The validator checks that all required (heading-level, heading-title) pairs are present as real Markdown headings (skipping fenced code blocks) and, with `--strict`, that they appear in the required order. It tolerates a leading UTF-8 BOM.

## Examples

### Good invocation ── produces `GO WITH RISK`

> User: 我要上线「客户日累计提款汇总视图」，`eng-review-finance` 已给 `READY`，请做发布评审。

The skill:

1. Scope: 只加一个视图 + 只读 API + 后台一个页面；feature flag=on 面向风控 3 人。
2. Schema: 只加视图，无 DDL 锁风险；backward + forward compatible。
3. Data: 消费现有 `dwd_withdraw_daily`；无 backfill；对账口径每天 03:30 vs `settlement.withdraw_daily`，差异阈值 0。
4. Metrics: `metrics-review` `READY`。
5. Permission: 新增 View + Export，仅风控角色；客服显式拒绝；default-deny for others。
6. Audit: 导出动作 actor + trace + 保留 7 年。
7. Rollback: 关闭 feature flag 即回退；无数据回滚需求。
8. Observability: 03:30 数据未到告警 + 权限拒绝率告警。
9. Open risk: 跨月对账用例未覆盖 → 严重度=medium，非 blocker，值班人接手。
10. Decision: `GO WITH RISK`，业务风险 owner=风控组长 王 X，工程 owner=数据平台组 李 Y。

### Anti-pattern ── must be blocked

> User: 明天上线一个后台改动，加了个「一键调整客户余额」按钮。

The skill must **not** return `GO` or `GO WITH RISK`. Correct behavior:

- Decision: `NO GO`.
- Gates that failed: Audit Gate (before + after 未定义)、Permission Gate (edit vs approve 未分离)、Rollback Gate (数据回滚未定义)、Ownership Gate (风险 owner 缺失)。
- List each gap under `## 10. Open Risks` with severity `blocker`, and route back to `eng-review-finance` for redesign.

## Language & Style

- Section headings, table headers, decision enums, severity enums: **English** (stable, machine-parseable, validator-friendly).
- Body content: match the user's language (this repo family defaults to 中文正文).
- Never conflate `GO` (release) with `APPROVE` (plan) or `READY` (engineering / metrics) ── they live on separate lines.
- Use `TODO` markers for uncertainties instead of guessing.
- Never delete required section headings, even when empty; leave `TODO` in place so gaps stay visible to the validator and reviewers.
- Prefer tables over prose for release checklist, open risks, and any per-area status.