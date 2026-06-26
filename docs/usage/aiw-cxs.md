# aiw cxs

Codex session manager and resume/exec helper.

## Usage

aiw cxs [global-options] <subcommand> [args...]

## Description

`aiw cxs` helps you inspect local Codex session logs, bind business aliases to sessions,
and run `codex exec` with optional session targeting.

Default paths:

- sessions dir: `~/.codex/sessions`
- workspace index: `.ai/sessions/index.json`

## Subcommands

- `list`: list recent sessions
- `show <ref>`: show a readable session preview
- `tail <ref>`: show the latest events
- `bind <alias> [ref]`: bind alias to a session
- `aliases`: list alias mappings
- `path <ref>`: print session jsonl path
- `resume <ref> [message]`: run `codex exec resume`
- `exec [message]`: run `codex exec`, optionally target a session

## Session Reference

`<ref>` can be:

- alias from `.ai/sessions/index.json`
- session id
- session id prefix

For explicit UUID-like values, `aiw cxs` can pass them through to `codex exec resume`
even when local session logs are missing.

## Examples

- `aiw cxs list -n 30`
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
