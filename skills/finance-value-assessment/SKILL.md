---
name: finance-value-assessment
description: Assess whether a financial, operations, reporting, risk, or analytics requirement has sufficient business value and an appropriately small scope. Use after problem discovery; not for technical design, Task creation, or implementation approval.
---

# Finance Value Assessment

Assess the value claim behind a clear requirement and produce a `Business Case` draft. This is a human decision aid, not a project approval or implementation gate.

## Use when

- A Problem Brief exists and a human needs evidence for priority, scope, or a validation experiment.

## Do not use when

- The decision flow is unclear; route to `finance-requirement-intake`.
- The request asks for metric definition, engineering design, Task creation, code, or release approval.

## Inputs

- Problem Brief or equivalent decision flow.
- Target system.

## Produce

Return a Requirement Artifact with `Artifact Type: Business Case` and local status `SUPPORTED`, `VALIDATE_FIRST`, or `NOT_SUPPORTED`.

Assess evidence for revenue, risk reduction, labour cost, decision efficiency, regulatory need, or customer experience. Compare smaller alternatives and identify a measurable validation path.

## Rules

- Evidence must name a person, incident, metric, or regulatory source; do not treat sentiment as evidence.
- Prefer a smaller scope or validation experiment where credible.
- Do not infer human approval from `SUPPORTED`.
- Do not create tasks, tickets, OpenSpec artifacts, or implementation plans. After user confirmation, AIW Requirement Management may capture this artifact; this Skill must not invoke capture.
- Use `%%` for missing evidence, owners, or decisions; recommend one next discussion stage without invoking it.

## Output

```markdown
# Requirement Artifact

## Metadata
- Artifact Type: Business Case
- Requirement ID: %% NOT_ASSIGNED
- Stage: value-assessment
- Status: SUPPORTED / VALIDATE_FIRST / NOT_SUPPORTED
- Based On: Problem Brief
- Human Approval Required: yes

## Facts
## Assumptions
## Value Evidence
| Dimension | Evidence | Confidence | Notes |
|---|---|---|---|

## Scope and Alternatives
## Validation Path
## Decision / Recommendation
## Open Questions
## Suggested Next Stage
```
