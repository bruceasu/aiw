# FD-025: Gateway graceful shutdown and stop script

**Status:** Complete
**Revision:** 13
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

用户已在 Windows 启动真实 Gateway，但没有可靠的停止命令。现有服务只能接收控制台信号，`scripts/start-gateway.bat` 还拼错了命令名。用户明确要求友好关闭和 `scripts/stop-gateway.bat`，允许使用 `build.bat bin` 安装及启动脚本启动。

## Options and decision

强制结束进程会绕过既有收敛与存储解锁；模拟 Windows 控制台信号可能影响同控制台的其他进程；本地文件轮询控制会增加持久化协议。采用受认证且仅允许 loopback 来源的内部 POST 控制入口，由 `stop --config` 命令调用；复用已有取消、进程树清理、HTTP Shutdown 与状态锁释放流程。

## Solution

新增 `POST /internal/shutdown`，要求现有 enabled 主体 Bearer Key、loopback peer、无 query、空 body、POST 方法。用户请求授权本次新增本地控制接口和关停连接；不扩展 OpenAI API 或认证凭据格式。所有 enabled 主体的 Key 均可用于本地控制，不提供远程关闭。停止处理即使审计存储不可写也应尽力取消服务，错误仅脱敏记录；不启动模型、不扣模型额度。

CLI 新增 `stop --config <file>`；仅在 loopback 或通配监听上发送本地请求，明确拒绝仅绑定远程网卡地址的配置。认证从配置读入内存，不输出 Key、原始响应或完整错误。接收 HTTP 202 后等待监听关闭及配置状态目录的 `gateway.lock` 释放（最多 15 秒）；不强杀、不删除残留锁，关停失败明确返回非零。保持原有 Ctrl+C/SIGTERM 入口。

修正启动脚本并新增停止脚本，两者转发参数及退出码。`build.bat bin` 增加编译和定向安装 Gateway 二进制与 Python 插件入口，保持现有 Windows/Linux AIW 构建；不覆盖 `gateway.json`。构建关闭依赖网络下载，用户授权最终安装产物。旧版本没有控制入口，不能用新 stop 假称已优雅结束旧进程；本轮检测监听为空，可以直接启动新版本。

## Scope

仅 Gateway 生命周期、两份批处理脚本、定向构建安装和相关文档/稳定规范。保留 Responses/Models/Usage 契约和现有状态格式；不下载依赖、不新增 Go 测试、不做 Git 写操作，不再调用真实模型。FD-021 客户端配置模式已实现并交独立 Reviewer；本 FD 不继续修改该客户端。

## Work items

- [x] 1.1 实现认证 loopback 控制入口。规模小、难度中、无依赖；完成条件：无效 Key、非 POST、query/body 拒绝；有效请求复用 cancel，保留关停审计与新 AI 请求拒绝。
- [x] 1.2 实现 stop CLI。规模小、难度中、依赖 1.1；完成条件：配置安全读取、无重定向/重试、有限等待、锁保留失败明确报告，不暴露凭据。
- [x] 1.3 提供启停脚本和定向安装。规模小、难度低、依赖 1.2；完成条件：`build.bat bin` 安装新 Gateway、不覆盖配置，脚本支持转发 `--config` 和退出码。
- [x] 1.4 文档及无产物编译。规模小、难度低、依赖 1.1–1.3；完成条件：记录本地权限、等待限制与异常恢复，Windows/Linux amd64 离线编译。
- [-] 1.5 独立聚焦运行与评审：取消作为 Worker Work Item 的安排，因为这些是后续独立 Tester/Reviewer 阶段，不能在 Worker 交接前假称完成；原验收仍保留于 Acceptance、TODO 和 Verification，活动模型取消只静态追踪。
- [x] 1.5a 独立验证交接材料准备。规模小、难度低、依赖 1.4；完成条件：交付已安装实现、报告与可观测测试契约（认证保护、正常/重复 stop、锁释放、零模型执行），后续 Tester 精确命令由 Planner 单独授权。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

