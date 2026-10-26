# Local delivery, recovery, release and archive

Read after recorded self-checks or a required current independent pass, before delivery or archive.
For a delivery failure, read the recovery section below before any recovery
action. Load [blockers.md](blockers.md) at a Gate and before archive; use
[handoffs.md](handoffs.md) for event recovery. Never expand explicit stage limits.

## Deliver a ready FD

   Apply skills/work-management.md's review policy. Self work needs resolved
   Work Items and actual Verification evidence; Independent and unmarked legacy
   work needs its current matching Reviewer pass. Do not use force for ordinary
   Self completion. Honor no-commit/no-merge and other explicit stage limits.

   In direct-workspace mode, the reviewed changes are
   already in the primary workspace; skip worktree delivery and proceed to
   archive. In isolated mode, read `.ai/fd/<id>/workspace.json` and take its
   `parent_branch`, `branch`, and `worktree` as the merge coordinates. Confirm
   the parent branch is clean and still matches the recorded parent. Run
   `aiw git wt local-merge <id>` directly for one squash commit on the parent
   branch. Do not proactively `sync` every FD before review or delivery; a
   clean parent advancing alone does not invalidate unchanged reviewed FD work. On a
   parent-side content conflict, the command resets the failed squash, verifies
   the parent is clean, and merges the parent into the FD worktree for resolution.
   Pause delivery and load `resolving-merge-conflicts` for the bounded
   recovery below. If any conflict cannot be safely resolved, preserve the
   state and report the unresolved gate without aborting or retrying. Do
   not blindly run commit/sync/retry for every error. If recovery fails, leave the FD active and preserve
   the worktree and branch. On success,
   `local-merge` verifies the clean FD HEAD, exact recorded coordinates, the
   single-parent squash, and its `FD-Source` before removing the worktree and
   branch. It safely detaches only a verified `.ai` junction to the primary
   workspace and preserves `.ai/fd/<id>` receipts. If cleanup partly fails,
   report the delivered commit and remaining resources; do not roll back the
   delivery or repeat the squash. Recover only through a known, authorized
   CLI cleanup path; if no safe supported path is known, stop and report the
   cleanup Gate rather than inventing a command. Defer close until required
   cleanup is verified. Then close with `aiw fd close <id> Complete` from the
   parent workspace and commit only the archive/index changes. A close rejection is a
   Gate, never a reason to edit a receipt or archive manually.
   Report the merge result, archive path, review count, commands actually run,
   unrun checks, and residual risks.


## Recover a failed local delivery

Keep full failure output and inspect both workspaces, the delivered commit if
any, `MERGE_HEAD`, and the unmerged index. Distinguish dirty-workspace or
coordinate errors, actual content conflicts, and delivery that succeeded but
failed cleanup. Diagnose first; a failed command alone does not call for sync.
Never discard unrelated changes or repeat an unchanged failed command.

For an actual conflict in this full FD lifecycle, load the standalone
`resolving-merge-conflicts` Skill and attempt one bounded resolution based on
the FD, both histories and local evidence. This is authorized recovery, not a
waiver of gates or runtime limits. If the Skill is unavailable, intent is
unclear, or any conflict cannot be safely resolved, preserve the in-progress
merge and remaining conflict entries, report resolved and unresolved paths,
and stop. Keep any clearly resolved edits documented; do not abort, reset,
force a side or commit a partial resolution. Explicit stage limits and foreign
role ownership still apply. CLI recovery may already have merged the parent
into the FD; finish or diagnose that merge before considering a separate sync.
If the cause
requires a separate sync, use `aiw git wt sync <id> <parent_branch>` with the
recorded parent explicitly. Inspect and resolve each conflict, including
modify/delete intent, before staging. `wt commit` runs `git add -A` and commits
all changes; it does not choose a correct resolution or remove conflict markers.
Use it only when every changed file is intentionally resolved and in scope.

Record the recovery commit, diff, commands and blocker resolution. Refresh
self-check evidence after recovery. When Independent work changed outside its
current pass, do not deliver under that old pass. With no foreign or in-flight
role, use `aiw fd request-review <id> --reason "review recovered result"`.
For an existing Pending Verification FD use request-review only when its
handoff is stale and unclaimed; otherwise honor the existing valid handoff.
Update recovery evidence before requesting review, never edit receipts, and
give the separate Reviewer the current FD, original Worker report, recovery
evidence, commit and diff. Keep the active cycle's review count and stage
limits; stop at the existing review limit rather than resetting it. Resume
normal delivery only after refreshed self-check evidence, or a required current
independent pass, covers the recovered implementation and all delivery gates pass.
Do not require a new review merely
because the parent advanced without changing the reviewed FD implementation.

`local-merge` verifies delivery and cleans Git resources. The host then runs
`aiw fd close <id> Complete` and commits the archive/index separately. If only
cleanup failed after a verified squash, follow the CLI cleanup recovery path;
do not create another squash, revert delivery or call an unarchived FD done.

## Local delivery versus release

A required fd-review pass gates Independent delivery; its shared implementation
review method includes actual source correctness, Standards, Spec and evidence.
Do not additionally invoke standalone code-review or add its two reviewers to
every FD. A planning design review is separate evidence, not that final pass.

Self work uses recorded self-checks. Local merge/archive is not release
approval. Only when a real release is
planned, route the release-readiness request and operational evidence to
`fd-release-gate`. Its GO/GO_WITH_RISK/NO_GO assessment is outside normal
FD local completion and never grants push, publishing or deployment authority.

## Close and archive

A Self FD with resolved Work Items and actual Verification evidence can use
`aiw fd close <id> Complete` without force or a Reviewer event. Independent
and unmarked historical FDs still require a current passed Reviewer event.
Deferred and Closed need `--reason "..."`. Native archives use
`docs/features/archive/<FD-ID>/<FD-ID>_SLUG.md`; matching reports and reviews
go into that FD directory. Archive changes the FD path and index; it does not
merge or publish code. Update a changelog only if the repository uses one. Do
not commit, push, or merge as a side effect of design or archive unless the
user authorized that action. A normal numbered FD implementation request
provides the scoped local authorization described in the default lifecycle;
design-only requests, standalone commits/reviews, and explicit stage limits do not.

An explicitly chosen PM force override remains an audited exception for skipped
required evidence. Ordinary Self completion is not an override and needs no
additional confirmation. Never describe a self-check as an independent pass.
