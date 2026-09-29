---
name: implement
description: "Implement the ready Work Items of one AIW Task in FD order, with optional OpenSpec specs."
---

# Implement

Follow `skills/reviewed-skill-contract.md` and
`skills/work-management.md`; read `docs/agents/work-management.md` when
present.

## When To Use

Use this Skill to carry one AIW Task through its ready FD Work Items in
dependency order. The user may select one Work Item as the scope; otherwise
continue through all ready items in the Task. Do not use it to discover an
Issue, design an FD, write a spec, or split the plan. The Task does not need an
OpenSpec change. Never create a change unless the user explicitly requests it.

## Resolve Work And Readiness

Resolve one Task from the explicit ID or the Task's session, workspace, Issue,
FD, or linked change. If several Tasks remain, ask for the Task ID. Read its
`task.toml`, approved Issue handoff, linked FD, ready Work Items, relevant
stable specs, and linked change only when present and relevant. For an older
Task without an FD, use its existing OpenSpec `design.md` and `tasks.md`.

Before each item, check its dependencies, outcome, acceptance evidence, and
required decisions. An FD marked `BLOCKED` prevents mapped work; an unrelated
`%% NEEDS_INPUT` in a ready FD does not stop a clear item. Route a missing
design decision to `fd-workflow` and an unclear item boundary to `to-tickets`. Use
confirmed evidence and known user preferences for routine choices. Ask only
when the missing answer changes scope, authorization, or the required behavior.
Older Tasks without an FD readiness section are actionable when their existing
design and specs make the selected item clear.

## Implement And Verify

Use the Task's declared workspace; ordinary sequential work stays in the
primary workspace. If isolation is selected, create or resolve it through
`aiw wt` after the FD is committed on the parent branch. Do not create a
worktree during Issue discovery or silently move work to another workspace.
Apply the smallest complete change for the current item, preserving unrelated
work and stable interfaces.

Follow the repository's validation budget. Review the changed paths
statically. After the Task's code edits, run one compile-only check: prefer a
`scripts/compile*` or root `compile*` command; otherwise use the narrowest
language-level compiler command without retaining a distributable artifact.
If it fails, fix the source and rerun the same command once. Report a second
failure. Do not run tests, final builds, formatters, linters, vet, or broad
verification without authorization. Writing a focused test is allowed when
the item needs it; running it follows the runtime authorization rules.

## Continue Through The Task

After each item, update TODO, Verification, and remaining `%%` notes. Mark its
FD checkbox, or the legacy `tasks.md` checkbox, only when its required
acceptance evidence is available. Use the supported AIW sync operation to
reconcile progress; do not directly set Task display status, claim a lease,
or fabricate an Attempt. A checked item does not prove formal acceptance.
Re-read the derived Work Item state and select the next ready item, including
an independent ready item when another is blocked. Continue without asking
again when scope and authorization remain the same. Stop when no eligible
items remain or a decision, authorization, workspace, or acceptance Gate
prevents further progress; report the exact blocker and work already done.

Do not start `aiw wf run --execute` or `aiw wf supervise` merely to continue
the Task or obtain acceptance evidence. Those are separate automated
execution modes and require an explicit request. Manual implementation can
record real static and compile evidence, but cannot invent an independent
Tester result or a supervised Attempt.

Mark the Task complete only when all work items and Verification are complete
and its lifecycle transition is authorized. Report changed files, static
evidence, commands run, skipped checks, unresolved Gates, and the next
authorized transition. Do not automatically commit,
merge, push, remove a worktree, delete a branch, or archive a change. A spec
change discovered during implementation is a `to-spec` handoff; update the FD
plan if its work items need to change.
