# FD-023: Agent Gateway and client configurable timeout

**Status:** Complete
**Revision:** 8
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

AI 请求可能持续数分钟。Gateway 原默认期限为 60 秒，`aiw ai` 固定等待 65 秒，可能过早取消请求。用户已批准延长默认期限并增加客户端参数。

## Options and decision

仅延长 Gateway 或客户端其中一端，另一端仍可能先到期。决定将 Gateway 默认期限改为 600 秒，客户端默认期限改为 660 秒，提供 `--timeout-seconds`。该参数只控制客户端，不发送给 Gateway。客户端使用 Node 原生 HTTP(S) 请求和 AbortSignal 控制请求及完整响应读取，避免 `fetch` 传输层的等待限制影响分钟级响应。

2026-10-04 用户澄清：本 FD 所称工作区配置为 `program/agent-gateway/gateway-example.json`。该示例已更新；安装目录中已有配置需运营者自行更新并重启。

## Solution

客户端参数只接受 1–3660 的十进制整数秒；上限覆盖 Gateway 现有 3600 秒配置上限并留 60 秒余量。缺值、重复和非法值在读取输入或发请求前以退出码 2 报错。保留认证、输入输出、有界响应、不重试、断连取消及 HTTP(S) 协议。默认等待变长可能延长并发占用；用户已批准该 CLI 契约变化。

## Scope

仅修改 `plugins/aiw-ai` 客户端、`program/agent-gateway` 默认值与 `gateway-example.json`、相关 README 和稳定规范。保留凭据与其它已有设置。不修改独立 `program/aiw-agent`，不新增后台任务，不构建部署或写入外部安装目录，不进行 Git 写操作。

## Work items

- [x] 1.1 Gateway 默认值及工作区 `program/agent-gateway/gateway-example.json` 改为 600 秒；规模小、难度低、无依赖；完成标准：保留现有 1–3600 秒校验和其它配置。
- [x] 1.2 客户端默认 660 秒并增加 `--timeout-seconds`；规模小、难度中、无依赖；完成标准：完整请求含响应读取受期限控制，非法或重复值退出码 2，保留 `--` 和其它参数行为。
- [x] 1.3 更新帮助、README 和稳定规范；规模小、难度低，依赖 1.1/1.2；完成标准：明确两端独立期限、配置重启生效、旧显式值仍优先，以及超时后不能恢复执行。
- [x] 1.4 记录静态证据、compile-only 和双文件实现报告；规模小、难度低，依赖 1.1–1.3；完成标准：诚实记录未运行测试、未部署及残余风险。

Keep item numbers stable after implementation starts. Use `- [-]` only for an explicitly cancelled item, with its reason on the same line.

## Acceptance

默认 Gateway 600 秒、客户端 660 秒；`aiw ai --timeout-seconds 300 "Explain this text"` 使用 300 秒客户端总期限。客户端期限不改变服务端执行上限，参数不得进入 Prompt 或 Responses JSON。非法、重复、缺值必须在请求前失败；认证、响应大小及输出契约不变。

## TODO

- [x] 完成上述实现和一次静态检查、compile-only；未运行测试、构建发布产物、部署或网络请求。
- [x] 独立 Tester 按两份 Planner 授权完成当前 10 个离线客户端场景；PM 接受覆盖率证据例外；独立 Reviewer 完成审查。

## Verification

- 已静态追踪参数解析、请求信号、响应读取及 Gateway 配置校验；检查了目标文件 diff。`--timeout-seconds` 仅用于创建 AbortSignal，未加入 Prompt/Responses JSON。
- `node --check plugins/aiw-ai/aiw-ai.mjs` 与 `python program/agent-gateway/scripts/compile.py` 已通过；仅编译，不执行应用或保留发布产物。
- 测试、长请求、服务断连、HTTP(S) 错误、真实模型调用及部署未运行验证。
- %% 工作区之外的安装配置与二进制尚未更新；延长默认期限后的资源占用需运营者观察。取消请求后网关实际清理效果仍待独立测试。
- 独立审查：`docs/features/reviews/FD-023-review-r1.md`，结果 `verification-passed`；来源 `FD-023-000007-test-accepted`。当前 10/10 客户端黑盒场景通过，22 个适用场景中覆盖 10 个；45.5% 需求场景覆盖率与未测分支覆盖率由 PM 接受为证据例外，未运行行为仍见报告的残余风险。

## Sources

- Issue: none
- 用户本轮澄清工作区配置指 `program/agent-gateway/gateway-example.json`。
- `openspec/specs/agent-proxy/spec.md`
- `openspec/specs/agent-proxy-client/spec.md`
- `docs/features/archive/FD-019/FD-019_AI_CLIENT_HELP_AND_DEFAULT_MODEL.md`

**Completed:** 2026-10-04
**Disposition reason:** DONE
