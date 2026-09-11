## Why

AIW is in its adoption phase. Users often spend time searching help and README
files before they can choose the right AIW workflow. `aiw ask` provides a
read-only, LLM-backed guidance surface and preserves private question-and-answer
memory for later human review.

## What Changes

- Add the core command `aiw ask "<prompt>"`, plus `--chat` and `--resume`.
- Reuse the existing core LLM implementation and require a versioned JSON answer
  contract before rendering friendly terminal output.
- Store personal sessions under `$HOME/.aiw/ask/`, outside Git.
- Add system-prompt overrides and a conservative filesystem-access policy.
- Keep automatic analysis, complex editing, tool calls, and provider refactoring
  out of scope.

## Capabilities

### New Capabilities

- `ask-guidance`: provide safe, structured AIW usage guidance and private local
  session memory.

## Impact

- Top-level CLI command registration and help.
- Existing `internal/ai` structured-output execution path.
- Local HOME-directory persistence and interactive terminal UI.
- No Task lifecycle, project-file, Git, or automatic-plugin mutation.
