---
name: office-hours-finance
version: 0.2.0
stability: beta
last_reviewed: 2026-07-06
owner: platform-finance
related_skills:
  - business-review
  - metrics-review
  - eng-review-finance
  - release-review
  - autoplan-finance
description: financial product intake and scope reduction workflow for admin platforms, operations tools, risk dashboards, finance reports, and analytics systems. use when a user wants to clarify a financial backend requirement, discover the real business problem, identify stakeholders, map the decision flow, reduce scope, and list unknowns before design or implementation. do not use for coding or implementation.
---

# Office Hours Finance

Use this skill to turn vague financial admin, operations, reporting, risk, finance, or analytics requests into a **decision-centric problem brief** — before any design, metric, permission, or engineering discussion begins.

## When To Use

- The request is vague or feature-centric ("加一个页面", "做一个按钮", "出一张报表").
- The requester cannot yet answer *who / when / what decision / what action / what downstream impact*.
- Multiple stakeholders may be affected but haven't been named.
- The user explicitly asks for intake, discovery, scoping, or a problem brief.

## When NOT To Use

- The problem brief already exists and is decision-clear — invoke `business-review` next.
- The user is asking for code, schema, or implementation — decline politely.
- Pure UI polish, wording change, or accessibility fix with no operational decision behind it.

## Hard Rules

- Do not design architecture, database schemas, APIs, UI, code, or implementation tasks.
- Do not accept feature-centric wording such as "add a page", "add a button", or "make a report" as sufficient.
- Require a clear decision flow: who is in what situation, sees what signal, makes what judgment, triggers what action, and what downstream impact follows.
- If the request has no decision or operational action, recommend `HOLD` or a smaller validation step.
- Prefer scope reduction over expansion.
- Do not invent stakeholders, frequency, or impact — mark unknown with `TODO`.

## Inputs

Required from caller:

- **Raw request** — the original ticket, message, or ask verbatim.
- **Target system(s)** — which admin platform, operations tool, or dashboard.

Optional but improves quality:

- Requester's role/team.
- Known downstream systems.
- Compliance / regulatory context.
- Links to related change tickets or specs.

If a required input is missing, ask **once**, then return a `HOLD` brief listing what is missing.

## Outputs

