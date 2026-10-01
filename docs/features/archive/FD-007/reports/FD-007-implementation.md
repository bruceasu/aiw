# FD-007 Worker Implementation Report

- FD: `FD-007`
- Source handoff: `FD-007-000003-design-ready`
- Reviewed plan: `docs/features/FD-007_BOUNDED_AUTOMATIC_FD_WORKFLOW.md`, Revision 3
- Workspace: primary workspace

## Implemented

- Added `$fd-workflow auto` as a host-operated Skill mode in the source and
  installed Skill copies. It resolves a new request or one numbered FD,
  follows the existing exact claim and source-event protocol, splits the FD
  into numbered Work Items, implements ready items, and closes only after a
  current Reviewer pass.
- Required each review to run in a separate subagent using `fd-review`.
  Reviewer outcomes are capped at three across resumes of the same active
  implementation cycle. Failed reviews return concrete findings to Worker;
  a third failure stops with the FD active.
- Updated the stable FD workflow spec, the user guide, and portable operation
  references. Manual CLI and Skill entry points remain available. There is
  no new `aiw fd auto` CLI command or background process.

## Static evidence

- The source Skill's routing follows the CLI's Design -> Open -> Pending
  Verification -> In Progress/Complete states, claims pending handoffs, and
  prohibits taking over in-flight/foreign-owned work. Reviewer event emission
  is assigned solely to a separate subagent.
- The installed Skill points to the source procedure and preserves its
  managed Task design-only path. The stable spec and usage guide agree on the
  three-review cap, stop gates, and lack of implicit test/build/Git permission.
- FD-007 creation, Planner claim, `design-ready`, and Worker claim succeeded
  through the existing CLI. This does not prove an end-to-end auto run.

## Commands and limits

- Ran `go run cmd/aiw/main.go fd new "Bounded automatic FD workflow"`.
- Ran `go run cmd/aiw/main.go fd claim FD-007 FD-007-000002-design-requested --session codex-root-fd007-20261001`.
- Ran `go run cmd/aiw/main.go fd emit FD-007 design-ready --producer planner --artifact docs/features/FD-007_BOUNDED_AUTOMATIC_FD_WORKFLOW.md --source-event FD-007-000002-design-requested`.
- Ran `go run cmd/aiw/main.go fd claim FD-007 FD-007-000003-design-ready --session codex-root-fd007-20261001`.
- Ran a focused `git diff --check` on the changed tracked Skill, spec, usage,
  and index files; it passed. Git warned that index line endings may be
  normalized on a later Git write.
- No tests, final builds, network calls, dependency changes, or Git writes were
  performed. A compile-only command is not applicable to Skill/spec/usage
  text. The auto loop itself has not yet been invoked end to end.

## Remaining risk

The first independent review (`docs/features/archive/FD-007/reviews/FD-007-review.md`)
requested a correction: the third failed review previously claimed Worker
before stopping. The host claimed the first `changes-requested` Worker event,
then changed the source and installed Skills, FD, spec, and usage guide so the
limit and repairability are checked before a subsequent Worker claim. A
second independent review is required for this fix.
The focused `git diff --check` after this correction passed; no tests, builds,
or network calls were run for the repair.

The second independent review (`docs/features/archive/FD-007/reviews/FD-007-review-r2.md`)
found that a new auto invocation could still claim the pending Worker handoff
at entry, bypassing the third-failure/no-actionable-repair stop gate. The host
claimed the second `changes-requested` Worker event and added the same count
and report check before any resumed Worker claim. The FD, stable spec, and
usage guide now describe this restart path. A third independent review is
required; if it fails, auto will stop without archiving.

The operation depends on a host with a separate subagent capability. Its
instructions stop at a pending Reviewer handoff when that capability is
absent. The three-review gate is procedural Skill behavior, not a new CLI
enforcement rule; the existing CLI still enforces claim, revision, digest,
and archive rules.
