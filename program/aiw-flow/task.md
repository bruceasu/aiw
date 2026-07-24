# 开发任务：实现 Codex Flow Python 工具

%% 历史设计文档：MCP 后端方案已撤销，当前实现仅支持 exec 后端。
%% 当前实现也不管理 Git Worktree；请使用 aiw/aiw-wt 准备工作区，再通过 --workspace 交给 aiw-flow。

你是一位资深 Python 工程师、CLI 工具架构师、Git 自动化专家，并熟悉 Codex CLI、异步进程管理和持久化工作流设计。

请在当前目录中实现一个可直接运行的 Python 项目：

```text
aiw-flow
```

该工具用于通过 Python 管理 Codex 代码执行 Agent，支持两种执行后端：

1. `codex exec`
2. `codex mcp`

两种后端必须共享统一的 Session、Thread、Prompt、Memory、Git Worktree、日志、状态和恢复机制。

不要只输出设计说明。请直接创建完整项目文件、实现代码、测试并运行验证。

---

# 一、项目目标

实现一个轻量但可靠的 Codex 工作流管理工具。

典型使用场景如下。

## Exec 模式

用于轮次较少、阶段明确的任务：

```text
分析
→ 实施
→ 修正
→ 验证
→ 结束
```

每一轮通过独立的 `codex exec` 进程执行。

首次运行创建新的 Codex Thread，后续通过明确保存的 `thread_id` 恢复同一个 Thread。

## MCP 模式

用于长时间、多任务、多阶段工作流：

```text
需求分析
├── 子任务 A
├── 子任务 B
├── 子任务 C
├── 实施
├── 审查
├── 修正
└── 汇总
```

MCP Server 可长期运行，管理多个 Session 和 Thread。

MCP 模式需要支持：

* 新建 Thread
* 继续指定 Thread
* 多 Session
* Thread 级互斥锁
* Workspace 级互斥锁
* MCP 进程生命周期管理
* MCP 服务异常后的重启和恢复
* 跨进程恢复已有 Thread
* 长时间任务状态持久化

---

# 二、技术要求

必须使用：

* Python 3.12+
* 标准库优先
* `asyncio`
* `pathlib`
* `dataclasses`
* 类型标注
* `argparse` 或轻量 CLI 框架
* JSON / JSONL
* 原子文件写入
* 文件锁
* Git 命令行
* pytest

允许使用少量必要依赖，但必须说明用途。

不要引入大型 Web 框架。

项目应支持：

* Windows
* Linux
* macOS

路径、进程启动、编码和文件锁必须考虑跨平台兼容性。

所有文本文件使用 UTF-8。

---

# 三、项目结构

请实现以下目录结构。允许根据实现需要增加文件，但不要把所有逻辑放进一个文件。

```text
aiw-flow/
├── pyproject.toml
├── README.md
├── AGENTS.md
├── .gitignore
├── src/
│   └── codex_flow/
│       ├── __init__.py
│       ├── __main__.py
│       ├── cli.py
│       ├── config.py
│       ├── models.py
│       ├── session_store.py
│       ├── event_store.py
│       ├── prompt_composer.py
│       ├── memory_manager.py
│       ├── lock_manager.py
│       ├── process_utils.py
│       ├── workspace_manager.py
│       ├── worktree_manager.py
│       ├── artifact_manager.py
│       ├── safety.py
│       └── backends/
│           ├── __init__.py
│           ├── base.py
│           ├── exec_backend.py
│           └── mcp_backend.py
├── tests/
│   ├── test_session_store.py
│   ├── test_prompt_composer.py
│   ├── test_exec_event_parser.py
│   ├── test_workspace_manager.py
│   ├── test_worktree_manager.py
│   ├── test_lock_manager.py
│   └── test_cli.py
└── examples/
    ├── coding-agent-instructions.md
    ├── analyze.md
    ├── implement.md
    └── fix-tests.md
```

---

# 四、统一核心模型

定义统一的后端协议。

参考模型如下，可合理调整，但不得破坏职责边界：

```python
@dataclass(frozen=True)
class TurnRequest:
    session_id: str
    prompt: str
    workspace: Path
    thread_id: str | None
    instructions: str
    memory: str
    phase: str
    timeout_seconds: int | None = None


@dataclass(frozen=True)
class TurnResult:
    thread_id: str | None
    final_output: str
    exit_code: int
    events_file: Path
    output_file: Path
    started_at: datetime
    completed_at: datetime
    interrupted: bool = False


class CodexBackend(Protocol):
    async def start(self) -> None:
        ...

    async def run_turn(self, request: TurnRequest) -> TurnResult:
        ...

    async def close(self) -> None:
        ...
```

