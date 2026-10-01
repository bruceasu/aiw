# FD-010 Supplemental Independent Review, recovery pass 2

- FD: `FD-010`, Revision 11.
- Source event: `FD-010-000011-review-requested`, claimed by session `agent-reviewer-fd010-recovery-final`.
- Reviewed base: `24c848a74fde49a6a6e92a80cc651352ebaa219f` (`HEAD`) plus the current working tree. Scope was the `wf` help discovery and routing repair.
- Worker report: `docs/features/archive/FD-010/reports/FD-010-implementation.md`.

## Findings

1. **Windows help can select the non-Windows `wf` binary.** `pluginExists` now delegates to `plug.DiscoverPlugin`, and `showPluginHelp` also uses that resolver, so the routing path is correct. However, `internal/plugin/discover.go` collects matching files from the plugin directory and selects the lowest `scoreExt`. It gives extensionless files priority 2 and `.exe` priority 3. When both the extensionless Linux binary and the Windows executable are present, Windows resolves `aiw-wf` first and attempts to launch that file, so `aiw help wf` cannot reach the intended Windows plugin. Earlier scoped inspection listed both `plugins/aiw-wf/aiw-wf` and `plugins/aiw-wf/aiw-wf.exe`; the directory is absent from the current working tree, so this review did not re-inspect the binary signatures. Make candidate priority platform-aware while retaining script priority and other-platform ordering.

## Reviewed evidence

- `internal/commands/help/super-help.go`: `Dispatch` checks `pluginExists` before built-ins; `pluginExists` and `showPluginHelp` both use `plug.DiscoverPlugin`. `listPlugins` now skips nested directories and recognizes `.py`, extensionless, and `.exe` names; it omits `wf` from the top-level plugin list as the FD requires.
- `internal/plugin/discover.go`: plugin-directory candidates are collected without platform filtering. `scoreExt` assigns `""` priority 2 and `.exe` priority 3, so the extensionless candidate wins when both are present, including on Windows.
- The FD records a successful narrow compile-only check by the Worker. This Reviewer did not rerun it.

## Review evidence and skipped checks

- Commands run: `aiw fd claim FD-010 FD-010-000011-review-requested --session agent-reviewer-fd010-recovery-final`; targeted `Get-Content`; targeted `git diff`; and scoped plugin-directory/file listing. The attempted read of the earlier binary paths found that they are absent from the current tree.
- Not run: tests, `aiw help wf`, builds/compiles, linters, formatters, or verification scripts.
- Residual risk: the Windows launch failure is statically derived from resolver scoring and the previously observed dual-binary layout; no runtime help command was run.

## Outcome

`CHANGES_REQUESTED`. The explicit `wf` help route still fails on Windows installations containing both the extensionless Linux binary and `.exe` candidate because the shared resolver ranks the wrong file first.
