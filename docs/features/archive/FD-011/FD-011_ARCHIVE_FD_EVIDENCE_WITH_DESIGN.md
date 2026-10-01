# FD-011: Archive FD evidence with design

**Status:** Complete
**Revision:** 5
**Priority:** Medium

## Problem

Closing an FD moves its design to `docs/features/archive/`, while Worker and
Reviewer reports stay in the active `reports/` and `reviews/` directories.
This separates historical evidence from the design it supports.

## Options and decision

Keep evidence in the active directories, or move it with the FD. Choose
`archive/reports/` and `archive/reviews/`: this retains the current FD file
layout and separates same-named report and review files.

## Solution

`aiw fd close` moves only matching Markdown evidence for the FD. It checks
destination collisions before writing and rolls back evidence moves if an
archive move fails. Reopening an FD leaves prior evidence archived; new
reports are archived at the next close. Existing completed FD evidence is
migrated. `.ai/fd/` receipts keep their original artifact paths as handoff
provenance.

## Scope

FD close behavior, existing numbered FD evidence, and affected workflow docs.
No change to event gates, CLI arguments, or Git history.

## Work items

- [x] 1.1 Archive matching Worker and Reviewer Markdown on FD close, preserving
  unrelated evidence and preventing destination overwrite.
- [x] 1.2 Migrate existing completed FD evidence and update path references.
- [x] 1.3 Update workflow docs and statically inspect close and re-review paths.

## Acceptance

- Close places this FD's reports and reviews under `docs/features/archive/`.
- Other FDs' evidence stays in place; a collision prevents overwrite.
- Re-review keeps prior evidence; subsequent close archives new evidence.
- Available evidence for existing archived FDs is migrated and linked.

## TODO

- Worker implementation complete; independent review remains pending.

## Verification

- `plugins/aiw-fd.py` close and request-review paths inspected statically.
- 32 existing evidence files moved; archive references were checked for old
  active evidence paths, with none remaining. Active evidence directories had
  zero files after migration.
- Python compile-only check passed for `plugins/aiw-fd.py` and the migration
  script; no bytecode artifact was written.
- Tests and runtime lifecycle checks were not run.
- Worker report: `docs/features/archive/FD-011/reports/FD-011-implementation.md`.

%% RISK: Historical archived FD text received path-only updates, so its current
%% digest differs from the older verification receipt. Those receipts still
%% identify the source content reviewed at the time; no receipt was changed.


- Independent Reviewer: `docs/features/archive/FD-011/reviews/FD-011-review.md` — verification-passed.
- Reviewer session: `fd011-reviewer-20261001-a7c39e`; source event: `FD-011-000004-implementation-ready`.
- Tests and runtime lifecycle checks were not run; rollback and migration recovery remain statically reviewed risks.
## Sources

- User request in this conversation (2026-10-01).
- `openspec/specs/fd-workflow/spec.md`.
- `skills/work-management.md`.

**Completed:** 2026-10-01
