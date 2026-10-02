# FD-001 independent re-review

- Source event: `FD-001-000005-implementation-ready`
- Reviewed state: current HEAD plus the uncommitted FD-001 working diff; the FD-002 no-runner message and related pilot artifacts are separate scope.
- Outcome: static verification passed. The newly added negative smoke scenarios were not run.

## Resolution of returned findings

1. `plugins/aiw-fd.py`, `prepare_event` now requires a preceding Planner, Worker, or Reviewer event to be in `dispatched` or `launching` state before its role can complete a handoff. It also requires that exact event in `--source-event` and checks the producer against the target role. Thus a still-pending role event cannot be skipped by a later role result. `scripts/fd_smoke.py` now includes an attempted `changes-requested` before the Reviewer claim, followed by a claimed handoff; this new case has not been executed.
2. `plugins/aiw-fd.py`, `close_locked` now checks the latest Reviewer `verification-passed` event's revision and `fd_sha256` against the FD content before a Complete archive. `scripts/fd_smoke.py` now includes an altered FD after review and an expected archive rejection; this new case has not been executed. The stable FD spec and usage notes reflect the digest requirement.

No remaining material finding in the reviewed fix. The existing implementation report correctly distinguishes earlier passing smoke evidence, real FD-002 role handoffs, and the new unrun scenarios. FD-002's report does not claim runtime verification of its changed CLI hint. Legacy Task/Core read compatibility and the decision not to remove old code are preserved; no OpenSpec change was created. The new FD path does not call Supervisor.

## Commands and evidence

- Claimed `FD-001-000005-implementation-ready` with `python plugins/aiw-fd.py claim FD-001 FD-001-000005-implementation-ready --session agent-reviewer-fd001-r2` before review.
- Read FD-001, its Worker report, prior independent review, FD stable spec, changed CLI code and smoke scenario, and the working diff.
- Worker reported passing Python compile-only checks and `git diff --check` after the fix. The Reviewer did not rerun them.

## Skipped checks and residual risk

No test, updated smoke scenario, final build, network command, or Git write was run during this review. The new negative paths are supported by static call-path review only; they are not recorded as runtime passes. The external Agent host adapter remains unimplemented, and a real legacy migration trial has not run. These limits are recorded in the FD and implementation report.
