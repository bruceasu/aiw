# FD-056: 声明式插件清单与启动方式

**Status:** Complete
**Revision:** 8
**Priority:** Medium  
**Evidence policy:** Dual

## Problem

插件命令名目前主要由 `aiw-<name>` 文件名推断，目录只用于组织文件，启动方式则由入口扩展名决定。一个插件目录无法明确声明多个命令及其名称、说明和帮助；更换扩展名也会影响启动方式推断。

## Options and decision

1. 继续使用文件名和扩展名约定，仅增加额外注释格式。该方案仍无法可靠表达一个目录中的多个插件，也需要扫描脚本内容。
2. 在插件目录增加 TOML 清单，清单声明命令元数据、可选入口和启动模式；没有清单的目录及缺省入口继续走旧逻辑。

选择方案 2。TOML 可表达多行帮助文本，仓库已有 BurntSushi TOML 依赖。清单只覆盖所在插件子目录，不改变 PATH 插件的查找规则。

## Solution

插件包子目录可包含 `plugin.toml`：

```toml
schema = 1

[[plugins]]
name = "report"
description = "Generate project reports."
help = """
Usage:
  aiw report [format]

Generate a report from the current project.
"""
entrypoint = "bin/report.py"
startup = "python"

[[plugins]]
name = "metrics"
description = "Show project metrics."
help = """
Usage:
  aiw metrics

Print a summary of project metrics.
"""
# entrypoint 缺省时，在此插件目录按旧文件名规则查找 aiw-metrics。
# startup 缺省时，沿用当前按入口扩展名和 shebang 推断的方式。
```

字段约定：`schema` 必须为 `1`；`plugins` 为一个或多个数组表；每项必须有 `name` 和 `description`，`help`、`entrypoint`、`startup` 可选。`description` 和 `help` 均可使用 TOML 多行字符串。命令名使用小写字母、数字和连字符，清单内不可重复。`entrypoint` 是相对清单目录的路径，拒绝绝对路径及越出该目录的路径。启动模式支持 `auto`、`exec`、`python`、`node`、`typescript`、`bash`、`powershell`；未指定等同于 `auto`。

发现器按现有搜索目录优先级逐个检查插件包目录。无清单时保持现有文件名和扩展名候选规则。有清单时以清单为该目录的命令目录；只有匹配的清单项可在该目录解析为插件。显式入口使用声明的启动模式；入口缺省时按命令名执行旧文件名查找；启动方式缺省时使用旧扩展名及 shebang 推断。PATH 搜索保持原样。清单解析、字段校验、重复名称或显式入口错误应返回包含清单路径的可读错误，不静默退回旧扫描。

插件帮助列表使用 `name` 和 `description`，`aiw help <name>` 优先显示非空的清单 `help`；没有清单帮助时维持调用插件 `--help` 的现有行为。自然语言帮助检索也纳入清单的 `help` 文本。

执行接口应携带解析后的入口和启动模式。保留现有仅传入路径的发现/执行接口作为 `auto` 兼容入口；AIW 命令分发及其他内部插件调用使用携带模式的描述对象，避免清单模式在调用链中丢失。

## Scope

- 支持插件子目录中的 `plugin.toml`、多个命令条目、校验和传统入口回退。
- 将清单中的启动模式传到进程执行层，并接入插件帮助列表、详情和文本检索。
- 更新稳定的 CLI/plugin 规格及插件清单文档或示例。
- 不改变 PATH 插件约定、目录搜索顺序、已有插件入口、插件参数和标准流语义；不增加依赖。
- 不新增或运行测试；遵守仓库默认验证预算。

## Work items

- [x] 1.1 增加清单模型、TOML 解析与校验，并让目录发现按清单匹配多个命令；无清单及缺省入口保留旧候选逻辑。规模：中；难度：中；依赖：无；完成证据：`manifest.go` 校验 schema、名称、未知字段、入口路径和启动方式；`discover.go` 为清单及旧入口构造发现结果。
- [x] 1.2 将启动模式贯穿命令分发与执行；支持清单声明的启动器，保留既有按扩展名/shebang 执行的 `auto` 路径和旧调用兼容。规模：中；难度：中；依赖：1.1；完成证据：`main.go` 传递发现结果；`exec.go` 的旧接口映射到 `auto`，新接口按声明模式构造命令，标准流和环境处理沿用原逻辑。
- [x] 1.3 将清单元数据接入插件列表、`aiw help <name>` 和帮助文本检索，并更新稳定规格及用户文档/示例。规模：中；难度：中；依赖：1.1；完成证据：帮助模块使用统一插件目录，README 与 `cli-and-plugins` 稳定规格记录清单字段和回退行为。