必须实现：

```text
ExecCodexBackend
McpCodexBackend
```

Session Manager 和 CLI 不应依赖后端内部实现。

---

# 五、Session 目录

默认数据目录：

```text
.aiw-flow/
```

每个 Session 使用独立目录：

```text
.aiw-flow/
├── sessions/
│   └── <session-id>/
│       ├── status.json
│       ├── instructions.md
│       ├── memory.md
│       ├── events.jsonl
│       ├── session.lock
│       ├── prompts/
│       │   ├── 0001-analyze.md
│       │   ├── 0002-implement.md
│       │   └── 0003-fix.md
│       ├── outputs/
│       │   ├── 0001-final.txt
│       │   ├── 0001-events.jsonl
│       │   └── 0001-stderr.log
│       └── artifacts/
│           ├── final.patch
│           ├── git-status.txt
│           ├── git-diff.txt
│           └── summary.json
├── worktrees/
│   └── <session-id>/
├── locks/
└── logs/
```

状态文件不得默认放在目标 Git 仓库内。

---

# 六、status.json

实现版本化状态结构。

至少包含：

```json
{
  "schema_version": 1,
  "session": {
    "id": "GWJ-1234-order-slip",
    "title": "Fix order slip output",
    "backend": "exec",
    "state": "active",
    "created_at": "ISO-8601",
    "updated_at": "ISO-8601"
  },
  "codex": {
    "thread_id": null,
    "thread_name": "GWJ-1234-order-slip",
    "codex_home": null,
    "model": null,
    "profile": null,
    "last_turn": 0
  },
  "workspace": {
    "source_repo": null,
    "workspace_path": null,
    "is_git": false,
    "use_worktree": false,
    "base_ref": null,
    "branch": null,
    "base_commit": null,
    "head_commit": null,
    "dirty": false
  },
  "instructions": {
    "system_file": "instructions.md",
    "memory_file": "memory.md",
    "instructions_hash": null,
    "memory_hash": null
  },
  "execution": {
    "current_phase": null,
    "last_command": [],
    "last_exit_code": null,
    "last_started_at": null,
    "last_completed_at": null
  },
  "result": {
    "status": "not_started",
    "tests_passed": null,
    "final_output_file": null,
    "patch_file": null,
    "error_message": null
  }
}
```

要求：

* 支持未来 schema migration
* 未知字段不能导致程序崩溃
* 写入必须原子化
* 写入前先生成临时文件
* flush 和 fsync
* 使用 `os.replace`
* 更新时持有 Session 文件锁
* 时间使用带时区的 ISO-8601 格式

---

# 七、事件日志

完整执行历史写入：

```text
events.jsonl
```

示例：

```json
{"time":"...","type":"session.created","session_id":"..."}
{"time":"...","type":"worktree.created","branch":"..."}
{"time":"...","type":"turn.started","turn":1,"phase":"analyze"}
{"time":"...","type":"thread.bound","thread_id":"..."}
{"time":"...","type":"process.started","pid":1234}
{"time":"...","type":"turn.completed","turn":1,"exit_code":0}
```

要求：

* 每条事件单独一行
* 每次 append 后及时 flush
* 保留原始 Codex JSONL 事件
* 区分内部事件和 Codex 原始事件
* 即使任务失败，也要记录最终失败事件

---

# 八、Prompt 设计

工具需要区分：

1. Persistent Instructions
2. Session Memory
3. Current Task Prompt

组合格式建议：

```text
[Persistent Execution Instructions]

<instructions.md>

[Session Memory]

<memory.md>

[Current Phase]

<phase>

[Current Task]

<user prompt>
```

要求：

* Prompt 通过 stdin 传入子进程
* 不把完整 Prompt 拼进 Shell 字符串
* 不使用 `shell=True`
* 保存每轮最终 Prompt 到 `prompts/`
* Prompt 文件包含轮次和阶段
* 计算 Prompt SHA-256
* 状态中记录 hash
* 保留用户原始问题
* 支持 `--prompt`
* 支持 `--prompt-file`
* 支持从 stdin 读取 Prompt

不要假设 Codex CLI 一定有标准 Chat API 的 system role。

