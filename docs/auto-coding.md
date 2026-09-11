# AIW `workflow supervise` 自动编码流程

本文档说明当前 AIW 自动编码流程的实际工作方式，包括
`workflow supervise` 的生命周期、Agent/LLM 调用、Skills 使用、工作区隔离、
协同约束、暂停条件和恢复方式。

## 1. 适用范围

`workflow supervise` 用于在一个已经建立的 AIW Task 上，持续执行多个受控的
Work Item。它不是无限制的自动编程循环，也不是后台调度器。

AIW 将以下职责分开：

| 组件 | 负责内容 |
|---|---|
| AIW Task | Task ID、分支、worktree、Session 和 handoff 生命周期 |
| OpenSpec | proposal、design、spec、`tasks.md` 等人工维护的工程输入 |
| Workflow Core | Work Item、Attempt、Gate、Evidence、写入租约和运行状态 |
| Supervisor | 前台循环、租约、下一步判断和暂停/恢复 |
| Session | 单次 Agent turn 的输入、输出、记忆和事件 |
| 外部 Agent CLI | 实际读取上下文、编辑代码和返回结果 |

正常链路如下：

```text
Requirement
  -> approved Task
  -> OpenSpec artifacts
  -> Work Items
  -> prepared Attempt
  -> one bounded Agent turn
  -> Session result
  -> Evidence / Gate
  -> next Work Item or human decision
```

## 2. 启动前准备

### 2.1 创建或确认 Task

普通 Task 通过 AIW 创建：

```powershell
aiw init
aiw new payment-retry
```

Requirement 驱动的工作应先完成需求批准和 promotion：

```powershell
aiw requirement chat payment-retry
aiw requirement approve payment-retry APPROVED --by alice --reason "scope approved"
aiw requirement promote payment-retry --task payment-retry
```

Task 至少应有：

```text
openspec/changes/<task-id>/
├── proposal.md       # 如果该变更有 proposal
├── design.md         # 如果存在设计决策
├── tasks.md          # 编号化实现清单
└── specs/            # 相关能力规格
```

`tasks.md` 是 Work Item 的来源。Workflow Core 会将尚未完成的清单项同步为可执行
Work Item，并依据依赖、Gate 和历史 Attempt 选择下一项。

### 2.2 检查计划

先生成或同步计划，再预览下一步：

```powershell
aiw task workflow plan payment-retry
aiw task workflow sync payment-retry
aiw task workflow advance payment-retry
aiw task workflow run payment-retry
```

`advance` 只准备一个 Attempt-bound 请求，不启动模型。

`run` 默认也是预览操作；它只计算下一步并投影状态，不创建 Agent turn。

## 3. Supervisor 的启动与停止

启动前台监督循环：

```powershell
aiw task workflow supervise payment-retry start
```

查看状态：

```powershell
aiw task workflow supervise payment-retry status
```

停止当前 Supervisor：

```powershell
aiw task workflow supervise payment-retry stop
```

Supervisor 运行在当前终端进程中。关闭终端、发送中断或进程异常退出后，不应直接
启动第二个 Supervisor；先查看状态和诊断结果。

## 4. Supervisor 每轮的实际流程

Supervisor 启动后会：

1. 从 `.ai/tasks/<task-id>/state.json` 加载 Workflow 状态。
2. 获取该 Task 唯一的 Supervisor lease，默认有效期约为一分钟。
3. 检查未解决的 projection repair；存在时先尝试修复，修复失败则暂停。
4. 同步 `tasks.md` 与 Workflow Core 的 Work Item 投影。
5. 计算 `workflow.NextRunnerOutcome`。
6. 如果当前状态允许执行，则续租并调用一次：

   ```text
   run --execute
     -> runBoundedTaskTurn
     -> aiw turn <task-id> --supervised
   ```

7. 等待该 Agent turn 返回，并读取对应 Session 结果。
8. 将 Session 结果记录为 Attempt Outcome 和 Evidence。
9. 更新 Workflow 状态及 OpenSpec/Task 的投影。
10. 重新评估下一步，而不是重复使用刚刚完成的 prepared request。
11. 记录 Supervisor observation。
12. 遇到 Gate、阻塞、无可执行 Work Item 或 repair-required 时暂停。

监督循环每轮只允许一个受控 Agent turn。它不会把多个 Agent turn 并行发起到
同一个 Task。

## 5. 工作区与写入协同

### 5.1 默认隔离 worktree

自动执行默认使用隔离 worktree：

```text
.wt/<task-id>/
```

如果需要明确在主工作区执行，必须显式指定：

```powershell
aiw task workflow run payment-retry --execute --primary
```

`--primary` 不能脱离 `--execute` 单独使用。Supervisor 启动时没有主工作区
默认切换行为，仍遵守自动执行的隔离策略。

### 5.2 写入租约

Workflow Core 为当前 Attempt 持有 write lease，用于防止两个写入型 Agent 同时
修改同一个工作区。以下情况会阻止继续执行：

