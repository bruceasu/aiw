# FD-010 Supplemental Independent Review

- FD: `FD-010`, Revision 8.
- Source event: `FD-010-000008-review-requested`, claimed by session `agent-reviewer-fd010-r3`.
- Reviewed base: `24c848a74fde49a6a6e92a80cc651352ebaa219f` (`HEAD`) plus the current working tree. Scope was the current FD, its help and completion changes, and their dispatch paths.
- Worker report: `docs/features/archive/FD-010/reports/FD-010-implementation.md`.

## Findings

1. **The `wf` compatibility plugin is not discoverable by help.** `plugins/aiw-wf/` contains executable `aiw-wf` and `aiw-wf.exe`, which the main plugin dispatcher can match. However, `internal/commands/help/super-help.go` discovers and routes plugins through `pluginScriptPath` and `pluginNameFromFile`, both limited to `aiw-<name>.py`. Therefore `listPlugins` does not list `wf`, and `Dispatch([]string{"wf"})` does not call `showPluginHelp`; it falls through to documentation search. This fails FD Work Item 1.3's requirement to keep `wf` discoverable and route its help to the plugin. Update help discovery to recognize the supported executable plugin forms and route `wf` through that result.

## Reviewed evidence

- `cmd/aiw/main.go` sends unhandled command names through `dispatchPlugin`; `plugins/aiw-fd.py` implements the FD CLI.
- `internal/commands/help/super-help.go` advertises FD operations and `wf`; its help-specific plugin lookup only tests `.py` paths, while `internal/plugin/discover.go` recognizes extensionless and `.exe` binaries.
- `plugins/aiw-wf/` contains `aiw-wf` and `aiw-wf.exe`; no `.py` implementation is present. Static call-path analysis shows why normal command dispatch can find the plugin while help discovery cannot.
- The FD records the Revision 6 compile script's partial outcome: `./cmd/aiw` compiled, then the script failed because `cmd/aiw-wf` no longer exists. The Worker report also preserves an earlier focused `go build` success. No compile was run by this Reviewer.

## Review evidence and skipped checks

- Commands run: `aiw fd claim FD-010 FD-010-000008-review-requested --session agent-reviewer-fd010-r3`; targeted `Get-Content`; scoped `git diff`; `Get-ChildItem` for plugin files; `rg` searches; and a UTF-8 read of `openspec/specs/command-help-consistency/spec.md`.
- Not run: tests, `aiw help wf`, `aiw completion`, builds/compiles, linters, formatters, or verification scripts.
- Residual risk: plugin help and shell completion behavior were statically traced but not executed.

## Outcome

`CHANGES_REQUESTED`. Work Item 1.3's `wf` help routing is unsupported by the current help resolver.
