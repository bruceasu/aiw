# AIW Work Management Contract

Use this contract for engineering Skills in an AIW/OpenSpec repository.

## Ownership

- AIW owns Task lifecycle status, branch, worktree, Session, handoff lineage,
  and external mappings.
- OpenSpec owns proposal, design, capability specs, and the detailed
  implementation checklist in `tasks.md`.
- GitHub and GitLab are optional projections used only on explicit request.

Do not create a second task tracker or let OpenSpec lifecycle state override the
resolved AIW Task.

## Discover Commands

Before an unfamiliar or mutating operation, inspect `aiw help --json` or the
specific command help once. Use the installed command surface rather than
inventing subcommands.

Current AIW task lifecycle commonly uses `aiw new`, `aiw show`, `aiw status`,
`aiw done`, and `aiw archive`. Worktrees use `aiw wt`.

Do not install AIW or OpenSpec automatically.

## Resolve Or Create The Task

Resolve context in this order:

1. Task ID explicitly supplied by the user.
2. Task established by the active AIW Session.
3. AIW Task associated with the current worktree or branch.
4. A unique matching OpenSpec change.

If planning work needs a new lifecycle, create it through AIW. `aiw new
<task-id> --backend auto` may delegate artifact creation to an installed
OpenSpec CLI and otherwise uses AIW's native backend.

If several Tasks match, stop and ask for the Task ID.

## Task And Change Mapping

Use one AIW Task and one OpenSpec change for work sharing the same goal and
archive lifecycle. A Task does not imply a dedicated branch or worktree.

Keep ordinary implementation slices as checklist items in `tasks.md`. Create a
separate AIW Task/change only when a slice needs an independent worktree,
delivery, or archive lifecycle, and only after the user approves the split.

## OpenSpec Change Creation

For a managed AIW change, AIW is the required entry point even when OpenSpec is
installed:

```text
aiw new <task-id> --backend auto
```

This creates the AIW lifecycle record and delegates proposal/spec artifact
creation to OpenSpec when available. The resulting change MUST contain
`openspec/changes/<task-id>/task.toml` with the Task ID, status, branch,
worktree, `parent_branch`, and Session mapping.

Do not use `openspec new change <task-id>` directly for a managed change. That
command creates OpenSpec artifacts but does not establish AIW lifecycle
metadata. If a third-party or upgradeable OpenSpec skill instructs direct
creation, this contract takes precedence; use AIW first and then continue with
the OpenSpec artifact steps.

An OpenSpec-only change is allowed only when the user explicitly accepts that
it is not tracked by AIW. It must not be handed to the AIW implementation or
completion workflow until the Task and `task.toml` have been reconciled.

## Workspace Rules

- Work in the primary Git checkout and current branch by default.
- A Task lifecycle does not imply a feature branch or linked worktree.
- Use isolation only for parallel writes, conflicting work, long-running work,
  disposable experiments, or an explicit user request. State the reason before
  creating it.
- Create or resolve isolated worktrees only through `aiw wt`; never create one
  silently.
- Require explicit Task or Session context when several Tasks share the primary
  workspace. Never select the most recent Task by guesswork.
- Treat `unassigned` and unknown workspace bindings as read-only until the user
  explicitly binds or repairs them.
- Specification artifacts created on the current branch must be committed
  before creating an isolated Task worktree. The new branch/worktree must inherit
  `task.toml`, proposal, design, specs, and `tasks.md` from that commit; do not
  copy them manually into the worktree.
- Record the source branch as `parent_branch` in the Task metadata before
  creating the Task branch/worktree. Completion merges only into that recorded
  parent branch; never infer the target from the current checkout.
- Do not use raw `git worktree` when AIW is available.
- Do not silently implement in a workspace that does not match the Task.
- Do not commit, merge, push, remove a worktree, or delete a branch merely
  because Task implementation is complete. Git delivery is separately
  authorized. Preserve Task resources on any failure or conflict.

If AIW is unavailable, report the missing capability and ask before using a raw
Git fallback.

## OpenSpec Operations

Use OpenSpec CLI for artifact apply, spec sync, or archive support when it is
already installed. Prefer AIW lifecycle commands that delegate with
`--backend auto` where supported.

If OpenSpec CLI is unavailable:

- continue from local artifacts when the requested operation can be completed
  safely;
- do not install it;
- report which OpenSpec operation was not executed.

For synchronization:

- OpenSpec may update AIW title, goal, and progress summary.
- AIW may update OpenSpec lifecycle status and Task/worktree references.
- Proposal, design, specs, and checklist content remain OpenSpec-owned.
- Stop on conflicts instead of overwriting either side.

Archive a primary Task after its checklist and Verification are complete; its
Git delivery remains unmanaged. Archive an isolated Task only after its branch
has been verifiably merged and its managed worktree and temporary branch have
been cleaned up. A cancelled discarded Task may archive with its reason.

## Sub-Agents

- At most two sub-agents may run concurrently.
- Use them only for bounded static analysis, code location, or independent
  implementation fragments.
- Sub-agents must not run tests, builds, network calls, permission escalation,
  commits, archive operations, or worktree operations.
- The main agent owns integration, lifecycle mutations, and the final report.
- Do not invoke `code-review` automatically after implementation.

## Tests

Writing or editing tests is allowed. Running tests, builds, type checks,
formatters, linters, vet, or verification scripts requires an explicit user
instruction to execute them.

After development is complete, ask once whether the user wants one focused test
command run. Include the exact command, scope, and expected duration. Default to
not testing when the user declines or does not respond.

Broader test or build scope requires separate approval.

## Completion

After implementation:

- update the selected `tasks.md` item, TODO, Verification, and remaining `%%`
  notes;
- synchronize the coarse AIW Task status without overwriting OpenSpec content;
- report static evidence and checks not run;
- report Task completion separately from Git delivery. Do not automatically
  commit, merge, push, clean a worktree, delete a branch, or archive. When Git
  delivery is explicitly requested, validate the recorded parent branch and
  preserve resources on partial or failed work.
