# FD-004: Re-review completed FDs after document changes

**Status:** Complete
**Revision:** 8
**Priority:** Medium

## Problem

Completed FDs live under `docs/features/archive/` and cannot emit another
review handoff. If an archived FD is edited after its accepted review, there is
no supported way to ask an independent Reviewer to review the current version
and restore valid completion evidence. Editing the index or writing an event
receipt manually would bypass the managed handoff contract.

## Options and decision

1. Permit ordinary `emit` on archived FDs. Rejected because it would make
   archive handling ambiguous and allow unrelated stage events on closed
   records.
2. Add `aiw fd request-review <FD-ID> --reason "..."`. Chosen: it is a narrow,
   explicit operation for requesting a fresh review of a Complete FD. The
   command owns the archive-to-active transition and records the reason in the
   event receipt.

## Solution

`aiw fd request-review FD-001 --reason "updated after acceptance"` MUST accept
an archived Complete FD. It MUST reject a non-Complete FD, a missing or
multiline reason, or a Complete FD without a latest acknowledged
`verification-passed` receipt from a Reviewer. The explicit request and reason
authorize reviewing the current content; the command does not infer whether
the document changed.

The operation MUST serialize with other FD mutations, increment the revision,
change status to `Pending Verification`, preserve the old completion date as
historical information, and create a `review-requested` receipt that captures
the current FD digest, reason, artifact, and Reviewer target. An archived FD
returns to the active feature directory and the index is rebuilt. The normal
claim/resume/dispatch protocol applies. A Reviewer then emits the existing
`verification-passed` or `changes-requested` result with the claimed source
event. A passed review can be archived again with `aiw fd close ... Complete`.

The command MUST NOT edit Work Items or synthesize a review result. Any new
implementation work found by the Reviewer is handled by an explicit FD edit.

## Scope

Add the CLI operation, its receipt and status transition, command help, usage
documentation, and the stable FD workflow specification. Preserve existing
`emit`, `claim`, `resume`, and `close` behavior for other event types. Do not
change legacy Task/Core behavior or add automatic Work Item reopening.

## Work items

- [x] 1.1 Implement `request-review`, receipt validation, and archived-to-active transition.
- [x] 1.2 Document the command and specify re-review lifecycle behavior.
- [x] 1.3 Perform static call-path review and compile-only validation; record checks not run.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

1. The example command creates a Reviewer handoff for the current FD version,
   records the one-line reason, increments the revision, updates the index,
   and makes the archived FD reviewable through the ordinary claim and emit
   flow.
2. A non-Complete FD, missing/multiline reason, or missing/unacknowledged prior
   Reviewer pass is rejected without creating an event or changing the FD.
3. A failed review returns the FD to In Progress through the existing
   `changes-requested` flow. A passed review records a digest for the reviewed
   version and permits a subsequent Complete archive.
4. No Work Item checkbox is changed by `request-review`.

## Verification

- Passed: Python in-memory compile-only check for `plugins/aiw-fd.py`.
- Passed: `git diff --check`.
- Not run: tests and `scripts/fd_smoke.py` under the default validation budget.
- Reopened Worker handoff `FD-004-000006-reopen-requested` was claimed by
  `codex-fd004-worker-20261001-r2`. Current `request-review` code, CLI
  documentation, and the stable FD spec were inspected against Work Items
  1.1鈥?.3 and the historical implementation report. No code change was needed.
- Historical Python compile-only and `git diff --check` results above were
  recorded before FD-004 was closed; this pass did not rerun them, tests, or
  the FD smoke script.
- Current Worker report: `docs/features/archive/FD-004/reports/FD-004-implementation-r2.md`.
- Independent Reviewer verification of the reopened FD is pending.

- Independent Reviewer: `docs/features/archive/FD-004/reviews/FD-004-review-r2.md` — verification-passed.
- Reviewer session: `fd004-reviewer-20261001-b6c0fe`; source event: `FD-004-000007-implementation-ready`.
- Tests and runtime lifecycle checks were not run in this review; historical compile-only and diff-check results were not rerun.
## TODO

- Independent Reviewer to assess the current FD, implementation, and both
  Worker reports. Do not mark Complete before a current Reviewer pass.

## Sources

- Issue: none
- `openspec/specs/fd-workflow/spec.md`
- `docs/usage/aiw-fd.md`
- Implementation report: `docs/features/archive/FD-004/reports/FD-004-implementation.md`

**Closed:** 2026-09-30
**Disposition reason:** 鐢ㄦ埛鏄庣‘璇锋眰鍏抽棴锛汧D-004 褰撳墠鏈畬鎴愮嫭绔?Verification锛屼笉澹版槑 Complete銆?
**Reopened:** 2026-10-01
**Reopen reason:** Continue independent verification and completion for FD-004

**Completed:** 2026-10-01
