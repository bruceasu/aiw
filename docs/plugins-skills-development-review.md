# AIW Plugins 与 Skills 开发辅助报告

日期：2026-09-11
审查范围：当前 `plugins/`、`skills/`、相关 Go 插件调度器及 README。
审查方式：静态阅读与一次只读索引检查。未运行测试、构建、Lint 或自动验证。

## 1. 结论摘要

AIW 已经具备较完整的交互式开发基础：Task、OpenSpec、Session、Worktree、Workflow Core 和插件机制彼此衔接，适合采用“人工决策、Agent 执行、Workflow 记录”的开发方式。

自动开发正式扩大使用前，建议优先处理以下问题：

1. Skill ZIP 导入的路径穿越风险。
2. Java 执行插件的 `file.encoding` 参数拼写错误。
3. 插件安全元数据没有被调度器强制执行。
4. `aiw git` 的原生 Git fallback 可以绕过破坏性操作保护。
5. Skills 缺少统一的完成、失败交接和验证边界。
6. `skills/README.md` 存在编码损坏，影响使用说明。

## 2. Review Findings

### P0：Skill ZIP 导入存在路径穿越风险

`extract_zip()` 直接使用 `ZipFile.extractall()`，没有校验 ZIP 成员路径。恶意 Skill 包可以使用 `../../` 将文件写出临时目录，理论上可能覆盖用户文件。

