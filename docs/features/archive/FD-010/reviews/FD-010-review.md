# FD-010 Independent Review

- FD: `FD-010`, Revision 1, Pending Verification.
- Source event: none was present in `.ai/fd/FD-010/`; this was a direct review request.
- Reviewed base: `24c848a` (`HEAD`) plus the current uncommitted working-tree diff. The worktree contains concurrent FD-008/009 changes; this review is limited to FD-010's files and call paths.
- Worker implementation report: not present under `docs/features/reports/`.

## Findings

1. **Prompt library sync never reads the checked-in prompt library.** In `internal/commands/initcmd/prompts.go`, `syncPromptLibrary` sets `sourceRoot` to `templatesRoot/.agents/prompts`, but the actual source directory is `docs/agent-templates/prompts/`. Since the code returns success when that mismatched source path does not exist, `aiw init --prompts` silently writes no reusable prompt files under `.agents/prompts/`. This fails the FD acceptance condition requiring the prompt library to be installed there and leaves generated instruction references pointing to missing files. Use the actual template source directory while preserving `.agents/prompts` as the destination.

2. **Worker verification provenance is missing.** The FD records a compile-only command as passed, but the required Worker implementation report is absent, so its command context and result cannot be independently traced. Record the report or otherwise provide the requested provenance before closing the FD.

## Review evidence

- Read FD-010, `skills/work-management.md`, the FD review Skill, the relevant CLI dispatch/help/completion changes, `internal/commands/initcmd/`, the modified prompt templates, and the existing FD-009 implementation report for its scope boundary.
- Static trace confirmed `cmd/aiw/main.go` dispatches `init` to the built-in handler and prompt-specific options require `--prompts`.
- Static trace found that the prompt library source path does not match the checked-in directory. No runtime command was run.
- Commands run: `Get-Content` on the listed instructions and source files; `rg` for FD references, prompt paths and source directories; `git status --short`, `git diff --stat`, `git log -5 --oneline --decorate`, and targeted `git diff`/`git show` reads.
- Not run: tests, runtime checks, build/compile, formatters, linters, and verification scripts. The FD's reported compile result was not rerun.

## Outcome

`CHANGES_REQUESTED`. Repair the prompt library source path and supply Worker verification provenance, then request another independent review. Residual risk: behavior of the corrected filesystem writes has not been exercised at runtime.
