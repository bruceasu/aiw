# Preflight and workspace selection

Read before starting or resuming implementation. Resolve shared resources via
[dependencies](dependencies.md). Read the current FD, relevant approved Issue
and stable specs, and repository limits. Inspect a receipt only when one exists.

## Workspace

Use the recorded isolated workspace by default. Before creating it, commit the
ready FD plan on the parent branch and require a clean parent. Do not include
unrelated edits. Use aiw git wt add <id>, then verify fd_id, parent_branch,
branch and worktree in .ai/fd/<id>/workspace.json. Keep Worker writes there.
The recorded parent is the only delivery target.

An explicit primary-workspace instruction skips worktree creation and its
workspace record. Preserve unrelated edits and the user's Git limits. A
workspace choice does not itself authorize implementation. Do not invent
Worker receipts. Keep Work Items, TODO and Verification in the FD current.

## Single host and managed handoffs

Without a runner or existing handoff, the host continues directly from the FD:
Design while planning, Open when ready, and In Progress during implementation.
No separate claim, role prompt or Worker report is needed. Do not mark Complete
before the applicable checks and review gate are satisfied.

With an existing managed handoff, read [handoffs](handoffs.md). Respect exact
event ownership, stale evidence and any in-flight role. If AIW_FD_ROLE_RUNNER
is set, the runner owns dispatch. Never take over an unknown session or reset
review/test budgets. Preserve legacy Pending Test/Test Acceptance states;
ordinary resume does not silently select that compatibility path.

Before delivery, decide the review policy under skills/work-management.md.
Routine Self work proceeds from recorded self-checks. Independent work uses
[review](review.md); request-review handles a ready direct-workspace FD without
a forced status transition. Missing decisions, scope conflicts or exhausted
repair budgets stop the affected work. Record routine blockers in the FD.
