# FD-108: Make FD directory creation and archive complete

**Status:** Complete
**Revision:** 6
**Priority:** Medium
**Review policy:** Independent
**Evidence policy:** Markdown

## Problem

`aiw fd new` creates both an active FD folder and an empty archive folder. The
archive process moves the design and selected evidence files but can leave
active directory trees behind and omit other FD-owned content.

## Solution

Create only the active index directory when a new FD needs it. On close, move
the complete FD-owned directory into the archive as one directory operation,
and retain rollback behavior if evidence/index updates fail. Remove empty
active folder trees after successful archival.

## Scope

Scope is limited to FD directory/index creation and `aiw fd close` archival in
`src/plugins/aiw-fd.py`. Existing CLI syntax and archive layout remain stable.

## Work items

- [x] 1.1 Avoid creating empty active/archive index directories during FD creation.
- [x] 1.2 Archive the complete FD-owned directory and clean up the active tree,
  with rollback on a failed close transaction.

## Acceptance

New FDs create an active folder without an empty archive counterpart. Closing
an FD preserves every item in its active folder under `archive/<FD-ID>/` and
leaves no empty active folder tree. Existing collision checks and rollback
restore the active tree if a close transaction fails.

## Verification

- Static review traced `new_fd` → `update_index` → `index_snapshot` and
  `close_locked` → evidence moves/index update/rollback. The active FD directory
  now moves as one unit; shared evidence rollback precedes restoration of it.
  Shared evidence that collides with FD-local evidence is rejected before moves.
- Compile-only check passed twice after the implementation and collision-guard
  edits: `python -c 'from pathlib import Path; p=Path("src/plugins/aiw-fd.py"); compile(p.read_text(encoding="utf-8"), str(p), "exec")'`.
- Tests are not authorized and were not run.
- Independent review is required because this changes filesystem persistence
  and archive rollback behavior.
- Independent review passed; report: `docs/features/FD-108/reviews/FD-108-000003-review.md`.
- The review report noted a policy wording inconsistency in its reviewed
  snapshot. This FD consistently requires Independent review; the historical
  report is retained unchanged.
- Focused re-review confirmed the wording reconciliation introduced no
  implementation drift; report: `docs/features/FD-108/reviews/FD-108-000005-review.md`.

## TODO

Archive FD-108 after this refreshed review pass.

## Sources

- Issue: none

Link relevant specs and evidence used for decisions.

**Completed:** 2026-10-27
