# .github/copilot-instructions.md

Read `AGENTS.md` first.

## Prompt Set

- load the four files in `prompts/core/`
- prefer the nearest local instructions
- add at most one repo-type, one domain, and one task-mode prompt
- do not load the whole prompt library

## Defaults

- use static analysis and minimal edits
- use at most three targeted discovery batches before editing unless blocked
- keep command output narrow
- do not run tests, builds, formatters, linters, type checks, vet,
  verification scripts, network calls, permission probes, escalation,
  `codex-auto-review`, or sub-agents without resource-budget authorization
- use at most one static/read-only post-edit command by default
- do not repeat equivalent commands

## Report

State the change, static evidence, exact commands, skipped runtime checks,
remaining risks, and optional checks requiring authorization.
