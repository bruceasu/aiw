---
name: business-review
version: 0.2.0
stability: beta
last_reviewed: 2026-07-07
owner: platform-finance
related_skills:
  - office-hours-finance
  - metrics-review
  - eng-review-finance
  - release-review
  - autoplan-finance
description: business value review workflow for financial admin platforms, operations tools, reporting systems, risk dashboards, and analytics products. use when a user asks whether a requirement, dashboard, report, workflow, or operational tool is worth building. evaluate revenue, risk reduction, labor cost, decision efficiency, regulatory needs, customer experience, cost-benefit, smaller alternatives, and validation paths. return APPROVE, REDUCE, or HOLD. read-only, do not write code, do not modify files, do not create pull requests.
---

# Business Review

Use this skill to decide whether a financial admin, reporting, risk, operations, finance, or analytics request is **worth pursuing**. This skill is a **read-only value gate**: it emits a structured `BUSINESS_REVIEW.md` and never writes code, tickets, or PRs.

## When To Use

- The problem brief is decision-clear (typically produced by `office-hours-finance`) and the caller needs a go/no-go on business value.
- The user asks "is this worth building?", "should we do this?", or "which of these should we prioritize?".
- `autoplan-finance` invokes this skill as its business-value step.

## When NOT To Use

- The requirement is vague or feature-centric ── route to `office-hours-finance` first.
- Metrics required to prove value are undefined ── route to `metrics-review` first (a value claim without a measurable metric is speculation).
- The user wants technical feasibility ── route to `eng-review-finance`.
- The user wants launch-readiness gate ── route to `release-review`.
- The user wants a full plan ── route to `autoplan-finance`.

## Hard Rules

- A request must hit **at least one** value dimension (revenue / risk reduction / labor cost / decision efficiency / regulatory / customer experience). If none, default to `HOLD`.
- Do not accept "improves visibility" / "everyone wants this" / "nice to have" as evidence ── all evidence must reference a named person, incident, metric, or regulator.
- Do not accept unquantified benefits when frequency and severity can plausibly be counted ── prefer `NEEDS_VALIDATION` via a lightweight measurement.
- Do not invent stakeholders, revenue impact, incident counts, regulatory citations, or existing alternatives.
- Prefer a **smaller version** or a **manual validation** over full build, whenever either is credible.
- If an existing report, query, or process solves ≥ 80% of the value, return `REDUCE` and route to that path.
- Do not upgrade a `HOLD` to `APPROVE` without new information from the user or a named business owner.
- Do not write code, SQL, migration, or implementation tickets.
- Do not modify repository files, even under `openspec/`.

## Inputs

Required from caller:

- **Problem brief or requirement one-liner** ── ideally from `office-hours-finance`.
- **Target system(s)** ── which admin platform, dashboard, report, or operational tool.

Optional but improves quality:

- Named business owner and their stated decision this change enables.
- Existing metric definitions or dashboard entries that would prove the value.
- Frequency / severity numbers for the current pain (incidents per week, minutes per case, loss per event).
- Regulatory citation (rule id, audit finding id) when regulatory dimension is claimed.
- Known alternative reports, queries, or processes that overlap.

If required inputs are missing, ask **once**, then proceed with `Decision: HOLD` and list gaps under `## 7. Required Conditions`.

## Outputs

