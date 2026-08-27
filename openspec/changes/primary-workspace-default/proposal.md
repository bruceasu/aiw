## Why

AIW currently treats every managed Task as if it needs a dedicated feature
branch and linked worktree. That adds fixed Git and workspace overhead to
ordinary sequential work, even when isolation provides no benefit.

## What Changes

- Make the primary Git checkout and its current branch the default workspace
  binding for a new Task.
- Add explicit workspace and delivery lifecycle metadata so destructive
  operations do not infer safety from path strings.
- Keep linked worktrees opt-in for parallel, conflicting, long-running, or
  disposable work, and centralize their creation in `aiw wt add`.
- Make fresh-agent handoff reuse the Task workspace unless isolation is
  explicitly requested.
- Separate Task completion from Git delivery and remove implicit Git delivery
  from ordinary completion.
- Add safe legacy inference, discard, repair, archive, and cleanup behavior.
- **BREAKING** Change the default metadata and workspace behavior of newly
  created Tasks; existing Task metadata remains compatible.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `workflow`: Change Task workspace defaults and define workspace/delivery
  state transitions, compatibility inference, and archive safety.
- `capabilities`: Change engineering Skill workspace selection and completion
  behavior from mandatory isolation to primary-workspace-first execution.
- `ai-support`: Change fresh-agent handoff to reuse the bound workspace by
  default and isolate only on explicit request.

## Impact

Affected areas include Task metadata serialization and registry projection,
native and delegated Task creation, the `aiw-wt` plugin, Task archive and agent
handoff commands, CLI help, engineering Skills, stable OpenSpec requirements,
and user-facing workflow documentation. No dependency, network, deployment, or
repository-wide migration is introduced.
