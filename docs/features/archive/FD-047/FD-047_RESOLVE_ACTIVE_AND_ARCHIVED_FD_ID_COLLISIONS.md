# FD-047: Resolve active and archived FD ID collisions

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

`aiw fd` scans both active and archived feature files to resolve an FD ID. After an archived FD is reopened, the same ID has one active file and one archived historical file. `resolve_fd` currently treats the two matches as an error, so commands such as `show`, `resume`, and `set-status` cannot target the active FD. The failure is confirmed by FD-022: `aiw fd set-status FD-022 Deferred ...` returned `expected one FD file for FD-022, found 2` before writing anything.

## Options and decision

1. Require an explicit active/archive selector on every command. This makes intent explicit but expands the CLI contract and complicates ordinary work on reopened FDs.
2. Prefer the single active file when one exists; use a single archived file only when no active file exists. Continue to reject multiple candidates within the selected class. This preserves existing ID-only commands and gives reopened FDs a clear current target.
3. Pick the newest file by modification time. This makes selection depend on filesystem timestamps and can silently target historical evidence.

Decision: use option 2. An active FD is the source of truth while it is reopened. Historical archive files remain unchanged and are still listed by `aiw fd list`; commands addressed by ID resolve to the active copy. If no active copy exists, existing archived-FD workflows continue to resolve a unique archive. Ambiguous candidates in the chosen class remain errors.

## Solution

Change the shared `resolve_fd` helper to partition matches into active and archived candidates. Select the active candidate when exactly one exists. If there are multiple active candidates, fail with an ambiguity error. If no active candidate exists, select the archive candidate only when exactly one exists; otherwise fail with a useful count and candidate paths. Preserve repository-boundary and symlink checks on the selected path.

This behavior allows inspection and active-status commands to operate on a reopened FD without changing archived history. It does not alter archive destination naming: closing a reopened FD can still be rejected if its canonical archive destination already exists. It also does not make `set-status` archive the FD or validate terminal evidence; status override behavior remains recorded by the existing audit mechanism.

## Scope

Includes `plugins/aiw-fd.py`, CLI usage documentation, and the stable FD workflow spec if it defines ID resolution. No changes to archive contents, status override semantics, FD lifecycle events, or other CLI commands beyond their use of the shared resolver.

## Work items

- [x] 1.1 Prefer a unique active FD in shared ID resolution, with unique-archive fallback and explicit ambiguity errors. Acceptance: one active plus any number of archived matches resolves active; multiple active matches fail; no active plus one archive resolves archived; no active plus multiple archives fail; path containment and symlink checks still apply. Size: small; difficulty: low; dependencies: none.
- [x] 1.2 Document reopened-FD ID resolution and its archive-close limitation. Acceptance: CLI usage describes active precedence, archive fallback, and unchanged archive collision behavior. The stable spec was checked and does not define ID resolution, so no spec change is needed. Size: small; difficulty: low; dependencies: 1.1.

Work items are independently reviewable; preserve their IDs after implementation starts.

## Acceptance

1. ID-based commands resolve the unique active FD when that ID also has archived history.
2. Archived-only IDs continue to resolve when exactly one archived FD exists.
3. Multiple active candidates and archived-only ambiguity continue to fail explicitly; no arbitrary timestamp-based selection occurs.
4. The chosen path remains inside the repository and must not be a symlink.
5. `set-status` can reach the active reopened FD and retains its existing audit and stale-receipt behavior.
6. Historical archived files are not modified or overwritten. Archive destination collision behavior remains explicit.

## Verification

- Static evidence: `plugins/aiw-fd.py` builds `all_files()` from active and archived paths and `resolve_fd()` currently rejects any total match count other than one. `set_status()` and `show` both use `resolve_fd()`.
- Reproduction already observed: `aiw fd set-status FD-022 Deferred --reason "..." --operator PM` failed with `expected one FD file for FD-022, found 2`; the active FD remained In Progress at revision 8 and no status audit was created.
- Implemented `resolve_fd()` now prefers exactly one active candidate, falls back to exactly one archive when no active candidate exists, and reports candidate paths for ambiguity within the selected class. The selected path retains canonical repository-boundary and symlink checks.
- Updated `docs/usage/aiw-fd.md` to describe active precedence, unique archive fallback, ambiguity errors, and the unchanged archive-close collision.
- Compile-only check passed: `python -B -c "from pathlib import Path; compile(Path('plugins/aiw-fd.py').read_text(encoding='utf-8'), 'plugins/aiw-fd.py', 'exec')"`.
- No tests or runtime validation were run, consistent with the repository validation budget.
- Independent Reviewer passed; report: `docs/features/reviews/FD-047-review.md` (source event `FD-047-000004-implementation-ready`).
- Archive close for a reopened FD whose canonical archive target exists remains unverified by runtime here and is explicitly outside this change.

## TODO

- [x] Implement 1.1 and update CLI usage docs in 1.2; no stable spec change was needed.
- [ ] After this FD is reviewed and delivered, preserve FD-022 archived and active content; use the corrected resolver for the authorized active-status correction.

## Sources

- `plugins/aiw-fd.py`: `active_files`, `archived_files`, `all_files`, `resolve_fd`, `set_status`, and `show`.
- `docs/usage/aiw-fd.md`: status override and archive behavior.
- `openspec/specs/fd-workflow/spec.md`.
- FD-022 active/archived duplicate and the captured `set-status` error in this conversation.

**Completed:** 2026-10-09
**Disposition reason:** Implementation and independent review passed; squash-delivered to develop.
