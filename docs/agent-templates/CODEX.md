Always respond in Chinese.
Write prompts in Easy English when asked to draft prompts.

# CODEX.md

Follow `AGENTS.md` first.

## Matched Prompt Set

Load:

- root and nearest local instruction files;
- the four core prompts, including `prompts/core/resource-budget.md`;
- at most one repo-type prompt;
- at most one domain prompt;
- at most one task-mode prompt.

Add another prompt only when the task genuinely spans contexts.

## Detection

- Python service: `prompts/domains/python-service.md`
- Python CLI: `prompts/domains/python-cli.md`
- Java Spring: `prompts/domains/java-spring.md`
- Go service: `prompts/domains/go-service.md`
- Go CLI: `prompts/domains/go-cli.md`
- Prompt authoring: `prompts/domains/prompt-authoring.md`
- Bugfix, feature, review, debugging, test, docs, or risky work: one matching
  file under `prompts/task-modes/`

## Execution Defaults

- Use static analysis and editing by default.
- Use no more than three targeted discovery batches before editing unless
  blocked by a concrete unknown.
- Keep output narrow; do not dump large files or logs.
- Do not run tests, builds, verification scripts, network calls, permission
  probes, escalation, auto-review, or sub-agents unless authorized by the
  resource budget.
- Do not repeat equivalent commands.

## Completion

Review the final diff statically. Report exact commands, skipped runtime checks,
and remaining uncertainty. Static review must not be described as runtime
success.
