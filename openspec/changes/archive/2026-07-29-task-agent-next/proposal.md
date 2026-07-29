## Why

AIW can persist Sessions, create handoff artifacts, resume an existing Codex
Thread, and manage task worktrees, but a user must currently connect those
steps manually. That makes sequential agent handoff fragile and leaves the
task, worktree, Session, and next Thread without one auditable transition.

## What Changes

- Add a sequential `task agent next` workflow for one existing Task.
- Resolve the Task's worktree and bound aiw-flow Session.
- Refresh a persistent handoff, then start a fresh Codex Thread in the same
  Session and worktree.
- Include the handoff, Session Memory, and Task context in the new Thread's
  initial prompt.
- Record parent/child Thread lineage and the handoff consumed by the next run.
- Keep same-Thread continuation, parallel agents, and multi-worktree forks out
  of this first tracer bullet.

## Capabilities

### New Capabilities

- `task-agent-handoff`: Sequentially hand a Task from one Codex Thread to a new
  Thread while preserving Session, worktree, handoff, and lineage state.

### Modified Capabilities

- `session`: Add explicit fresh-Thread handoff lineage and Task context binding.

## Impact

- Affected Go task orchestration and command/help surfaces.
- Affected aiw-flow Session metadata, handoff, and new-Thread execution path.
- Affected `aiw-wt` integration and Task metadata compatibility.
- No new third-party dependencies or external service requirements.
