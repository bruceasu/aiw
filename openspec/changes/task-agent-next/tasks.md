## 1. Task and Session metadata

- [x] 1.1 Define the minimal Task↔Session binding and fresh-Thread lineage fields with backward-compatible decoding.
- [x] 1.2 Add atomic persistence and migration-safe diagnostics for parent/child Thread IDs, handoff path/hash, and transition timestamps.
- [x] 1.3 Add a per-Task/worktree execution lease integrated with the existing Session lock.

## 2. aiw-flow handoff execution

- [x] 2.1 Extract or expose an aiw-flow execution seam that can start a fresh Thread while reusing Session phase, memory, workspace, and timeout behavior.
- [x] 2.2 Build the child Thread prompt from Task context, Session Memory, and a bounded handoff reference without copying large artifacts.
- [x] 2.3 Persist successful and failed transition states, release the lease on every exit path, and preserve existing `run`, `continue`, and `loop` behavior.

## 3. Task command integration

- [x] 3.1 Add `aiw task agent next TASK_ID` routing and help text, resolving the Task worktree and Session before invoking Codex.
- [x] 3.2 Fail closed for missing bindings, invalid worktrees, completed/running Sessions, lease conflicts, and Codex startup failures.
- [x] 3.3 Add status/diagnostic output showing the consumed handoff and parent-to-child Thread lineage.

## 4. aiw-wt compatibility

- [x] 4.1 Make `aiw-wt` read canonical `task.toml` and retain `tasks.toml` as a legacy fallback with an actionable diagnostic.
- [x] 4.2 Verify Task worktree resolution and Session binding across Windows and Linux path conventions.

## 5. Verification and documentation

- [x] 5.1 Add unit tests for metadata migration, handoff prompt composition, lease conflicts, and failure cleanup.
- [x] 5.2 Add seam/integration tests through the highest-level `aiw` CLI covering successful handoff, lineage inspection, and unchanged existing workflows.
- [x] 5.3 Run formatting, static checks, unit/integration tests, and build verification; update CLI and workflow documentation.

## 6. Interactive fork

- [x] 6.1 Add `/fork` parsing and Loop handling that creates a handoff, starts a fresh Thread with the handoff prompt, and exits.
- [x] 6.2 Add Loop tests and document the single-handoff-context behavior.
