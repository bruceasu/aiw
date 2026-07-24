## 1. Command Surface

- [x] 1.1 Add metadata and help for `file`, `blame`, `file-at`, and `lines` views.
- [x] 1.2 Implement validated Git argument construction and dispatch for all file-history views.

## 2. Tests

- [x] 2.1 Add unit tests for every file history mode, rename opt-out, blame, historical content, and `log -L`.
- [x] 2.2 Add unit tests for missing arguments, conflicting modes, help output, and existing-view compatibility.

## 3. Verification

- [x] 3.1 Run Python compilation and the focused `aiw-git` unit-test suite.
- [x] 3.2 Inspect the final diff and confirm no unrelated changes or dependency updates.

## Notes

%% Git itself remains responsible for validating revisions, paths, and `-L` selector syntax.
%% Verification: Python compilation passed; 11 focused unit tests passed; OpenSpec strict validation passed; a real rename-aware file log smoke test passed.
