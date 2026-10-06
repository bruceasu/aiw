# FD-027: Make `aiw wt` use FD worktrees

**Status:** Complete
**Revision:** 11
**Priority:** High  
**Test policy:** Independent  
**Evidence policy:** Dual

## Problem

`aiw wt` still resolves worktrees through the legacy Task lifecycle. New
engineering work is planned and identified by numbered FDs, so passing an FD
ID to `wt` fails or follows Task-specific handling instead of the FD worktree
contract. The public `aiw wt` command surface must use FD IDs directly; the
old `aiw fd worktree` subcommand is retired rather than retained as an alias.

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
commit, and `local-merge`. `wt add` directly creates the Git worktree and
writes the single managed workspace record. Remove `aiw fd worktree` from the
FD plugin, help, and completions so `aiw wt` is the sole worktree interface.
Task IDs are not accepted. Do not recreate or require Workflow Core (`wf`).
Align help, completion, and user documentation with the FD-only command
contract.

## Scope

- The repository-owned `aiw-wt` plugin and its argument handling.
- FD-ID validation and workspace-record checks for `wt` operations.
- FD worktree add, status, and commit commands; remove the duplicate
  `aiw fd worktree` command path.
- An explicit FD `local-merge` flow, using the recorded parent branch and
  recovering content conflicts in the FD worktree after safely aborting the
  failed parent merge.
- Help and user-facing FD worktree documentation.
- No Workflow Core restoration, Task ID compatibility, Task data migration,
  worktree removal, or unrelated Git lifecycle changes.

## Work items

- [x] 1.1 Implement FD-ID and workspace-record validation. Size: S; difficulty:
  Low; dependencies: none. Complete when malformed, missing, or mismatched
  records fail before Git state changes.
- [-] 1.2 Initial design routed `wt add` through `aiw fd worktree`; cancelled
  by the user's 2026-10-06 direction to retire that command and unify the
  implementation under `aiw wt`.
- [x] 1.3 Implement `local-merge` preflight and successful FD-to-parent merge.
  Size: S; difficulty: Medium; dependencies: 1.1. Complete when dirty or
  wrong-branch worktrees are rejected before merge and success targets only
  the recorded parent.
- [x] 1.4 Detect a parent-side content conflict, safely abort it, and confirm
  parent recovery. Size: S; difficulty: Medium; dependencies: 1.3. Complete
  when non-conflict Git failures and abort/recovery failures stop without
  starting the reverse merge.
- [x] 1.5 Merge parent into the FD worktree after confirmed recovery and report
  the manual resolution/retry path. Size: S; difficulty: Medium; dependencies:
  1.4. Complete when the parent is clean after recovery, conflicts remain
  isolated in the FD worktree, and no delivery retry happens automatically.
- [x] 1.6 Align help and docs, update the stable FD worktree contract, and
  record static/compile evidence. Size: S; difficulty:
  Low; dependencies: 1.1, 1.3-1.5, 1.7. Complete when all current command guidance
  describes FD identity and local-merge behavior.
- [x] 1.7 Implement direct `wt add` creation and metadata writing, then remove
  `fd worktree` dispatch and help. Size: M; difficulty: Medium; dependencies:
  1.1. Complete when `wt add` enforces a committed FD and clean parent,
  creates `feature/<fd-id>` at `.wt/<fd-id>`, records all four coordinates,
  and `aiw fd worktree` is no longer exposed. Implemented in the plugin and FD
  dispatch; R1 Reviewer checked this path and requested the 1.8 follow-up.
- [x] 1.8 Guard `wt add` against unignored FD management paths. Size: S;
  difficulty: Low; dependencies: 1.7. Added after Reviewer R1 found that a
  clean repository without `.ai/` and `.wt/` ignore rules becomes dirty after
  add. Complete when add checks both paths before any Git mutation and gives
  the required `.gitignore` rules on failure. R1 evidence: first Tester run
  observed `?? .ai/` and `?? .wt/`; the repaired fixture covered only ignored
  paths. The previous 1.7 remains completed; this is its follow-up guard.

