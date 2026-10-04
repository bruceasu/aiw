# FD-018: Agent Gateway plugin entry

**Status:** Complete
**Revision:** 7
**Priority:** Medium
**Test policy:** External
**Evidence policy:** Dual

## Problem

用户需要通过 `aiw agent-gateway` 调用预先编译的网关程序。

## Options and decision

沿用现有 Python 插件入口机制，无需修改 Go 插件发现器或添加依赖。
相比平台专用脚本，维护一份入口并复用现有 Python 运行时。

## Solution

新增 `plugins/aiw-agent-gateway/aiw-agent-gateway.py`，定位同目录中的
Windows `agent-gateway.exe` 或 Linux `agent-gateway`，原样转交参数。
未指定配置时，入口传入程序同目录 `gateway.json` 的绝对路径；显式配置优先。
Linux 使用 exec 替换进程，Windows 等待子进程并返回其退出码。

## Scope

仅新增入口、说明和产物忽略规则。无网关 API、持久化或依赖变化。
产物来源为 `program/agent-gateway`，本轮读取确认该目录已存在。

## Work items

- [x] 1.1 新增跨平台入口；规模小、难度低，无依赖。完成标准：同目录定位、参数和标准流传递、明确失败退出。
- [x] 1.2 新增说明与产物忽略规则；规模小、难度低，依赖 1.1。完成标准：文件名、运行时和路径差异可查。
- [x] 1.3 增加默认配置路径；规模小、难度低，依赖 1.1。完成标准：未指定配置使用同目录 gateway.json，显式配置优先，保持 start 在首位。
- [x] 1.4 修正网关构建目标；规模小、难度低，依赖 1.1。完成标准：在独立模块 program/agent-gateway 编译并输出正确文件，plugins 安装包含网关构建，错误路径恢复工作目录。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 插件发现器可发现一层子目录中的 Python 入口。
- Windows/Linux 调用同目录指定文件，参数不经 shell 拼接。
- 缺失或启动失败写标准错误，返回非零；Windows 保留子进程退出码。
- 不自动编译或下载程序。
- 默认配置不依赖调用者工作目录；支持显式 `--config` / `-config` 和等号写法。

## TODO

- 独立静态审查已完成，见 `docs/features/reviews/FD-018-review-r6.md` 及同名 JSON；实际启动仍由外部安排，不宣称运行验证通过。
- 运营者需按现行稳定规格配置 `principals[].keys` 明文网关 Key；原 Key 哈希说明属于历史配置记录，不生成或披露凭据。

## Verification

- 2026-10-04 独立 Reviewer `codex-reviewer-FD018-20261004-r6` 已领取
  `FD-018-000006-review-requested`，静态审查工作项 1.1–1.4 通过；报告为
  `docs/features/reviews/FD-018-review-r6.md` 及同名 JSON，审查提交为
  `c9b6d51cd2308e1ffbaf51755d697f6acda3ad83`。本轮未运行编译、测试、最终构建或网络。
  External 实际启动、跨平台标准流/退出码和 Windows 信号行为仍未运行验证；
  现有二进制可能保留历史错误构建，本轮未重建或替换，部署更新由运营者另行安排。
  无历史 Worker 报告，不补造完成事件；状态恢复记录与真实当前代码作为独立审查依据。
- 2026-10-04 用户一次性授权处理本 FD 状态。修订 5 的工作项已全部勾选，
  但最新交接仍为修订 2 的待领取 Planner 请求 `FD-018-000002-design-requested`。
  本次仅恢复为 Pending Verification，并通过 `aiw fd request-review` 创建当前
  Reviewer 请求，由 CLI 取消并保留过期交接；不补造 Worker 完成事件。
- 以下编译结果是此前记录，本次状态修复未重复执行。当前代码以独立静态审查为准，
  测试策略保持 External，未授权实际启动、测试、最终构建、网络或部署。

- 本轮静态确认 Windows 默认配置使用 Linux 示例路径。已将工作区和
  `C:/green/aiw` 安装目录下 gateway.json 的路径字段改为本机绝对路径，
  Codex 使用已存在的 node.exe + codex.js；其它字段保留，未启动服务。

- 本轮用户反馈 `req: unknown requirement command: --config`。静态读取三个二进制的
  `go version -m`，工作区 Windows/Linux 和安装目录 Windows 产物的 path 均为
  `aiw/cmd/aiw-req`。`build.bat` 网关命令错误地编译了需求 CLI，根因已确认。
- 修正为在独立网关模块目录执行 `go build .`；未执行 build.bat 或替换现有二进制。
- 本轮使用 `python program/agent-gateway/scripts/compile.py` 做 Windows/Linux amd64
  离线编译，退出码 0，不保留产物；已静态检查构建目标、输出路径和 popd 错误分支。

- 静态依据：`internal/plugin/discover.go` 支持一层子目录和 `.py`，
  `internal/plugin/exec.go` 支持 Python 入口并传递标准流和环境。
- 入口仅使用标准库；已运行 Python `compile()` 内存语法编译，通过，不保留字节码产物。
- 默认配置参数插入在可选 `start` 之后；已静态对照网关 `main.go` 的 Go flag 解析契约。
- 未运行网关、测试、最终产物构建或网络请求。
- 风险：Windows 入口额外等待进程，服务关停建议直接运行 exe；实际信号行为未测试。

## Sources

- Issue: none

用户本轮直接要求新增插件入口。参考 `openspec/specs/agent-proxy/spec.md`
和 `docs/features/archive/FD-017/FD-017_GO_SHARED_AI_GATEWAY.md`；网关行为不变。

**Completed:** 2026-10-04
