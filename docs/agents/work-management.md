# Work Management

AIW is authoritative for Task lifecycle, branch, worktree, Session, and handoff
state. OpenSpec is authoritative for proposal, design, capability requirements,
and the detailed checklist in `tasks.md`.

Local work normally uses:

- `openspec/changes/<task-id>/` for active Task artifacts;
- `openspec/specs/` for stable capability requirements;
- the primary Git checkout for ordinary sequential implementation;
- `.wt/<task-id>/` for automated execution and explicitly isolated Task
  worktrees.

Use AIW lifecycle commands for Task creation, status, completion, and archive.
Automated execution creates or reuses the Task worktree by default. Use `aiw wt`
for explicit isolation or repair. AIW's automatic backend may delegate
supported artifact operations to an installed OpenSpec CLI.

Workflow Core runtime state is not projected into Git-tracked `task.toml` or
OpenSpec `tasks.md`: it owns Attempts, leases, retry counts, evidence, and
diagnostics in its runtime store. A retry-exhausted Work Item stays blocked
until an operator reopens it with a reason. A force-close records `CANCELLED`
and an explicit merged or discarded delivery outcome; it is not a substitute
for validation or checklist completion. For a worktree pull conflict,
`aiw wt pull <task-id> --resolve=agent` is opt-in and yields a reviewable
proposal only for eligible text files; protected paths and unsafe file types
remain manual, and proposal application never commits or completes the merge.

Do not create new canonical work under `.scratch`. GitHub and GitLab remain
optional external projections.

For complete Skill behavior, read `skills/work-management.md`.
