# FD-016 Worker implementation report

## Result

- Updated native archived FD discovery, index generation, close, re-review,
  reopen, dispatch, claim, resume, and worktree guards to recognize
  `docs/features/archive/<FD-ID>/`.
- New close operations put the FD design and its current reports/reviews in
  the same per-FD archive directory. Close preflights destinations and rolls
  back file, receipt, and index changes on filesystem errors.
- Migrated 14 flat archived FD designs and 46 evidence files with
  `python scripts/migrate_fd_evidence.py`. The migration preflights design and
  evidence destinations, rewrites current Markdown path references, and rolls
  back moved files and rewritten documents if a filesystem write fails.
- Kept `.ai/fd/` event receipt artifact paths unchanged and left Task archive
  paths outside this migration.
- Updated the FD workflow stable spec, usage guide, work-management contract,
  and portable workflow reference.

## Evidence

- The migration command completed and reported 14 designs and 46 evidence
  files moved.
- Static inventory found zero flat archived FD designs, 14 nested designs, and
  no FD-scoped evidence left in the shared archive folders. Index links point
  to per-FD paths.
- Static search found no shared archive report paths in current feature docs,
  specs, or Skills.
- Static review confirmed archived FD resolution and indexing include the
  direct legacy layout and the new per-FD layout, while evidence files are
  excluded from FD design discovery.
- Static review traced nested archive checks through close, request-review,
  reopen, claim, resume, event emission, and worktree creation.
- Python compile-only check passed for `plugins/aiw-fd.py` and
  `scripts/migrate_fd_evidence.py` with in-memory `compile()`; no bytecode was
  written. `git diff --check` found no whitespace errors in the scoped
  tracked source/docs.
- Tests and disposable archive/close/reopen lifecycle checks were not run.
- Go compilation was not run because this FD changes Python-only code and docs.

## Residual risk

The migration and close rollback paths have not been exercised with injected
filesystem failures. Re-review/reopen behavior is statically inspected only.
