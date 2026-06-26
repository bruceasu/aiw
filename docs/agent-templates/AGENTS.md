Always respond in Chinese.
Write prompts in Easy English when asked to draft prompts.

# AGENTS.md

## Purpose
This is the root entry file for a mixed-language repository used with coding agents.
Keep this file short.
Use it for routing and for rules that must apply everywhere.

## Design Rules For Instruction Files
- Write constraints, not tutorials.
- Do not repeat content that already lives in local files or project docs.
- Load the smallest useful prompt set.
- Prefer local rules over broad global rules.

## Load Order
1. Apply this file first.
2. Prefer the nearest `AGENTS.md` or `CODEX.md` in the current subtree.
3. Load `prompts/core/*.md`.
4. Add at most one repo-type prompt, one domain prompt, and one task-mode prompt unless the task truly spans more than one context.
5. When rules overlap, use this precedence:
   `project-local > domain > language > repo-type > root`

## Task Size
- Local change: one module, one package, one command, or one endpoint.
- Subtree change: several related files in one project.
- Repository-level change: shared contracts, shared libraries, or multiple deployable projects.

For subtree and repository-level changes, plan before editing.
For repository-level changes, split work into phases and validate after each phase.

## Context Discipline
- Start with the nearest code, tests, config, and docs.
- Expand search only when current evidence is not enough.
- Avoid full-repo scans or loading every prompt file by default.

## Universal Rules
- Plan first for any non-trivial task.
- Change only what the task needs.
- Keep diffs small, local, and reviewable.
- Preserve current contracts unless the task explicitly requires a change.
- Explain risky actions before doing them.
- Validate meaningful changes before claiming success.
- Update nearby docs when behavior or workflow changes.

## Stop And Ask
Pause and ask when:
- local instruction files conflict
- the expected external behavior is unclear
- two designs have very different blast radius
- a migration, rollout, or compatibility policy is missing

## High-Risk Areas
Call out risk before changing:
- dependencies or build tooling
- auth, permission, billing, or security logic
- schema, migration, or persistence contracts
- public APIs, events, or CLI contracts
- CI/CD, infra, or deployment config
- concurrency, async, retry, timeout, or shutdown behavior

## Required Working Structure
For non-trivial tasks, provide:
- Understanding
- Relevant Areas
- Assumptions
- Plan
- Validation
- Risks

## Prompt Routing
- For repo-wide habits, load `prompts/core/*.md`
- For mixed-language repository concerns, load `prompts/repo-types/monorepo.md`
- For Python work, also use `python/AGENTS.md` or `python/CODEX.md`
- For Java work, also use `java/AGENTS.md` or `java/CODEX.md`
- For Go work, also use `go/AGENTS.md` or `go/CODEX.md`
- For prompt-system work, also use `prompts/domains/prompt-authoring.md`
- For bugfix, feature, review, debugging, test, docs, or risky-change requests, also use the matching file in `prompts/task-modes/`

## Validation Preference
Prefer the nearest `./scripts/verify.sh`.
If the current subtree has its own verification script, prefer that local script over the repo root script.

## Harness Feedback
If the task reveals unclear ownership, missing checks, or weak instructions, report:
`Harness improvements suggested`
