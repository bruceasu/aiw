# Monorepo

## Scope
- Work inside the smallest relevant subtree.
- Do not assume one language rule applies to the whole repository.

## Routing
- Prefer the nearest local `AGENTS.md`, `CODEX.md`, and `scripts/verify.sh`.
- Use project-local validation before repo-root validation when possible.

## Cross-Project Changes
- Map the affected projects and shared contracts first.
- Plan the work in phases.
- Validate after each project or phase, not only at the end.

## Shared Rules
- Keep shared prompt rules in `prompts/`.
- Keep project-specific rules near the project, not at the root.
