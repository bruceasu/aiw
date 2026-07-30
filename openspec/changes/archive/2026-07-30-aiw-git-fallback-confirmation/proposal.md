# Problem Statement

The `aiw-git` plugin currently rejects every subcommand that is not defined by
the plugin. Passing an ordinary Git command through would improve coverage,
but doing so silently could execute destructive Git operations or installed
`git-<command>` extensions outside the plugin's documented safety controls.

# Solution

When a command is not defined by `aiw-git`, the dispatcher will offer an
explicit, interactive fallback to the native `git <command> ...` invocation.
The fallback will be refused by default, including whenever an interactive
terminal is unavailable. Only an affirmative response will allow delegation.

# User Stories

1. As an `aiw-git` user, I want known plugin commands to keep their existing
   behavior, so that documented workflows remain stable.
2. As an `aiw-git` user, I want an unknown plugin command to show me the exact
   native Git command that would run, so that I can review its effect before
   execution.
3. As an `aiw-git` user, I want fallback to be refused by default, so that a
   typo or unexpected command cannot silently mutate repository state.
4. As a script or CI operator, I want non-interactive invocations to refuse
   fallback without blocking for input, so that automation remains safe and
   deterministic.
5. As an `aiw-git` user, I want native Git output and its exit status preserved
   after I approve fallback, so that Git remains the source of truth for native
   command behavior.
6. As an `aiw-git` user, I want `aiw-git` help and diagnostics to explain when
   fallback was refused or selected, so that the command's behavior is
   understandable and auditable.

# Implementation Decisions

- The existing local-command lookup remains authoritative. A command that is
  known locally is never redirected because its arguments are invalid.
- An unknown command is considered for fallback only after local lookup fails.
- Fallback confirmation is a separate confirmation flow; a Git argument such
  as `--force` must never implicitly approve it.
- The proposed command is displayed before confirmation, with arguments kept as
  distinct process arguments when executed and safely rendered in the prompt.
- Native execution uses a subprocess argument list without shell invocation.
- Refusal, EOF, a non-interactive stdin, or an unavailable terminal prevents
  execution and returns a non-zero cancellation result.
- Approved fallback delegates to `git` and preserves native stdout, stderr, and
  exit status. Git aliases and installed `git-<command>` extensions are part of
  the explicitly approved native delegation surface.
- Local help for a known command remains local. Help for an unknown command is
  treated as a native fallback candidate only after confirmation.
- Top-level help documents that unknown commands require explicit native Git
  fallback confirmation.

# Testing Decisions

- Tests will assert observable dispatch behavior rather than helper internals.
- The dispatcher tests will cover known-command precedence, affirmative
  fallback, default refusal, EOF/refusal, non-interactive refusal, exact argv
  forwarding, and native exit-code propagation.
- Help tests will cover known local help and unknown native-help fallback.
- Tests will mock the confirmation input and subprocess boundary so no real
  repository mutation or external Git extension is invoked.
- Existing dispatcher tests in `plugins/aiw-git/test_aiw_git_dispatch.py` are
  the prior art and should remain the primary test seam.

# Out of Scope

- Adding wrappers for every native Git command.
- Reimplementing Git's command discovery, alias resolution, or safety policy.
- Making fallback opt-out or automatically approved by `--force`.
- Adding a persistent configuration switch that changes the default refusal.
- Changing confirmation behavior of already-defined `aiw-git` commands.
- Running tests, builds, or verification as part of specification authoring.

# Further Notes

The implementation uses exit code `2` for refused fallback. A future
non-interactive caller that needs native delegation should receive a separately
named explicit opt-in rather than reinterpreting Git flags.
