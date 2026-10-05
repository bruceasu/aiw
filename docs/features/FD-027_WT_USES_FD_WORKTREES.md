# FD-027: Make `aiw wt` use FD worktrees

**Status:** Open
**Revision:** 3
**Priority:** High  
**Test policy:** Independent  
**Evidence policy:** Dual

## Problem

`aiw wt` still resolves worktrees through the legacy Task lifecycle. New
engineering work is planned and identified by numbered FDs, so passing an FD
ID to `wt` fails or follows Task-specific handling instead of the FD worktree
contract. `aiw fd worktree add/status` already provides an FD-oriented entry
point, but the familiar `wt` command surface remains Task-oriented.

## Options and decision

Replace the removed Task/Workflow-backed `wt` implementation with an FD-native
plugin. Every worktree operation takes a numbered FD ID and resolves its
coordinates from `.ai/fd/<fd-id>/workspace.json`. Reject Task IDs with a clear
message to use an FD; do not recreate `wf` or migrate legacy Task records.
This follows the repository's current FD-first contract and avoids keeping an
entry point whose required Task workflow no longer exists.

Add `aiw wt local-merge <fd-id>` for delivery. Use only the FD's recorded
`parent_branch`, `branch`, and `worktree`; do not infer the target from the
current checkout. Require both worktrees to be on their recorded branches and
clean before starting. First attempt to merge the FD branch into its recorded
parent. If Git reports a content conflict, confirm the failed parent merge has
`MERGE_HEAD` and unmerged index entries, abort that merge, and verify the
parent worktree is clean again. Only then merge the recorded parent branch
into the FD worktree so conflicts can be resolved there. If abort or the
cleanliness check fails, stop and preserve the current state. If the initial
merge failed without a content conflict, report the Git error and stop without
starting the reverse merge. After resolution and commit in the FD worktree,
the user explicitly reruns `local-merge`; never retry delivery automatically.

## Solution

Restore a repository-owned `plugins/aiw-wt.py` that resolves FD IDs through
`.ai/fd/<fd-id>/workspace.json` and implements FD worktree creation, status,
commit, and `local-merge`. `wt add` delegates to the managed `aiw fd worktree`
path rather than creating a second workspace record format. Task IDs are not
accepted. Do not recreate or require Workflow Core (`wf`). Align help,
completion, and user documentation with the FD-only command contract.

## Scope

- The repository-owned `aiw-wt` plugin and its argument handling.
- FD-ID validation and workspace-record checks for `wt` operations.
- FD worktree add, status, and commit commands.
- An explicit FD `local-merge` flow, using the recorded parent branch and
  recovering content conflicts in the FD worktree after safely aborting the
  failed parent merge.
- Help and user-facing FD worktree documentation.
- No Workflow Core restoration, Task ID compatibility, Task data migration,
  worktree removal, or unrelated Git lifecycle changes.

## Work items

- [ ] 1.1 Implement FD-ID and workspace-record validation. Size: S; difficulty:
  Low; dependencies: none. Complete when malformed, missing, or mismatched
  records fail before Git state changes.
- [ ] 1.2 Restore the `wt` plugin with FD add, status, and commit routing. Size:
  M; difficulty: Medium; dependencies: 1.1. Complete when operations use the
  recorded FD branch/path and reject Task IDs without invoking Task/`wf` code.
- [ ] 1.3 Implement `local-merge` preflight and successful FD-to-parent merge.
  Size: S; difficulty: Medium; dependencies: 1.1. Complete when dirty or
  wrong-branch worktrees are rejected before merge and success targets only
  the recorded parent.
- [ ] 1.4 Detect a parent-side content conflict, safely abort it, and confirm
  parent recovery. Size: S; difficulty: Medium; dependencies: 1.3. Complete
  when non-conflict Git failures and abort/recovery failures stop without
  starting the reverse merge.
- [ ] 1.5 Merge parent into the FD worktree after confirmed recovery and report
  the manual resolution/retry path. Size: S; difficulty: Medium; dependencies:
  1.4. Complete when the parent is clean after recovery, conflicts remain
  isolated in the FD worktree, and no delivery retry happens automatically.
- [ ] 1.6 Align help and docs, update the stable FD worktree contract, and
  record static/compile and independent review evidence. Size: S; difficulty:
  Low; dependencies: 1.2–1.5. Complete when all current command guidance
  describes FD identity and local-merge behavior.

## Acceptance

Acceptance: `aiw wt` uses FD IDs and has no Task or `wf` dependency. Add
creates `feature/<fd-id>` at `.wt/<fd-id>` and writes the managed workspace
record with all four coordinates. Status and commit use the validated record.
Task IDs, unknown FDs, malformed records, wrong branches/paths, and dirty
worktrees fail actionably before changing Git state. `local-merge` merges only
the recorded FD branch into the recorded parent. On a content conflict, it
aborts the parent merge only after confirming the initial clean preflight,
verifies parent recovery, then merges parent into the FD worktree. Abort or
recovery failure stops with the current state preserved. Conflict resolution
occurs in the FD worktree; after committing resolution, delivery requires a
second explicit command. Non-conflict Git failures do not trigger reverse
merge. No command automatically removes a worktree or branch.

## Sources

- User report: `aiw wt` does not recognize FDs and remains on the old TASK flow.
- `cmd/aiw/main.go` dispatches `wt` to the external plugin fallback.
- `docs/usage/aiw-fd.md` describes `aiw fd worktree add/status` as the FD path.
- `README.md` documents legacy `aiw wt` operations using `<task-id>`.
- Commit `599b11d` removed the repository-owned Task/Workflow `wt` plugin;
  `C:\green\aiw\plugins\aiw-wt.py` is an installed copy still describing the
  old contract and is not the source to edit.

## Verification

- Not run; implementation has not started.
- Planned: focused Go compile-only command for the changed CLI/plugin package.
- Planned: static trace of FD workspace resolution and both merge directions.
- Planned: independent review of diff and merge failure preservation behavior.
- Runtime tests not authorized by repository defaults.
