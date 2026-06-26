# .github/copilot-instructions.md

Read `AGENTS.md` first.
Use this file as the GitHub Copilot entry point.

## Load A Small Prompt Set
- prefer the nearest local `AGENTS.md` or `CODEX.md`
- load all files in `prompts/core/`
- add at most one repo-type prompt, one domain prompt, and one task-mode prompt unless the task truly spans more
- use this precedence when rules overlap:
  `project-local > domain > language > repo-type > root`

## Routing
- mixed-language repo work:
  `prompts/repo-types/monorepo.md`
- prompt-system work:
  `prompts/domains/prompt-authoring.md`
- bugfix, feature, review, debugging, test, docs, or risky work:
  the matching file in `prompts/task-modes/`
- Python work:
  `python/AGENTS.md`
- Java work:
  `java/AGENTS.md`
- Go work:
  `go/AGENTS.md`

## Defaults
- plan first for non-trivial tasks
- keep changes small and easy to review
- inspect local code, tests, and config before broad exploration
- explain risky changes before making them
- report exact validation commands and limits

## Validation
Prefer the nearest `./scripts/verify.sh`.
If none exists, use the repository-standard commands for the detected language.
