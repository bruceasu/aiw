# FD-046: Force archive FDs as Complete

**Status:** Pending Verification
**Revision:** 4
**Priority:** Medium
**Evidence policy:** Dual

## Problem

An operator may need to archive an FD after completion when the current Reviewer `verification-passed` event is unavailable. `set-status` can force the status to Complete, but `close` still rejects the missing review receipt, so it cannot perform the archive transaction.

## Options and decision

Keep normal `close` validation unchanged and add an explicit `--force` / `-f` path for Complete archive. The forced path will set Complete as part of the archive transaction, so the FD need not already be Complete. It will record that status and Reviewer checks were skipped and will never create Reviewer evidence. Keep the existing guard against archiving while a role execution has an unknown result; this protects the archived files from later writes by a still-running role.

## Solution

- Add `--force` with `-f` alias to `aiw fd close <id> Complete`.
- Forced Complete close requires a non-empty, single-line `--reason` of at most 500 characters. Other close outcomes reject `--force`.
- Under the existing FD lock, set status to Complete and increment revision, append the completion date and disposition reason, move the plan and matching report/review evidence, update the index, and write one operation audit. Treat the audit, FD, index, pending receipt, and evidence moves as one rollback-capable transaction.
- Audit the local operator, previous status/event, resulting status/revision, reason, skipped status and Reviewer checks, and `review_verified: false`. Do not emit or alter a `verification-passed` receipt.
- Continue rejecting an already archived FD, archive destination collisions, unsafe paths, launching/dispatched latest receipts, and filesystem failures. Roll back all changes on failure.
- Keep non-forced `close` behavior unchanged.

## Scope

- `plugins/aiw-fd.py`: CLI option validation, forced close transaction, audit and rollback.
- `openspec/specs/fd-workflow/spec.md`: forced close contract and scenarios.
- `docs/usage/aiw-fd.md`: usage and explicit review-evidence limitation.
- This FD and its implementation report.

No changes to Reviewer event semantics, `set-status`, `force-emit`, Agent cancellation, or concurrent role recovery.

## Work items

- [x] 1.1 Add forced Complete close with required reason, status update, truthful operation audit, and transactional rollback while retaining archive safety checks.
- [x] 1.2 Update the stable FD workflow specification and CLI usage documentation.
- [x] 1.3 Record static review and one narrow compile-only check; prepare the independent Reviewer handoff.

## Acceptance

- `aiw fd close FD-XXX Complete --force --reason "..."` and the `-f` alias archive an active FD from any defined status without a current Reviewer pass, set its archived plan status to Complete, increment revision, update the index, and move only matching evidence.
- Audit identifies the previous status/event, local operator, reason, resulting revision and skipped checks; `review_verified` is false and no Reviewer receipt is created.
- Missing/invalid reason, `--force` with Deferred or Closed, already archived FD, collision, unsafe path, latest launching/dispatched receipt, or any move/write failure leaves original state intact.
- Normal Complete close still requires the current, unforced Reviewer verification-passed event. Normal Deferred/Closed behavior is unchanged.

## Verification

- Static review traced parser/dispatch, forced and normal close branches, audit rollback, latest-receipt preservation, evidence moves, index update and path checks; docs and stable spec describe the same behavior.
- Compile-only command passed with no bytecode output: `python -c "from pathlib import Path; compile(Path('plugins/aiw-fd.py').read_text(encoding='utf-8'), 'plugins/aiw-fd.py', 'exec'); print('Python syntax compile passed; no bytecode written.')"`.
- As requested earlier, FD-044 was archived with the new workspace plugin script; the CLI printed that Reviewer verification was skipped and not recorded. See `reports/FD-046-implementation-r1.md`.
- The PATH `aiw` executable used an older external plugin copy and rejected `--force`; no state changed in that failed invocation. Updating that external copy was not part of this workspace implementation.
- Tests, fault injection, final builds, formatters, lint, and network checks were not run.
- Independent Reviewer handoff remains pending. Force archive is not Reviewer evidence.

## TODO

- [ ] Obtain an independent Reviewer result; do not treat forced archive as verification.

## Sources

- [Issue Plan](../issues/ISSUE-001/issue-plan.md)
- [Approved recovery Issue](../requirements/REQ00008-fd-force-recovery/requirement-plan.md)
- [Stable FD workflow specification](../../openspec/specs/fd-workflow/spec.md)
- [CLI usage](../usage/aiw-fd.md)
- `plugins/aiw-fd.py`: `force_transaction`, `close_locked`, and `evidence_moves`.
