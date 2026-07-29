# CODEX.md

Follow `AGENTS.md` first.

## Normal Python Prompt Set
- root `AGENTS.md` and `CODEX.md`
- `../prompts/core/*.md`
- `../prompts/repo-types/monorepo.md` only when the task spans more than one project
- one Python domain prompt
- one task-mode prompt

## Routing
- if service markers exist, also load `../prompts/domains/python-service.md`
- if CLI markers exist, also load `../prompts/domains/python-cli.md`
- if the request is a bugfix, review, debugging, test, docs, feature, or risky change, also load the matching file in `../prompts/task-modes/`

## Execution
For non-trivial work, provide:
- Understanding
- Relevant Modules
- Assumptions
- Plan
- Validation
- Risks

Inspect tests, schemas, and config before broad exploration.
Follow the shared resource budget. Use static review by default; do not run
formatters, linters, type checks, tests, builds, or verification scripts
automatically. When authorized, run one path-focused command and ask before
widening scope.
