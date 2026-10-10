# FD-053: Automatic recovery for missing FD handoffs and privileged state operations

**Status:** Complete
**Revision:** 7
**Priority:** Medium
**Evidence policy:** Dual

## Problem

Auto mode can encounter an active `Open` FD with no receipt or handoff. The
normal Worker claim path cannot proceed, and `refresh-worker` requires an
existing pending Worker event. Stopping to ask whether to create the missing
state repeats a decision that the user has now asked the workflow to handle
automatically. The workflow also exposes privileged state commands whose
effects differ and must not become blanket permission to skip other gates.

## Options and decision

1. Keep asking the user whenever an `Open` FD has no event. This preserves the
   current gate but does not meet the requested automatic recovery.
2. Use privileged commands for any missing or stale state. This is too broad:
   `set-status` creates no handoff, `cancel-event` does not stop its Agent, and
   `force-emit` skips workflow checks.
3. Add a narrow Auto recovery for an active `Open` FD with no event: record a
   blocker pair, then use audited `force-emit decision-recorded` to create a
   pending Worker handoff. Choose this option. It directly fixes the observed
   state, preserves the operation audit, and leaves unrelated privileged
   operations behind their existing specific gates.

## Solution

Update only `.agents/skills/fd-workflow/SKILL.md`. In Auto preflight, when the
FD is active, its status is `Open`, and no receipt exists, inspect the FD,
current branch/worktree, role runner, and repository cleanliness. Write and
commit the required blocker feedback pair with `source_event: null` before
creating the handoff. If the legacy FD lacks `**Revision:**`, add the initial
revision metadata and commit that plan update before emitting. Preserve the
existing runner rule: if `AIW_FD_ROLE_RUNNER` is configured, stop before
creating or claiming the handoff because that runner owns dispatch.

Use `aiw fd force-emit <id> decision-recorded --producer human` with the
blocker report as its artifact, a factual reason, and the declared operator.
For an `Open` FD, the current CLI routes this event to Worker. Claim that exact
event and continue in the recorded FD worktree. Keep the force operation audit
and report every skipped check; Worker must resolve ordinary scope, Work Item,
evidence, and validation requirements before `implementation-ready`. Skipped
checks are never evidence of completion.

Do not use `set-status`, `cancel-event`, or another `force-emit` type as
general-purpose recovery. `set-status` changes status without creating a
handoff or terminal evidence. `cancel-event` cancels a receipt but does not stop
its Agent; never use it for an unknown or in-flight session. Use other
privileged operations only for their documented condition after inspecting the
exact receipt and recording the reason, skipped checks, and residual risk.

## Scope

In scope: the Auto preflight recovery for an active `Open` FD with no event,
legacy Revision metadata needed to emit safely, and concise guardrails for
`set-status`, `cancel-event`, and `force-emit` in the same skill.

Out of scope: changing `aiw fd` command behavior, CLI authorization, generic
recovery for other statuses, replacing any claimed/in-flight event, changing
test/build authorization, or changing other skills and stable specs.

## Work items

- [x] 1.1 Add the audited no-receipt `Open` recovery and legacy Revision step to Auto preflight. (Size: S; Difficulty: Medium; Depends: none; Done when: exact conditions, blocker evidence, `decision-recorded` command, claim, worktree, and audit follow-up are explicit.)
- [x] 1.2 Define narrow usage boundaries for `set-status`, `cancel-event`, and other `force-emit` cases. (Size: S; Difficulty: Medium; Depends: none; Done when: command effects, in-flight restrictions, and non-passing skipped checks are stated.)
- [x] 1.3 Review the revised workflow for consistency with receipt, blocker, worktree, and authorization rules. (Size: S; Difficulty: Low; Depends: 1.1-1.2; Done when: no duplicate/conflicting rules remain and skipped evidence stays truthful.)

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

1. An active `Open` FD with no event is recovered automatically in Auto mode:
   blocker feedback is committed, a `decision-recorded` event is created with
   `force-emit`, and the exact pending Worker event is claimed.
2. If the FD has no Revision field, Auto adds and commits initial revision
   metadata before the force operation; a rejected command is not retried
   unchanged.
3. The force audit and blocker report retain skipped checks as skipped. Normal
   Worker evidence, review, merge, and archive gates still apply.
4. Pending standard events continue through their standard claim/refresh path.
   Claimed, launching, or dispatched events are not replaced by this recovery.
5. `set-status`, `cancel-event`, and unrelated `force-emit` uses are not
   automatically inferred from the no-receipt case; their distinct effects and
   recovery boundaries are documented.

## Verification

- Static review of the skill and relevant CLI contracts. Do not run tests or
  skill validation scripts under the repository's default zero-validation
  budget; report them as not run.
- Static review completed: the new recovery is restricted to active `Open`
  FDs with no event, preserves the role-runner and clean-parent/worktree gates,
  records skipped checks as unperformed, and leaves other receipt states on
  standard paths. Privileged command boundaries describe their distinct
  effects and do not imply that cancellation stops an Agent.
- `git diff --check develop...HEAD` passed. Tests and skill validation scripts
  were not run.
- Independent Reviewer passed `FD-053-000006-implementation-ready`; see
  `docs/features/reviews/FD-053-review-r1.md` and its JSON sidecar. Review was
  static; tests, runtime workflow, builds, and skill validation scripts were
  not run.

## Sources

- Issue: none
- Prior incident: `docs/features/archive/FD-051/reports/FD-051-blocker-20261010T151854+0900-no-handoff.md`
- Current command behavior: `src/plugins/aiw-fd.py`
- Workflow being updated: `.agents/skills/fd-workflow/SKILL.md`

**Completed:** 2026-10-10
