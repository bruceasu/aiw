## Why

`aiw git show` can inspect repository-level history, but it does not provide a focused way to inspect how one file or one section of a file evolved. Users currently need to remember and assemble several Git commands for file logs, rename tracking, blame, historical content, and line history.

## What Changes

- Add a file-history view with selectable concise, patch, statistics, graph, and full-evolution output.
- Follow file renames by default, while allowing callers to disable rename tracking.
- Add a blame view for line-level attribution.
- Add a file-at-revision view backed by `git show <revision>:<path>`.
- Add a line/function history view backed by `git log -L`.
- Document all new views in command help and cover Git argument construction with tests.

## Capabilities

### New Capabilities

- `git-file-history`: Read-only inspection of a file's commit history, line attribution, historical content, and line/function evolution.

### Modified Capabilities

None.

## Impact

- Affected command: `plugins/aiw-git/git-show.py`.
- Affected tests: focused tests for `git-show.py` command dispatch and Git argument construction.
- No dependency, persistence, public library API, or write-operation changes.
- Existing `aiw git show` views remain backward compatible.
