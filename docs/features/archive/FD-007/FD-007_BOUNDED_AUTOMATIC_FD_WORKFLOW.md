# FD-007: Bounded automatic FD workflow

**Status:** Complete
**Revision:** 9
**Priority:** Medium

## Problem

The numbered FD workflow has separate design, implementation, review, repair,
and archive handoffs. A user currently has to request each stage. A long
implementation can stall after a review finding or be archived without a
current independent review when a host bypasses the event protocol.

## Options and decision

1. **Add a host-operated `$fd-workflow auto` operation (chosen).** The host
   agent plans and implements the FD, invokes the existing `aiw fd` lifecycle,
   delegates every Reviewer pass to an independent subagent, and stops after
   three review attempts. This uses the host's actual subagent capability and
   leaves event ownership with the existing CLI.
2. Add a Python `aiw fd auto` daemon. The CLI cannot itself create a host
   subagent, and the optional role runner is an external executable. A daemon
   would either depend on a specific agent product or silently fall back to a
   same-session review.
3. Repeatedly invoke existing Skills by hand. This preserves the current
   workflow but does not deliver the requested one-operation experience.

## Solution

- Add an `auto` operation to the installed and source `fd-workflow` Skills.
  Accept either a concrete new feature request or one unambiguous numbered FD.
  For a new request, create an FD, split it into ordered numbered Work Items,
  resolve design decisions from evidence, and produce the normal design-ready
  handoff. Resume an existing FD from its recorded state without restarting
  completed work.
- The host acts as Planner and Worker, claiming exact pending handoffs before
  writing and citing the source event on each emitted result. It implements
  all ready Work Items, records real Verification evidence and unrun checks,
  and emits `implementation-ready` only after scoped items are resolved.
- Each Reviewer pass runs in a separate subagent with the `fd-review` Skill.
  The subagent claims the exact Reviewer event and writes a report and either
  `changes-requested` or `verification-passed`. The host does not review its
  own implementation or fabricate a Reviewer event.
- Count at most three independent Reviewer passes in the active implementation
  cycle, including earlier interrupted or resumed `auto` invocations.
  After a failed pass, the host fixes the reported findings and requests the
  next pass through normal events. If pass three still requests changes, stop
  before claiming the new Worker handoff, leave it pending, preserve findings,
  and report the remaining gate. A failed pass with no actionable repair also
  stops before claim; it is never marked passed.
- On a passed review, close as `Complete` with the existing `aiw fd close`
  operation only when its current revision and digest gate accepts it. The
  operation does not commit, push, merge, publish, or run restricted validation
  without separate authorization.
- Stop for material human choices, missing runtime authorization, an existing
  in-flight or foreign-claimed handoff, unavailable subagent capability, and
  unsafe or unsupported Task/legacy state. Report the exact resumable state;
  do not create a second writer or downgrade the review requirement.
- On every resumed `In Progress` Worker handoff, inspect the latest
  `changes-requested` report and prior Reviewer outcomes before claim. The
  third-failure or no-actionable-repair gate applies at entry as well as
  immediately after a Reviewer returns.

## Scope

In scope: one numbered FD per invocation, new FD allocation or unambiguous
resume, automatic Work Item decomposition and implementation, independent
Reviewer subagents, bounded repair, evidence-based Complete archive, and
source/installed Skill plus stable-spec documentation.

Out of scope: an autonomous background daemon, legacy Task/Core migration,
automatic Git writes, permission escalation, implicit test/build/network
authorization, bypassing human decisions, and changing existing CLI event
semantics.

## Work items

- [x] 1.1 Define the `auto` entry and state transitions, including new and
  resumed FD resolution, exact event claim/source rules, and stop conditions.
- [x] 1.2 Add independent Reviewer subagent orchestration with a maximum of
  three review attempts and explicit Worker repair between failed attempts.
- [x] 1.3 Update the stable FD workflow spec and usage instructions; verify
  both source and installed Skill copies describe the same operation.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## TODO

- [x] Add the source and installed Skill entry points and three-review gate.
- [x] Record the state and recovery contract in the stable spec and user guide.
- [x] Prepare the Worker report for independent review.

## Acceptance