- **Primary**: a single `BUSINESS_REVIEW.md` document following the [Output Format](#output-format) below. Return as a message; do not write to disk unless the caller explicitly asks.
- **Secondary (when OpenSpec Profile is requested)**: contents targeted at the `[business]` block of `task.toml` and an appended section in `task.md`. See [OpenSpec Handoff](#openspec-handoff).

## Handoff

- **Upstream**: `office-hours-finance` (problem brief, decision flow, stakeholders).
- **Downstream**: `metrics-review` (when `APPROVE` needs a measurable metric to prove value), `eng-review-finance` (when `APPROVE` proceeds to design), manual validation (when `NEEDS_VALIDATION` via a measurement task).
- **Aggregator**: `autoplan-finance` embeds this skill's output as `## 4. Business Value` of the master `PLAN.md`.

## Workflow

Run these passes in order. A failed earlier gate blocks all later ones.

1. Restate the request and the specific **decision** it enables (from the upstream Problem Brief).
2. Fill the **Value Matrix** ── one row per claimed value dimension with evidence + strength (strong / weak).
3. Estimate **Cost vs Benefit** ── development, maintenance, data governance, operational training, compliance/audit overhead.
4. **Challenge scope** ── is there a smaller useful version? Which parts are optional or downstream?
5. Identify **Smaller Alternatives** ── existing report / query / manual process that already covers ≥ 80%.
6. Define a **Validation Path** ── how would we know this delivered the claimed value?
7. List **Required Conditions** ── stakeholders, metrics, permissions, audit, regulatory dependencies that must exist for the claim to hold.
8. Emit the **Final Recommendation** per the Decision Model below.

## Review Gates

Ordered; a failed earlier gate blocks all later ones.

| Gate | Trigger | Decision Effect |
|---|---|---|
| Value Dimension Gate | Zero value dimensions hit | `HOLD` |
| Evidence Gate | All claimed dimensions rated `weak` | `HOLD` or `NEEDS_VALIDATION` |
| Quantification Gate | Benefit is unquantified when it could be counted (frequency × severity × cost) | `NEEDS_VALIDATION` (route to measurement task) |
| Existing Alternative Gate | An existing report, query, or process covers ≥ 80% of the claim | `REDUCE` (route to that path) |
| Scope Gate | Must-Have list from upstream brief > 5 items | `REDUCE` (split into phases) |
| Owner Gate | No named business owner accountable for the decision this change enables | `HOLD` |
| Regulatory Evidence Gate | Regulatory dimension is claimed but no rule id, audit finding id, or compliance sign-off is cited | `HOLD` until citation is provided |
| Cost-Benefit Gate | Development + maintenance + compliance cost obviously exceeds benefit within 12 months | `HOLD` or `REDUCE` |
| Everything-Clear Gate | All gates passed with credible quantification and named owner | `APPROVE` |

## Decision Model

- `APPROVE` ── value is clear, cost is proportionate, scope is constrained, owner is named, and required conditions are satisfied or explicitly listed as follow-ups.
- `REDUCE` ── value exists but scope is too large, or an existing alternative solves ≥ 80%; break into phases or route to the alternative.
- `HOLD` ── value is unclear, no dimension is met, evidence is only weak, or owner / regulatory citation is missing.
- (Return `HOLD` with a `Next: NEEDS_VALIDATION` pointer when the correct action is a measurement first ── this keeps the axis binary while still capturing the recommendation.)

Alignment with sibling skills:

| Business Review Decision | Consistent with `office-hours-finance` Recommendation | Maps to `autoplan-finance` Plan Status |
|---|---|---|
| `APPROVE` | `PROCEED` | `APPROVE` |
| `REDUCE`  | `REDUCE`, `PROCEED` | `REDUCE` |
| `HOLD`    | `HOLD`, `NEEDS_VALIDATION` | `HOLD`, `BLOCKED` |

Any other combination is a bug in the review and must be fixed before returning to the caller.

## Output Format

```markdown
# Business Review

## Decision
APPROVE / REDUCE / HOLD

## 1. Context
- Upstream Problem Brief:
- Requesting stakeholder:
- Target system(s):
- Decision this change enables (one sentence):
- Named business owner (person, not team):

## 2. Value Matrix
| Dimension | Evidence (person / incident / metric / regulator) | Strength | Notes |
|---|---|---|---|
| Increase revenue                       |  | strong / weak | |
| Reduce risk                            |  | strong / weak | |
| Reduce manual labor cost               |  | strong / weak | |
| Improve decision efficiency            |  | strong / weak | |
| Satisfy regulatory / audit / compliance|  | strong / weak | |
| Improve customer experience            |  | strong / weak | |

## 3. Cost vs Benefit
| Cost Type | Estimate | Notes |
|---|---|---|
| Development                     |  |  |
| Maintenance                     |  |  |
| Data governance                 |  |  |
| Operational training            |  |  |
| Compliance / audit overhead     |  |  |

- Estimated benefit (quantified when possible):
- Payback horizon:
- Cost > benefit within 12 months? yes / no

## 4. Scope Challenge
- Must-Have list size (from upstream brief):
- Which items are actually optional for the first useful decision?
- Which items can move to Should-Have or Non-Goals?

## 5. Smaller Alternatives
| Alternative | Coverage of Claim | Cost | Recommendation |
|---|---|---|---|
| Existing report / query / process |  |  | reuse / extend / discard |
| Manual workflow (short-term)      |  |  | reuse / extend / discard |
| Vendor / SaaS option              |  |  | reuse / extend / discard |

## 6. Validation Path
- How will we know this delivered value? (Metric name from `metrics-review`, target value, target date)
- Lightweight validation before full build? (Spreadsheet, manual pull, spike):
- Kill criteria if validation fails:

## 7. Required Conditions
| Condition | Owner | Blocks Decision? |
|---|---|---|
| Metric definition (`metrics-review`) |  | yes / no |
| Permission model (`eng-review-finance`) |  | yes / no |
| Audit coverage (`eng-review-finance`) |  | yes / no |
| Regulatory citation |  | yes / no |
| Business owner sign-off |  | yes / no |

## 8. Final Recommendation
- Decision: APPROVE / REDUCE / HOLD
- Next: metrics-review | eng-review-finance | release-review | manual-validation | back-to-office-hours
- Rationale:
- Named business owner (person):
- Next review date if `REDUCE` or `HOLD`:
```

Use `references/business-review-template.md` when you need to hand the raw template to the user.

## References

- `references/business-review-template.md` ── the full raw template mirroring [Output Format](#output-format).
- `references/value-matrix.md` ── detailed rules for the six value dimensions, strong vs weak evidence, and the review questions to ask.
- `references/openspec-business-review-mapping.md` ── how to place this review into an OpenSpec-lite TOML repo.

## OpenSpec Handoff

When this skill runs **inside** `autoplan-finance`, its output populates `## 4. Business Value` of `PLAN.md`. No file emission is needed.

When this skill runs **standalone** and the caller asks for OpenSpec output, append a `## Business Review` section to `openspec/changes/<change-id>/task.md` and update the `[business]` block in `task.toml`:

```text
openspec/
  changes/
    <change-id>/
      task.toml          # [business] decision + named owner
      task.md            # append Business Review section
      tasks.md           # each Required Condition with Blocks Decision = yes becomes a task
```

Full rules in `references/openspec-business-review-mapping.md`.

## Validator

Structural coverage of an emitted `BUSINESS_REVIEW.md` can be verified with:

```bash
python scripts/validate_business_review.py <path-to-BUSINESS_REVIEW.md>
python scripts/validate_business_review.py <path-to-BUSINESS_REVIEW.md> --strict
python scripts/validate_business_review.py <path-to-BUSINESS_REVIEW.md> --json
```

The validator checks that all required (heading-level, heading-title) pairs are present as real Markdown headings (skipping fenced code blocks) and, with `--strict`, that they appear in the required order. It tolerates a leading UTF-8 BOM.

## Examples

### Good invocation ── produces `APPROVE`

> User: 客服每天 20 起「客户抱怨入金没到账」→ 已有 Problem Brief，请评审是否值得建入金对账看板。

The skill:

1. Context: 客户抱怨 → 客服判断是否漫延 → 决定入金失败 / 客户误报。
2. Value Matrix: 减少人工 (strong: 客服 20 起 × 5 min = 100 min/day)；提升决策效率 (strong: 5 min → 30 sec)；改善客户体验 (strong: 平均等待 20 min → < 5 min)。
3. Cost vs Benefit: 开发 3 人周 + 维护 0.1 FTE；预计回收 < 3 个月。
4. Smaller Alternatives: 现有 T+1 报表覆盖 30%；不够用。
5. Validation Path: metric = 客服入金查询平均处理时长；目标从 5 min → 1 min；30 天验证。
6. Required Conditions: metric 定义（→ metrics-review）、客服权限（→ eng-review-finance）、业务 owner=客服组长 张 X。
7. Decision: `APPROVE`，Next: `metrics-review`。

### Anti-pattern ── must be blocked

> User: 我想加一个「实时全球业务大屏」，管理层想看。

The skill must **not** return `APPROVE`. Correct behavior:

- Value Matrix: 提升决策效率 (weak: 无具体决策)；改善客户体验 (weak: 内部使用)；其他维度未命中。
- Existing Alternative Gate: 现有日报覆盖 80% → 建议 `REDUCE`.
- Owner Gate: 无命名业务 owner → 建议 `HOLD`.
- Decision: `HOLD`，Next: `back-to-office-hours`（让 office-hours-finance 重新问清"看到这个数管理层会做什么决策"）。

## Language & Style

- Section headings, table headers, decision enums, strength enums: **English** (stable, machine-parseable, validator-friendly).
- Body content: match the user's language (this repo family defaults to 中文正文).
- Never conflate `APPROVE` (business value) with `READY` (engineering / metrics) or `GO` (release) ── they live on separate tracks.
- Use `TODO` markers for uncertainties instead of guessing.
- Never delete required section headings, even when empty; leave `TODO` in place so gaps stay visible to the validator and reviewers.
- Prefer tables over prose for value matrix, cost matrix, smaller alternatives, and required conditions.
- All evidence must reference a named person, incident id, metric name, or regulatory citation ── never anonymous sentiment.