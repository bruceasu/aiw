# FD-058：为内置插件补充 `plugin.toml` 元数据

**Status:** Complete
**Revision:** 11
**Priority:** Medium  
**Evidence policy:** Dual

## Problem

AIW 已支持通过 `plugin.toml` 显式声明插件名称、描述和帮助文本，但仓库自带插件仍主要依赖入口文件名与目录名识别。用户通过 `aiw help` 和插件列表查看内置能力时，无法统一获得这些清单元数据。

## Options and decision

1. 只给文档目录中的单文件插件逐个补清单：改动少，但无法覆盖仓库内按目录打包的插件。
2. 只给正式的插件子目录分别添加清单；`src/plugins/` 根目录的平铺单文件插件没有清单时沿用旧规则发现。选择方案 2。

清单只声明插件名称、说明和帮助。省略 `entrypoint` 和 `startup`，使现有发现逻辑继续按旧文件名、扩展名和 shebang 选择启动入口；这样此次变更只补充元数据，不改变插件启动行为。示例目录和共享库/构建辅助文件不作为插件登记。

## Solution

- 为每个正式插件子目录添加各自的 `plugin.toml`：`aiw-ai`、`aiw-cz`、`aiw-git`、`aiw-github`、`aiw-req`、`aiw-say`、`aiw-skills`。
- 不在 `src/plugins/` 根目录创建清单；根目录平铺的单文件插件没有 `plugin.toml` 时，继续按现有文件名和扩展名规则发现。
- 每项填写调用名、准确简洁的说明和多行帮助；帮助文本概述主要用途及 `--help` 用法，不复制完整命令手册。
- 不声明 `entrypoint` / `startup`，保留旧入口文件候选和自动启动方式。

## Scope

- 只覆盖七个插件子目录中的各自主入口（`aiw-git` 是包含多条子命令的一个插件）。
- 根目录平铺入口（`ai-code-index`、`ai-gen-index`、`bcc`、`cxs`、`exec-java`、`fd`、`file`、`hello`、`notify`、`patch`、`plugin`、`setup-project`、`tcc`、`vc`）不生成清单并继续使用旧规则发现。
- 排除 `examples/`、`__pycache__/`、`aiw_codec.py`、`send_teams_msg.py`、插件内部模块和构建产物。
- 不修改插件发现代码、启动方式、依赖、测试或部署配置；仅为七个正式子目录补充说明清单，不新增 OpenSpec change。

## Work items

- [x] 1.1 确认根目录平铺单文件插件继续使用旧规则发现，不为其创建 `plugin.toml`。
- [x] 1.2 为七个正式插件子目录分别添加 `plugin.toml`，登记对应插件名称、说明和多行帮助；省略入口和启动字段以保留旧行为。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

1. `src/plugins/` 根目录不存在 `plugin.toml`；其平铺单文件插件继续按原文件名、扩展名规则发现。
2. 七个正式子目录各有一个有效 `plugin.toml`，且声明的名称与旧目录/插件入口名称一致。
3. 每个条目都有非空 `description` 与可读 `help`；多行帮助使用 TOML 多行字符串。
4. 所有清单均省略 `entrypoint` 和 `startup`，插件启动继续由原有候选发现与自动模式决定。
5. 示例、辅助库和构建产物不会被误登记；插件代码、发现行为和既有命令名不变。

## TODO

- [x] 按用户补充收敛范围：不为根目录平铺插件新增清单，移除临时根清单发现逻辑及稳定规格改动。
- [x] 完成七个子目录清单并回退临时根目录发现代码与稳定规格改动，生成最新实施报告。
- [x] 按 Reviewer r1 意见从 `discover.go`、`manifest.go` 和稳定规格移除搜索根清单支持，保留子目录清单与根目录旧式发现。
- [x] Reviewer r2 确认修正后范围符合验收要求。

## Verification

- 静态核对七个子目录清单与现有入口文件、目录和代码中的命令元数据；发现根清单支持未完全回退后，按基线版本恢复发现代码与稳定规格，并再次检查相关差异。
- 编译命令 `$env:GOPROXY='off'; $env:GOTOOLCHAIN='local'; python scripts/compile.py` 曾在范围调整前运行，退出码 1。Go 编译子进程因系统分页文件/内存不足无法启动或分配内存；未观察到源码诊断，不重试。最终范围不修改 Go 源码。
- 未运行测试、最终构建、格式化、lint 或验证脚本。
- 不验证插件运行时行为；省略启动字段以保持由已审查的现有兼容路径处理。
- Reviewer r1（来源事件 `FD-058-000008-implementation-ready`）请求修改：`discover.go`、`manifest.go` 和稳定规格仍包含搜索根清单支持，与本次收敛后的范围及 Worker 报告不符。详见 `docs/features/reviews/FD-058-review-r1.md`。未运行测试、编译或 runtime 验证。
- Worker 修正：以上三个文件恢复至搜索根支持变更前的 `2ce3f71` 基线；本次没有再次运行 compile-only 命令。
- Reviewer r2（来源事件 `FD-058-000010-implementation-ready`）确认 `discover.go`、`manifest.go` 和稳定规格不再支持搜索根清单；根目录清单不存在，七个子目录清单保留。静态审查通过，详见 `docs/features/reviews/FD-058-review-r2.md`。未运行测试、编译或 runtime 验证。

## Sources

- 用户要求为当前插件补充 `plugin.toml` 说明。
- `openspec/specs/cli-and-plugins/spec.md`：清单字段、目录发现、入口和启动缺省行为。
- `src/plugins/` 当前入口文件、各正式插件目录的 README / `META` 元数据。
- `README.md`：插件搜索路径和源码插件命名约定。

**Completed:** 2026-10-11
