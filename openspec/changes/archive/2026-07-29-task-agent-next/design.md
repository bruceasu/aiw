# Design: Task Agent Next

## Context

AIW already persists Sessions, handoff artifacts, Codex Thread IDs, and task
worktrees. The current pieces are exposed as separate commands, while `aiw
task ai` invokes the low-level cxs adapter directly. A user therefore has to
manually create a handoff, find the task worktree, and start a fresh Thread.

## Goals

- Provide one explicit sequential handoff command for an existing Task.
- Keep the same Task, worktree, and aiw-flow Session while starting a fresh
  Codex Thread.
- Make the parent/child Thread relationship and consumed handoff auditable.
- Reuse aiw-flow execution and Session locking instead of duplicating cxs
  invocation logic.
- Preserve existing one-shot, loop, and same-Thread continuation behavior.

## Non-Goals

- Parallel agents writing to one worktree.
- Automatic multi-worktree branching or task decomposition.
- Replacing cxs as the low-level Codex session adapter.
- Moving the handoff source of truth to the OS temporary directory.

## Decisions

### Sequential fresh-Thread workflow

`aiw task agent next TASK_ID` resolves the Task, its worktree, and its bound
Session. It creates or refreshes the persistent Session handoff, then starts a
new Codex Thread in that same worktree and Session. The initial prompt includes
the Task context, Session Memory, and a reference to the handoff artifact.

Same-Thread continuation remains `aiw-flow continue`; a fresh Thread is an
explicit handoff operation.

### Durable lineage

The Session/task metadata records task ID, parent Thread ID, child Thread ID,
handoff path and content hash, and transition timestamps. The handoff is
created before the child Thread starts. If resolution or handoff creation
fails, no child Thread is launched.

### Capability boundaries

Task orchestration calls the aiw-flow execution seam. cxs remains responsible
for Codex CLI process details. `aiw-wt` remains responsible for worktree
lifecycle, but its task metadata reader accepts canonical `task.toml` first
and the legacy `tasks.toml` as a compatibility fallback.

### Concurrency guard

The transition uses the existing Session lock plus a task/worktree execution
lease. A git worktree lock alone is insufficient because it protects
worktree lifecycle, not simultaneous file writes.

## Risks and Mitigations

- [Risk] A stale or missing Task↔Session binding could launch work in the
  wrong context. → Mitigation: validate all identifiers and worktree paths
  before starting a child Thread; fail closed.
- [Risk] Duplicate handoff requests could race. → Mitigation: hold the
  Session lock and use an idempotent lineage transition record.
- [Risk] Existing `tasks.toml` users could break. → Mitigation: read
  `task.toml` first and retain legacy fallback with a diagnostic.
- [Risk] Large handoffs could bloat the next prompt. → Mitigation: pass
  references plus bounded excerpts, reusing Session handoff limits.

## Migration Plan

1. Add metadata and execution seams behind the new command; do not alter
   existing `run`, `continue`, or `loop` semantics.
2. Add the `aiw task agent next` command and diagnostics.
3. Update `aiw-wt` metadata compatibility and integration tests.
4. Document the sequential workflow and keep parallel/fork behavior explicitly
   unsupported until a separate change.

## Open Questions

- Whether the public command should eventually be exposed as a plugin alias in
  addition to the built-in `aiw task` group.
- Whether a future release should allow a child Thread to change Session phase
  explicitly; the first slice inherits the current phase.
