---
name: implement
description: "Implement one selected work item from an AIW Task and its matching OpenSpec change."
disable-model-invocation: true
---

# Implement

Follow `skills/reviewed-skill-contract.md`; lifecycle ownership remains in
`skills/work-management.md`.

Read `skills/work-management.md` and
`docs/agents/work-management.md` when present.

## Resolve Work

1. Resolve one AIW Task.
2. Resolve its matching OpenSpec change and selected `tasks.md` item.
3. Read only the applicable `task.toml`, proposal, design, capability specs,
   checklist, and notes.
4. Stop and ask for the Task ID when several Tasks match.

## Prepare The Workspace

Use the Task's declared workspace. Ordinary sequential work stays in the
primary workspace; verify its project root and branch before modifying files.
Use `aiw wt` only when isolation is explicitly authorized and beneficial.

Do not use raw `git worktree` when AIW is available. Do not silently implement
in another workspace. If the active agent cannot safely continue in the
resolved worktree, report its path and stop.

## Implement

- Apply the smallest complete change for the selected checklist item.
- Use at most two bounded sub-agents under the shared contract.
- Use `aiw patch` as the preferred path for AI-generated patches when
  available. Use direct editing only when the patch command cannot represent
  the change or is unavailable, and report the fallback.
- Writing or updating tests is allowed, but do not run them automatically.
- Do not invoke `/tdd` or `/code-review` automatically.
- Do not run type checks, formatters, linters, vet, builds, or verification
  scripts without explicit execution instructions from the user.

## Complete Development

Update the selected checklist item, TODO, Verification, and remaining `%%`
risks or questions. Synchronize the coarse AIW Task status without overwriting
OpenSpec-owned content.

Perform one static review of the changed paths and mark Task completion
separately from Git delivery. Do not automatically commit, merge, push, remove
a worktree, delete a branch, synchronize, or archive. If the user explicitly
requests Git delivery, validate the recorded `parent_branch` and preserve all
resources on failure.

After development is complete, ask once whether the user wants one focused test
command run. Show the exact command, scope, and expected duration. Default to no
test when the user declines or does not respond. Ask again before broader tests
or builds.
