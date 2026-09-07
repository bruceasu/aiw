## Problem Statement

`aiw task agent next <task-id>` 当前只能处理已经存在且完整绑定了 Session 和
worktree 的 Task。用户直觉上把它理解为“消费 handoff，启动下一个 agent”：
当 Task 不存在时，应自动建立新的 Task、Session、branch、worktree 和 Thread；
当 Task 已存在时，应继续使用已有资源并只启动新的 Thread。当前行为无法表达
这两种情况，也没有定义 handoff 来源、资源冲突、部分初始化和失败重试规则。

## Solution

扩展 `task agent next`，使其根据 Task 是否存在自动选择“创建新 Task”或“继续
已有 Task”的路径。新 Task 必须从可追溯的 handoff 创建；已有 Task 复用已有
Session、branch 和 worktree。命令在启动新 Thread 前验证资源、名称、状态和
handoff，并记录完整的父子任务与 Thread 谱系。

## User Stories

1. As an engineer, I want `task agent next <task-id>` to create a missing Task from a handoff, so that I can hand work to a fresh agent without manually creating lifecycle resources.
2. As an engineer, I want an existing Task to reuse its Session, branch, and worktree, so that the next Thread continues in the established engineering context.
3. As an engineer, I want missing resources for a partially initialized Task to be created automatically, so that recoverable lifecycle gaps do not block handoff.
4. As an engineer, I want resource conflicts to stop the command without overwriting existing resources, so that another Task cannot be damaged accidentally.
5. As an engineer, I want handoff sources to have a deterministic precedence, so that the next agent consumes the context I intended.
6. As an engineer, I want a new Task to copy and record its handoff source, so that the original handoff remains intact and the new context is auditable.
7. As an engineer, I want invalid Task IDs to produce a confirmable valid candidate name, so that naming can be repaired without silently changing my intent.
8. As an engineer, I want running Sessions to reject automatic handoff, so that two agents cannot write the same Task worktree concurrently.
9. As an engineer, I want failed creation to clean up only resources created by that attempt, so that retrying is safe and existing work is preserved.
10. As an engineer, I want the new Task to record its parent Task, Session, Thread, handoff path, and hash, so that the handoff lineage can be traced later.
11. As an engineer, I want successful handoff consumption to be recorded separately from handoff creation, so that pending and consumed transitions are distinguishable.

## Implementation Decisions

- `task agent next <task-id>` first resolves whether the Task exists, then selects the existing-Task or new-Task path.
- For an existing Task, the command preserves the Task, Session, branch, and worktree, creates a fresh Thread, and does not create a replacement Session by default.
- For a missing Task, the command creates the Task, Session, branch, worktree, and fresh Thread from a handoff.
- A partially initialized existing Task is retained and missing lifecycle resources are created. Existing resource name/path conflicts fail closed.
- Handoff precedence is: explicit `--handoff`, existing Task `artifacts/handoff.md`, current Session handoff. If no source exists, the command refuses to create a new Task.
- A new Task copies the selected handoff into its own `artifacts/handoff.md`; the source remains unchanged.
- A new Task records the source Task, Session, Thread, source path, content hash, and creation time when those values are available.
- A Task ID that violates naming rules produces a deterministic candidate mapping for user confirmation. Confirmation is required; the default is refusal.
- A running Session blocks the transition unless the user explicitly requests takeover.
- Creation is transactional: preflight occurs before mutation, failures clean up only resources created by the current attempt, and original resources remain unchanged.
- The old Session is marked `handed-off` or `completed`; the new Task enters `running` and owns the new Session and Thread.
- The new Task records `handoff_status = pending` until the new Thread successfully starts and consumes the copied handoff, then records consumption time, consumer Thread, and hash.

## Testing Decisions

- Test through the highest available CLI seam and assert external behavior, resource ownership, diagnostics, and persisted lineage rather than helper implementation details.
- Cover new Task creation from each handoff source, missing-handoff refusal, existing Task reuse, partial-resource repair, resource conflict refusal, invalid-name confirmation and default refusal.
- Cover running-Session refusal, explicit takeover, transactional cleanup, retry after failure, handoff copy/hash recording, and pending-to-consumed transition.
- Cover preservation of existing same-Thread continuation and unrelated Task workflows.
- Reuse the existing task-agent handoff, Session, worktree, and handoff test fixtures where possible.

## Out of Scope

- Parallel agents writing to the same Task worktree.
- Automatic deletion of old branches or worktrees.
- Implicit external publication to GitHub or GitLab.
- Changing ordinary `run`, `continue`, or `loop` behavior.
- Arbitrary Session reuse without an explicit user option.

## Further Notes

%% The exact command-line spelling for explicit takeover and handoff source options should follow the existing AIW CLI option conventions during implementation.
%% The exact lifecycle state name for the old Session should reuse the canonical AIW state vocabulary if `handed-off` is not already supported.
