# AGENTS.md
This repository uses OpenSpec-lite workflow.
Before coding, ALWAYS read:
1. relevant files under:
   - openspec/changes/<task>/
   - openspec/specs/
2. especially:
   - task.md
   - design.md (if exists)
   - spec.md
Rules:
- Work on ONE task at a time.
- Keep changes scoped.
- Do not refactor unrelated modules.
- Preserve backward compatibility unless explicitly required.
- Prefer small reviewable diffs.
- Update TODO and Verification before finishing.
- Add risks/questions using `%%` notes instead of guessing.
- If stable requirements changed, update openspec/specs/.
- If architecture decisions changed, update design.md.
Git worktree convention:
- one task = one branch = one worktree
- branch: feature/<task-id>
- worktree: .wt/<task-id>
