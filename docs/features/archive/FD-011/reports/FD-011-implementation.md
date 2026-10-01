# FD-011 Worker implementation report

- Source handoff: `FD-011-000003-design-ready`, claimed by
  `codex-fd011-20261001`.
- Scope: archive Worker and Reviewer evidence with its numbered FD.

## Changes

- `aiw fd close` selects only matching Markdown files in active `reports/`
  and `reviews/`, rejects destination collisions, moves them under `archive/`,
  and rolls back moved files and FD text if a move fails.
- Prior evidence remains archived during `request-review`; later close archives
  newly written evidence.
- One-time migration moved 32 existing report and review files for completed
  FDs into `archive/reports/` and `archive/reviews/`, and updated their archive
  document references. The migration script can be rerun after a partial move.
- FD workflow specification and working guidance now describe evidence archive.

## Verification

- Static inspection of close and request-review call paths, archive references,
  and scoped source diff. No archived document still references an active
  report or review path, and the active evidence directories were empty after
  the migration.
- Python compile-only check passed for the FD command and migration script.
- No tests, runtime lifecycle check, final build, network call, or Git write
  was run.

## Residual risk

Path-only edits to historical archived FDs change their present content digests
relative to older verification receipts. The receipts retain the original
reviewed revision and artifact path. This migration did not create a new
independent review for those historical FDs.
