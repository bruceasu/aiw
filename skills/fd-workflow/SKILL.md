---
name: fd-workflow
description: Design an AIW Task from an Issue or refine its Feature Design, including decisions and ordered work items. Also supports explicitly requested standalone FD storage.
---

# Feature Design Workflow

Read `skills/work-management.md`. A managed FD is the Task's primary
engineering plan. Resolve the Issue and Task when they exist, then read the
approved Issue evidence, relevant code and stable specs. An OpenSpec change is
created only when the user explicitly requests it. Stable spec updates do not
require a change directory.

Choose the FD mode before writing: an AIW Task uses the managed FD below;
`FD-XXX` numbering, `FEATURE_INDEX.md`, and the portable template belong only
to standalone FD storage. Do not place an FD under `docs/requirements/`;
that directory holds Issue records. A managed FD uses its Task ID for identity
and numbered Work Items within the document.

## Managed FD

Use `docs/features/<task-id>.md` for a managed Task FD; the Task's
`.ai/tasks/<task-id>/` record remains the execution center. Move an older FD
stored elsewhere to this path before mapping it to Core, and preserve completed
items. A legacy Task with only an OpenSpec `design.md` and `tasks.md` may
continue from them until its plan is deliberately moved to an FD.

Record the goal and source Issue, constraints, relevant stable specs, decisions
with evidence and rationale, compatibility effects, acceptance evidence,
ordered work items, TODO, Verification, and `%% NEEDS_INPUT` for unresolved
material choices. Use numbered checklist items such as `- [ ] 1.1 ...` so AIW
can map them to Workflow Core Work Items. Keep item IDs stable after mapping.
The FD owns their meaning and ordering; Workflow Core owns execution state.

Choose a clearly better approach from evidence and known user preferences.
Use a bounded sub-agent comparison only when delegation is authorized and
useful. Ask the human when options are materially close, critical information
is missing, or the choice changes authorization or accepted scope. Record the
choice and remaining trade-offs in the FD. Mark design readiness as
`FD_APPLIED`, `FD_NOT_REQUIRED`, or `BLOCKED`. A `BLOCKED` FD cannot map its
numbered work items yet; resolve material decisions before setting it ready.

When stable behavior changes, update the relevant `openspec/specs/` capability
spec. If a linked OpenSpec change exists, keep its proposal and spec delta in
sync with the FD. Do not make its `tasks.md` the Task's required work source.

After writing the FD, use the supported AIW planning/sync operation to map its
items into `.ai/tasks/<task-id>/`. Report any failed mapping as a Gate. Do not
create Attempts, claim leases, change Task status, commit, or run verification
as part of design.

## Standalone FD

Use this mode only when the user explicitly requests standalone FD storage or
there is no managed AIW Task. Read `references/portable-operations.md` for its
layout and operations. Do not silently reconcile a standalone FD into a Task.