- **Primary**: a single `# Problem Brief` document using the [Output Format](#output-format) below.
- **Recommendation** must be exactly one of: `PROCEED` | `HOLD` | `REDUCE` | `NEEDS_VALIDATION`, with an explicit `Next:` pointer to the sibling skill or manual step that should follow.
- **Secondary (when OpenSpec Profile is requested)**: contents targeted at `tasks.md` under `openspec/changes/<change-id>/`. See [OpenSpec Handoff](#openspec-handoff).

## Handoff

- **Upstream**: raw user request, ticket, or Slack message.
- **Downstream**: `business-review` (when the brief is decision-clear and needs value approval), `metrics-review` (when the decision depends on undefined numbers), `eng-review-finance` (rarely, only when the brief is a technical scoping question), manual validation (when `NEEDS_VALIDATION`).
- **Aggregator**: `autoplan-finance` embeds this skill's output as sections `## 1. Problem`, `## 2. Decision Flow`, `## 3. Stakeholders`, `## 5. Scope`, `## 14. Open Questions` of the master `PLAN.md`.

## Workflow

1. **Identify the problem.** Who has it? How often? What is the current workaround? What is the loss, risk, delay, or manual effort?
2. **Map the decision flow.** Actor / Situation / Sees / Decides / Acts / Downstream Impact — every column must be filled.
3. **Map stakeholders.** Operations, customer support, risk, finance, audit, management, engineering/data — plus regulator when applicable.
4. **Reduce scope.** Must Have (needed for the first useful decision) / Should Have / Nice To Have / Explicit Non-Goals.
5. **List unknowns.** Missing data, undefined business rules, undefined metrics, undefined permissions, undefined audit/retention, unclear operational owner — mark each with `Blocks Next Step? yes/no`.
6. **Emit brief.** Assemble sections 1–6 and set `Recommendation` per the [Decision Gates](#decision-gates).

## Decision Gates

| Gate | Trigger | Recommendation |
|---|---|---|
| Feature-Centric Gate | Only "add X" / "show X" language; no decision described | `HOLD` — request re-intake with decision flow |
| Decision Flow Gate | Any of Actor / Situation / Sees / Decides / Acts / Downstream Impact is empty | `HOLD` |
| Business Impact Gate | Frequency × severity × cost not stated or unquantified | `NEEDS_VALIDATION` — propose a lightweight measurement first |
| Owner Gate | No operational owner identified for the acting step | `HOLD` |
| Scope Gate | Must-Have list > 5 items | `REDUCE` — split into phases |
| Everything-Clear Gate | All gates above passed and unknowns are non-blocking | `PROCEED` 窶・set `Next:` to the appropriate downstream skill |

## Status Model

Recommendation axis (produced by this skill):

- `PROCEED` ── all gates passed; the brief is decision-clear and ready to advance to the next skill.
- `REDUCE` ── value is real but scope is too broad; break into phases before advancing.
- `NEEDS_VALIDATION` ── the business impact is not yet measurable; propose a lightweight measurement first.
- `HOLD` ── one or more gates failed (feature-centric, decision flow empty, owner missing, no impact quantified).

Alignment with sibling skills:

| Office Hours Recommendation | Maps to `business-review` Decision | Maps to `autoplan-finance` Plan Status |
|---|---|---|
| `PROCEED`          | `APPROVE` (subject to value review)   | `APPROVE` (subject to full plan) |
| `REDUCE`           | `REDUCE`                              | `REDUCE` |
| `NEEDS_VALIDATION` | `HOLD` (validate first)               | `HOLD` |
| `HOLD`             | `HOLD`                                | `BLOCKED: incomplete intake` |

Any other combination is a bug in the brief and must be fixed before returning to the caller.

## Output Format

```markdown
# Problem Brief

## Recommendation
PROCEED / HOLD / REDUCE / NEEDS_VALIDATION
Next: business-review | metrics-review | eng-review-finance | release-review | manual-validation

## 1. Problem
### 1.1 Who
### 1.2 Frequency
### 1.3 Impact
### 1.4 Current Workaround

## 2. Decision Flow
| Actor | Situation | Sees | Decides | Acts | Downstream Impact |
|---|---|---|---|---|---|

## 3. Stakeholders
| Stakeholder | Need | Decision Rights | Operational Owner? |
|---|---|---|---|

## 4. Scope
### 4.1 Must Have
### 4.2 Should Have
### 4.3 Nice To Have
### 4.4 Explicit Non-Goals

## 5. Unknowns
| Unknown | Why It Matters | Owner | Blocks Next Step? |
|---|---|---|---|

## 6. Next Review
```

Use `references/intake-template.md` when handing the raw template to the user.

## References

- `references/intake-template.md` ── the full raw template mirroring [Output Format](#output-format).
- `references/openspec-office-hours-mapping.md` ── how to place this brief into an OpenSpec-lite TOML repo.

## OpenSpec Handoff

When this skill runs **inside** `autoplan-finance`, its output populates sections 1, 2, 3, 5, and 14 of `PLAN.md`. No file emission is needed.

When this skill runs **standalone** and the caller asks for OpenSpec output, emit `tasks.md` under `openspec/changes/<change-id>/`:

```text
openspec/
  changes/
    <change-id>/
      task.toml          # machine-readable intake, one field per Recommendation
      tasks.md            # produced here (Problem Brief body)
      tasks.md           # each Unknown row with "Blocks Next Step? yes" becomes a task
```

Full rules in `references/openspec-office-hours-mapping.md`.

## Validator

Structural coverage of an emitted brief can be verified with:

```bash
python scripts/validate_office_hours.py <path-to-brief.md>
python scripts/validate_office_hours.py <path-to-brief.md> --strict
python scripts/validate_office_hours.py <path-to-brief.md> --json
```

The validator checks that all required (heading-level, heading-title) pairs are present as real Markdown headings (skipping fenced code blocks) and, with `--strict`, that they appear in the required order. It tolerates a leading UTF-8 BOM.

## Examples

### Anti-pattern — must be blocked

> User: 帮我在管理后台加一个按钮，冻结账户。

Correct behavior: `Recommendation: HOLD`. Missing: who fires it (运营？风控？), when (什么信号触发？), what post-freeze workflow (通知？复核？), reversibility. Return brief with these questions in `## 5. Unknowns` marked `Blocks Next Step? yes`.

### Good invocation — produces PROCEED

> User: 客服每天遇到 20+ 起客户投诉「昨天入金没到账」，我们要一个入金对账查看页给客服。

The skill:

1. Identifies actor (客服), frequency (20/day), impact (客户流失风险 + 客服 5 min/case).
2. Fills decision flow: 客服看到投诉 → 查看入金对账页 → 判断是渠道延迟/入金失败/客户误报 → 回复客户或转风控.
3. Stakeholders: 客服（Operational Owner）、支付渠道运维、风控.
4. Scope reduced: Must Have = 按订单号/客户 ID 查询 + 最近 24h 入金状态；Non-Goals = 手动补单.
5. Recommendation: `PROCEED`, Next: `metrics-review` (to define "入金到账 SLA" metric first).

## Language & Style

- Section headings: English (stable, machine-parseable).
- Body content: match the user's language.
- Use `TODO` markers for uncertainties instead of guessing.
- Never delete required section headings, even when empty; leave `TODO` in place so gaps stay visible.