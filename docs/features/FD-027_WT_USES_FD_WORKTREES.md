# FD-027: Make `aiw wt` use FD worktrees

**Status:** Planned  
**Revision:** 1  
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

Keep the established `aiw wt` command and make FD IDs its primary identity.
Retain legacy Task support where it already exists, selected by the canonical
ID namespace (`FD-<number>` versus existing Task ID); reject ambiguous or
unknown IDs with a message that points to the correct FD or Task workflow.
This avoids breaking existing Task worktrees while ensuring new FD work does
not need a Task or Workflow Core record.

Add `aiw wt local-merge <fd-id>` for delivery. Use only the FD's recorded
`parent_branch`, `branch`, and `worktree`; do not infer the target from the
current checkout. Require the parent checkout to be clean and on the recorded
parent branch. First attempt the FD branch merge into the parent. If it fails,
leave the parent merge state and files untouched, return to the FD worktree,
and merge the recorded parent branch into the FD branch there. Report that
delivery did not complete and that conflicts must be resolved in the FD
worktree; after resolution, a new explicit `local-merge` invocation retries
the FD-to-parent merge. Preserve both worktrees and branch data on every
failure. This direction keeps conflict resolution isolated and avoids
discarding the failed parent merge state.

## Solution

Make `aiw wt` resolve numbered FD IDs through
`.ai/fd/<fd-id>/workspace.json`, implementing add/status and delivery against
the recorded FD workspace coordinates. Keep existing Task-ID behavior under
the Task namespace. Do not recreate or require Workflow Core (`wf`) as an
intermediary. Provide the explicit `local-merge` delivery and failure-recovery
behavior decided above. Align command help, plugin behavior, shell completion,
and user documentation.

## Scope

- The `aiw-wt` command/plugin and its dispatch/argument handling.
- FD worktree metadata and operations needed for `wt` to accept FD IDs.
- An explicit FD `local-merge` flow, using the recorded parent branch and
  recovering failed parent merges into the FD worktree for conflict resolution.
- Help, completion, and user-facing command documentation affected by the
  chosen interface.
- No Workflow Core restoration, Task data migration, or unrelated Git
  lifecycle changes.

## Work items

- [ ] 1.1 Add FD-ID resolution and workspace-record loading for `wt`, with
  clear unknown/malformed-record errors; keep Task ID dispatch intact.
- [ ] 1.2 Implement `wt` add and status for FD IDs using recorded branch and
  worktree paths, with focused acceptance evidence.
- [ ] 1.3 Implement `local-merge` clean-parent preflight and successful
  FD-to-parent delivery using only recorded branch coordinates.
- [ ] 1.4 On failed FD-to-parent merge, preserve parent state and merge parent
  into FD worktree; report recovery state without auto-retrying delivery.
- [ ] 1.5 Align help, shell completion, and docs; record compile/static and
  independent review evidence.

## Acceptance

Acceptance: FD IDs resolve without Task/Workflow Core records; add creates
`feature/<fd-id>` at `.wt/<fd-id>` from the recorded parent and writes a
workspace record with all four coordinates. Status reports the recorded branch,
parent, path, and Git status. `local-merge` succeeds only after a clean-parent
preflight and merge into the recorded parent. If delivery merge fails, the
parent remains unmodified, the parent branch is merged into the FD worktree,
and the command explains how to resolve conflicts and rerun delivery. Unknown
IDs, malformed workspace records, wrong branch/worktree, dirty parent, and
Git failures produce actionable errors and preserve user data. Existing Task
IDs keep their current behavior. No operation removes worktrees or branches.

## Sources

- User report: `aiw wt` does not recognize FDs and remains on the old TASK flow.
- `cmd/aiw/main.go` dispatches `wt` to the external plugin fallback.
- `docs/usage/aiw-fd.md` describes `aiw fd worktree add/status` as the FD path.
- `README.md` documents legacy `aiw wt` operations using `<task-id>`.

## Verification

- Not run; implementation has not started.
- Planned: focused Go compile-only command for the changed CLI/plugin package.
- Planned: static trace of FD workspace resolution and both merge directions.
- Planned: independent review of diff and merge failure preservation behavior.
- Runtime tests not authorized by repository defaults.
