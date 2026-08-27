# Work Management

AIW is authoritative for Task lifecycle, branch, worktree, Session, and handoff
state. OpenSpec is authoritative for proposal, design, capability requirements,
and the detailed checklist in `tasks.md`.

Local work normally uses:

- `openspec/changes/<task-id>/` for active Task artifacts;
- `openspec/specs/` for stable capability requirements;
- the primary Git checkout for ordinary sequential implementation;
- `.wt/<task-id>/` only for explicitly isolated Task worktrees.

Use AIW lifecycle commands for Task creation, status, completion, and archive.
Use `aiw wt` only when isolation is needed. AIW's automatic backend may delegate supported
artifact operations to an installed OpenSpec CLI.

Do not create new canonical work under `.scratch`. GitHub and GitLab remain
optional external projections.

For complete Skill behavior, read `skills/work-management.md`.