默认启动与 Ctrl+C 保持兼容；有效本地 stop 返回成功且服务监听结束、锁释放；认证/方法/输入错误不关闭服务；停止失败不自动删除锁或强杀进程；安装不覆盖现有配置；Windows 脚本可从任意工作目录使用。仅 loopback 控制，shared 绑定具体非 loopback 地址时仍使用控制台信号。旧版本 404 给出明确升级说明。

## TODO

- R2 正常 stop 返回非零和 R3 默认配置入口问题均已修复；独立 R4 聚焦测试 12 项通过，R1 独立 Reviewer 未发现实质问题。历史失败与未运行报告保留，当前结论见 `reviews/FD-025-review-r1.md`。
- PM R3 明确接受需求覆盖 37.5% 与分支未测量的验证例外；20 项未运行仍保留为风险，不作为运行通过。
- 活动模型进程关停尚无本轮运行证据；需静态追踪已有 cancellation/Job Object/进程组流程。

## Verification

- 独立 Reviewer 已 claim `FD-025-000012-test-accepted`，会话 `fd025-reviewer-20261004-fd021_reviewer`，与 Worker/Tester 分离；报告 `reviews/FD-025-review-r1.md` 及同名 JSON 结论为验证通过、无实质发现。
- 复核 R4 raw、精确命令版本绑定授权、三文件摘要及 PM R3 例外；真实运行 12 passed / 0 failed / 20 unrun，需求覆盖 37.5%，分支未测量。静态追踪认证/loopback、取消/进程树、HTTP 与周期清理后解锁、失败保锁、无强杀/删锁/重定向/重试及安装配置保护；Reviewer 未追加运行验证。

- 保留 `reports/FD-025-test-report-r2.md` 与 `reports/FD-025-test-decision-r1.md`：6 项通过、1 项失败；Windows Winsock 10061 与 syscall 应用错误码不同，已按平台拆分识别，Linux 判定不变。修复后仅允许同一范围的一次针对性验证，不扩大范围、不调用模型。
- 修复后无产物编译与用户授权 `build.bat bin` 安装均退出 0，详见 `reports/FD-025-implementation-r2.md`；独立测试/PM/Reviewer 的最终结论记录于后续版本化报告及正式事件。
- 默认 stop 入口修正后 Python 内存 compile 和用户授权 `build.bat bin` 安装退出 0，见 `reports/FD-025-implementation-r3.md`。独立运行计划增加临时默认配置入口场景，不接触真实配置或真实模型。

- 用户已授权 `build.bat bin` 安装和 `scripts/start-gateway.bat` 启动；停止脚本是明确请求的配套操作。先读被调用代码再运行，关闭依赖下载；不触发模型。
- 默认只执行 `python program/agent-gateway/scripts/compile.py` 做 Windows/Linux 无产物编译。
- 独立 Tester 的精确聚焦启停验证命令需 Planner 审阅并记录版本绑定授权，再执行一次；不得自动扩大或重复未变更失败命令。
- 已执行 `python program/agent-gateway/scripts/compile.py`，退出 0，Windows/Linux amd64 输出 NUL，无下载或构建产物。
- 按用户明确授权执行 `cmd.exe /d /c build.bat bin`，退出 0，AIW 与 Gateway Windows/Linux 二进制定向安装到 `C:\green\aiw`，不复制或覆盖 `gateway.json`。此次最终构建为用户提供的明确例外，未运行 build all/plugins 等更大范围操作。

## Sources

- Issue: none

- 用户本轮明确请求及安装/启动授权。
- `openspec/specs/agent-proxy/spec.md` 的生命周期、审计与状态锁要求。
- `program/agent-gateway/main.go`、`server.go`、`runner.go` 和 Windows/Linux 进程清理。
- `scripts/start-gateway.bat`、`build.bat`、插件入口。

**Completed:** 2026-10-04
**Disposition reason:** done