1. `$fd-workflow auto` from a new request creates and designs one numbered FD,
   decomposes it into reviewable Work Items, implements them, and archives only
   after a separate Reviewer subagent records a current pass.
2. An existing FD resumes from its actual handoff and preserves completed
   items; a claimed/in-flight or stale handoff is not duplicated or skipped.
3. Review attempts never exceed three across resumes of one active cycle; every failed pass
   has its report preserved, each retry follows a Worker repair, and a third
   failure stops without archive.
4. The operation obeys repository authorization budgets and stops for missing
   decisions or capabilities rather than treating unrun checks as passed.
5. Existing manual FD commands and Skills remain usable.

## Verification

- Design evidence: `skills/work-management.md`, `skills/fd-workflow/SKILL.md`,
  `.agents/skills/fd-workflow/SKILL.md`, `plugins/aiw-fd.py` handoff/close gates,
  `openspec/specs/fd-workflow/spec.md`, and `fd-review` Skill.
- After implementation: inspect the Skill and spec diff, trace the new and
  resumed paths against CLI states, and run one static diff check. This
  documentation-only change has no compile target; no test, build, or network
  command is required.
- Worker report: `docs/features/archive/FD-007/reports/FD-007-implementation.md`. Source and
  installed Skill copies both expose `auto`; the source procedure routes
  Design, Open/In Progress, Pending Verification, and Complete states to the
  corresponding existing CLI handoffs. It requires a separate Reviewer
  subagent, preserves the three-review limit across resumes, and stops on
  missing authorization or an in-flight event. The stable spec and usage guide
  describe the same boundary.
- The `aiw fd new`, `claim`, and `emit design-ready` path was exercised for
  FD-007 itself, yielding the claimed Worker handoff
  `FD-007-000003-design-ready`. No automatic review/fix loop has been run yet.
- `git diff --check` on the changed tracked Skill, spec, usage, and index
  files passed. No compile-only check applies to this text-only change.
- Independent Reviewer report: `docs/features/archive/FD-007/reviews/FD-007-review.md`,
  `changes-requested` from `FD-007-000004-implementation-ready`. The Auto
  repair step must check the third-failure/no-actionable-repair stop gate
  before claiming the next Worker handoff; otherwise it leaves that handoff
  dispatched without a continuing Worker Session. This is a static finding;
  no runtime check was performed by Reviewer.
- Worker fixed that ordering in both Skill copies, the FD design, stable spec,
  and usage guide after claiming `FD-007-000005-changes-requested`. A focused
  `git diff --check` on the corrected tracked files passed. A fresh Reviewer
  pass is required; the first report remains historical evidence.
- Second independent Reviewer report: `docs/features/archive/FD-007/reviews/FD-007-review-r2.md`,
  `changes-requested` from `FD-007-000006-implementation-ready`. It found
  that a later auto invocation could still claim the intentionally pending
  Worker event before applying the review limit. Worker claimed
  `FD-007-000007-changes-requested` and added the count/actionability gate to
  resume preflight, source and installed Skills, spec, and usage guide. The
  third independent pass will assess this correction; no runtime test or
  build was run for the text-only repair.
- Second independent Reviewer report: `docs/features/archive/FD-007/reviews/FD-007-review-r2.md`,
  `changes-requested` from `FD-007-000006-implementation-ready`. The immediate
  repair path now checks before Worker claim, but resumed `auto` still routes an
  `In Progress` FD directly to Worker. Preflight must inspect the previous
  Reviewer outcome count and repairability before claiming a deliberately
  pending Worker handoff. No runtime check was performed by Reviewer.
- Third independent Reviewer report: `docs/features/archive/FD-007/reviews/FD-007-review-r3.md`,
  `verification-passed` from `FD-007-000008-implementation-ready`. Static
  review confirmed that both resumed preflight and immediate repair apply the
  review-count/actionability gate before a Worker claim. No end-to-end auto
  invocation, test, build, or network call was performed; the three-review
  limit remains a host Skill procedure rather than a CLI-enforced counter.

## Sources

- Issue: none; user request on 2026-10-01 for one operation covering split,
  design, implementation, repair, independent subagent review, and archive,
  with at most three Review/Fix rounds.
- Stable spec: `openspec/specs/fd-workflow/spec.md`.

**Completed:** 2026-10-01
