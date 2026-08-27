# Goal

Make the primary checkout the default Task workspace while preserving explicit,
safe linked-worktree isolation.

# Scope

Included:

- Task workspace and delivery metadata.
- Native and delegated Task creation defaults.
- Worktree add, remove, discard, repair, and archive safety.
- Fresh-agent handoff workspace behavior.
- Skills, stable specs, help, README, ADR, and glossary updates.
- Focused unit tests for the changed behavior.

Out of scope:

- Multiple worktrees per Task.
- Automatic Git commit, checkout, merge, push, fetch, or stash.
- Repository-external lifecycle storage or bulk metadata migration.

# Constraints

- Preserve existing Task metadata compatibility.
- Do not overwrite the user's unrelated working-tree changes.
- Do not run tests, builds, formatters, linters, or network operations without
  separate authorization.

# TODO

- [x] 1. Add workspace kind and delivery metadata with safe legacy inference.
- [x] 2. Default native and delegated Task creation to the primary workspace.
- [x] 3. Make worktree transitions explicit, lossless, offline, and safe.
- [x] 4. Separate archive and completion from Git delivery and add cancellation.
- [x] 5. Make fresh-agent handoff reuse the bound workspace by default.
- [x] 6. Update Skills, AGENTS, stable specs, help, README, ADR, and glossary.
- [x] 7. Add focused test coverage without executing it.

# Verification

- [x] Static diff confirms all metadata writers preserve lifecycle and list fields.
- [x] Static call-path review confirms primary, isolated, unassigned, unknown,
  merged, and discarded branches fail safely.
- [x] Stable specs and Skill instructions agree with the program behavior.
- [x] Tests/builds intentionally not run unless separately authorized.

# Notes

%% Existing user deletions under `.agents/skills/` are out of scope and must not
be restored or overwritten.

%% Runtime behavior remains unverified until a focused test command is explicitly
authorized.
