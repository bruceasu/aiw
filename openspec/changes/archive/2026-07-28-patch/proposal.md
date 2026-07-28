# Proposal: Cross-platform Git Patch Application

## Problem Statement

On Windows, patch application frequently fails before Git can validate the patch because PowerShell and process pipes may use different encodings. UTF-8, UTF-8 with BOM, UTF-16, and legacy Windows encodings can be interpreted inconsistently.

## Solution

Add an aiw patch command that normalizes patch input to UTF-8, validates it with Git, and converts supported AI patch syntax to standard unified diff, then delegates patch application to git apply. The command preserves safe defaults and presents actionable diagnostics. AI code-editing workflows and AI support surfaces SHOULD route generated patches through this command rather than writing patch changes through a separate direct-edit path.

## User Stories

1. As a Windows developer, I want to apply a patch without manually changing terminal encodings, so that code changes work consistently.
2. As a developer, I want check-only mode, so that I can detect conflicts before modifying files.
3. As a developer, I want useful file and Git diagnostics, so that I can fix an invalid patch quickly.
4. As a developer, I want Git to remain responsible for patch semantics, so that behavior follows familiar Git rules.
5. As a developer, I want index and three-way behavior to be explicit, so that applying a patch does not unexpectedly change repository state.
6. As an AI coding agent, I want one patch application path, so that the AI support layer has a consistent file-edit route for every code change.

## Implementation Decisions

- Provide check, apply, and reverse operations.
- Accept a patch file and standard input.
- Normalize supported Windows input encodings before invoking Git.
- Use git apply --check before normal application.
- Keep --3way and --index opt-in.
- Return Git's non-zero status and actionable stderr.
- Make aiw patch the default application backend for AI-generated patches.
- Return structured success and failure information to the AI caller, including changed paths when available.

## Testing Decisions

- Test command construction at the subprocess boundary.
- Test UTF-8, UTF-8 BOM, and UTF-16 inputs.
- Test check-only, apply, reverse, invalid input, and safe defaults.
- Add a real Git integration test in a temporary repository.

## Out of Scope

- Reimplementing unified diff parsing.
- Automatically staging changes.
- Automatically enabling three-way merge.
- External publication.

## Further Notes

%% Legacy Windows code-page detection is ambiguous. Prefer an explicit encoding option when automatic detection is inconclusive.
%% Verify BOM and newline preservation empirically before documenting it as a strict guarantee.
