---
name: finance-requirement-synthesis
description: Synthesize Issue discussion artifacts in any domain into a human decision package. Use to resume, focus, or consolidate discovery, value, metric, and engineering-option discussions; not to create an FD, Task, OpenSpec change, or release decision.
---

# Requirement Synthesis

Use this method across domains. The `finance-` skill ID is retained for compatibility; it does not restrict activation to finance. Apply domain-specific rules only when relevant to the supplied evidence.

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
- Do not create a Task, OpenSpec change, implementation checklist, branch, worktree, or release decision. After user confirmation, AIW Issue Management (or legacy Requirement Management) may capture this artifact; this Skill must not invoke capture.
- Release status is always `NOT_STARTED` until an implemented FD or legacy AIW Task reaches the separate release stage.
- `READY_FOR_HUMAN_DECISION` is not engineering readiness. Use `%%` for all unresolved gates.
- Include goals, explicit non-goals, key rules, and verifiable acceptance
  examples, with sources. Empty headings do not establish completeness.
- Separate business blockers from proposed engineering postponements. Write
  each postponement and its reason as an exact Plan statement for human review.
  It becomes non-blocking only after human confirmation; it never waives an
  unresolved business rule or conflict. A draft may be saved before it is ready.

## Output

### When loaded by Issue Management or legacy Requirement Management

Apply this method to the current Issue and discussion gap in any domain.
Use domain-specific checks only when the supplied context requires them. The current host JSON contract overrides the Markdown template below;
use the template only for a requested draft or standalone discussion.
Use loaded source bodies and exact references. Do not load sibling Skills or
linked materials, invent missing evidence, or treat capture as confirmation.
Ask only the highest-impact open questions; keep conflicts and proposed
conclusions separate from host-confirmed fragments. Report missing evidence
without claiming readiness. The host owns persistence and human checkpoints.
This method does not authorize tests, external calls, or lifecycle operations.

### Draft artifact

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
## Goals
## Scope
## Non-Goals
## Key Rules
## Acceptance Examples
## Remaining Decisions and Proposed Design Postponements
## Risks and Open Questions
## Earliest Blocking Decision
## Human Decision Requested
## Suggested Next Stage

## Release
Status: NOT_STARTED
Trigger: after an FD or legacy AIW Task exists, implementation evidence is available, and a release is planned.
```
