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

Use one AIW Task and one OpenSpec change for work sharing the same goal, branch,
worktree, delivery, and archive lifecycle.

Keep ordinary implementation slices as checklist items in `tasks.md`. Create a
separate AIW Task/change only when a slice needs an independent worktree,
delivery, or archive lifecycle, and only after the user approves the split.

## Worktree Rules

- Create or resolve implementation worktrees through `aiw wt`.
- Default to one Task, one `feature/<task-id>` branch, and one
  `.wt/<task-id>` worktree.
- Specification artifacts created on the current branch must be committed
  before creating the Task worktree. The new branch/worktree must inherit
  `task.toml`, proposal, design, specs, and `tasks.md` from that commit; do not
  copy them manually into the worktree.
- Record the source branch as `parent_branch` in the Task metadata before
  creating the Task branch/worktree. Completion merges only into that recorded
  parent branch; never infer the target from the current checkout.
- Do not use raw `git worktree` when AIW is available.
- Do not silently implement in a workspace that does not match the Task.
- After all checklist items are complete, merge the Task branch into its
  recorded parent branch and verify the merge. Then remove the Task worktree
  and delete the Task branch. Only after cleanup succeeds, run sync and archive.
  Preserve Task resources on any failure or merge conflict before cleanup.

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

Archive through `aiw archive <task-id> --backend auto` only after the completed
Task branch has been merged into its recorded parent branch, the merge has been
verified, and the worktree and feature branch have been cleaned up. Sync and
archive are the final lifecycle steps. Stop and preserve resources if merge or
cleanup fails.

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
- commit the completed implementation changes on the AIW Task branch after the
  static review. When every checklist item is complete, run the completion
  protocol: merge into the recorded parent branch, verify the merge, remove the
  worktree and delete the Task branch, then synchronize and archive. Do not
  clean up partial or failed work.
