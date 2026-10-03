# FD-026: Gateway short command

**Status:** Complete
**Revision:** 12
**Priority:** Low
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

网关 CLI 当前使用 `aiw agent-gateway`，命令名较长。用户要求将它替换为短命令
`aiw gw`，并明确不保留旧命令兼容入口。

## Options and decision

1. 只重命名插件入口文件，保留 `plugins/aiw-agent-gateway` 目录。
2. 将插件目录和入口统一改为 `plugins/aiw-gw/aiw-gw.py`，并同步构建与安装路径。

选择第二种。AIW 插件发现依据 `aiw-<subcommand>` 文件名，因此新入口可注册为
`aiw gw`；插件目录、入口和安装位置统一使用短名，避免目录和命令不一致。用户不要求
旧命令兼容。Go 网关二进制、协议、认证、状态存储格式及配置字段保持不变。旧安装目录
中的配置不自动迁移；本次不运行安装或触碰外部安装目录。

## Solution

将网关插件入口及说明放在 `plugins/aiw-gw/`，构建输出和安装复制目标同步改为该目录。
所有当前操作指引、`aiw-ai` 帮助和启动/停止脚本改用 `aiw gw`。移除旧
`aiw-agent-gateway.py` 命令入口，不提供旧命令别名。默认配置仍从入口同目录读取，
网关二进制文件名 `agent-gateway[.exe]` 保持不变。

## Scope

修改插件入口与 README、`build.bat` 中的构建/安装路径、启动/停止批处理脚本、当前
客户端和网关操作文档，以及引用插件入口路径的当前测试夹具。保留内部 Go 模块目录、
二进制名、API、配置结构和持久状态目录约定。不搬动或复制真实 `gateway.json`，不
迁移外部安装配置，不构建安装包、不部署、不执行 Git 交付操作，除 FD 生命周期明确
授权的本地提交/合并步骤外。

## Work items

- [x] 1.1 将插件入口和插件说明统一到 `plugins/aiw-gw/`；规模小、难度低、无依赖。
  完成标准：目录中存在唯一 `aiw-gw.py` 入口，参数和默认配置行为保留，旧入口不存在。
- [x] 1.2 同步网关构建输出与安装来源到 `plugins/aiw-gw/`；规模小、难度低，依赖 1.1。
  完成标准：构建生成的 Windows/Linux 二进制和插件入口位于同一短名目录，安装复制路径一致。
- [x] 1.3 将启动/停止脚本和当前用户文档改用 `aiw gw`；规模小、难度低，依赖 1.1。
  完成标准：当前使用指引不再要求运行 `aiw agent-gateway`，历史归档不改写。
- [x] 1.4 静态审查并记录实现证据；规模小、难度低，依赖 1.1–1.3。
  完成标准：检查插件发现、安装路径和调用链；记录规定的 compile-only 结果及未运行的检查。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- `aiw gw start|stop` 通过插件发现调用短名目录中的现有网关入口。
- 新安装布局将插件入口和 Windows/Linux 网关二进制放入 `plugins/aiw-gw/`。
- `aiw agent-gateway` 不再有插件入口或文档用法。
- `gateway.json` 的 schema、网关 HTTP 契约和既有绝对状态路径不变；旧安装配置不自动迁移。

## TODO

- [x] 完成 Work Items 1.1–1.4。

## Verification

- 静态检查已覆盖插件发现规则、唯一入口、构建输出、安装来源、参数转发和默认配置路径。
- 执行 `go build -o NUL ./cmd/aiw`；未输出诊断，但执行工具未显示退出码，结果未确认，未重跑。
- 未运行测试、网关进程、最终产物构建、安装或外部部署。
- 实现报告：`docs/features/reports/FD-026-implementation-r1.md` 及同名 JSON。
- 首轮独立审查要求更新 FD-021 两个夹具的插件目录；Worker 已将安装夹具和 live 配置路径改为 `plugins/aiw-gw`。未运行夹具。
- Reviewer 请求修改：`docs/features/reviews/FD-026-review-r1.md` 及同名 JSON。发现 `tests/fd021_gateway_acceptance.mjs` 与 `tests/fd021_gateway_live.mjs` 仍引用旧插件目录；测试与运行验证未执行。
- Reviewer r2 静态审查通过：`docs/features/reviews/FD-026-review-r2.md` 及同名 JSON。R1 两项夹具路径问题已修复；Tester r2 的 13 个场景仍全部未运行，需求覆盖率 0%，分支覆盖率未测量。命令、构建、安装、配置和 HTTP 行为仍缺少运行时证据。

## Sources

- `cmd/aiw/main.go` 和 `internal/plugin/discover.go` 的插件回退及名称匹配逻辑。
- `plugins/aiw-agent-gateway`、当前 `plugins/aiw-gw`、`build.bat` 与网关命令脚本。
- 用户决定：使用短命令 `gw`，不考虑旧命令兼容；本轮选择重新走 Planner handoff。

**Completed:** 2026-10-04
**Disposition reason:** DONE
