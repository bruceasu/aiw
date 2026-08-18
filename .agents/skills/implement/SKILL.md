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

1. Resolve one AIW Task when AIW is available; otherwise resolve one matching
   standalone OpenSpec change or explicitly selected work item.
2. Resolve its matching OpenSpec change and selected `tasks.md` item when
   those artifacts exist.
3. Read only the applicable `task.toml`, proposal, design, capability specs,
   checklist, and notes.
4. Stop and ask for the Task ID when several Tasks match.

## Prepare The Workspace

In managed mode, create or resolve the Task worktree through `aiw wt` and
verify that its branch and path match AIW Task metadata. If AIW is unavailable
but Git is available and isolation is useful, use an explicit native
`git worktree` and branch. Otherwise work in the user's current workspace or
selected branch after checking for conflicting changes.

Do not use raw `git worktree` when AIW is available. Do not silently implement
in another workspace. A native Git worktree is a valid fallback only when AIW
is unavailable; report its path and branch. If isolation is unavailable,
continue in place when safe and report the limitation.

## Implement

- Before exploring implementation paths, apply the shared Repository Index
  Context procedure. Refresh `.ai/` when a generator is available; otherwise
  continue with existing index data or live search (`rg`, then `fd`, then
  `grep`) and report the fallback.
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
risks or questions. Synchronize the coarse AIW Task status when AIW is
available; in standalone modes, update OpenSpec-owned records without
fabricating AIW state.

Perform one static review of the changed paths, then commit the completed
implementation changes on the managed Task branch when AIW is active. In
standalone modes, commit on the selected Git branch when appropriate; do not
claim an AIW Task branch.

If every implementation checklist item in `tasks.md` is complete and managed
mode is active, run the completion protocol automatically:

1. Merge the Task branch into its validated `parent_branch`.
2. Verify the merge succeeded.
3. Remove the Task worktree and delete the Task branch.
4. Synchronize AIW and OpenSpec state.
5. Archive the completed change through the automatic backend.

Do not start this protocol when any checklist item is incomplete. In standalone
modes, do not attempt these AIW-only operations. If sync,
archive, merge, or verification fails, stop and preserve the Task branch and
worktree for recovery. Do not clean up a partial task or a conflicted merge.
The merge target must be the validated `parent_branch` recorded in the Task
metadata.

After development is complete, ask once whether the user wants one focused test
command run. Show the exact command, scope, and expected duration. Default to no
test when the user declines or does not respond. Ask again before broader tests
or builds.
