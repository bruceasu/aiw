# FD-056 独立审查报告

<!-- aiw-data: FD-056-review-r1.json -->

**FD：** FD-056
**Reviewer Session：** `fd056-reviewer-20261011-7c91e4b2`
**来源事件：** `FD-056-000005-implementation-ready`
**审查基线：** `3a9d37b..5324eb13ec29d4d47d502a460945e649af5968dd`（`feature/FD-056`）

## 结论：Changes Requested

### 发现

1. **[P2] 帮助文本搜索会静默吞掉无效清单错误。** [`src/internal/commands/help/super-help.go:359`](../../../src/internal/commands/help/super-help.go:359) 使用 `pls, _ := listPlugins()` 忽略错误。若插件目录中的 `plugin.toml` 无效，`aiw help <多词查询>` 会把清单解析失败当作没有插件结果继续执行，用户看不到包含清单路径的诊断。这与 Acceptance 5 要求清单错误可定位、不能被伪装成未找到相冲突。应让 `searchDocs` 返回错误并由帮助分发层呈现，或用其他方式把该错误传回调用方。

## Acceptance 核对

- **1，旧发现兼容：** `discover.go` 在无清单目录沿用 `matchPluginName`、`bestCandidate`；`manifest.go` 的列表兼容路径保留旧帮助名解析范围。静态实现符合要求。
- **2，多命令与多行文本：** `manifest.go` 使用 TOML 数组表并解析 `description`/`help`；帮助列表按 `PluginInfo` 输出 `name` 与 `description`。README 记录清单字段。
- **3，入口和启动模式：** 显式入口经过 `resolveManifestEntrypoint` 的相对路径、符号链接边界与文件类型检查；缺省入口回退到包目录中的旧文件名候选。`main.go` 将启动模式传入执行层，旧接口映射到 `auto`。
- **4，帮助正文：** `showPluginHelp` 优先打印非空清单帮助，否则按既有插件帮助参数调用执行层；`searchDocs` 检索清单说明和帮助。自然语言帮助搜索的错误路径见上述发现。
- **5，配置错误：** TOML 解析、未知字段、schema、命令名、重复名、说明、启动方式及入口路径错误均产生带清单路径的错误；但帮助文本搜索会吞掉 `listPlugins` 错误，尚未满足所有入口的错误可见性。
- **6，调用兼容：** 静态检查显示执行仍复用环境合并、标准流连接和子进程退出码处理；发现结果路径顺序与 PATH 搜索代码保留。未进行运行验证，因此平台运行行为未验证。

## 检查范围与命令

- 已阅读 FD-056、稳定规格 `openspec/specs/cli-and-plugins/spec.md`、Worker 双格式实现报告及其 JSON。
- 已检查 `3a9d37b..5324eb13ec29d4d47d502a460945e649af5968dd` 的插件解析、发现、执行、帮助、命令分发、README 和规格差异，并静态追踪相关调用链。
- 已运行只读命令：`aiw fd --help`、`aiw fd show FD-056`、`aiw fd resume FD-056`、`aiw fd emit --help`、`aiw fd claim FD-056 FD-056-000005-implementation-ready --session fd056-reviewer-20261011-7c91e4b2`、`git status --short`、`git rev-parse HEAD`、`git diff --stat 3a9d37b..HEAD`、针对实现文件和文档的 `git diff`、针对相关调用和行号的 `rg -n` 与 PowerShell 文件摘录、`git log -5 --oneline --decorate`、`git diff --cached --stat`、`git diff --cached --check`。最后一项发现报告元数据行有尾随空格，已在报告提交前移除。
- 未运行测试、编译、最终构建、格式化、lint、验证脚本或运行时插件检查。Worker 报告和 FD 均记载 compile-only 命令的最终退出码未保留，因此本审查不把它算作编译通过。

## 剩余风险

启动器在不同操作系统上的可用性、Node TypeScript 参数兼容性、TOML 解析和子进程实际行为均未运行验证。初始审查时 worktree 已有 FD 状态与索引状态同步改动；审查证据提交不包含索引文件。发出 Reviewer 结果后，AIW 将 FD 状态更新为 `In Progress`，随后只提交了 FD-056 对应的 FD 与索引状态同步。
