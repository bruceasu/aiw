# FD-010 Independent Review, pass 2

- FD: `FD-010`, Revision 4.
- Source event: `FD-010-000004-implementation-ready`, claimed by session `agent-reviewer-fd010-r2`.
- Reviewed base: `24c848a74fde49a6a6e92a80cc651352ebaa219f` (`HEAD`) plus the current working tree. Review scope was FD-010 and its initialization call paths; concurrent FD-008/FD-009 changes were excluded.
- Worker report: `docs/features/archive/FD-010/reports/FD-010-implementation.md`.

## Findings

1. **Automatic template detection still recommends a removed command.** `internal/commands/initcmd/prompts.go:253` tells users to run `aiw prompts go --merge`, but `aiw prompts` is no longer a built-in command and there is no `aiw-prompts` plugin. This guidance cannot recover from the detection error and conflicts with the restored `aiw init` interface. Update it to a valid invocation such as `aiw init --prompts --template go --merge`.

## Reviewed evidence

- `cmd/aiw/main.go` dispatches `init` to `initcmd.Dispatch` before plugin fallback.
- `syncPromptLibrary` resolves the source as `docs/agent-templates/prompts/` (`templatesRoot` plus `syncedPromptLibrarySourceDir = "prompts"`) and retains `.agents/prompts/` as its destination.
- Generated instruction templates and the checked-in root instruction files reference `.agents/prompts/`; no old source-relative prompt paths were found in those templates.
- `parseInitOptions` requires `--prompts` for `--merge`, `--force`, and `--template`. With both flags, instruction synchronization clears `Force` while retaining `Merge`, and prompt-library synchronization applies `Force`, satisfying the requested merge-and-refresh behavior by static trace.
- The `init` call path contains no OpenSpec, Task runtime, or worktree writes; the old setup plugin is not invoked.
- The Worker report records `GOPROXY=off` and `go build -o NUL ./cmd/aiw`, exit code 0, after the source-path fix. This is accepted as Worker-reported compile evidence; Reviewer did not rerun it.

## Review evidence and skipped checks

- Commands run: `aiw fd claim FD-010 FD-010-000004-implementation-ready --session agent-reviewer-fd010-r2`; `aiw fd --help`; `aiw fd emit --help`; `git rev-parse HEAD`; targeted `Get-Content`, `git status --short`, `git diff`, and `rg` inspections.
- Not run: tests, `aiw init`, runtime/build checks by Reviewer, linters, formatters, or verification scripts. Repository instructions do not authorize these for this review.
- Residual risk: installation and file-write behavior remain unexercised at runtime.

## Outcome

`CHANGES_REQUESTED`. Replace the stale command example at `internal/commands/initcmd/prompts.go:253`, then request a fresh independent review. The previous prompt-library source-path and Worker compile-provenance findings are resolved.
