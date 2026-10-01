# FD-013: Reopen archived Closed and Deferred FDs

**Status:** Complete
**Revision:** 9
**Priority:** Medium

## Problem

Archived `Closed` and `Deferred` FDs cannot resume under the managed FD
workflow. `request-review` accepts only archived `Complete` FDs, so continuing
FD-004 would otherwise require a new ID or manual file and receipt edits.

## Options and decision

Create a new FD for every continuation, or add a managed reopen operation.
Choose `aiw fd reopen <fd-id> --reason <text>` for archived `Closed` and
`Deferred` FDs. It preserves the original FD ID and history, while keeping
`request-review` as the path for archived `Complete` FDs.

## Solution

Reopen requires a one-line reason and an archived terminal FD with no
in-flight role. It increments the revision, preserves earlier close metadata
and Work Item checkboxes, records the reopen reason, moves the FD to the
active directory as `In Progress`, rebuilds the index, and creates a
`reopen-requested` Worker handoff bound to the new FD digest. Existing
archived evidence stays in the archive; new evidence is written to active
reports and reviews. Worker claims the exact event before writing and then
uses the normal `implementation-ready` and independent review path.

## Scope

FD CLI, command help, FD workflow spec and guidance, plus a one-time invocation
to reopen FD-004 as explicitly requested. No Git delivery or automatic
recovery of other historical work. A `Complete` FD uses `request-review`.

## Work items

- [x] 1.1 Add safe `reopen` validation and file/event transition for archived
  `Closed` and `Deferred` FDs.
- [x] 1.2 Document the new command and continuation lifecycle in CLI help,
  user guidance, and stable spec.
- [x] 1.3 Statically trace the new handoff through claim, emit, and close;
  run one compile-only check.
- [x] 1.4 Use the new operation to reopen FD-004, preserving its prior
  disposition and creating a pending Worker handoff.
- [x] 1.5 Repair the Reviewer finding by adding managed correction of an
  unclaimed reopen reason and correcting FD-004's unreadable reason.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- Archived `Closed` or `Deferred` FD reopens as active `In Progress` with a
  new Worker handoff, preserved prior close metadata and checkboxes, and an
  updated index.
- Missing/multiline reason, an active FD for ordinary reopen, archived
  `Complete` FD, in-flight receipt, or active destination collision is
  rejected before mutation.
- A failed file or receipt write does not leave an active FD and archived
  FD with the same ID, or a claimable orphan event.
- Prior evidence remains archived; a later close archives new evidence.
- FD-004 is active again with its historical closure recorded and a claimable
  Worker handoff; its unfinished verification remains to be handled separately.
- A mistaken reopen reason can be corrected only while its Worker event is
  pending; the FD and receipt retain matching digests and correction history.

## Verification

- Static call-path and scoped diff inspection traced `reopen` through
  `claim`, `implementation-ready`, and `close`.
- The initial FD-004 reopen succeeded, but the Chinese reason became unreadable
  during Windows argument transport. `python plugins/aiw-fd.py reopen FD-004
  --reason "Continue independent verification and completion for FD-004"
  --correct-reason` then corrected the pending handoff through the managed
  command. Read-only inspection found the FD and receipt reasons identical,
  their digest matching, one correction history entry, and the event still
  pending for Worker. FD-004 remains active at revision 6 with its prior
  close metadata preserved and is listed as In Progress.
- Go compile-only check passed for the changed help and completion packages:
  `go build ./internal/commands/help ./internal/commands/completion`, with
  network disabled and no distributable artifact. Python source parsed and
  ran during the actual FD-004 reopen. A separate Python compile-only command
  did not execute: managed PowerShell failed to load, then the allowed `cmd`
  retry misparsed the `-c` quoting.
- Tests, disposable lifecycle checks, and final builds were not run.
- Go help and completion packages compiled again after the reason-correction
  help text changed; the check passed with network disabled.
- Worker report: `docs/features/archive/FD-013/reports/FD-013-implementation.md`.
- Review round 2 provenance check: UTF-8 JSON decoding of FD-004's receipt
  gives `reason_corrections[0].previous_reason` as
  `\u7ee7\u7eed\u72ec\u7acb\u9a8c\u8bc1\u5e76\u5b8c\u6210 FD-004` (ASCII-escaped
  Unicode). `correct_reopen_reason` copies the pre-correction `event["reason"]`
  directly into this field. The review's unreadable display is inconsistent
  with that decoded value; the pre-correction receipt bytes were not retained,
  so the exact past bytes cannot be independently re-read. No FD-004 event was
  edited for this review. See `docs/features/archive/FD-013/reports/FD-013-implementation-r3.md`.

- Independent Reviewer round 3: `docs/features/archive/FD-013/reviews/FD-013-review-r3.md` — verification-passed.
- Reviewer session: `fd013-reviewer-20261002-73a9c1`; source event: `FD-013-000008-implementation-ready`.
- The round 2 apparent reason mismatch was a PowerShell decoding artifact; explicit UTF-8 parsing confirms the correction history value and the managed code path preserves the prior receipt reason.
- FD-004 completed and archived at revision 8 after its own current Reviewer pass. No FD-004 receipt was changed in this review.
- Tests, runtime lifecycle checks, and builds were not run in this review; round 3 changed no source code.
## TODO

- Independent review passed; no scoped TODO remains.


- Independent Reviewer: `docs/features/archive/FD-013/reviews/FD-013-review.md` — changes-requested.
- Reviewer session: `fd013-reviewer-20261001-c4b9d2`; source event: `FD-013-000004-implementation-ready`.
- Finding: FD-004 reopen reason is mojibake in the active FD and its handoff receipt; see the review report.
- Worker repair: the current FD-004 reason and handoff receipt are readable and
  digest-matched; the previous value remains in receipt correction history.
- Tests and builds were not run during review. Repository-built `aiw` plugin integration remains unverified.

- Independent Reviewer round 2: `docs/features/archive/FD-013/reviews/FD-013-review-r2.md` — changes-requested.
- Reviewer session: `fd013-reviewer-20261001-e71f4a`; source event: `FD-013-000006-implementation-ready`.
- Finding: the correction history's `previous_reason` differs from the actual prior reopen receipt value; see the review report.
- FD-004 has since completed and archived at revision 8; its reopen receipt is
  acknowledged. Do not alter its historical event to address this finding.
- Tests and builds were not run during review. Repository-built `aiw` plugin integration remains unverified.
## Sources

- User request in this conversation (2026-10-01).
- FD-004: `docs/features/archive/FD-004/FD-004_RE_REVIEW_COMPLETED_FDS_AFTER_DOCUMENT_CHANGES.md`.
- `openspec/specs/fd-workflow/spec.md`.

**Completed:** 2026-10-01
