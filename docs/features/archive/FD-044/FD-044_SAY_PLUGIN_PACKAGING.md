# FD-044: 补齐 AIW Say 插件构建与安装

**Status:** Complete
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

Say CLI 已由 FD-036 实现，但 `plugins/` 无 Say 发行目录，`build.bat plugins`
只构建 req、gateway、cz，导致 `all` 安装也遗漏 Say。当前环境找不到
`aiw-say`，插件分发已有对应发现机制。

## Options and decision

选择复用现有 Go 插件构建与 `plugins` 复制流程，不修改命令分发。
用户已明确要求新增 `plugins/aiw-say/` 并补充 `build.bat` 处理。

## Solution

- 新增 `plugins/aiw-say/README.md`，说明生成物和安装方式。
- 新增 `build.bat say`，生成该目录下的 `aiw-say.exe`（Windows amd64）和
  `aiw-say`（Linux amd64），沿用版本注入、离线 Go 设置与错误退出约定。
- 构建阶段复制 `program/aiw-say/aiw.toml.example` 和四个 profile 样例；
  不生成实际 `aiw.toml`，不修改用户 profile。
- 在 `build_plugins` 调用 Say 构建，专用复制步骤排除实际 `aiw.toml` 并安装到
  `%INSTALL_DIR%/plugins/aiw-say/`；`all` 通过该入口包含 Say。
  `say` 单独调用仅构建，保持与 `req` 一致的约定。
- `.gitignore` 忽略 Linux 无扩展名生成物；Windows exe 已全局忽略。
- 更新构建帮助、根 README 与 Say 手册。默认配置位于 Say 可执行文件同目录，
  并非自动读取主程序目录；其他位置使用 `--config`。

## Scope

仅包含 Say 发行目录、构建安装接入、生成物忽略规则及对应文档。
不修改翻译逻辑、插件分发、依赖、用户配置、GUI 或已有 `bin` 行为。
脚本编辑已获用户请求；执行最终构建、安装或翻译尚未获授权。

## Work items

- [x] 1.1 新增 Say 发行目录说明与生成物忽略规则。规模：小；依赖：无；证据：目录 README 与 .gitignore 差异。
- [x] 1.2 新增双平台 Say 构建入口、样例复制、plugins 接入和帮助。规模：小；依赖：1.1；证据：批处理调用及失败返回路径静态核对。
- [x] 1.3 同步文档与配置路径，静态核对分发/构建/安装链路，记录实际验证。规模：小；依赖：1.2；证据：实现报告与 Say 编译结果。

## Acceptance

1. `say` 构建入口声明两个平台产物，任一构建失败返回非零并清理临时平台变量。
2. `plugins` 和 `all` 包含 Say，安装目录符合现有插件发现路径。
3. 样例不替换实际 `aiw.toml` 或用户 profile，生成二进制不进入版本控制。
4. 文档明确独立构建与完整安装的区别以及默认配置查找位置。

## Verification

- 已静态检查 `cmd/aiw/main.go`、`internal/plugin/discover.go`、
  `internal/say/config.go`、`build.bat` 和 FD-036。
- 实施完成；`git diff` 已静态核对脚本入口、双平台构建、setlocal 失败退出、
  样例复制与专用安装排除 `aiw.toml` 的路径。
- 离线执行 `go build -o NUL ./cmd/aiw-say`，退出码 0，未保留发行二进制。
  仓库 `scripts/compile.py` 只包含 aiw 和 aiw-req，因此使用 Say 窄范围编译。
- 未运行测试、批处理最终构建、安装或翻译请求；Windows 批处理执行及 Linux
  交叉编译未作运行验证。实现报告：`reports/FD-044-implementation-r1.md`。
- 用户已明确授权本次直接在主工作区修改，不进行 Git 写操作；这是专用工作树规则的显式例外。

## Sources

- 本会话用户请求：补充 `plugins/aiw-say/`，并在 `build.bat` 增加处理。
- `docs/features/archive/FD-036/FD-036_AIW_SAY_CLI.md`
- `openspec/specs/cli-and-plugins/spec.md`
- `program/aiw-say/README.md`

**Completed:** 2026-10-09
**Disposition reason:** User confirms Say plugin is complete and runtime validated; forced archive requested.