证据：[plugins/aiw-skills/aiw-skills.py](../plugins/aiw-skills/aiw-skills.py#L228-L235)

建议：

- 解压前逐个检查成员路径是否位于目标目录内。
- 拒绝绝对路径、`..` 路径和符号链接。
- 增加 ZIP 路径穿越回归测试。
- 修复前不要安装来源不可信的 `bundle.zip`。

### P1：Java 执行插件的编码参数拼写错误

三处使用了 `-Dfile.ecnoding=UTF-8`，正确参数应为 `-Dfile.encoding=UTF-8`。

证据：[plugins/aiw-exec-java.py](../plugins/aiw-exec-java.py#L407-L477)

这会导致用户以为编码已经设置，但 Maven 实际忽略该参数。命令行 `--encoding` 还涉及全局常量传递，建议一并改为显式参数传递。

### P1：插件元数据中的安全属性没有被调度器执行

`aiw-plugin.py` 会读取 `readOnly`、`mutatesFiles`、`requiresConfirmation`，但这些字段主要用于展示：

证据：[plugins/aiw-plugin.py](../plugins/aiw-plugin.py#L38-L60)

真正执行插件时，Go 主程序直接调用插件：

证据：[main.go](../main.go#L90-L112)

因此：

- `requiresConfirmation: true` 不会自动产生确认闸门。
- `mutatesFiles: true` 不会阻止自动流程调用。
- `readOnly: true` 不是强制只读保证。

建议把描述性元数据与调度安全策略分开，后者由 Go 调度器或 Workflow Core 强制执行。

### P1：`aiw git` 的未知命令会绕过安全包装

`aiw git` 对未知子命令直接回退到原生 Git：

证据：[plugins/aiw-git/aiw-git.py](../plugins/aiw-git/aiw-git.py#L208-L226)

例如以下命令可能绕过插件确认逻辑：

```powershell
aiw git reset --hard
aiw git clean -fd
aiw git push --force
```

建议：

- 对破坏性原生 Git 命令增加统一确认。
- 或仅允许显式 `--native` 才启用 fallback。
- 自动 Workflow 禁止调用未知 Git 子命令。

### P1：插件发现和插件列表行为不一致

Go 调度器支持顶层插件以及一级嵌套目录：

证据：[internal/plugin/discover.go](../internal/plugin/discover.go#L41-L80)

但 Python 的 `aiw plugin list` 只扫描顶层 `aiw-*.py`：

证据：[plugins/aiw-plugin.py](../plugins/aiw-plugin.py#L43-L61)

结果是某些插件可以运行，却不会出现在插件列表中。建议统一两套发现规则和扩展名优先级。

### P2：Skills reviewed contract 覆盖不足

当前 34 个 canonical Skill 都有 frontmatter，但多数没有统一、明确地写出：

- required inputs
- outputs
- completion criteria
- unresolved-input behavior
- verification scope

例如：[skills/tdd/SKILL.md](../skills/tdd/SKILL.md#L1-L30) 和 [skills/research/SKILL.md](../skills/research/SKILL.md#L1-L20) 都缺少完整的状态和完成边界表达。

建议每个 Skill 至少使用以下结构：

```markdown
## Inputs
## Outputs
## Completion
## Unresolved Inputs
## Verification Boundary
```

### P2：Skills README 存在编码损坏

[skills/README.md](../skills/README.md#L1-L20) 当前显示为乱码，不能作为可靠的中文 Skill 索引或教程。建议统一为 UTF-8，并增加静态编码检查。

### P2：`setup-project` 会自动写入用户 Shell 配置

`aiw setup-project` 会修改 PowerShell、Bash、Zsh 或 Fish 的 profile：

证据：[plugins/aiw-setup-project.py](../plugins/aiw-setup-project.py#L118-L153)

建议增加：

```text
aiw setup-project --dry-run
aiw setup-project --install-completion
```

并在写入用户级文件前进行显式确认。

## 3. 交互式开发教程

### 3.1 初始化项目

```powershell
aiw init
aiw skills list
aiw skills install implement
aiw skills install tdd
```

安装前建议预览：

```powershell
aiw skills install implement --dry-run
```

检查已经安装的 Skill：

```powershell
aiw skills discover
aiw skills discover --json
```

如果目标目录已有人工维护的 Skill，可以登记为 AIW 管理副本：

```powershell
aiw skills adopt
```

不要直接覆盖来源未知的 `.agents/skills/<name>`。

### 3.2 普通交互式 Task

适合需求仍在澄清、需要边讨论边实现的工作：

```powershell
aiw new payment-retry
aiw show payment-retry
aiw context payment-retry
aiw chat payment-retry
```

推荐节奏：

```text
需求讨论
  -> aiw new
  -> aiw context
  -> aiw chat
  -> 人工确认设计
  -> 编辑代码
  -> git status
  -> 人工验证
  -> aiw done
```

需要持久化需求讨论时：

```powershell
aiw requirement chat daily-report
aiw requirement show daily-report
aiw requirement approve daily-report APPROVED --by alice --reason "scope approved"
aiw requirement promote daily-report --task daily-report
```

### 3.3 交互式 Session

`aiw ask` 用于只读咨询：

```powershell
aiw ask "How do I start a Task?"
aiw ask --chat
aiw ask --resume
```

在 Chat 中：

- 每行输入问题。
- 使用 `/send` 提交当前问题。
- 使用 `/exit` 退出。
- `Ctrl+C` 只取消当前输入行。

需要读取额外目录时必须显式授权：

```powershell
aiw ask --allow-path C:\docs "Review the deployment guide"
```

### 3.4 并行交互式开发

需要多个 Agent 同时工作时，给每个 Task 单独的 Worktree：

```powershell
aiw wt add payment-retry main
cd .wt\payment-retry
aiw wt commit payment-retry "implement retry logic"
```

合并前检查：

```powershell
aiw wt status payment-retry
aiw wt pull payment-retry
```

只有确认分支已合并后，才清理资源：

```powershell
aiw status payment-retry DONE
aiw archive payment-retry --cleanup-wt --delete-branch
```

## 4. 自动开发教程

自动开发应使用 Workflow Core，把工作拆成有边界的 Work Item，避免让 Agent 无限循环修改代码。

### 4.1 标准流程

```powershell
aiw new payment-retry
aiw task workflow plan payment-retry
aiw task workflow advance payment-retry
aiw task workflow run payment-retry
```

第一次 `run` 是预览。确认 Work Item、权限和目标目录后，才执行：

```powershell
aiw task workflow run payment-retry --execute
```

自动执行默认使用隔离 Worktree。只有明确需要修改父工作区时才使用：

```powershell
aiw task workflow run payment-retry --execute --primary
```

每次完成一个 Work Item 后检查：

```powershell
aiw task workflow diagnose payment-retry
aiw task workflow repair payment-retry
aiw task workflow recover payment-retry
```

### 4.2 自动开发前的检查清单

- Task ID 与 OpenSpec change ID 相同。
- `tasks.md` 中有编号且可执行的 checklist item。
- `design.md` 没有阻塞性的 `%% NEEDS_INPUT`。
- 当前工作区和目标分支正确。
- 自动执行是否允许创建或修改 Worktree 已明确。
- 是否需要运行测试、构建或验证命令已经明确授权。
- 不可信的 Skill ZIP 尚未进入安装流程。
- 自动 Agent 不会调用未知的破坏性 Git 命令。

## 5. 自动开发故障恢复

### 5.1 需求或设计不完整

不要猜测，保留明确的未决输入：

```text
%% NEEDS_INPUT: <missing decision>
```

然后回到需求或设计阶段，补充决策后再继续 `advance`。

### 5.2 Work Item 被阻塞

先查看诊断：

```powershell
aiw task workflow diagnose payment-retry
aiw show payment-retry
```

处理对应 Gate 后再继续：

```powershell
aiw task workflow advance payment-retry
```

不要使用强制完成绕过未解决 Gate。

### 5.3 Agent 中断或 Session 失效

```powershell
aiw task workflow recover payment-retry
aiw task workflow repair payment-retry
aiw session status
aiw session list
```

确认 Task、Session、branch 和 Worktree 一致后再恢复。不要直接创建第二个 Task 来“绕开”旧状态。

### 5.4 Worktree 有未提交修改

```powershell
aiw wt status payment-retry
git -C .wt\payment-retry status
```

确认修改属于当前 Task 后保存：

```powershell
aiw wt commit payment-retry "save implementation progress"
```

未检查 diff 前不要使用 `--force`，也不要删除 Worktree。

### 5.5 合并冲突

默认先保留 Git 冲突状态：

```powershell
aiw wt pull payment-retry
```

需要 Agent 生成有限范围的文本冲突建议时：

```powershell
aiw wt pull payment-retry --resolve=agent
aiw wt resolve review payment-retry
```

人工确认补丁后才执行：

```powershell
aiw wt resolve apply payment-retry --confirm
git status
```

该操作不会自动提交，也不会自动完成 merge。受保护的 Task 元数据、锁文件、二进制文件和敏感路径仍需人工处理。

### 5.6 插件或编译器不可用

先查看帮助和插件发现结果：

```powershell
aiw --help
aiw plugin list --json
aiw tcc --help
aiw vc --help
aiw bcc --help
```

检查常用环境变量：

```powershell
$env:AIW_ROOT
$env:TCC_HOME
$env:VC_HOME
$env:BCC_ROOT
$env:JAVA_HOME
```

Java Maven 工具建议先预览命令：

```powershell
python plugins\aiw-exec-java.py --single -c com.example.Main --dry-run
```

在 `file.encoding` 拼写错误修复前，不要把命令输出中的编码参数当作已生效。

## 6. Skill 编写建议

新 Skill 应明确说明：

```markdown
---
name: example-skill
description: Use when the user wants ...
---

## Trigger

何时使用，何时不使用。

## Inputs

需要哪些文件、Task、用户决定或权限。

## Process

按顺序执行，每一步都有可检查的完成条件。

## Outputs

输出哪些文件、报告、Evidence 或 Gate。

## Unresolved Inputs

未知信息使用 `%% NEEDS_INPUT: ...`，阻塞时明确标记 `BLOCKED` 或 `INCOMPLETE`。

## Verification Boundary

说明执行了什么验证，哪些测试、构建或运行检查没有执行。
```

Skill 不应自行伪造 Task 状态、Attempt 结果、Lease 或发布结果。生命周期应由 AIW 和 Workflow Core 管理。

## 7. 审查限制与建议验证

本次没有运行测试、构建、格式化、Lint 或网络操作。建议修复 P0/P1 问题后，按以下顺序授权验证：

1. 运行 `plugins/aiw-skills/tests` 中的 ZIP、安装和 manifest 测试。
2. 运行 `plugins/aiw_wt_test.py` 与 Git 插件 dispatch 测试。
3. 对 `aiw-exec-java.py` 做 dry-run，确认生成 `-Dfile.encoding`。
4. 做一次插件列表与实际插件调用的一致性检查。
5. 最后再进行一次自动 Workflow 的隔离 Worktree 演练。
