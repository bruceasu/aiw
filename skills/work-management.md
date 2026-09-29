# AIW Work Management Contract

Use this contract for engineering Skills in an AIW repository.

## Ownership

- An Issue describes a bug, feature, or modification and may be split into
  smaller Issues. Existing Requirement records and `aiw req` remain compatible
  Issue entry points until their data is migrated explicitly.
- Feature Design (FD) owns engineering decisions and planned work items.
  An approved Issue may lead directly to an FD and AIW Task.
- `.ai/tasks/<task-id>/` is the center of execution. AIW owns Task lifecycle,
  branch, worktree, Session, handoff lineage, and external mappings. Workflow
  Core owns Work Items, Attempts, Gates, Evidence, write leases, and derived
  execution, validation, and readiness state there.
- New Tasks use the Schema 9 execution path.
- OpenSpec owns stable capability specs in `openspec/specs/`. Create an
  OpenSpec change only when the user explicitly requests one. Stable spec
  updates and Task creation do not imply a change directory.
- GitHub and GitLab are optional Issue projections used only on explicit request.

Do not create a second Task tracker or treat `task.toml.status` as an
independently writable execution source. Request Workflow Core transitions
through the managed Task context and report the resulting summary. Keep
human-authored FD work items distinct from Core-owned generated state.

## Discover Commands

Before an unfamiliar or mutating operation, inspect `aiw help --json` or the
specific command help once. Use the installed command surface rather than
inventing subcommands. Do not install AIW or OpenSpec automatically.

## Resolve Or Create The Task

Resolve context in this order:

1. Task ID explicitly supplied by the user.
2. Task established by the active AIW Session.
3. AIW Task associated with the current worktree or branch.
4. Unique Task linked from the current Issue or FD.
5. Unique Task linked to an OpenSpec change, when one exists.

If several Tasks match, ask for the Task ID. Create a new lifecycle through
AIW. Use its native backend unless the user explicitly asks for an OpenSpec
change. Do not create a change just to satisfy Task creation.

One Issue may be split into smaller Issues when each has independently useful
scope. One Issue or FD may lead to multiple Tasks when delivery or archive
lifecycles differ. Record the lineage; do not silently duplicate work.

## Issue Decisions

Use confirmed user preferences and current evidence to choose a clearly better
approach within the approved scope. A bounded sub-agent may compare options;
the main agent remains responsible for the choice and records the rationale in
the Issue or FD. Ask the human when options are materially close, a critical
fact is missing, or the decision changes scope, risk, or authorization. Do not
interrupt for routine implementation choices.

## Feature Design And Specs

FD is the primary engineering planning artifact for a Task. It records the
goal, constraints, decisions, compatibility effects, acceptance evidence,
work items, TODO, Verification, and unresolved `%% NEEDS_INPUT` notes. Generate
or update Task work from its selected FD items. Implementation does not depend
on an OpenSpec `tasks.md` checklist.

Read relevant stable specs before changing behavior. Update `openspec/specs/`
when stable requirements change. Link an OpenSpec change only on explicit user
request. If linked, keep its artifacts
consistent with the FD, but do not let its checklist or lifecycle override the
Task. A legacy Task whose only plan is `tasks.md` remains actionable; migrate
its plan deliberately rather than discarding completed items.

## Workspace Rules

- Work in the primary Git checkout and current branch for ordinary sequential
  work. An isolated worktree is an explicit Task execution choice for parallel
  writes, conflicting work, long-running automation, or a user request.
- A Task lifecycle does not imply a feature branch or linked worktree.
- Use isolation for parallel writes, conflicting work, long-running work,
  disposable experiments, or an explicit user request. State the reason first.
- Create or resolve isolated worktrees only through `aiw wt`.
- Require explicit Task or Session context when several Tasks share primary.
- Treat `unassigned` and unknown workspace bindings as read-only until bound.
- The Task's FD, or a legacy `tasks.md`, must be committed on the parent
  branch before creating an isolated Task worktree, so it inherits the plan.
- Record `parent_branch` before creating the Task branch/worktree. Delivery
  targets that branch, never an inferred current checkout.
- Do not silently implement in a workspace that does not match the Task.
- Do not commit, merge, push, remove a worktree, or delete a branch merely
  because implementation is complete. Git delivery is separately authorized.
  Preserve Task resources on any failure or conflict.

If AIW is unavailable, report the missing capability and ask before using a raw
Git fallback.

## Delivery And Completion

`local-merge` is a delivery operation, independent of Task completion status.
After a successful merge, an unfinished Task remains unfinished and resumes in
the primary workspace. Preserve its Work Items, Evidence, Gates, and Session
lineage; do not mark it done or archive it as a side effect of delivery.

After implementation, update the selected FD work item, TODO, Verification,
and remaining `%%` notes. Request the appropriate Core transition and report
Task completion separately from Git delivery. Archive only under the Task's
documented archive rules. Archive a managed FD with its Task at
`docs/features/archive/<date>-<task-id>.md`; a linked OpenSpec change remains
optional. Do not automatically commit, merge, push, clean a
worktree, delete a branch, or archive.

## Validation

Static review is the default. After implementation, run one compile-only check:
prefer `scripts/compile*` or root `compile*`, otherwise the narrowest language
compiler command without retaining a final distributable artifact. Do not run
tests, final-artifact builds, formatters, linters, vet, or verification scripts
without authorization under the repository rules. Report actual commands,
static evidence, skipped checks, and unresolved risks.
