---
name: autoplan-finance
version: 0.2.0
stability: beta
last_reviewed: 2026-07-06
owner: platform-finance
related_skills:
  - office-hours-finance
  - business-review
  - metrics-review
  - eng-review-finance
  - release-review
description: planning orchestrator for financial product decision workflows across office hours, business review, metrics review, engineering review, and release readiness review. use when a user wants a complete plan.md for a financial admin, operations, reporting, dashboard, risk, finance, or analytics requirement. enforce decision flow, metrics governance, permissions, auditability, and release gates. generate structured planning output only. do not code, modify files, or create pull requests.
---

This Skill follows [`skills/reviewed-skill-contract.md`](../reviewed-skill-contract.md)
and [`skills/work-management.md`](../work-management.md). It is read-only and
does not create or mutate lifecycle artifacts.

# Autoplan Finance

Use this skill to generate a complete planning document (`PLAN.md`) for a financial admin, operations, reporting, finance, risk, or analytics requirement. This skill is a **read-only planning orchestrator**: it consolidates the outputs of five sibling review skills into one decision-centric plan.

## When To Use

- The user describes a financial admin / operations / reporting / risk / analytics change and asks for a plan, review, scoping, or brief.
- The user asks for a decision-flow, metrics-governance, permissions-audit, or release-readiness bundle.
- The user wants OpenSpec-compatible planning output for a finance change.

## When NOT To Use

- The user wants code, a schema migration, a PR, or a deployment.
- The user asks for a single narrow artifact (only metrics doc, only permission matrix, etc.) — invoke the corresponding sibling skill directly instead.
- The requirement is non-financial and has no metrics / permission / audit governance concern.

## Hard Rules

- Do not write code.
- Do not modify repository files, even under `openspec/`.
- Do not create pull requests, branches, or commits.
- Do not invent missing business rules, data sources, permissions, or audit requirements.
- Do not upgrade a `HOLD` / `BLOCKED` plan to `APPROVE` without new information from the user.
- If inputs are incomplete, produce a plan with `%% NEEDS_INPUT: ...` notes and a decision gate that blocks downstream sections.

## Inputs

Required from caller:

- **Requirement one-liner** — the business ask in one sentence.
- **Target system(s)** — which admin platform, service, or module.
- **Requesting stakeholder(s)** — role or team asking for the change.

Optional but improves quality:

- Existing OpenSpec artifacts or issue links.
- Known upstream / downstream systems.
- Known metrics, permissions, or audit contracts.
- Compliance / regulatory context.

If required inputs are missing, ask **once**, then proceed with `Status: BLOCKED: incomplete intake` and list what is missing under `## 14. Open Questions`.

## Completion Criteria

Return a complete `PLAN.md`-shaped response with both status axes, all required
section headings, the five review stages in order, and every missing or
unverified input represented by `%%` or a blocking status. Report whether
runtime validation or sibling Skill invocation was actually performed.

## Authorization Boundary

The Skill may read caller-provided context and return planning content. It may
not write `PLAN.md` or OpenSpec files, create or mutate Tasks, branches,
worktrees, commits, pull requests, or external publications unless the caller
explicitly requests that separate action and the appropriate lifecycle Skill
handles it.

## Outputs

