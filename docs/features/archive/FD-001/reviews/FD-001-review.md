# FD-001 independent review

- Source event: `FD-001-000003-implementation-ready`
- Reviewed base: current HEAD plus the uncommitted FD-001 working diff; `plugins/aiw-fd.py` and `scripts/fd_smoke.py` also contain the separate FD-002 no-runner hint and fixture files, which are excluded from these findings.
- Outcome: changes requested.

## Material findings

1. `plugins/aiw-fd.py`, `prepare_event`: the previous handoff is checked for `--source-event` and producer only when its state is `dispatched` or `launching`. A `pending` Planner, Worker, or Reviewer handoff can be completed by a later `emit` with no claim and no source event. For example, after `aiw fd new`, a manually edited FD can emit `design-ready --producer planner` while `design-requested` is still pending. This bypasses the stable spec's requirement that a host Agent atomically claims the exact event before writing and cites that event at the next handoff. It also permits the role transition with no bound writer Session. Require the matching claimed/dispatched event for host role completion, and preserve a deliberate recovery path for old or manually migrated FDs if needed. The current smoke scenario itself emits `changes-requested` from a pending Reviewer handoff without a claim, so it does not prove the stricter contract.
2. `plugins/aiw-fd.py`, `close_locked`: `Complete` checks that the latest event is a Reviewer `verification-passed` with the same FD revision, but does not compare the FD content to the event's `fd_sha256`. A later edit to Work Items, Verification, or acceptance text that leaves `**Revision:**` unchanged can still be archived under the old review. Bind completion to the reviewed FD content, or explicitly reconcile and review the changed content before archiving. This is a gap in evidence-based completion and current-reviewer protection.

## Evidence and scope

The FD format, index projection, role Skills, optional worktree, local-commit boundary, and legacy Task/Core guidance are present. `docs/features/archive/FD-001/reports/FD-001-implementation.md` accurately separates the disposable smoke runner from the real FD-002 role pilot and records unrun checks. FD-002's independent review and archive are cited as pilot evidence; its no-runner output scenario was not run. The FD-001 smoke check reportedly passed, but the Reviewer did not rerun it. No OpenSpec change was created for this work.

I read FD-001, its Worker report, `openspec/specs/fd-workflow/spec.md`, `skills/fd-review/SKILL.md`, `skills/work-management.md`, the FD CLI paths (`dispatch`, `prepare_event`, `claim`, `resume`, `close_locked`), `scripts/fd_smoke.py`, the FD template, index and usage documentation. I inspected `git status --short` and the working diff. I claimed the exact Reviewer handoff with `python plugins/aiw-fd.py claim FD-001 FD-001-000003-implementation-ready --session agent-reviewer-fd001` before review. No tests, runtime scenarios, builds, network commands, or Git writes were run during this review.

Residual risk: the real host Agent adapter remains absent; the pilot used manual host sessions. Legacy Task/Core records are preserved, but no real migration trial was run. These limits were already disclosed and are separate from the two blocking findings above.
