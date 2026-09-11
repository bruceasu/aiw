---
name: requirement-management
description: Start or continue one AIW Requirement conversation.
disable-model-invocation: true
---

# Requirement Management Conversation

Use this as the single human entry point for requirement work. Start or resume
`aiw requirement chat [requirement-id]`; the linked AIW Session owns the
current conversation phase and selects the next useful discussion. Requirement
Management owns the conversation, requirement state, confirmation checkpoints,
and durable capture; supporting Skills supply the discussion method and draft
artifact for the selected phase.

## Supporting Skill Routing

Automatically load and apply the relevant support Skill; do not require the
human to name Skills, artifacts, files, or CLI commands.

| Requirement gap or phase | Support Skill | Expected artifact or result |
|---|---|---|
| New, vague, or decision-flow gap | `finance-requirement-intake` | Problem Brief |
| Value, priority, smallest-scope, or validation-evidence gap | `finance-value-assessment` | Business Case |
| Finance, risk, reporting, or analytics metric is material or disputed | `finance-metric-brief` | Metric Brief |
| Feasibility, system-boundary, permission, audit, or trade-off gap before promotion | `finance-engineering-options` | Engineering Options |
| Existing artifacts need to resume, focus, or consolidate into a decision package | `finance-requirement-synthesis` | Requirement Plan |
| Business terms, actors, entities, relationships, or domain boundaries are unclear or inconsistent | `domain-modeling` in Requirement mode | Confirmed vocabulary and domain boundaries returned to the Requirement Artifact |
| Blocking ambiguity, conflict, irreversible choice, or `%% NEEDS_INPUT` | `grill-with-docs` in Requirement mode | One decision-centred deep-discovery result |
| Promotion will prepare or supplement OpenSpec planning artifacts with material design gaps | `fd-workflow` in managed AIW/OpenSpec mode | `design.md` Design Readiness |

Load only the Skill needed by the earliest unresolved gap. Apply its artifact
schema and rules, then return its confirmed facts and open questions to this
conversation. Do not make the human repeat settled evidence or automatically
run every phase. `finance-release-gate` is not a Requirement-stage support
Skill: use it only after a promoted Task has implementation evidence and a
release is planned.

Use `domain-modeling` before or alongside a phase Skill when the requirement's
business language is unstable. In Requirement mode, return the agreed
vocabulary, relationships, and boundaries to the Requirement Artifact; do not
create an ADR or write engineering-owned files directly.

When promotion invokes the managed adapter to prepare OpenSpec proposal,
design, capability specs, or `tasks.md`, assess Design Readiness first. Load
`fd-workflow` for material unresolved design decisions; otherwise record
`FD_NOT_REQUIRED` with its rationale in `design.md`. Record `FD_APPLIED` or
`BLOCKED` in the same section as applicable. A `BLOCKED` Design Readiness
prevents promotion from handing the Task to implementation, but does not
replace the separate human confirmation required for promotion.

## Conversation contract

1. Establish whether this is a new or existing Requirement. For an existing
   record, resume from the earliest missing or blocking phase.
2. Select the earliest unresolved phase using Supporting Skill Routing and
   lead its smallest useful discussion. The human supplies evidence and
   decisions in ordinary language rather than Skill names, artifact types,
   files, or CLI arguments.
3. If terms, actors, entities, relationships, or boundaries remain unclear,
   apply `domain-modeling` before recording the affected requirement
   conclusion. Enter deep discovery only for a blocking ambiguity, conflict, irreversible
   decision, or `%% NEEDS_INPUT`. Return confirmed conclusions to the affected
   Requirement Artifact; formal ADRs begin only after engineering work exists.
4. Before every durable action, present a confirmation checkpoint containing
   the action, target, content summary, and write scope. After explicit human
   confirmation (`confirm` or `确认` in the active conversation), invoke the
   corresponding Requirement domain operation.
5. Continue through the Requirement lifecycle until its deliverable state is
   reached: capture the Requirement Plan, obtain the human's approve, defer,
   or reject decision, and, when approved and requested, promote it to one
   Task with its handoff and initial OpenSpec planning artifacts. Requirement
   delivery ends there; implementation and release remain subsequent workflows.

## Authorization boundary

The conversation may prepare artifact content and command parameters. Creating
a Requirement, capturing an artifact, recording a decision, and promotion each
require their own explicit human confirmation. Promotion creates or reuses one
Task, its handoff, and any initial OpenSpec planning artifacts; implementation
and release remain subsequent workflows.

## Output

At each turn, return one concise checkpoint:

```markdown
## Requirement Conversation

- Phase:
- Support Skill applied:
- Requirement state:
- Confirmed facts:
- Open question or decision:
- Proposed durable action: none | create | capture | approve | defer | reject | promote
- Target and write scope:
- Confirmation required: yes | no
- Next question:

%% NEEDS_INPUT: <blocking fact, when applicable>
```

## Completion

Complete the conversation phase only when its artifact is confirmed and
captured, or a specific human decision blocks further progress. Report only
actions actually confirmed and completed.