- **Primary**: a single `PLAN.md` document following the [Output Format](#output-format) below. Return as a message; do not write to disk unless the caller explicitly asks.
- **Secondary (when OpenSpec Profile is requested)**: a fileset described in [OpenSpec Finance Profile](#openspec-finance-profile). Return as multiple fenced code blocks, one per file, each preceded by the intended relative path.

## Workflow

Run the planning sequence by invoking the sibling skills below in order. If a sibling skill is unavailable in the current environment, run the same conceptual pass internally and annotate the corresponding plan section with `(self-run, no sibling skill invoked)`.

1. `office-hours-finance` — clarify problem, stakeholders, decision flow, scope, unknowns.
2. `business-review` — decide `APPROVE`, `REDUCE`, or `HOLD`.
3. `metrics-review` — define metrics, formulas, sources, owners, consistency risks.
4. `eng-review-finance` — define architecture, data flow, permissions, audit, failure modes, observability, testing.
5. `release-review` — assess schema, data, metrics, permissions, audit, rollback, monitoring readiness (only when the plan is near launch).
6. Consolidate all outputs into a single `PLAN.md` using the format below.

## Decision Gates

Ordered from earliest to latest; a failed earlier gate blocks all later ones.

| Gate | Trigger | Plan Status Effect |
|---|---|---|
| Decision Flow Gate | `## 2. Decision Flow` empty or missing Actor / Sees / Decides / Acts | `BLOCKED: unclear decision flow` |
| Business Value Gate | No revenue, risk reduction, labor saving, decision efficiency, regulatory need, or CX benefit | `HOLD` or `REDUCE` |
| Metrics Gate | Any metric lacks business definition, formula, source fields, owner, currency/precision, or time-window semantics | Mark `## 6. Metrics: INCOMPLETE`, block later engineering commitments |
| Permission Gate | View / export / edit / approval / field-level / data-scope permissions undefined | Mark `## 9. Permissions: INCOMPLETE`, block release readiness |
| Audit Gate | Sensitive actions lack actor / timestamp / before / after / reason / trace id / retention | Mark `## 11. Release Readiness: BLOCKED` |
| Release Gate | Missing rollback, migration verification, monitoring, or data reconciliation | Mark `## 11. Release Readiness: NO GO` |

## Status Model

Two independent status axes:

- **Plan Status** (business track): `APPROVE` | `REDUCE` | `HOLD` | `BLOCKED`
- **Release Readiness** (engineering track): `GO` | `GO WITH RISK` | `NO GO` | `NOT YET REVIEWED`

Allowed combinations:

| Plan Status | Allowed Release Readiness |
|---|---|
| `APPROVE` | `GO`, `GO WITH RISK`, `NOT YET REVIEWED` |
| `REDUCE`  | `GO WITH RISK`, `NOT YET REVIEWED` |
| `HOLD`    | `NOT YET REVIEWED`, `NO GO` |
| `BLOCKED` | `NO GO`, `NOT YET REVIEWED` |

Any other combination is a bug in the plan and must be fixed before returning to the caller.

## Output Format

```markdown
# PLAN.md

## Status
APPROVE / REDUCE / HOLD / BLOCKED

## 1. Problem

## 2. Decision Flow
| Actor | Situation | Sees | Decides | Acts | Downstream Impact |
|---|---|---|---|---|---|

## 3. Stakeholders

## 4. Business Value

## 5. Scope
### Must Have
### Should Have
### Nice To Have
### Explicit Non-Goals

## 6. Metrics
### 6.1 Metric Registry
| Metric | Definition | Formula | Unit | Time Dim | Refresh | Source (system.table.field) | Owner | Currency | Precision | Rounding | Cut-off | Consistency Check |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
### 6.2 Source Mapping
### 6.3 Financial Correctness Notes
### 6.4 Consistency Review

## 7. Data Definition

## 8. Architecture

## 9. Permissions
### 9.1 Roles
### 9.2 Permission Matrix
| Role | View | Export | Edit | Approve | Field-level Restrictions | Data Scope |
|---|---|---|---|---|---|---|
### 9.3 Field-level Restrictions
### 9.4 Data Scope Rules

## 10. Audit Requirements
### 10.1 Audited Actions
| Action | Actor | Timestamp | Target | Before | After | Reason | Trace/Request Id | Queryable | Retention |
|---|---|---|---|---|---|---|---|---|---|
### 10.2 Retention Policy
### 10.3 Audit Query Requirements

## 11. Release Readiness
GO / GO WITH RISK / NO GO / NOT YET REVIEWED
### 11.1 Release Checklist
### 11.2 Rollback Plan
### 11.3 Open Release Risks

## 12. Testing Strategy

## 13. Risks

## 14. Open Questions

## 15. Milestones

## 16. Next Review
```

Use `references/plan-template.md` when you need to hand the raw template to the user.

## OpenSpec Finance Profile

When the user asks for OpenSpec-compatible output, emit one file per document in this **logical** layout (host project decides the actual on-disk location):

```text
specs/
requirements.md
design.md
tasks.md
metrics.md
permissions.md
audit.md
release.md
```

Per-file minimum contents are specified in `references/openspec-finance-extension.md`. Structural coverage can be checked with:

```bash
python scripts/validate_openspec_finance.py <specs-dir>
python scripts/validate_openspec_finance.py <specs-dir> --strict
python scripts/validate_openspec_finance.py <specs-dir> --json
```

## OpenSpec-lite TOML Host Variant

If the host repository uses the OpenSpec-lite TOML workflow (`openspec/changes/<change-id>/task.toml` + `tasks.md` + optional `design.md`, with long-lived specs under `openspec/specs/<capability>/`), map the profile as follows:

| Profile file | OpenSpec-lite location |
|---|---|
| `requirements.md` | `openspec/changes/<change-id>/tasks.md` (or the `[requirements]` block in `task.toml`) |
| `design.md` | `openspec/changes/<change-id>/design.md` |
| `tasks.md` | `openspec/changes/<change-id>/tasks.md` |
| `metrics.md` | `openspec/specs/<capability>/metrics.md` |
| `permissions.md` | `openspec/specs/<capability>/permissions.md` |
| `audit.md` | `openspec/specs/<capability>/audit.md` |
| `release.md` | `openspec/changes/<change-id>/release.md` |

Full rules in `references/openspec-lite-toml-mapping.md`.

## Examples

### Good invocation — produces `APPROVE / GO WITH RISK`

> User: 我要在管理后台加一个「异常客户资金调整审批」流程，让运营在 T+0 就能发现异常并冻结账户。

The skill:

1. Confirms Actor (运营/风控), Situation (T+0 异常发现), Decides (冻结/放行), Downstream Impact (客户资金流转).
2. Fills every gate; metric registry lists 异常检出率 / 误报率 with owners and formulas; permission matrix separates 查看/导出/冻结/复核; audit table covers 冻结/解冻 sensitive actions with retention 7 年.
3. Emits `PLAN.md` with `Status: APPROVE`, `Release Readiness: GO WITH RISK` due to reconciliation task pending.

### Anti-pattern — must be blocked

> User: 帮我加一个报表页显示所有客户余额。

The skill must **not** return `Status: APPROVE`. Correct behavior: return `Status: BLOCKED: unclear decision flow`, list the missing decision (what判断/什么动作) under Open Questions, and stop.

## Language & Style

- Section headings: English (stable, machine-parseable).
- Body content: match the user's language.
- Never conflate `APPROVE` (plan) and `GO` (release) semantics — keep them on separate lines.
- Use `TODO` markers for uncertainties instead of guessing.
- Never delete required section headings, even when empty; leave `TODO` in place so gaps stay visible.
