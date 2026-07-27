---
name: implement
description: "Implement one selected work item from an OpenSpec change."
disable-model-invocation: true
---

Implement one selected work item from the resolved OpenSpec change. Read
`skills/work-management.md` and resolve the change before modifying files.

Read the applicable `task.toml`, `proposal.md`, `design.md`, capability specs,
`tasks.md`, and `notes.md` before starting. If no unique change can be resolved,
stop and ask for its identifier.

When the change declares an AIW branch or worktree, verify that the current
workspace matches it before changing implementation files. Do not silently
switch workspaces or infer a target from `.scratch`.

Use /tdd where possible, at pre-agreed seams.

Run typechecking regularly, single test files regularly, and the full test suite once at the end.

Once done, use /code-review to review the work.

Update the selected checklist item, TODO, Verification, and remaining `%%`
risks or questions. Commit according to the repository's task workflow after
verification; do not create or publish an external Issue automatically.