## Acceptance

Acceptance: `aiw wt` is the sole worktree command, uses FD IDs, and has no Task
or `wf` dependency. Add directly creates `feature/<fd-id>` at `.wt/<fd-id>`
and writes the managed workspace record with all four coordinates. Status and
commit use the validated record. `aiw fd worktree` is absent from command
dispatch, help, and shell completion. Before creating Git state, add confirms
the FD worktree and metadata paths are ignored; otherwise it reports the
missing `.wt/` or `.ai/` rule and leaves the parent unchanged.
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
- User direction (2026-10-06): `wf` no longer exists; use FD in its place and
  retire `aiw fd worktree` in favor of one `aiw wt` interface.
- `cmd/aiw/main.go` dispatches `wt` to the external plugin fallback.
- `docs/usage/aiw-fd.md` previously described `aiw fd worktree add/status` as the FD path.
- `README.md` documents legacy `aiw wt` operations using `<task-id>`.
- Commit `599b11d` removed the repository-owned Task/Workflow `wt` plugin;
  `C:\green\aiw\plugins\aiw-wt.py` is an installed copy still describing the
  old contract and is not the source to edit.

## Verification

- Static review: plugin validation and merge paths were inspected against the
  acceptance criteria; help, completion, guides, and stable specs were aligned.
- Compile-only: Python syntax compilation passed for `plugins/aiw-wt.py` and
  `plugins/aiw-fd.py` using `python -c` and in-memory `compile()`.
- R1 Worker handoff: `FD-027-000004-implementation-ready`; implementation
  report: `docs/features/reports/FD-027-implementation-r1.md`.
- R1 Tester report: `FD-027-000005-test-report-ready`; first run 7/9 because
  the temporary repository omitted `.ai/` and `.wt/` ignore rules; authorized
  fixture repair and one rerun yielded 9/9. Of 19 acceptance scenarios, 14
  had runtime evidence (73.68%); branch coverage was not measured. See
  `docs/features/reports/FD-027-test-report-r1.md` and both authorization
  records. Uncovered S15-S19 were not counted as passed.
- PM accepted the bounded report under event `FD-027-000006-test-accepted`;
  the missing branch coverage and five uncovered scenarios remain exceptions,
  not behavior waivers. Decision: `docs/features/reports/FD-027-test-decision-r1.md`.
- Independent Reviewer R1 emitted `FD-027-000007-changes-requested`; report:
  `docs/features/reviews/FD-027-review-r1.md`. The three findings concern
  unignored management paths, obsolete README commands, and stale FD evidence.
- R2 repair: `wt add` now rejects missing ignore rules before Git mutation;
  README removes obsolete commands and explains the ignore prerequisite.
  `git diff --check` found no whitespace errors, and in-memory Python
  `compile()` of `plugins/aiw-wt.py` exited 0. Focused R2 test and independent
  review are pending.

## TODO

- [x] Submit the formal Worker completion through the primary worktree's
  `.ai/fd/FD-027` receipts, referencing the report committed in this FD
  worktree. Do not synthesize or copy role receipts.
- [x] Complete the first independent Tester report and PM decision under
  exact, revision-bound Planner authorization.
- [ ] Complete R2 Tester and independent Reviewer verification, then deliver/close via
  the recorded `aiw wt` workflow.

%% RISK: FD role receipts are stored under the primary worktree's ignored
%% `.ai` directory and are not shared with `.wt/FD-027`. Submit lifecycle
%% events from the primary worktree against the genuine receipt chain, and
%% keep implementation artifacts in this FD worktree. The primary workspace
%% contains unrelated FD-028 edits; preserve them during review and delivery.
%% RISK: R1 left S15-S19 without runtime evidence and did not measure branch
%% coverage. The R2 Tester and Reviewer must assess the new ignore preflight
%% and these remaining evidence gaps without treating unrun checks as passed.

**Completed:** 2026-10-06
