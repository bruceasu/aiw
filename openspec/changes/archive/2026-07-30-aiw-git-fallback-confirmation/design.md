# Design

## Dispatch seam

The highest useful seam is the existing `aiw-git` dispatcher after it has
discovered the bundled command map and failed to find the requested
subcommand. This keeps local command precedence and avoids changing each
wrapper. The dispatcher owns the fallback decision and the native subprocess
boundary.

## Decision flow

1. Discover the bundled command map.
2. Preserve the existing handling for overview help and known local commands.
3. If the requested subcommand is absent, resolve whether `git` is available.
4. If no interactive terminal is available, report that fallback requires
   confirmation and return without starting Git.
5. Render the exact candidate argv and ask for confirmation on stderr. Empty
   input, EOF, and every answer other than `y` or `yes` refuse the fallback.
6. After affirmative confirmation, execute `git` with the original argument
   vector and return its exit status.

The confirmation must happen before any native Git process is started. The
implementation must not run a probe command first, because even a probe may
trigger repository hooks, aliases, or an installed Git extension.

## Help behavior

`aiw git help <known-command>` keeps rendering the local command help. For an
unknown name, the dispatcher presents the native `git help <name>` invocation
as a fallback candidate. The prompt must make clear that this is native Git
delegation, not an `aiw-git` help page.

## Safety and portability

- Use an argv list and `subprocess.run`; never use `shell=True` or concatenate a
  command string for execution.
- Use terminal detection before reading confirmation input to prevent CI and
  piped invocations from blocking.
- Render arguments defensively in diagnostics because an argument can contain
  whitespace, quotes, or control characters.
- Do not interpret `--force`, `--yes`, or any other forwarded Git argument as
  fallback approval.
- If the native executable is missing, return a clear diagnostic without
  prompting for an action that cannot run.

## Compatibility

Existing local commands, their help handling, their confirmation semantics,
and their subprocess argv remain unchanged. Only the current unknown-command
error path gains an interactive fallback decision.
