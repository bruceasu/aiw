# FD-002: Actionable FD handoff recovery hints

**Status:** Complete
**Revision:** 6
**Priority:** Medium

## Problem

When `AIW_FD_ROLE_RUNNER` is unset, `aiw fd emit` and `aiw fd resume`
leave a role handoff pending. The current message says to use the role Skill,
but omits the command that binds the pending event to the host Session. An
operator must discover the exact event ID and `claim` syntax elsewhere before
the role can safely write. This adds friction to the FD-001 real Agent pilot.

## Options and decision

Option A: change only the pending handoff message to print the exact `claim`
command with the FD and event IDs and a Session ID placeholder. This reuses
the existing receipt and claim rules. Option B: add automatic host-session
discovery and claiming. That would introduce host-specific state and change
the ownership boundary. Choose A because the host already knows its Session
ID and can run the existing atomic `claim` operation.

## Solution

In the no-runner branch of role dispatch, print
`aiw fd claim <fd-id> <event-id> --session <host-session-id>` using the actual
FD and event IDs. Keep the role name and pending state visible. The placeholder
means the operator supplies the current host Session ID; AIW does not invent
or infer one. `emit` and `resume` share this branch, so both paths give the
same next action. Keep the receipt pending until a separate `claim` succeeds.

Do not change the `claim` command, event schema, dispatch state transitions,
runner invocation, or behavior for launched, dispatched, human, or PM events.
Those events must continue to direct the operator to the original Session,
process, or log. No new persistence, permission, dependency, or migration is
required. The CLI output gains a more specific hint; existing command forms
remain valid.

## Scope

Scope: the no-runner pending-role output in `plugins/aiw-fd.py`, the matching
CLI usage text if needed, and focused verification of the printed instruction.
Excluded: automatic Agent launch or claim, recovery of unknown in-flight work,
Supervisor/Core changes, and OpenSpec change creation.

## Work items

- [x] 1.1 Make the no-runner role handoff message include the exact `aiw fd
  claim` invocation for its FD and event, with `<host-session-id>` as a
  placeholder. Evidence: inspect the changed `dispatch` branch and capture
  the output from a focused local FD scenario when runtime validation is
  authorized.
- [x] 1.2 Check the pending `emit` and `resume` paths and the in-flight
  recovery path against the acceptance criteria. Update usage notes only if
  they would otherwise contradict the CLI. Evidence: static call-path review;
  any runtime command and result must be recorded in Verification.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

1. With no role runner, a newly emitted pending Planner, Worker, or Reviewer
   handoff prints its actual FD ID, event ID, role, and a copyable `aiw fd
   claim ... --session <host-session-id>` command. The receipt remains pending.
2. Resuming the same pending event without a runner prints the same command
   and does not create another event or writer.
3. A launched or claimed event with an unknown outcome keeps its existing
   instruction to inspect its original Session, process, or log; it does not
   suggest a new claim or redispatch.
4. The displayed command is executable after substituting a valid host
   Session ID, and the existing claim rule still rejects another Session.

## Verification

- Planner stage: read `plugins/aiw-fd.py` dispatch, resume, and claim paths;
  read FD-001 and `openspec/specs/fd-workflow/spec.md`. No code or runtime
  validation performed in this stage.
- Worker: perform the narrowest compile-only check after coding. A focused
  runtime scenario may be run only under repository authorization rules.
- Worker: the no-runner branch now interpolates `fd_id`, `event_id`, and the
  target role into the pending message. `emit` and pending `resume` both call
  `dispatch`; launching/dispatched `resume` still uses its separate original
  session/process/log message. The existing `claim` accepts a valid supplied
  Session ID once and rejects another Session. The usage note already describes
  this command, so no documentation edit was needed. Compile-only check passed:
  `python -c "from pathlib import Path; p=Path('plugins/aiw-fd.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec')"`.
  Focused runtime output
  was not captured because runtime validation was not authorized for this FD.
- Reviewer: independently compare the implementation and actual evidence
  with Acceptance, including pending and in-flight behavior. Record checks
  that were not run.
- Reviewer: static review passed; see `docs/features/archive/FD-002/reviews/FD-002-review.md`.
  The exact Reviewer handoff was claimed by `agent-reviewer-fd002`.
  No focused `emit`/`resume` output scenario or competing claim was run;
  those runtime results remain unverified.

## Sources

- Issue: none; FD-001 real Agent pilot identified the usability gap.
- `docs/features/FD-001_EVENT_DRIVEN_FD_WORKFLOW.md`: explicit handoffs,
  single writer, and safe recovery are required.
- `openspec/specs/fd-workflow/spec.md`: pending receipts may be claimed by one
  host Session; unknown in-flight work must not be dispatched twice.
- `plugins/aiw-fd.py`: current `dispatch`, `resume`, and `claim` behavior.

**Completed:** 2026-09-29
