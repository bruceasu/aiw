---
name: handoff
description: Compact the current conversation into a handoff document for another agent to pick up.
argument-hint: "What will the next session be used for?"
disable-model-invocation: true
---

Follow `skills/reviewed-skill-contract.md`. Read `skills/work-management.md`
when present. Resolve the AIW Task, its
worktree, Session, and matching OpenSpec change before writing the handoff.

Write a handoff document summarising the current conversation so a fresh agent
can continue the work. Prefer the AIW Session artifact location. Use the
current workspace's `.ai/tmp` directory only when no AIW Session artifact store
is available; use a unique filename and tell the user its absolute path.

Include a "suggested skills" section in the document, which suggests skills that the agent should invoke.

For managed work, include the Task ID, Work Item, Attempt, relevant Evidence,
and unresolved Gates when known. A handoff records facts and recommendations;
it does not complete an Attempt, release a lease, or advance Task state.

Do not duplicate content already captured in other artifacts (OpenSpec specs,
proposal, design, tasks, plans, ADRs, external Issues, commits, or diffs).
Reference them by path or URL instead.

Redact any sensitive information, such as API keys, passwords, or personally identifiable information.

If the user passed arguments, treat them as a description of what the next session will focus on and tailor the doc accordingly.

Do not start a new Thread automatically. When the user explicitly asks to hand
off execution, use `aiw turn <task-id>` so AIW preserves the Task,
worktree, Session, lease, and lineage.
