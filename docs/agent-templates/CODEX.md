Always respond in Chinese.
Write prompts in Easy English when asked to draft prompts.

# CODEX.md

Follow `AGENTS.md` first.

## Role
Use this file to decide which prompt set to load.
Keep the active prompt set small.

## Normal Matched Set
- this file
- root `AGENTS.md`
- the nearest local `AGENTS.md` or `CODEX.md`
- `prompts/core/*.md`
- zero or one repo-type prompt
- zero or one domain prompt
- zero or one task-mode prompt

Add another prompt only if the task truly spans two contexts.
Do not load the whole prompt library by default.

## Domain Detection
- Python service markers: `fastapi`, `django`, `flask`, `pydantic`, `app/`, `tests/`
  - also load `prompts/domains/python-service.md`
- Python CLI markers: `typer`, `click`, `argparse`, `__main__.py`
  - also load `prompts/domains/python-cli.md`
- Java Spring markers: `spring-boot`, `@RestController`, `application.yml`, `src/main/java`
  - also load `prompts/domains/java-spring.md`
- Go service markers: `cmd/server`, `internal/`, HTTP or gRPC handlers, service config packages
  - also load `prompts/domains/go-service.md`
- Go CLI markers: `cobra`, `urfave/cli`, command trees under `cmd/`, single-binary tools
  - also load `prompts/domains/go-cli.md`
- Prompt-authoring markers: edits in `AGENTS.md`, `CODEX.md`, `prompts/`, `.github/`, or `tasks/`
  - also load `prompts/domains/prompt-authoring.md`

## Task Mode Detection
- bug, defect, regression, broken behavior
  - also load `prompts/task-modes/bugfix.md`
- new behavior, endpoint, command, feature, or extension
  - also load `prompts/task-modes/feature-dev.md`
- review, audit, inspect, or code review
  - also load `prompts/task-modes/code-review.md`
- debug, investigate, flaky, trace, reproduce, logging
  - also load `prompts/task-modes/debugging.md`
- test, coverage, regression test, contract test
  - also load `prompts/task-modes/test-work.md`
- docs, readme, guide, runbook, migration note
  - also load `prompts/task-modes/docs-work.md`
- dependency, schema, auth, deployment, data, async, concurrency, or transaction changes
  - also load `prompts/task-modes/risky-change.md`

## Repository-Level Trigger
If the task touches shared packages, shared schemas, generated clients, or multiple deployable projects:
- produce a phase plan first
- change one phase at a time
- validate after each phase

## Execution Defaults
- Start in planning mode for non-trivial work.
- Inspect the nearest tests, configs, and external contracts first.
- Use the smallest change that solves the task.
- Separate facts, assumptions, and inferences.
- Report exact validation commands and limits.

## Validation
Prefer the nearest `./scripts/verify.sh`.
If none exists, use the repository-standard commands for the detected language.

## Definition of Done
A task is not done until:
- the requested change is implemented
- relevant tests or checks are updated when needed
- validation has been run, or limits are stated clearly
- prompt and doc changes stay aligned

## AIW Capability Discovery

Before using an unfamiliar or mutating AIW tool, inspect docs/agents/aiw-tools.md and runtime discovery:

- aiw help --json
- aiw plugin list --json

Prefer aiw file for text file operations and aiw patch for AI-generated code changes. Runtime metadata is authoritative; do not depend on absolute plugin paths.