# CLI 与扩展入口

## Purpose

固定 AIW 原生命令、初始化、插件扩展及独立问答/提交辅助入口的公共行为。插件各自的完整业务语义不在本基线枚举范围内。

## Requirements

### Requirement: 内置与插件分派

主入口 MUST 优先处理内置命令；Task 命令支持顶层入口及 `task <command>` 路径。未知顶层名称 MUST 尝试发现 `aiw-<name>` 插件。wt MUST 经插件入口执行。

#### Scenario: 调用工作流命令

- **WHEN** 使用 `aiw workflow ...` 或 `aiw task workflow ...`
- **THEN** 两者进入同一 Task workflow 分派。

#### Scenario: 调用外部扩展

- **WHEN** 顶层命令没有内置处理器且发现了对应插件
- **THEN** 系统把剩余参数交给插件执行，并传入插件名称、路径和 AIW 调用环境。

### Requirement: 插件发现与子进程结果

插件发现 MUST 搜索可执行文件旁和当前目录的 plugins（含一级子目录），以及 PATH 上匹配名称的候选，并按扩展名优先级选取。执行 MUST 使用对应解释器或可执行文件，连接标准输入输出，并将启动错误或非零退出作为失败反馈。

#### Scenario: 插件启动失败

- **WHEN** 插件解释器不存在或子进程无法启动
- **THEN** AIW 报告执行错误，而不是把调用视为成功。

### Requirement: 基础初始化与可选设置

init MUST 创建 OpenSpec、Task 运行目录、工作树及相关指令目录，并仅在缺失时生成基础指令文件。可选 prompts 同步按显式选项执行；官方 setup 插件不可用或失败 MUST 给出提示并允许基础初始化保留。

#### Scenario: 已有项目指令

- **WHEN** 执行基础 init 且基础指令文件已经存在
- **THEN** write-if-missing 路径不覆盖该文件。

#### Scenario: 官方 setup 不可用

- **WHEN** 未跳过 setup，但无法发现或运行官方插件
- **THEN** 系统报告该情况，继续保留已完成的基础初始化。

### Requirement: ask 的系统提示与路径边界

ask MUST 按命令行文本、命令行文件、个人 `~/.aiw/ask/config.toml` 的顺序解析系统提示。显式命令行文件可直接授权读取；个人配置引用的外部文件 MUST 通过工作区、个人 ask 目录、allow-path 或交互确认获得许可。

#### Scenario: 非交互读取外部配置文件

- **WHEN** 配置引用的系统提示文件位于默认范围外，且没有 allow-path
- **THEN** 非交互调用返回授权错误，不静默读取该文件。

### Requirement: ask 的结构化结果

ask MUST 向共享 AI 层请求 ReadOnly 模式并校验响应 schema_version、status、capability 和 safety 等已实现字段。无效 JSON、无效状态或缺少必需对象 MUST 保存错误记录并返回失败。

#### Scenario: 返回无效回答

- **WHEN** 模型输出无法通过 ask 响应校验
- **THEN** 系统保留诊断，不能将无效内容打印为正常成功回答。

### Requirement: cz 的暂存区与审阅

cz MUST 在生成提交草稿前要求存在 staged changes。它 MUST 支持 LLM 草稿或交互向导，并把草稿交给 ReviewAndCommit 流程后再调用提交函数。

#### Scenario: 没有暂存改动

- **WHEN** staged changes 为空
- **THEN** cz 返回需要先 git add 的错误，不生成提交。

### Requirement: Git export 导出提交引用

git export MUST 调用 `git archive --format=zip` 导出指定 Git ref，未指定时使用 HEAD。导出内容 MUST 来自该引用的提交树，而不是未提交的工作区修改。

#### Scenario: 导出默认引用

- **WHEN** 不指定 ref 执行 export
- **THEN** 导出当前 HEAD 的内容。

#### Scenario: 指定其他分支、标签或提交

- **WHEN** 提供可解析的 Git ref
- **THEN** 导出该 ref，无需把工作区切换到该分支。

## 实现依据

- [主入口](../../../main.go)、[Task 分派](../../../internal/commands/task/command.go)。
- [插件发现](../../../internal/plugin/discover.go)、[插件执行](../../../internal/plugin/exec.go)、[初始化](../../../internal/commands/task/init.go)。
- [ask 配置](../../../internal/commands/ask/config.go)、[路径策略](../../../internal/commands/ask/path_policy.go)、[问答执行](../../../internal/commands/ask/command.go)。
- [cz](../../../internal/commands/cz/command.go)、[Git export](../../../plugins/aiw-git/git-export.py)。
- [ask 回归材料](../../../internal/commands/ask/command_test.go)；本次未执行。

%% ReadOnly 是传给 provider 的执行配置；外部 CLI 的实际文件访问隔离仍取决于其实现，allow-path 不能被泛化为所有后端的严格读取白名单。
