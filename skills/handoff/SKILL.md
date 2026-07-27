---
name: handoff
description: Compact the current conversation into a handoff document for another agent to pick up.
argument-hint: "What will the next session be used for?"
disable-model-invocation: true
---

Write a handoff document summarising the current conversation so a fresh agent
can continue the work. In an AIW Session, prefer the Session artifact location
and reference the active OpenSpec change. Use the temporary directory of the
user's OS only when no AIW Session artifact store is available.

Include a "suggested skills" section in the document, which suggests skills that the agent should invoke.

Do not duplicate content already captured in other artifacts (OpenSpec specs,
proposal, design, tasks, plans, ADRs, external Issues, commits, or diffs).
Reference them by path or URL instead.

Redact any sensitive information, such as API keys, passwords, or personally identifiable information.

If the user passed arguments, treat them as a description of what the next session will focus on and tailor the doc accordingly.
