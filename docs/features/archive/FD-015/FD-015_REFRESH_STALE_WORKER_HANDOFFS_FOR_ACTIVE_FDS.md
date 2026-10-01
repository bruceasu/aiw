# FD-015: Refresh stale Worker handoffs for active FDs

**Status:** Complete
**Revision:** 5
**Priority:** High

## Problem

PM revised FD-014 after its `design-ready` event. The FD is now revision 8,
while the latest pending Worker receipt is bound to revision 6. `claim` and
`resume` correctly reject this mismatch, and the existing `request-review`
recovery command applies only to `Pending Verification` FDs. Without a
managed replacement, Worker cannot safely continue FD-014.

## Options and decision

Closing and reopening FD-014 would record a false terminal disposition.
Editing its receipt by hand would bypass provenance and claim safeguards.
Choose a narrow PM operation, `aiw fd refresh-worker <fd-id> --reason <text>`,
which supersedes only a stale pending Worker handoff. This FD is the separate
prerequisite requested by the user; FD-014 remains untouched until the new
operation is reviewed.

## Solution

The operation accepts an active `Open` or `In Progress` FD and a one-line,
nonblank reason of at most 500 characters. Its latest event must target
`worker`, be `pending`, and differ from the current FD revision or normalized
content digest. Reject a current handoff, an archived FD, missing or in-flight
receipt, path collision, and malformed reason before mutation. Do not create a
new event by claiming or acknowledging the stale Worker receipt.

Under the FD mutation lock, increment the current FD revision and create a
PM-produced `work-requested` event targeting Worker. Include the reason,
`supersedes` old event ID, current FD path, revision and digest. Mark the old
pending receipt `cancelled` with `superseded_by`. Preserve its original fields
and the FD's status and Work Items. If any write or index update fails, restore
the original FD and old receipt and remove the new receipt so no claimable
orphan remains. The new event follows ordinary claim, resume, and
`implementation-ready --source-event` rules; generic `emit` does not create
`work-requested`.

Document the command and recovery rule in Python CLI help, Go help and shell
completion, the stable FD workflow spec, source Skill guidance, and usage docs.
Keep existing FD event and `request-review` behavior compatible. No migration
or external dependency is needed.

## Scope

In scope: the managed stale Worker handoff replacement and its command/docs.
Out of scope: implementing FD-014's Tester lifecycle, changing FD-014's
authored content or receipt during this FD, replacing an in-flight handoff,
and automatic dispatch when no role runner is configured.

## Work items

- [x] 1.1 Implement guarded `refresh-worker` transition with rollback and
  supersession provenance.
- [x] 1.2 Add CLI help, completion, stable spec, Skill, and usage guidance.
- [x] 1.3 Inspect the scoped diff and call path, run one compile-only check,
  record limitations, and hand off to an independent Reviewer.

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- A stale pending Worker event for an active `Open` or `In Progress` FD is
  cancelled and linked to one new PM `work-requested` event bound to the new
  FD revision/digest; the FD remains active with Work Items intact.
- The new Worker event can be claimed and used as the exact source of
  `implementation-ready`; the old event cannot be claimed again.
- A current, claimed, launching, or dispatched handoff; a non-Worker handoff;
  an archived FD; invalid reason; or event collision is rejected without
  mutation. Failure during writes restores prior FD/receipt state and leaves
  no claimable orphan.
- Documentation identifies the operation as a PM recovery action and keeps
  `request-review` for stale Reviewer handoffs in `Pending Verification`.

## Verification

- Before handoff, statically inspect validation, mutation, rollback, claim,
  resume, and `implementation-ready` paths and the scoped diff.
- After code edits, run the narrowest compile-only check for the changed Go
  help/completion packages; Python source must parse via a compile-only check
  if available without generating a final artifact.
- Tests, disposable lifecycle runs, final builds, network calls, and Git write
  operations are outside the default authorization and are not planned.
- Static inspection traced `refresh-worker` through receipt creation,
  cancellation, rollback, `claim`, `resume`, and the existing
  `implementation-ready --source-event` gate. The repository already had
  unrelated uncommitted changes, so its aggregate diff is not an isolated
  FD-015 patch.
- Independent review passed: `docs/features/archive/FD-015/reviews/FD-015-review-r1.md`.
  Review source event: `FD-015-000004-implementation-ready`.
- `python scripts/compile.py` could not complete because its target
  `cmd/aiw-wf` directory is absent. The narrow Go package compile attempt
  `go build ./internal/commands/help ./internal/commands/completion` then
  failed with access denied in the default Go cache. No alternate cache or
  escalation was attempted after that permission failure. Go compilation
  remains unverified.
- Implementation report: `docs/features/archive/FD-015/reports/FD-015-implementation.md`.

## TODO

- Independent Reviewer to inspect the implementation and the reported
  compile limitation. Apply the command to FD-014 only after FD-015 passes
  review; do not claim FD-014's stale event.

## Sources

- User decision to use a separate prerequisite FD (2026-10-02).
- FD-014 and its stale `FD-014-000006-design-ready` receipt.
- `openspec/specs/fd-workflow/spec.md` and `docs/usage/aiw-fd.md`.
- `plugins/aiw-fd.py` existing claim, request-review, emit, and resume paths.

**Completed:** 2026-10-01