Persistent Instructions 应作为明确的执行规则前置到任务 Prompt。

同时保留对仓库 `AGENTS.md` 的自然支持。

---

# 九、Memory 管理

实现 `memory.md`。

建议结构：

```markdown
# Session Memory

## Goal

## Confirmed Findings

## Decisions

## Modified Files

## Validation State

## Open Issues
```

要求：

* Session 创建时生成初始 Memory
* 每轮可以通过 CLI 更新
* 支持人工编辑后继续
* 保存 memory hash
* 不要让程序自动无条件覆盖用户手工修改
* 支持将本轮结果追加为 Memory Note
* 第一版不需要让模型自动完全重写 Memory
* 提供显式命令更新或追加 Memory

CLI 示例：

```bash
aiw-flow memory show SESSION_ID
aiw-flow memory edit SESSION_ID
aiw-flow memory append SESSION_ID --text "..."
aiw-flow memory replace SESSION_ID --file memory.md
```

---

# 十、Exec Backend

实现 `codex exec` 后端。

## 新 Thread

概念命令：

```text
codex exec --json
```

Prompt 从 stdin 传入。

## 恢复 Thread

概念命令：

```text
codex exec resume <thread-id> --json
```

实际参数顺序和有效选项必须通过当前环境中的：

```bash
codex exec --help
codex exec resume --help
```

进行确认。

不要盲目假设命令格式。

如果当前环境没有安装 Codex：

* 测试不能依赖真实 Codex
* 使用 mock/fake process
* README 中说明安装要求
* CLI 应给出明确错误

## ExecBackend 要求

必须：

* 使用 `asyncio.create_subprocess_exec`
* 设置 cwd 为 Session Workspace
* Prompt 通过 stdin
* 分别读取 stdout 和 stderr
* 避免 stdout/stderr 管道死锁
* 实时解析 stdout JSONL
* 从 `thread.started` 或等价事件提取 `thread_id`
* 保存原始 JSONL
* 保存 stderr
* 保存最终输出
* 记录退出码
* 支持 timeout
* 支持取消
* timeout 后先 graceful terminate
* 超时后再强制 kill
* Windows 和 POSIX 分别正确处理
* 不使用 `shell=True`
* 不记录环境变量中的密钥

首次运行：

```text
thread_id 为空
→ 新建 Thread
→ 捕获 Thread ID
→ 原子写入 status.json
```

继续运行：

```text
已有 thread_id
→ 明确 resume 指定 ID
→ 禁止使用 --last
```

禁止使用 `resume --last`，因为并发 Session 下不可靠。

## Ephemeral

可选支持：

```text
--ephemeral
```

但必须明确：

* Ephemeral Session 不保证后续 resume
* status.json 中记录该属性
* 使用 ephemeral 时，如果用户尝试继续，应给出明确提示

---

# 十一、MCP Backend

实现一个长期运行的 Codex MCP 后端。

启动命令必须在运行时通过 Codex CLI help 验证。

可能的命令名称包括但不限于：

```text
codex mcp
codex mcp-server
```

不要硬编码未经验证的假设。

## MCP Backend 职责

至少实现：

* 启动 Codex MCP 子进程
* 建立 MCP STDIO 客户端连接
* 初始化 MCP Session
* 列出或确认可用工具
* 新 Thread 调用 Codex 创建工具
* 已有 Thread 调用 Codex reply 工具
* 提取和持久化 thread_id
* 多 Session 共用一个 MCP 进程
* 每个 Thread 使用 `asyncio.Lock`
* 每个 Workspace 使用 `asyncio.Lock`
* MCP 进程退出检测
* 自动重启
* 重启后继续使用已有 thread_id
* 记录 server pid
* 记录 server start time
* 正常 close
* 应用退出时停止子进程
* 不因 MCP 关闭而删除 Session 状态
* 不假设 MCP 关闭会删除磁盘中的 Codex Thread

## 并发规则

```text
同一个 thread_id：
一次只允许一个请求

同一个 workspace：
一次只允许一个写操作

不同 Thread + 不同 Workspace：
允许并行
```

如果两个 Session 指向同一个 Workspace，必须拒绝或串行化。

## MCP 生命周期

支持：

```bash
aiw-flow daemon start
aiw-flow daemon status
aiw-flow daemon stop
```

第一版允许 MCP daemon 与 CLI 处于同一 Python 进程中，但架构必须允许后续独立成常驻进程。

如果完整 MCP daemon 需要超出 MVP 范围：

