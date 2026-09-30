# AIW Work Management Contract

Use this contract for AIW engineering Skills.

This file is the canonical shared contract. A Skill that needs these rules
should reference `skills/work-management.md` in its `SKILL.md` and keep its
own instructions focused on its domain. Do not copy lifecycle rules into each
Skill. `aiw skills install` and `sync` materialize this file beside the Skill
when that reference is present; treat the installed copy as generated output.
The same applies to `skills/reviewed-skill-contract.md`, so user-level
installations remain usable outside the AIW repository.

## New FD workflow

- A numbered file under `docs/features/FD-XXX_SLUG.md` is the source of truth
  for design, Work Items, status, Verification, and unresolved `%%` notes.
- `docs/features/FEATURE_INDEX.md` is an index. Rebuild it from FD files; do
  not treat it as a second state owner.
- An Issue may link to an FD. An AIW Task is not required for a new FD.
- `aiw fd` is the FD command surface. Read `aiw fd --help` before an unfamiliar
  or mutating operation. `aiw fd emit` records an explicit handoff; ordinary
  file saves and Git commits do not dispatch roles.
- `.ai/fd/<fd-id>/` contains dispatch receipts and logs. It does not own FD
  status or replace the Markdown plan. Never write a fake event or review.
- OpenSpec owns stable capability specs in `openspec/specs/`. Create an
  OpenSpec change only when the user explicitly asks for one.

## Roles and handoffs

PM creates and grooms an FD. Planner records options, decisions, acceptance,
and numbered Work Items. Worker implements all ready items in dependency order.
Reviewer checks the actual diff and evidence in a separate session. A failed
review returns concrete findings to Worker. Human decisions and authorization
stop automatic dispatch until answered.

Use an explicit stage event after the producing role has written its result.
The event must name an existing project-relative artifact. If the host has a
configured role runner, AIW starts the next role. Otherwise the event stays
pending for a Skill or later `aiw fd resume`. Before a host session writes for
a pending handoff, claim its exact event with `aiw fd claim <id> <event-id>
--session <host-session-id>`; include that event in `--source-event` at the
next handoff. Do not redispatch an unknown in-flight result; inspect the
recorded session or log first.

FD Work Item checkboxes record authored progress. Mark one complete only when
the required change and real evidence exist. Report checks that were not run.
Do not mark the FD Complete until all scoped items are resolved and the
Reviewer has recorded a passed result.

## Workspace and Git

Use the primary workspace for ordinary sequential work. Use an FD worktree
only for parallel writes, conflicting changes, or a user request. Record the
parent branch and keep each FD's writing role in one workspace at a time.
Worktree creation requires the FD plan to be committed so it is present in the
new worktree. Do not mix unrelated files into an FD commit.

The user permits local Git commits for the new FD workflow. A Worker may make
a focused commit for an independently reviewable slice. A Reviewer checks a
specific commit or diff. Commit does not authorize push, merge, release,
deployment, worktree deletion, or archive. Follow any narrower local rule.

## Legacy Task compatibility

Existing `.ai/tasks/<task-id>/` records and Core state remain readable. A
legacy Task may continue from its existing `docs/features/<task-id>.md` or
OpenSpec `tasks.md`. Do not rewrite its completed items, evidence, Sessions, or
Git lineage to make it look like a new FD. The old `aiw issue promote --task`
and `aiw wf` commands remain compatibility paths until migrated explicitly.
Do not start `aiw wf supervise` for new work.

## Validation

Static review is the default. After code edits, run one compile-only check
using a repository `compile*` script when available. Tests, final builds,
formatters, linters, vet, network calls, and deployment follow the repository's
authorization rules. Report actual commands, skipped checks, and risks.