- 当前工作区已有活动的写入 Attempt；
- Task 的 isolated worktree 丢失或绑定无效；
- Session 仍在运行；
- Supervisor lease 已被其他进程持有；
- 状态文件或事件投影需要恢复。

Supervisor 自身也有唯一 lease。模型调用可能超过 lease 的初始有效期，因此
Supervisor 会在提交 Session 结果前续租，确保结果仍由当前监督进程提交。

### 5.3 并行协同边界

AIW 的自动流程不是多 Agent 并行编程模型。协同方式是：

```text
Supervisor
  -> 一个 Attempt
  -> 一个 Session / Agent turn
  -> 持久化 Evidence
  -> 下一轮再选择下一个 Work Item
```

工程 Skill 可以按任务需要使用受控 sub-agent，但 sub-agent 只能执行有界的静态
分析、代码定位或独立实现片段；不得自行运行测试、构建流程、网络操作、提交、
归档或 worktree 操作。主 Agent 负责集成和生命周期变更。

## 6. LLM 调用方式

### 6.1 Provider 入口

`aiw turn` 会根据 Session 和 Provider 配置调用外部 CLI。当前共享的 Provider
配置同时服务于：

- `aiw ask`
- Managed Workflow turn
- CZ 提交向导

Supervisor 启动时可以临时指定 Provider 和模型：

```powershell
aiw task workflow supervise payment-retry start `
  --provider copilot `
  --model gpt-5.6
```

`--provider` 和 `--model` 只对 `start` 有效。

也可以在配置文件中设置默认值：

```toml
[ai]
provider = "copilot"
model = "gpt-5.6"
codex_command = "codex"
copilot_command = "copilot"
```

配置优先级和命令支持的 Provider 取决于当前 AIW 配置实现；环境变量
`OPENAI_API_KEY`、`OPENAI_MODEL`、`OPENAI_BASE_URL` 可用于 OpenAI 兼容 Provider。

### 6.2 单次 Agent turn

Supervisor 不直接拼接并执行任意 Shell 命令，也不直接管理模型对话。它将受控
请求交给：

```text
aiw turn
  -> Session backend
  -> Codex CLI 或 Copilot CLI
  -> Session output
  -> Attempt Outcome / Evidence
```

Agent turn 会读取：

- Task 的 `task.toml`；
- 当前 OpenSpec proposal、design、spec 和 `tasks.md`；
- 当前选中的 Work Item；
- Task-local `artifacts/handoff.md`；
- 当前 Session 的 instructions 和 memory；
- 相关历史 Evidence 和 Gate。

Agent 的目标是完成一个 Work Item，而不是自行决定整个 Task 的完成、发布或归档。

### 6.3 输出持久化

Session 输出保存到：

```text
.ai/sessions/<session-id>/
├── status.json
├── events.jsonl
└── outputs/
    ├── 0001-live.jsonl
    ├── 0001-final.txt
    ├── 0001-events.jsonl
    └── 0001-stderr.log
```

`0001-live.jsonl` 用于 Supervisor 前台显示运行进度；最终结果、事件和标准错误
分别保存在对应文件中。模型输出是可审阅 Evidence，不等于自动通过 Gate。

## 7. Skills 的加载和使用

### 7.1 AIW 不在 Supervisor 层直接执行 Skill

`workflow supervise` 本身不读取或执行 `SKILL.md`。Skill 必须先安装到项目的
`.agents/skills/`，或者由宿主 Agent 使用用户级 `~/.agents/skills/` 目录发现：

```powershell
aiw skills install implement
aiw skills install tdd
aiw skills install code-review
```

安装后的 Skill 是 Agent 可读取的工作流说明，不是 Supervisor 的内置插件。
自动执行时，managed handoff 会要求 Agent 遵循 `implement` Skill；该 Skill
允许宿主 Agent 根据当前 Work Item 隐式选择它。`tdd` 和 `code-review` 仍然
不会被 `implement` 或 Supervisor 自动调用。

### 7.2 推荐 Skill

#### `implement`

这是自动编码最主要的工程 Skill，负责：

- 解析一个 Task 和一个 Work Item；
- 读取对应 OpenSpec 输入；
- 检查 Design Readiness；
- 在 Task 声明的工作区实施最小完整修改；
- 更新 checklist、TODO、Verification 和 `%%` 风险；
- 报告 Evidence、未解决 Gate 和下一步；
- 修改后执行 compile-only 检查：优先使用 `scripts/` 或仓库根目录中的
  `compile` 脚本；没有脚本时使用语言级编译命令；
- 编译失败时修复实现并重新 compile，无法解决时报告阻塞；
- 不自动提交、合并、推送、归档、运行测试或执行广义 build 流程。

#### `tdd`

`tdd` 是显式选择的测试先行 Skill。它不会被 `implement` 或 Supervisor 自动
调用。只有用户或明确的 Agent 流程要求测试先行时才应使用，并且运行测试仍需
明确授权。

