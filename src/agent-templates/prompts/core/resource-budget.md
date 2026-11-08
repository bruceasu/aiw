# Resource Budget

## Default Budget

For an ordinary implementation request:

- tests and final-artifact builds: `0`
- formatter, linter, type-checker, vet, and verification runs: `0`
- compile-only checks: allowed after implementation; first use a
  `scripts/compile*` or root `compile*` script, otherwise the narrowest
  language-level compiler command. The command must not retain a final
  distributable artifact.
- unrelated external network calls and standalone executable downloads: `0` by default
- permission probes and privilege escalation requests: `0`
- `codex-auto-review` and repeated review passes: `0`
- sub-agents: allowed for bounded delegated work; each follows this same
  runtime and validation budget
- post-edit command validation: at most `1` static/read-only command

Implementation does not imply authorization for tests, final-artifact builds,
or other runtime validation. Follow the repository's instructions for any
project-specific test workflow.

## Discovery Budget

- Use up to four targeted discovery batches for ordinary tasks. Continue one
  focused batch at a time only to resolve a concrete uncertainty; stop when
  enough evidence exists to make the scoped change.
- Batch related reads and searches.
- Read relevant symbols or excerpts instead of entire large files.
- Do not dump full logs, generated output, vendor trees, or lockfiles.
- Stop searching when enough evidence exists to make the scoped change.

## Retry Budget

- Do not repeat an unchanged failed command. If a compile-only check fails due
  to a source error, fix the source and rerun the same compile-only command
  once. If that retry fails, stop and report the remaining error.
- Allow one cheap corrected retry only for a command spelling, shell entrypoint,
  or path mistake.
- After a permission failure, stop that path. Do not try alternate shells,
  escalation, or broader commands unless the user authorizes the required
  action.

## Network And Permissions

- Read-only searches of public web pages and documentation are allowed when
  relevant to the task. Keep queries limited to the question and non-sensitive
  context; do not send local file contents or secrets unless the user explicitly
  asks to share them.
- When needed for the task, use the project's package manager to resolve or
  download project library dependencies, including normal dependency install
  and build hooks. Examples include Maven, pip, uv, and npm. Keep installation
  scoped to the project and respect its declared versions and lockfiles.
- Installing or downloading a standalone executable, CLI, compiler, browser,
  or other tool remains gated on explicit user authorization, even if a package
  manager or install hook would fetch it.
- Network access by the program or test being run, including requests to
  external APIs/services, transmitting local or user data, and remote side
  effects, remains off by default and follows Runtime Authorization.
- Do not probe permissions with a command expected to fail.
- Ask for permission only when it is essential to the requested outcome.
- Explain the exact action, scope, expected cost, and risk before asking.

## Reporting

State what was changed, what static evidence was reviewed, what commands ran,
what was intentionally skipped, and what optional focused check the user may
authorize.
