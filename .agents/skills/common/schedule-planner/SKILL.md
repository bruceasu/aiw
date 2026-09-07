---
name: schedule-planner
description: A generic planning skill to create, prioritize, and schedule tasks or projects. Works across any domain, not limited to finance.
---
# Schedule Planner Skill

## Overview
Provides commands to:
- `init <plan-name>`: create a new plan file under `docs/plans/PLAN_NAME.md`.
- `add <plan-name> "task description" [--due <date>] [--owner <name>]`: append a task entry.
- `list <plan-name>`: show all tasks with status.
- `prioritize <plan-name> <task-id> <high|medium|low>`: set priority.
- `complete <plan-name> <task-id>`: mark task as done.

## Usage Example
```bash
aiw-flow schedule init sprint-2024-Q1
aiw-flow schedule add sprint-2024-Q1 "Implement login flow" --due 2024-01-15 --owner alice
aiw-flow schedule list sprint-2024-Q1
```
