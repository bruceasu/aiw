# aiw cxs

Codex session manager and resume/exec helper.

## Usage

aiw cxs [global-options] <subcommand> [args...]

## Description

`aiw cxs` helps you inspect local Codex session logs, bind business aliases to
sessions, browse workspace sessions in a desktop GUI, and continue Codex work.

Default paths:

- sessions dir: `~/.codex/sessions`
- workspace index: `.ai/sessions/index.json`

## Subcommands

- `list`: list recent sessions
- `gui`: browse current-workspace sessions in a desktop window
- `show <ref>`: show a readable session preview
- `tail <ref>`: show the latest events
- `bind <alias> [ref]`: bind alias to a session
- `aliases`: list alias mappings
- `path <ref>`: print session jsonl path
- `resume <ref> [message]`: run `codex exec resume`
- `exec [message]`: run `codex exec`, optionally target a session

## Desktop GUI

Run:

```text
aiw cxs gui
```

The GUI shows only sessions from the current workspace by default. Enable
`Show all workspaces` to include global sessions. Select a session to preview
user and assistant messages, edit its alias, or resume interactive Codex.

Interactive resume uses:

```text
codex resume <session-id>
```

Codex starts in the session's original working directory. If the directory is
unknown, missing, or no interactive terminal is available, the GUI shows a
copyable command instead of launching Codex.

## Session Reference

`<ref>` can be:

- alias from `.ai/sessions/index.json`
- session id
- session id prefix

For explicit UUID-like values, `aiw cxs` can pass them through to `codex exec resume`
even when local session logs are missing.

## Examples

- `aiw cxs list -n 30`
- `aiw cxs list --current-workspace --json -n 20`
- `aiw cxs gui`
- `aiw cxs bind payment-retry`
- `aiw cxs aliases`
- `aiw cxs show payment-retry`
- `aiw cxs tail payment-retry -e 20`
- `aiw cxs resume payment-retry "continue with tests"`
- `aiw cxs exec "summarize current diff"`
- `aiw cxs exec --session payment-retry "continue implementation"`
- `aiw cxs exec --last "continue latest session"`
- `aiw cxs exec --session 123e4567-e89b-12d3-a456-426614174000 --dry-run`

## Notes

- `--dry-run` prints the exact `codex` command without executing it.
- `--session` and `--last` are mutually exclusive for `exec`.
- `resume` always uses `codex exec resume`.
- Existing `list` behavior remains global by default; use
  `--current-workspace` for workspace-only output.
- Invoke `$resume-ext` to get a compact numbered selection workflow inside
  Codex. The skill prepares a command and never starts nested Codex.
