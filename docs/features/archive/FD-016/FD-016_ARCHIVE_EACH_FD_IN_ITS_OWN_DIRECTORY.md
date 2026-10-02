# FD-016: Archive each FD in its own directory

**Status:** Complete
**Revision:** 5
**Priority:** Medium

## Problem

Archived FD files and their reports/reviews currently share flat directories:
the FD Markdown files sit directly in `docs/features/archive/`, with evidence
under shared `archive/reports/` and `archive/reviews/`. This makes the archive
harder to browse by FD and does not match the requested per-FD archive path.

## Options and decision

1. Keep the flat FD files and shared evidence directories. This preserves the
   current implementation but does not satisfy the requested layout.
2. Put each archived FD and its evidence under
   `docs/features/archive/<FD-ID>/`. Choose this option so one directory
   contains the FD and the evidence needed to review its lifecycle.

## Solution

Keep active FD paths unchanged at `docs/features/FD-XXX_SLUG.md`. On archive,
place the design at
`docs/features/archive/<FD-ID>/<FD-ID>_SLUG.md`, with matching reports and
reviews under `reports/` and `reviews/` inside that FD directory. Update FD
lookup, listing/index generation, close, re-review, and reopen to resolve this
layout. Migrate existing flat archived FD files and shared evidence into
per-FD directories with collision preflight and rollback; never overwrite
existing files. Keep `.ai/fd/` event receipts and their original artifact paths
unchanged as provenance. Preserve active FD layout, IDs, lifecycle gates, and
CLI arguments. Legacy Task-owned FD archival under its separate date/task
layout is out of scope.

## Scope

FD archive path resolution and index projection, close/review/reopen lifecycle,
existing native FD archive/evidence migration, and the related workflow/spec
documentation. Excludes Task archive paths, event schema/CLI arguments, active
FD locations, Git history, and unrelated workspace modifications.

## Work items

- [x] 1.1 Update FD archive lookup, index, close, request-review, and reopen to
  use `docs/features/archive/<FD-ID>/` while retaining lifecycle and receipt
  semantics.
- [x] 1.2 Migrate existing flat archived FD files and matching evidence into
  per-FD directories with collision checks and rollback; update current path
  references and stable workflow guidance.
- [x] 1.3 Statically review affected path consumers and run one compile-only
  check; report tests and runtime lifecycle checks that were not run.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- An archived FD design is stored at
  `docs/features/archive/<FD-ID>/<FD-ID>_SLUG.md`; its reports and reviews are
  under that FD directory.
- `aiw fd list/show/close/request-review/reopen` resolve active and archived
  FDs correctly, and the generated index links to the nested archive path.
- Closing an FD preflights all per-FD destinations, rejects collisions before
  moving files, and rolls back partial moves on filesystem errors.
- Re-review preserves older evidence in that FD's archive directory; newly
  authored evidence is archived with the next close. Event receipt artifact
  paths remain unchanged.
- Existing flat native FD archives and shared archived evidence are migrated
  to the corresponding per-FD directory without loss or overwrite. Legacy
  Task-owned FD archive paths are unchanged.
- Current workflow docs and the stable FD workflow spec describe the new path;
  active-FD and Task-archive guidance remains accurate.

## Verification

- Migrated 14 flat archived FD designs and 46 evidence files with
  `python scripts/migrate_fd_evidence.py`; the command preflighted all
  destinations before moving files and rewrote references transactionally.
- Static review found no flat archived FD designs or FD-scoped evidence left
  in the shared archive folders; the index links to the nested FD paths.
- Static search found no remaining shared archive report path references in
  current feature docs, specs, and Skills.
- Python compile-only check passed for `plugins/aiw-fd.py` and
  `scripts/migrate_fd_evidence.py` using in-memory `compile()`; no bytecode was
  written.
- `git diff --check` reported no whitespace errors in the scoped source/docs.
- Tests and disposable archive/close/reopen lifecycle checks were not run.
- Go compilation was not run because FD-016 changes Python-only code and docs.
- Independent review: passed in `docs/features/archive/FD-016/reviews/FD-016-review-r1.md`.

## TODO

- Worker implementation and independent review are complete; pending Complete archive decision.

## Sources

- Issue: none
- `docs/features/archive/FD-011/FD-011_ARCHIVE_FD_EVIDENCE_WITH_DESIGN.md`
- `docs/features/archive/FD-013/FD-013_REOPEN_ARCHIVED_CLOSED_AND_DEFERRED_FDS.md`
- `openspec/specs/fd-workflow/spec.md`
- `plugins/aiw-fd.py`
- User direction in this conversation: `docs/features/archive/<FD-ID>/FD-ID_XXXX.md`.

**Completed:** 2026-10-01
