---
name: implement
description: Implement ready Work Items in one numbered FD, obtain independent review, and continue automatic local delivery and archive when gates pass, unless the user limits the stage.
---

# Implement

Read `skills/reviewed-skill-contract.md`, `skills/work-management.md`, and the
repository instructions. Use this Skill after a numbered FD is Open, or when
returning from a failed review. A user may limit work to one item; otherwise
continue through all ready items in dependency order. A normal FD implementation
request includes the shared contract's default local lifecycle; a separate
`fd-workflow auto` request is not needed after review. Preserve explicit
implementation-only, review-only, partial-item, and no-merge limits. Do not
design a missing decision during implementation.

## Read and prepare

Resolve the FD ID from the user, active session, or unique Issue link. Read
the FD, approved Issue evidence when present, relevant stable specs, code,
tests, and workspace rules. Read a linked OpenSpec change only when it exists
and affects the work. For an existing legacy AIW Task, continue under its
original Task and Core contract; do not silently convert it.

Before each item, check its outcome, dependencies, acceptance, and unresolved
`%%` notes. Use the FD's recorded workspace, isolated by default unless the
user explicitly selected primary-workspace implementation. Ensure the ready
plan is committed before creating its worktree. Stop at a real scope, authorization, or workspace
conflict. Continue independent ready items when one item is blocked.

When taking a pending Worker handoff in a host session, first claim its exact
event with `aiw fd claim <id> <event-id> --session <host-session-id>`. A role
runner dispatched by AIW has already claimed its event. Never take over an
event that names another active or unknown session.

## Implement and record

Make the smallest complete change for each item. Update its checkbox only
when the change and required evidence are real. Record TODO, Verification,
unresolved `%%` notes, and paths or commits that a Reviewer should inspect.
Focused local commits are allowed for independently reviewable slices when
the repository rules permit them. Stage only files that belong to this FD and
use `FD-XXX: description` unless the repository has a stronger convention.

Follow the repository's validation budget. Review the final diff statically.
After code edits, run one compile-only check. Do not run tests, final builds,
formatters, linters, vet, or network checks without their required authority.
Never write a passing result for an unrun command.

Before any next-stage event or managed review command, check the user's stage
and scope limit. For implementation-only or no-review requests, or selected
Work Items without authorization to review them, save the scoped result and
evidence and stop before dispatching Reviewer. A no-merge limit may permit
review but never delivery. An intentional stage limit is a requested stop,
not a failed gate or permission to fabricate completion of the whole FD.
If review is separately authorized for part of an unfinished FD, keep it scoped
without emitting a full-FD implementation-ready event or marking that FD ready.

Only when the stage permits independent review and the whole FD implementation
scope is complete, emit `implementation-ready` with
`--producer worker` and a project-relative implementation report as the
artifact. Include the source event ID when completing a dispatched handoff.
For the default path, `implementation-ready` routes directly to Reviewer after
the Worker completes the repository-authorized compile-only check and static
review. Optional testing is a separate `$fd-test` invocation and does not
create a workflow event or gate review. Existing FDs with the legacy
`Test policy: Independent` marker retain their recorded CLI routing, but the
default implementation flow does not dispatch a Tester or assess a test
report. If a new decision is needed, emit `needs-decision` with a written
question instead, then stop.

An independent Reviewer uses `fd-review` to check the FD, diff or commit, and actual evidence.
It emits `changes-requested` with findings or `verification-passed` with its
review report. A failed review returns to Worker; continue from the same FD.
Do not claim verification passed because Worker checked the FD box.

## Continue after review

The host PM continues through `fd-workflow`'s delivery and close procedure when
the full FD scope and current independent pass satisfy the shared contract's
gates. Do not proactively sync the parent; attempt reviewed delivery directly.
If it fails, follow fd-workflow recovery: diagnose before any sync, load
`resolving-merge-conflicts` for one bounded attempt at clearly supported
resolutions, preserve unresolved conflicts, and obtain a new independent review
when recovery changes implementation beyond the current pass. Verify the
recorded coordinates, clean workspaces, current FD digest,
and no implementation drift or unresolved blocker. Automatically local-merge,
use verified CLI cleanup, close Complete, and commit the archive/index changes.
For an explicitly selected primary workspace, skip worktree merge and follow
the managed review/archive path instead. Do not ask for another auto command.

An explicit stage limit ends at that stage; a partial FD does not qualify for
delivery. A dispatched Worker only emits its assigned handoff and the host PM
owns continuation. Do not take over a runner or foreign claim. A failed gate,
unresolved merge conflict, or delivery/cleanup/archive error stops the affected
step with state and blocker feedback preserved. The bounded conflict Skill
attempt does not authorize guessing, gate bypasses or unchanged retries.
This continuation grants no tests, builds, network, push, release, or deployment.

