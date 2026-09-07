## 1. Resolve command intent and identity

- [x] 1.1 Add existing-vs-new Task resolution before lifecycle mutation.
- [x] 1.2 Define and implement valid Task ID normalization and confirmation flow.
- [x] 1.3 Add deterministic diagnostics showing the resolved Task, Session, branch, worktree, and Thread plan.

## 2. Resolve and preserve handoff

- [x] 2.1 Implement handoff precedence: explicit option, Task artifact, current Session artifact.
- [x] 2.2 Refuse new-Task creation when no handoff exists.
- [x] 2.3 Copy the selected handoff into a new Task without modifying the source.
- [x] 2.4 Persist source path, hash, timestamps, pending status, and successful consumption metadata.

## 3. Create or repair lifecycle resources

- [x] 3.1 Create a new Task, Session, branch, worktree, and Thread from a confirmed handoff.
- [x] 3.2 Reuse all bound resources for an existing Task and start only a fresh Thread.
- [x] 3.3 Repair missing resources for a partial Task without replacing existing resources.
- [x] 3.4 Reject unrelated branch, worktree, Session, and lease conflicts.

## 4. State, lineage, and atomicity

- [x] 4.1 Add parent/child Task, Session, and Thread lineage fields.
- [x] 4.2 Enforce running-Session refusal and explicit takeover diagnostics.
- [x] 4.3 Implement compensating cleanup for resources created by a failed attempt.
- [x] 4.4 Transition source and child lifecycle states according to the specification.

## 5. Verification and documentation

- [x] 5.1 Add focused command-layer coverage for Task creation/reuse, parsing, and conflict-path helpers; full CLI execution remains unrun.
- [x] 5.2 Add unit coverage for handoff lineage hash/status persistence; full lifecycle execution remains unrun.
- [x] 5.3 Add unit coverage for invalid-name normalization, takeover options, and lineage diagnostics; full CLI execution remains unrun.
- [x] 5.4 Update CLI help and workflow documentation to describe automatic Task resolution.
- [x] 5.5 Update TODO and Verification records after implementation.

## Verification

- [x] Static review confirms lifecycle state fields and focused unit coverage are connected to `task agent next`.
- [x] Runtime tests passed: `go test ./internal/commands/task`.
