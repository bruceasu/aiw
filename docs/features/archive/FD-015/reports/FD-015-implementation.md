# FD-015 Worker implementation report

- Source handoff: `FD-015-000003-design-ready`, claimed by host session
  `01a0f7b5-5fc4-70b0-ab29-69d25ce53239`.
- Scope: managed replacement of a stale pending Worker handoff. FD-014 and
  its receipt were not modified.

`plugins/aiw-fd.py` now exposes `refresh-worker` and creates a PM
`work-requested` receipt only for an active `Open` or `In Progress` FD with a
stale, pending Worker event. It validates the reason, role, state, FD path,
revision, digest, and new event path before mutation. The old receipt is
cancelled with a successor link; the new one records `supersedes`. The new
receipt stays unclaimable during writes and becomes pending last. On a write
or index error, the command attempts to restore the original FD and receipt
and remove the new event, reporting any incomplete rollback.

Go command help/completion, source FD Skills, usage guidance, and the stable
FD workflow spec now describe the recovery operation. Existing claim, resume,
and `implementation-ready --source-event` logic accepts the new Worker event
through its target role and event identity.

Static inspection covered the scoped code and call paths. The working tree
contains other uncommitted changes, so the aggregate Git diff is not solely
FD-015. `python scripts/compile.py` failed because `cmd/aiw-wf` is absent.
The corrected narrow compile command
`go build ./internal/commands/help ./internal/commands/completion` failed on
default Go cache access denied. Go compilation is unverified; no further
permission or cache workaround was attempted. Tests, disposable lifecycle
checks, final builds, network calls, and Git writes were not run.

## Compile follow-up (2026-10-02)

At the user's explicit request, reran the exact command
`python scripts/compile.py` with `GOTOOLCHAIN=local` and `GOPROXY=off`. It
completed successfully (exit code 0, no output). This supersedes the earlier
statement that Go compilation was unverified. The earlier failed attempts
remain recorded above. No FD content or receipt was changed, so the pending
Reviewer handoff remains bound to the same FD revision and digest.
