---
name: finance-requirement-synthesis
description: Synthesize financial requirement discussion artifacts into a human decision package. Use to resume, focus, or consolidate discovery, value, metric, and engineering-option discussions; not to create an AIW Task, OpenSpec change, or release decision.
---

# Finance Requirement Synthesis

Consolidate existing Requirement Artifacts into a `Requirement Plan` that lets a human decide whether to promote the requirement into managed engineering work.

## Modes

- `focused`: address one named discussion gap.
- `resume`: identify and address the earliest blocking gap in supplied artifacts.
- `synthesize`: consolidate supplied artifacts without running missing stages.
- `full`: discuss intake, value, metric, and engineering-option stages only when the user explicitly requests all of them.

`full` never runs `finance-release-gate`.

## Inputs

- Requirement one-liner and target system.
- Any available Problem Brief, Business Case, Metric Brief, or Engineering Options.

## Produce

Return a Requirement Artifact with `Artifact Type: Requirement Plan` and local status `READY_FOR_HUMAN_DECISION`, `BLOCKED`, or `DEFERRED`.

The plan indexes available evidence, separates facts from assumptions, identifies the earliest blocking decision, gives scope and risks, and states whether a human should approve, defer, reject, or request more discussion.

## Rules

- Do not claim missing sibling discussions were performed. A mode may only summarize supplied artifacts or discuss the requested scope.
- Do not call sibling Skills automatically.
- Do not create a Task, OpenSpec change, implementation checklist, branch, worktree, or release decision. After user confirmation, AIW Requirement Management may capture this artifact; this Skill must not invoke capture.
- Release status is always `NOT_STARTED` until an implemented AIW Task reaches the separate release stage.
- `READY_FOR_HUMAN_DECISION` is not engineering readiness. Use `%%` for all unresolved gates.

## Output

```markdown
# Requirement Artifact

## Metadata
- Artifact Type: Requirement Plan
- Requirement ID: %% NOT_ASSIGNED
- Stage: synthesis
- Status: READY_FOR_HUMAN_DECISION / BLOCKED / DEFERRED
- Based On: list supplied artifacts
- Human Approval Required: yes

## Facts
## Assumptions
## Evidence Index
## Scope
## Risks and Open Questions
## Earliest Blocking Decision
## Human Decision Requested
## Suggested Next Stage

## Release
Status: NOT_STARTED
Trigger: after an AIW Task exists, implementation evidence is available, and a release is planned.
```