#### `code-review`

`code-review` 用于实现后的 Standards/Spec 审查，也不是自动回调。需要在
Workflow 暂停或实现完成后由人类明确选择。

#### `handoff`

`handoff` 用于把当前对话、Task、Work Item、Attempt、Evidence 和 Gate 压缩成
可由新 Agent 继续使用的文档。它不会完成 Attempt、释放 lease 或推进 Task 状态。

### 7.3 Skill 边界

自动编码时，Skill 应遵守以下边界：

- 不创建第二套 Task tracker；
- 不直接修改 AIW 生命周期状态；
- 不伪造 Attempt、Evidence、Gate 或 lease；
- 不自动运行测试、广义 build、格式化、Lint、vet、网络或发布操作；
- 允许且要求执行 compile-only 检查：先查找 `scripts/` 和仓库根目录的
  `compile`、`compile.bat`、`compile.cmd` 或 `compile.ps1`；不得把
  `build.bat` 等综合构建脚本当作 compile；无脚本时使用语言级编译命令；
- compile 失败时将编译器输出作为修复输入并重试，仍失败则报告 `BLOCKED`；
- 不自动执行 Git commit、merge、push、branch 删除或 archive；
- 缺少关键事实时写 `%% NEEDS_INPUT`，无法安全继续时报告 `BLOCKED` 或
  `INCOMPLETE`。

## 8. Gate、Evidence 和暂停机制

Supervisor 不会自行解决以下问题：

- 需求尚未批准或 promotion；
- Design Readiness 未通过；
- 需要人工决定的设计、权限、迁移或发布问题；
- 测试或构建授权缺失；
- Focused Test 授权过期；
- Session 返回失败或不完整结果；
- write lease、worktree 或 projection repair 异常；
- Attempt 达到重试上限；
- 没有依赖满足的下一个 Work Item。

常见暂停结果包括：

```text
gate
blocked
repair-required
runner-paused
no-work
```

Agent 输出只能作为 Evidence。只有在 Evidence 满足规则、Gate 被人工
`resolved` 或 `waived`，并且 Workflow Core 允许时，Work Item 才能完成。

## 9. 失败、诊断和恢复

Supervisor 出错后，先执行：

```powershell
aiw task workflow diagnose payment-retry
aiw task workflow supervise payment-retry status
```

根据诊断结果再选择：

```powershell
aiw task workflow recover payment-retry
aiw task workflow repair payment-retry
aiw wt repair
```

处理原则：

1. 不要立即启动第二个 Supervisor。
2. 不要手动删除 `.ai/tasks/<task-id>/state.json` 或 Session 输出。
3. 保留失败 Attempt 和 Session 结果，以便审计。
4. 先修复状态、worktree 或 projection，再恢复 Supervisor。
5. Session 结果未知时，不要创建第二个 Attempt 覆盖原结果。

如果 Attempt 失败，Supervisor 会记录失败结果并按 retry policy 暂停或继续。
默认每个 Work Item 最多尝试三次；可通过受控 Workflow 命令为单个 Work Item
设置一到五次的上限：

```powershell
aiw workflow retry-policy payment-retry wi-0001 3
```

达到上限后不会静默重试，也不会把 Work Item 标记为完成。需要明确理由后才能
reopen。

## 10. 自动化不会替代的人工操作

Supervisor 不会自动：

- 批准或 promotion Requirement；
- 解决 Gate 或授权验证；
- 运行测试、构建、迁移或广泛验证；
- 提交、推送、合并、删除分支或发布外部变更；
- 创建后台 Scheduler；
- 将 `aiw done` 或 `aiw archive` 当作 Git 已交付。

一个完整的交付流程仍需人工审阅：

```text
Supervisor 完成 Work Items
  -> 检查 Evidence / Gate / Verification
  -> 必要时执行一次授权验证
  -> 人工审查 diff
  -> 单独授权 Git delivery
  -> 合并、清理和 archive
```

## 11. 推荐的完整示例

```powershell
# 1. 准备 Task 和 OpenSpec
aiw new payment-retry
aiw task workflow plan payment-retry
aiw task workflow sync payment-retry

# 2. 预览下一步，不启动 LLM
aiw task workflow run payment-retry

# 3. 明确选择 Provider/Model 后启动前台 Supervisor
aiw task workflow supervise payment-retry start `
  --provider copilot `
  --model gpt-5.6

# 4. 另一个终端查看状态
aiw task workflow supervise payment-retry status

# 5. 如果暂停，先诊断和修复
aiw task workflow diagnose payment-retry
aiw task workflow recover payment-retry

# 6. Supervisor 完成后，人工审阅并单独处理 Git delivery
aiw show payment-retry
git status
```

默认建议先使用 `run` 预览，再使用 `supervise start`。不要把 Supervisor 当作
无边界的自动发布系统；它的核心目标是把一个 Task 拆成可审计、可暂停、可恢复
的单步 Agent 执行。
