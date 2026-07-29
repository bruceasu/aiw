# Design: Task Agent Next Auto-Creation

## Context

The existing sequential handoff command is bound to a pre-existing Task,
Session, and worktree. The desired user model is handoff-oriented: a handoff
document is the input to the next agent, and the command should decide whether
to create lifecycle resources or continue an existing Task based on Task
identity.

## Decision: Two Resolution Paths

The command resolves the supplied Task ID before any mutation:

```text
Task exists
  -> validate and repair missing Task resources
  -> reuse Task / Session / branch / worktree
  -> start a fresh Thread

Task does not exist
  -> resolve and validate handoff
  -> confirm normalized identity when needed
  -> create Task / Session / branch / worktree
  -> copy handoff
  -> start a fresh Thread
```

The Task ID is the primary discriminator. An existing Session is not by itself
permission to create a new Task; it is used only when the resolved Task already
binds to it or when the user explicitly supplies it as the handoff source.

## Decision: Handoff Resolution

The command resolves handoff sources in this order:

1. Explicit `--handoff` path.
2. Existing Task `artifacts/handoff.md`.
3. Current Session handoff.
4. Refuse when creating a new Task if no source exists.

For a new Task, the selected handoff is copied into the new Task's artifact
directory. The source is never moved, deleted, or overwritten. The copy records
source path, content hash, and timestamp. The new Thread reads the copy so its
context is independent of the source Session.

## Decision: Identity and Resource Naming

The supplied ID is used unchanged when valid. When invalid, AIW computes a
deterministic normalized candidate and presents the complete mapping for Task,
Session, branch, worktree, and Thread context. The command proceeds only after
explicit confirmation; refusal is the default.

Existing names and paths are never silently replaced. A conflict stops the
operation before mutation unless the existing resource is the one already bound
to the resolved Task.

## Decision: Partial Resources and Atomicity

An existing Task with missing Session, branch, or worktree metadata is repaired
by creating only the missing resources. A conflict or invalid existing binding
fails closed.

For a new Task, resource creation is a compensating transaction. The command
tracks resources created by this invocation and removes only those resources if
later creation or Thread startup fails. It must not alter the source handoff,
existing Task, or existing Session during cleanup.

## Decision: State and Lineage

The source Session transitions to the canonical handed-off/completed state. The
new Task and Session enter running state once the new Thread starts. The Task
lineage records parent Task, parent Session, parent Thread, child Session, child
Thread, handoff path, hash, and timestamps.

The copied handoff starts as `pending`. After successful child Thread startup and
handoff consumption, the new Task records `consumed_at`, `consumer_thread`, and
the consumed hash. Startup failure leaves the handoff pending and records the
failure without claiming successful consumption.

## Decision: Concurrency

The command uses the existing Session lock and per-Task/worktree execution
lease. A running Session refuses automatic transition. An explicit takeover is
an exceptional path and must be visible in diagnostics and lineage.

## Failure and Retry

All preflight checks happen before creating resources: Task identity, handoff,
Session state, paths, branch names, worktree availability, and existing leases.
Failures return actionable diagnostics. A retry after cleanup sees the original
handoff and can repeat the operation deterministically.

## Compatibility

Existing `run`, `continue`, `loop`, and same-Thread behavior remain unchanged.
The existing command remains valid for existing Tasks while gaining the new
creation path for missing Task IDs.
