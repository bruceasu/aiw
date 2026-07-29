# CODEX.md

Follow `AGENTS.md` first.

## Normal Java Prompt Set
- root `AGENTS.md` and `CODEX.md`
- `../prompts/core/*.md`
- `../prompts/repo-types/monorepo.md` only when the task spans more than one project
- `../prompts/domains/java-spring.md` when Spring markers exist
- one task-mode prompt

## Routing
- if the request is a bugfix, review, debugging, test, docs, feature, or risky change, also load the matching file in `../prompts/task-modes/`

## Execution
For non-trivial work, provide:
- Understanding
- Relevant Packages / Classes
- Assumptions
- Plan
- Validation
- Risks

Inspect contracts, tests, and configuration before broad exploration.
Follow the shared resource budget. Use static review by default; do not run
Maven, Gradle, tests, builds, or verification scripts automatically. When
authorized, run one module-focused command and ask before widening scope.
