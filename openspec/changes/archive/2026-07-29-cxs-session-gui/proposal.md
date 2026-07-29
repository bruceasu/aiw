## Why

`aiw cxs` can inspect Codex sessions and manage aliases, but selecting the right
session still requires several CLI commands and does not preserve the session's
original working directory when resuming interactive Codex. A workspace-scoped
visual selector and a lightweight Codex skill will make session discovery,
preview, alias maintenance, and continuation faster and less error-prone.

## What Changes

- Add a desktop GUI entry point for browsing Codex sessions, previewing readable
  conversation content, and creating, renaming, or removing aliases.
- Extract and cache each session's original working directory.
- Show only sessions belonging to the current workspace by default in the GUI
  and `resume-ext`, with an explicit option to show all workspaces.
- Resume the selected session with interactive `codex resume <session-id>` from
  its original working directory.
- Add a small `resume-ext` skill that presents a compact, GUI-like session
  selection workflow and produces a safe handoff to interactive resume.
- Preserve the existing `aiw cxs` commands and stored alias format.

## Capabilities

### New Capabilities

- `codex-session-navigation`: Workspace-scoped Codex session discovery,
  preview, alias editing, GUI selection, interactive resume, and the
  `resume-ext` skill workflow.

### Modified Capabilities

None.

## Impact

- Affected implementation: `plugins/aiw-cxs.py` and its session metadata/cache,
  command parser, process-launch behavior, and UI layer.
- Affected documentation: `docs/usage/aiw-cxs.md` and top-level command help.
- New reusable workflow: a repository-managed `resume-ext` skill.
- Runtime integration: local Codex session JSONL files and the installed
  `codex resume` command.
- Dependencies: prefer the Python standard library GUI toolkit; adding an
  external dependency requires a separate decision and approval.