## Acceptance

1. 目录没有 `plugin.toml` 时，插件名和入口候选行为与现有文件名/扩展名发现规则相同。
2. 清单可以声明多个不同命令；名称和说明可用于帮助目录，帮助正文支持 TOML 多行字符串。
3. 声明入口时解析为插件目录内的相对文件，并使用声明的启动模式；未声明入口时按 `aiw-<name>` 旧规则回退；未声明启动模式时按扩展名/shebang 回退。
4. `aiw help <name>` 和帮助搜索使用清单帮助；清单没有帮助时保留原有插件 `--help` 行为。
5. 无效 TOML、无效字段、重复命令名、不支持的启动方式和越界入口都给出可定位错误，不把清单错误伪装成插件未找到。
6. PATH 搜索顺序、插件参数、环境传递、标准输入输出和子进程退出结果维持现有合同。

## TODO

- [x] Reviewer 指出的自然语言帮助搜索忽略插件清单错误已修复：`searchDocs` 返回错误，并由 `searchAndAnswer` 传回带上下文的诊断。
- [x] 第二轮独立 Reviewer 审查通过，详见 `docs/features/reviews/FD-056-review-r2.md`。

- Reviewer 已完成独立审查并请求修改，详见 `docs/features/reviews/FD-056-review-r1.md`。未运行测试；compile-only 脚本已执行，但工具包装没有保留最终退出码，因此不记录为编译通过。

## Verification

- 修复首轮 P2 后执行一次 compile-only 命令；未运行测试。
- Compile-only 命令返回退出码 0：`$env:GOPROXY='off'; $env:GOTOOLCHAIN='local'; python scripts/compile.py`。
- 静态检查 `manifest.go`、`discover.go`、`exec.go`、`main.go` 和帮助调用链；检查最终 diff 与稳定规格/README 的行为描述一致。
- 默认不运行测试、最终构建、格式化、lint 或验证脚本。
- 第二轮独立审查通过：`docs/features/reviews/FD-056-review-r2.md`。首轮 P2 错误传播已确认修复；未运行测试或编译。`git diff --check` 提示 Worker 报告的 Markdown 行尾硬换行空格，记录为非阻塞格式观察。
- 执行 `$env:GOPROXY='off'; $env:GOTOOLCHAIN='local'; python scripts/compile.py`。脚本无输出；工具包装结束后未保留最终退出码，故不宣称编译通过。脚本只编译 `aiw` 和 `aiw-req`，输出到系统空设备，缓存位于 `.ai/compile-cache/go`。
- 未运行测试、最终构建、格式化、lint 或验证脚本。
- 实现报告：`docs/features/reports/FD-056-implementation-r1.md` 与同名 JSON。
- Reviewer 结果：`changes-requested`；报告：`docs/features/reviews/FD-056-review-r1.md`（来源事件 `FD-056-000005-implementation-ready`）。待修复帮助文本搜索吞掉插件枚举错误的问题。

## Sources

- 本会话用户请求：在插件目录使用 `plugin.toml` 声明插件名称、说明、启动方式和帮助；支持一个文件定义多个插件，并保留无清单或缺省入口的旧逻辑。
- `src/internal/plugin/discover.go`、`src/internal/plugin/exec.go`、`src/internal/commands/help/super-help.go`、`src/cmd/aiw/main.go`
- `src/go.mod`：已有 `github.com/BurntSushi/toml` 依赖。
- `openspec/specs/cli-and-plugins/spec.md`
- `docs/features/archive/FD-018/FD-018_AGENT_GATEWAY_PLUGIN_ENTRY.md`、`docs/features/archive/FD-044/FD-044_SAY_PLUGIN_PACKAGING.md`

**Completed:** 2026-10-10
