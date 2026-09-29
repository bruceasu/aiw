---
name: implement
description: "Implement one selected work item from an AIW Task and its Feature Design, with optional OpenSpec specs."
---

# Implement

Follow `skills/reviewed-skill-contract.md` and
`skills/work-management.md`; read `docs/agents/work-management.md` when
present.

## When To Use

Use this Skill to implement one selected, ready Work Item in an AIW Task. Do
not use it to discover an Issue, design an FD, write a spec, or split the plan.
The Task does not need an OpenSpec change.

## Resolve Work And Readiness

Resolve one Task from the explicit ID or the Task's session, workspace, Issue,
FD, or linked change. If several Tasks remain, ask for the Task ID. Read its
`task.toml`, approved Issue handoff, linked FD, selected Work Item, relevant
stable specs, and linked change only when present and relevant. For an older
Task without an FD, use its existing OpenSpec `design.md` and `tasks.md`.

Check that the selected item has a clear outcome, acceptance evidence, and
required decisions. An FD marked `BLOCKED` prevents mapped work; an
unrelated `%% NEEDS_INPUT` in a ready FD does not stop the selected item. Route a missing design
decision to `fd-workflow` and an unclear item boundary to `to-tickets`. Use
confirmed evidence and known user preferences for routine choices. Ask only
when the missing answer changes scope, authorization, or the required behavior.
Older Tasks without an FD readiness section are actionable when their existing
design and specs make the selected item clear.

## Implement And Verify

Use the Task's declared workspace; ordinary sequential work stays in the
primary workspace. Use `aiw wt` for explicitly authorized isolation. Do not
silently move work to another workspace. Apply the smallest complete change
for the selected item, preserving unrelated work and stable interfaces.

Follow the repository's validation budget. Review the changed paths
statically. After code changes, run one compile-only check: prefer a
`scripts/compile*` or root `compile*` command; otherwise use the narrowest
language-level compiler command without retaining a distributable artifact.
If it fails, fix the source and rerun the same command once. Report a second
failure. Do not run tests, final builds, formatters, linters, vet, or broad
verification without authorization. Writing a focused test is allowed when
the item needs it; running it follows the runtime authorization rules.

## Complete The Item

Update the selected FD checkbox, TODO, Verification, and remaining `%%` notes.
For a legacy Task, update its selected `tasks.md` checkbox. Use the supported
AIW sync operation to reconcile item progress; do not directly set Task
display status, claim a lease, or fabricate an Attempt. Completing one Work
Item does not complete the Task. Report the selected item, changed files,
static evidence, commands run, skipped checks, unresolved Gates, and the
recommended next Core transition.

Mark the Task complete only when all work items and Verification are complete
and its lifecycle transition is authorized. Do not automatically commit,
merge, push, remove a worktree, delete a branch, or archive a change. A spec
change discovered during implementation is a `to-spec` handoff; update the FD
plan if its work items need to change.