* 先实现 `McpCodexBackend` 生命周期类
* 使用测试替身验证行为
* README 明确已实现和未实现的部分
* 不得伪装为已经支持

---

# 十二、Git 和 Worktree

如果指定目录是 Git 仓库，并启用 `--worktree`，则创建独立 worktree。

示例：

```text
git worktree add
  -b agent/<session-id>
  <worktree-path>
  <base-ref>
```

## 创建要求

* 验证源仓库
* 获取仓库根目录
* 验证 base ref
* 默认分支名：

```text
agent/<sanitized-session-id>
```

* 防止 Session ID 注入 Git 参数
* 禁止 Session ID 包含路径穿越
* 使用参数数组，不使用 Shell 拼接
* 记录 base commit
* 记录 branch
* 记录 workspace path
* 记录 HEAD

## 恢复要求

每次继续前验证：

* Worktree 目录存在
* `.git` 指向有效
* 当前 branch 与 status.json 一致
* Git 仓库根目录正确
* HEAD 可读取
* 没有被其他 Session 占用
* Worktree 未被 Git prune
* 状态文件中的 branch 仍存在

如 worktree 丢失：

* 如果 branch 存在，支持显式 repair
* 不要自动静默创建全新 branch
* 提供：

```bash
aiw-flow workspace repair SESSION_ID
```

## 非 Git 模式

支持：

```text
--workspace <directory>
```

此时：

* 不创建 worktree
* status.json 中 `is_git=false` 或按实际检测
* 仍使用 Workspace Lock

## Finish 时

支持：

```bash
aiw-flow finish SESSION_ID --create-patch
```

生成：

* `git status --short`
* `git diff`
* `git diff --binary`
* patch 文件
* 当前 commit
* changed file list

默认不要：

* commit
* push
* merge
* 删除 branch

## Cleanup

支持：

```bash
aiw-flow delete SESSION_ID
```

必须要求显式确认，或使用：

```text
--yes
```

清理选项：

* 删除 Session 状态目录
* 删除 worktree
* 可选删除 branch
* 不自动删除 Codex Thread 持久化数据
* README 中解释 Thread 与 Session 状态的区别

---

# 十三、CLI 命令

实现：

```text
aiw-flow new
aiw-flow run
aiw-flow continue
aiw-flow status
aiw-flow list
aiw-flow inspect
aiw-flow finish
aiw-flow archive
aiw-flow delete
aiw-flow memory
aiw-flow workspace
aiw-flow daemon
```

## new

示例：

```bash
aiw-flow new \
  --id GWJ-1234-order-slip \
  --title "Fix order slip output" \
  --backend exec \
  --repo D:/repos/mt5-report \
  --base origin/develop \
  --worktree \
  --instructions examples/coding-agent-instructions.md
```

要求：

* Session ID 必须唯一
* 校验 Session ID
* 创建目录
* 创建 status.json
* 创建 instructions.md
* 创建 memory.md
* 可选创建 worktree
* 不自动执行 Codex

## run

第一轮：

```bash
aiw-flow run GWJ-1234-order-slip \
  --phase analyze \
  --prompt-file examples/analyze.md
```

要求：

* 若已有 thread_id，默认拒绝，提示使用 continue
* 支持 `--force-new-thread`
* 运行前加锁

## continue

```bash
aiw-flow continue GWJ-1234-order-slip \
  --phase implement \
  --prompt "根据分析实施修改并运行测试"
```

要求：

* 自动读取 thread_id
* 不允许使用其他 Session 的 Thread
* 若 thread_id 缺失，明确报错
* 支持失败后再次 continue

## status

清晰输出：

* Session ID
* Backend
* State
* Thread ID
* Workspace
* Worktree branch
* Current phase
* Last exit code
* Last turn
* Dirty status
* Last output file
* Last error

支持：

```text
--json
```

## list

输出所有 Session。

支持筛选：

```text
--backend
--state
--repo
```

## inspect

显示：

* status
* 最近事件
* Git 状态
* Prompt 路径
* Output 路径
* Memory 摘要
* Thread ID

## finish

* 生成 artifacts
* 状态改为 completed
* 默认保留 worktree
* 默认保留 Thread
* 不自动删除任何重要数据

## archive

* 状态设为 archived
* 可选移动 Session 目录到 archive
* 不删除 Codex Thread

## delete

