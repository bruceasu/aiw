# FD-011: Merge branches in an isolated worktree

**Status:** Completed  
**Revision:** 1  
**Priority:** Medium

## Problem

Users need to merge branches into another branch without changing their current checkout, files, or index.

## Options and decision

Switching in the current workspace requires restoring local state. Use an independent temporary worktree, as requested. Reject a target already checked out anywhere; never bypass Git's branch protection.

## Solution

- `aiw git merge-to target_branch`: merge the current local branch.
- `aiw git merge-to target_branch source_branch`: merge the selected branch.
- `aiw git merge-to --new new_branch source_branch1 source_branch2 ...`: create from the first source and merge the others sequentially. One source is sufficient.

Targets are local branch names. Sources are local or already available remote-tracking branches; no fetch or push. Validate all arguments and sources before mutation. Reject invalid input, missing branches, an existing --new target, detached HEAD without an explicit source, and checked-out existing targets.

Create a unique temporary worktree outside the invoking workspace. Resolve source commits before checkout. Use completed, noninteractive merges, preventing autostash and squash configuration from leaving unfinished state. Remove the worktree without force only after success. On merge failure, interruption, or cleanup failure, return failure and print the absolute recovery path. Never reset or abort automatically. Earlier successful merges and any newly created branch remain on later failure.

## Scope

New Python Git wrapper, README, stable CLI spec, and this FD/index. No dependencies, Go routing changes, unrelated refactoring, implementation-session Git writes, or workflow dispatch.

## Work items

- [x] 1.1 Add metadata, argument validation, and branch preflight before mutation.
- [x] 1.2 Implement isolated checkout, ordered merges, cleanup, and recovery output.
- [x] 1.3 Document all forms and recovery semantics.
- [x] 1.4 Record static evidence and one focused compile-only check, including unavailable checks.

## Acceptance

- The caller's checkout, files, and index stay unchanged, including local uncommitted changes.
- Checked-out existing targets are rejected, including the caller's branch.
- New targets start at the first source, then merge remaining sources in order.
- Success removes the worktree without force; failure retains recovery paths.
- Invalid input causes no branch/worktree mutation.
- Plugin overview and detailed help discover the new wrapper.

## Verification

- Follow-up confirmation: the existing wrapper and documentation satisfy the agreed command forms. Re-read the full wrapper and checked the scoped Git diff; the focused in-memory Python syntax compile passed. No real merges, tests, network calls, or Git writes were performed.
- Static review traced plugin discovery, all argument forms, branch preflight, ordered merges, conservative cleanup, interruption handling, and recovery output. Compared documentation additions with the original file contents.
- Python syntax compilation passed twice: once before, and once after correcting recovery arguments and PowerShell command rendering found during static review. No bytecode or final executable was retained.
- Command: `python -c "from pathlib import Path; p = Path('plugins/aiw-git/git-merge-to.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec'); print('Python syntax compile passed; no bytecode or executable retained.')"`.
- The initial PowerShell discovery command could not start because the terminal helper reported a setup refresh error. File access tools completed discovery and edits; no escalation was requested.
- Tests, real merge scenarios, formatters, linters, Go compilation, final builds, and network operations were not run. Only the Python plugin changed; the Go compile script is outside that scope.

%% Independent review and runtime conflict/recovery validation remain pending. Keep this FD In Progress; do not mark it Complete.

## Sources

- User request in this session, 2026-10-02.
- [Work management](../../skills/work-management.md).
- [Stable CLI spec](../../openspec/specs/cli-and-plugins/spec.md).
- [Git dispatcher](../../plugins/aiw-git/aiw-git.py).
