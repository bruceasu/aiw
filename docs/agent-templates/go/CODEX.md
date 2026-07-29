# CODEX.md

Follow `AGENTS.md` first.

## Normal Go Prompt Set
- root `AGENTS.md` and `CODEX.md`
- `../prompts/core/*.md`
- `../prompts/repo-types/monorepo.md` only when the task spans more than one project
- one Go domain prompt
- one task-mode prompt

## Routing
- if service markers exist, also load `../prompts/domains/go-service.md`
- if CLI markers exist, also load `../prompts/domains/go-cli.md`
- if the request is a bugfix, review, debugging, test, docs, feature, or risky change, also load the matching file in `../prompts/task-modes/`

## Execution
For non-trivial work, provide:
- Understanding
- Relevant Packages
- Assumptions
- Plan
- Validation
- Risks

Inspect tests, configs, and public contracts before broad exploration.
Follow the shared resource budget. Use static review by default; do not run
tests, vet, builds, or verification scripts automatically. When authorized,
run one package-focused command and ask before widening scope.
