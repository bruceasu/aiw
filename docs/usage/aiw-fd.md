# FD-first development workflow

Use `aiw fd` for new engineering work. A numbered FD contains the design,
stable Work Item IDs, progress, acceptance, and Verification. The FD file is
the source of truth; `FEATURE_INDEX.md` is a lookup index. An approved Issue
can be linked with `--issue`; a Task is optional.

```text
aiw fd new "Improve report export" --issue REQ-123
aiw fd list
aiw fd show FD-002
aiw fd claim FD-002 FD-002-000002-design-requested --session <host-session-id>
aiw fd emit FD-002 design-ready --producer planner --artifact docs/features/FD-002_IMPROVE_REPORT_EXPORT.md
aiw fd resume FD-002
aiw fd emit FD-002 implementation-ready --producer worker --artifact docs/features/reports/FD-002-implementation.md --source-event FD-002-000003-design-ready
aiw fd emit FD-002 verification-passed --producer reviewer --artifact docs/features/reviews/FD-002-review.md --source-event FD-002-000004-implementation-ready
```

The creation event routes to Planner. Planner writes options, decision,
acceptance, and numbered Work Items before `design-ready`. Worker implements
all ready items and records a report; then it emits `implementation-ready`.
An independent Reviewer emits `changes-requested` with findings or
`verification-passed` with a review report. A role must claim or receive its
handoff before completing it, then pass `--source-event <event-id>`.
`aiw fd show` displays the last ID.

The event receipt includes FD ID, revision, type, producer, target role, and
artifact path. `AIW_FD_ROLE_RUNNER` may name an executable that receives
`<role> <fd-path> <event-json-path>`. AIW starts it when a new event is ready.
The runner should read those files, work in the named role, and emit the next
event with the original event ID. Its stdout and stderr go to the event log.
If no runner is configured, the event stays pending. Before a host Agent starts
writing, bind that exact event with `aiw fd claim <fd-id> <event-id> --session
<host-session-id>`. The claim is atomic and idempotent for the same session;
another session cannot take it. When that role emits its result, pass the
claimed event ID through `--source-event`. If a runner is configured, AIW
claims the event as it starts that runner. A pending event may also be
dispatched later by `aiw fd resume`.

`resume` does not launch a second writer for a launching or dispatched event.
Inspect its original session or event log and reconcile the result first.
Changing the FD manually does not itself emit an event. Stage operations
increase `**Revision:**`; a pending role event must be claimed before its role
can complete it, and a changed FD cannot be claimed until reconciled. A
`Complete` archive also requires the FD content to match the Reviewer's
`verification-passed` receipt. Reconcile changed content with a new review.
If a process crashes while holding `.ai/fd/<id>/.mutation-lock`, inspect the
original process and event receipt before removing the stale lock. Never
remove it while the role may still be writing.

For parallel writes, commit the FD plan, then run `aiw fd worktree add FD-002`.
`aiw fd worktree status FD-002` shows Git worktrees. Sequential work may stay
in the primary checkout. Local commits may be made for focused slices under
repository rules; push, merge, release, and archive are separate actions.

After a passed review and a separate close decision, run `aiw fd close FD-002
Complete`. Use `--reason "..."` for `Deferred` and `Closed` outcomes.
This archives the FD file and rebuilds the index; it does not change Git
delivery state.

Old `aiw issue promote --task`, `aiw task`, and `aiw wf` remain compatibility
commands for existing Task records. Do not use `aiw wf supervise` for a new FD.
Stable specs still live in `openspec/specs/`; create an OpenSpec change only
when explicitly requested.

## Legacy migration

An existing Task can keep its current FD, Core records, Session IDs, evidence,
and Git lineage. Read it through the legacy commands; a new numbered FD does
not require conversion. To move unfinished design work deliberately, create a
new numbered FD, cite the old Task and FD in `Sources`, and copy only decisions
that still apply. Keep old Work Item IDs and completed evidence in the legacy
record. Give the new FD its own Work Item IDs and record what remains to be
verified. Never synthesize FD events or Reviewer approval from historical Core
records. Archive or delete the old Task only through a separate explicit
decision after reconciling its outstanding work.