* 明确显示将删除的内容
* 默认交互确认
* `--yes` 跳过确认
* 默认不删除 Git branch
* 默认不删除 Codex Thread 数据

---

# 十四、状态机

实现基本状态：

```text
created
active
running
paused
failed
completed
archived
deleted
```

合法转换至少包括：

```text
created → running
running → active
running → failed
active → running
active → paused
paused → running
active → completed
failed → running
completed → archived
archived → deleted
```

禁止明显无效转换。

发生异常时，不允许 Session 永久停留在 `running`。

程序启动或 inspect 时，若发现：

```text
state = running
但对应进程不存在
```

标记为：

```text
failed
```

并记录恢复事件。

---

# 十五、锁

实现：

1. Session Lock
2. Workspace Lock
3. Thread Lock
4. Daemon Lock

要求：

* 跨进程文件锁
* 同进程内可结合 `asyncio.Lock`
* Windows 和 POSIX 支持
* 锁文件包含：

  * pid
  * hostname
  * acquired_at
  * session_id
* 检测陈旧锁
* 不可随意删除仍有效的锁
* 支持合理 timeout
* 锁异常提供清晰错误

---

# 十六、安全要求

必须遵循：

* 禁止 `shell=True`
* 禁止命令字符串拼接
* 所有外部命令使用参数列表
* 校验 Session ID
* 防止路径穿越
* 防止 branch name 注入
* 不把 API Key 写入日志
* 不输出完整环境变量
* 不读取或修改用户未指定目录
* 默认禁止 commit
* 默认禁止 push
* 默认禁止访问生产系统
* 默认不执行清理性 Git 命令
* 不自动运行 `git reset --hard`
* 不自动运行 `git clean -fd`
* 不自动覆盖未提交修改
* 不自动删除 worktree
* 不自动删除 branch
* 不自动删除 Codex Thread 数据

如果 Workspace 在没有 worktree 的情况下存在未提交修改，必须警告。

---

# 十七、Codex 配置支持

支持配置：

```text
model
profile
sandbox
approval policy
CODEX_HOME
timeout
additional codex args
```

但需要：

* 对危险参数进行限制
* 不允许用户通过 additional args 覆盖由程序管理的关键参数
* 配置优先级清晰：

```text
CLI
→ Session config
→ Project config
→ Global config
→ Defaults
```

实现配置文件：

```text
~/.config/aiw-flow/config.toml
```

Windows 使用合适的用户配置目录。

Session 状态中保存最终解析后的关键配置。

---

# 十八、测试要求

所有核心逻辑必须有测试。

不得要求测试环境安装真实 Codex。

使用 Fake Backend、Fake Process 或 Mock Subprocess。

至少测试：

## Session Store

* 创建状态
* 原子更新
* schema_version
* 非法状态转换
* 异常恢复
* 未知字段兼容

## Prompt Composer

* Instructions + Memory + Current Prompt 顺序
* UTF-8
* Hash 稳定
* 空 Memory
* Prompt 文件保存

## Exec Event Parser

输入模拟 JSONL：

```json
{"type":"thread.started","thread_id":"abc"}
{"type":"item.completed","item":{"type":"agent_message","text":"done"}}
```

验证：

* 提取 Thread ID
* 提取最终输出
* 未知事件不报错
* 非法 JSON 单独记录
* 不丢失原始数据

## Worktree

使用临时 Git 仓库：

* 创建 worktree
* branch 命名
* 恢复
* Git 状态
* patch
* repair
* 路径含空格
* 非法 Session ID

## Lock

* 同 Session 冲突
* 陈旧锁
* timeout
* 跨进程基本行为

## CLI

* new
* status
* list
* run with fake backend
* continue
* finish
* delete confirmation

## MCP

使用 Fake MCP Server 或 Mock Client 测试：

* 新 Thread
* reply
* Thread Lock
* Workspace Lock
* Server restart
* 已有 Thread 恢复
* close

---

# 十九、README

README 必须包含：

1. 项目用途
2. Exec 与 MCP 模式区别
3. 安装方法
4. Codex CLI 前置条件
5. 快速开始
6. Git Worktree 示例
7. Session 恢复
8. Thread ID 的含义
9. status.json
10. memory.md
11. events.jsonl
12. finish 和 cleanup
13. Windows 使用说明
14. Linux/macOS 使用说明
15. 安全边界
16. 当前限制
17. 故障排查

必须明确说明：

