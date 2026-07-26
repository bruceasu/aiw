# Business Review

> Mirror of `SKILL.md` Output Format. Copy this file, fill each section, then
> run `python scripts/validate_business_review.py <this-file>` to check
> structural coverage. `--strict` also enforces section order.

## Decision
APPROVE / REDUCE / HOLD

<!-- Any failed gate in SKILL.md "Review Gates" forces HOLD or REDUCE.
     Document each gate outcome in the sections below. -->

## 1. Context

- Upstream Problem Brief:
- Requesting stakeholder:
- Target system(s):
- Decision this change enables (one sentence):
- Named business owner (person, not team):

## 2. Value Matrix

| Dimension | Evidence (person / incident / metric / regulator) | Strength | Notes |
|---|---|---|---|
| Increase revenue                        |  | strong / weak |  |
| Reduce risk                             |  | strong / weak |  |
| Reduce manual labor cost                |  | strong / weak |  |
| Improve decision efficiency             |  | strong / weak |  |
| Satisfy regulatory / audit / compliance |  | strong / weak |  |
| Improve customer experience             |  | strong / weak |  |

<!-- Strength enum: strong / weak
     Strong = named person, incident id, metric with value, or regulator citation.
     Weak   = generic sentiment, "everyone wants this", "improves visibility". -->

## 3. Cost vs Benefit

| Cost Type | Estimate | Notes |
|---|---|---|
| Development                 |  |  |
| Maintenance                 |  |  |
| Data governance             |  |  |
| Operational training        |  |  |
| Compliance / audit overhead |  |  |

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

<!-- If any alternative covers ≥ 80% of the claim, the decision is REDUCE. -->

## 6. Validation Path

- How will we know this delivered value? (Metric name from `metrics-review`, target value, target date):
- Lightweight validation before full build? (Spreadsheet, manual pull, spike):
- Kill criteria if validation fails:

## 7. Required Conditions

| Condition | Owner | Blocks Decision? |
|---|---|---|
| Metric definition (`metrics-review`)     |  | yes / no |
| Permission model (`eng-review-finance`)  |  | yes / no |
| Audit coverage (`eng-review-finance`)    |  | yes / no |
| Regulatory citation                      |  | yes / no |
| Business owner sign-off                  |  | yes / no |

## 8. Final Recommendation

- Decision: APPROVE / REDUCE / HOLD
- Next: metrics-review | eng-review-finance | release-review | manual-validation | back-to-office-hours
- Rationale:
- Named business owner (person):
- Next review date if `REDUCE` or `HOLD`: