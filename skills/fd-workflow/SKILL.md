---
name: fd-workflow
description: Create and design a numbered FD, record decisions and Work Items, and hand it to the next role.
---

# Feature Design Workflow

Read `skills/work-management.md` and the repository instructions. Use this
Skill when the user wants to create, refine, review, or close a Feature Design.
It does not implement code. A new FD does not require an AIW Task or OpenSpec
change. Read an Issue and stable specs when they affect the design.

## Resolve the FD

Use an explicit FD ID first, then a unique FD linked from the current Issue or
conversation. Ask when several FDs match. For a new FD, use `aiw fd new
"<title>" [--issue <id>]`; this allocates an unused `FD-XXX` number, creates
the file from `docs/features/TEMPLATE.md`, updates the index, and records a
`design-requested` event. If an AIW Task already owns a legacy FD, keep its
current path and use the legacy workflow until a deliberate migration.

## Design

Read the FD, approved Issue evidence when present, relevant code, prior FDs,
and stable specs. Write the problem, credible options, decision and reason,
scope, compatibility effects, numbered Work Items, acceptance, and a realistic
Verification plan. Use `%% NEEDS_INPUT: ...` for material unknowns. Do not
turn a routine technical choice with clear evidence into a human question.
Keep Work Item IDs stable after implementation starts.

`Design` becomes `Open` only when the FD contains all required sections,
numbered Work Items, and no material `%% NEEDS_INPUT` note. The Planner then
emits `design-ready` with `--producer planner --artifact <fd-path>` and the
current source event ID when one is dispatched. This records the handoff to
Worker. Do not emit an event merely because a heading or template exists.

Use `aiw fd list`, `show`, and `resume` for status and recovery. A pending event
can be handled by the named role in the current host after `aiw fd claim <id>
<event-id> --session <host-session-id>`. A dispatched event with unknown outcome
must be reconciled against its original session or log.

## Close and archive

A passed Reviewer event is required before `aiw fd close <id> Complete`.
Deferred and Closed need `--reason "..."`. Archive changes the FD
path and index; it does not merge or publish code. Update a changelog only if
the repository uses one. Do not commit, push, or merge as a side effect of
design or archive unless the user authorized that action.

Read `references/portable-operations.md` for numbered FD conventions and
`references/templates.md` for layout examples. When they conflict with the
FD-first contract above, use this Skill and the current CLI behavior.