* `codex exec` 默认可能创建持久化 Thread
* 后续恢复必须保存明确的 `thread_id`
* 不要依赖 `--last`
* `codex mcp` 关闭不等于删除所有 Thread
* `aiw-flow delete` 默认只删除自身状态和可选 worktree
* Codex Thread 的物理持久化由 Codex 自身管理
* 若需要彻底隔离，可为 Session 指定独立 `CODEX_HOME`
* 若删除独立 `CODEX_HOME`，其中所有 Codex 状态都会被清除
* 不要直接依赖 Codex 内部 rollout 文件名

---

# 二十、AGENTS.md

为本项目创建 `AGENTS.md`，至少包含：

```markdown
# Development Rules

- Python 3.12+
- Use typed Python.
- Prefer standard library.
- Use asyncio for subprocess execution.
- Never use shell=True.
- Never concatenate untrusted input into commands.
- All state writes must be atomic.
- All session-changing operations require locks.
- Tests must not require a real Codex installation.
- Do not weaken tests to make them pass.
- Do not commit or push.
```

---

# 二十一、示例 Prompt 文件

创建：

```text
examples/coding-agent-instructions.md
```

内容包括：

* 先阅读代码
* 最小修改
* 不 commit
* 不 push
* 不访问生产
* 不声称未实际执行的测试成功
* 输出 root cause、changed files、commands、tests、risks

创建：

```text
examples/analyze.md
```

要求：

```text
只分析，不修改文件。
识别根因、涉及文件、建议修改、测试计划和风险。
```

创建：

```text
examples/implement.md
```

要求：

```text
根据已批准的分析实施最小修改。
增加回归测试。
运行相关测试。
不要 commit 或 push。
```

创建：

```text
examples/fix-tests.md
```

要求：

```text
分析当前失败测试。
修复实际根因。
不要删除测试、跳过测试或降低断言强度。
重新运行相关测试。
```

---

# 二十二、实现策略

按以下顺序实施。

## Phase 1：Exec MVP

优先完整实现：

* Session Store
* Event Store
* Prompt Composer
* Memory
* Exec Backend
* Worktree
* CLI
* Tests

## Phase 2：可靠性

实现：

* 原子写入
* 锁
* timeout
* cancel
* 异常恢复
* patch
* repair

## Phase 3：MCP

实现：

* McpBackend
* 长期 MCP 进程
* 多 Session
* Thread Lock
* Workspace Lock
* 重启和恢复

如果 MCP 的当前 CLI 或 Python SDK 接口与预期不同：

1. 先检查本地帮助和可用包
2. 使用实际接口
3. 在 README 中记录差异
4. 不得虚构不存在的 API
5. 保持 `McpCodexBackend` 接口稳定
6. 对暂时无法完成的部分提供明确的受控异常，不得静默失败

---

# 二十三、完成标准

任务完成前必须：

1. 创建所有项目文件
2. 实现 Exec MVP
3. 实现 Worktree 支持
4. 实现 status.json
5. 实现 events.jsonl
6. 实现 Prompt 和 Memory
7. 实现统一 Backend 接口
8. 实现 MCP Backend，或明确实现真实可运行的最小版本
9. 运行格式检查
10. 运行类型检查
11. 运行全部测试
12. 修复测试失败
13. 检查 Git diff
14. 更新 README
15. 输出最终报告

最终报告必须包含：

```text
Summary
Implemented Features
Project Structure
Important Design Decisions
Commands Executed
Test Results
Known Limitations
Recommended Next Steps
```

不得声称没有实际运行过的测试成功。

---

# 二十四、重要约束

* 不要只提供伪代码
* 不要只提供架构图
* 不要等待用户逐步确认
* 不要把全部逻辑放进一个 Python 文件
* 不要依赖全局可变状态
* 不要把 Session 状态放进目标仓库
* 不要覆盖用户已有文件
* 不要创建 commit
* 不要 push
* 不要自动安装 Codex
* 不要自动登录 Codex
* 不要输出密钥
* 不要静默忽略异常
* 不要在异常时留下永久 `running` 状态
* 不要依赖 `codex exec resume --last`
* 不要假设关闭 MCP 会删除 Thread
* 不要把可读名称当作真实 Thread ID
* 不要直接操作 Codex 内部 Session JSONL 文件

现在开始：

1. 检查当前目录
2. 创建项目结构
3. 实现 Phase 1
4. 运行测试
5. 再实现 Phase 2 和 Phase 3
6. 修复所有可以修复的问题
7. 输出最终实施报告
