# FD-049: Agent Gateway 优先支持 Chat Completions 文本接口

**Status:** Complete
**Revision:** 8
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户要求优先实现 `/v1/chat/completions`。`aiw say` 已使用该接口，发送 `model`、system/user 文本 `messages` 和 `stream:false`，并读取单个 `choices[0].message.content`、`finish_reason:stop`。现有 Codex Gateway 仅提供 Responses，导致 Say 无法接入。

本 FD 从 FD-048 的 1.9 拆出，作为独立首批交付。FD-048 的真实上游代理、互斥模式及统计/计费预留继续保留为后续设计；本 FD 不声称这些能力已经实现。

## Options and decision

1. 修改 Say 使用 Responses：能够接入现状，但用户已指定先提供 Chat Completions。
2. 独立实现另一套 Codex 执行链：会重复认证、额度、取消、审计和清理逻辑。
3. 将 Chat Completions 文本请求映射为内部 Request，并共享现有 Responses 执行链：采用此方案；按接口选择最终响应对象，保持 Responses 的非流式/SSE 行为兼容。

## Solution

新增 `POST /v1/chat/completions`，支持本期明确列出的非流式文本子集：

- 顶层仅接受 `model`、`messages`、`stream`、`n`。`model` 必填，`messages` 为非空数组；`stream` 省略或 false，`n` 省略或 1。未知/重复字段、null、尾随 JSON、非法 UTF-8、超限主体与不支持选项在执行前拒绝。
- 消息仅接受 `role` 与字符串 `content`；角色支持 system、developer、user、assistant。system/developer 仅允许在对话开头，按顺序合并为 instructions；user/assistant 按顺序映射为内部消息。消息内容非空白，至少有一条 user 消息。不支持图片/音频、content 数组、tool 消息、function/tools、流式和生成参数控制。
- 主体上限沿用 1 MiB；映射后的指令和对话总量沿用 64 KiB UTF-8。共享内部 `decodeRequest` 的边界校验；role 和内容不由拼接重新解释为控制字段。
- 复用 Gateway Key/主体认证、模型授权、RPM、并发、日额度、timeout、客户端取消、App Server preflight、独立 thread/cwd 和进程树清理，不新增配置或依赖。
- 成功返回 `object:chat.completion`、稳定 `chatcmpl_` ID、created、请求逻辑 model，以及一个 index=0、role=assistant、字符串 content、finish_reason=stop 的 choice。内容只来自成功执行的最终文本；错误保留现有脱敏 envelope/HTTP 状态，不伪装 stop。
- 已知 usage 映射为 prompt_tokens/completion_tokens/total_tokens；缺失保持 `usage:null`，不补零。请求审计 ID 与新响应 ID 可通过同一随机后缀关联；执行/额度只发生一次。
- 扩展 HTTP 路由观测为 `/v1/chat/completions`，沿用现有记录结构。新增 route 枚举值意味着旧二进制可能不识别新记录；禁止将这一点描述为可直接降级，文档必须注明此限制。本 FD 不改变元数据字段或存储 schema 版本。

## Scope

仅实施 Codex 模式的非流式 Chat Completions 文本子集及相关 spec/文档/证据。Say 客户端代码保持现状，使用其已有 `OPENAI_BASE_URL`、`OPENAI_API_KEY` 和模型设置接入。FD-048 的代理模式、WebSocket、统计/计费实现和新 SDK 依赖不属于本次交付。

## Work items

- [x] 1.1 实现严格 Chat Completions 文本请求映射及响应对象。验收：支持的角色/字段/大小明确，失败在执行前拒绝，输出/usage 与 Say 读取方式匹配。大小：小；难度：中；依赖：无。
- [x] 1.2 接入路由并共享现有授权与推理执行链。验收：不重复扣额/执行；错误、取消与 Responses 行为一致；Responses SSE 路径保持兼容；新路由观测可识别。大小：小；难度：中；依赖：1.1。
- [x] 1.3 更新稳定 spec、Gateway/Say 接入文档、TODO/Verification 和 Dual 报告。验收：支持矩阵、旧二进制读取新 route 的降级限制、compile-only 与未运行验证一致。大小：小；难度：低；依赖：1.1、1.2。

## Acceptance

1. Say 当前 model/system/user/stream:false 请求可以被 Gateway 接受，成功时返回一个完整文本 choice 和 stop。
2. 不支持字段/角色/content/stream/n、未知模型和非法 JSON 在 App Server 执行前拒绝；方法、查询、媒体类型、大小错误沿用现有响应策略。
3. Chat Completions 与 Responses 使用同一认证、额度、执行/取消/清理路径，不另起执行、不重试、不重复计数。
4. 已知 usage 正确重命名，未知 usage 为 null；异常结果返回错误，不伪称成功。
5. HTTP 观测包含新 route；现有数据由新版本继续读取，直接降级旧版本的 route 兼容限制明确记录。
6. 本次交付不改变现有 Responses 成功对象、错误及 SSE 事件契约，不要求新增统计/计费功能。

## Verification

- 独立 Reviewer r2 已通过静态验收：会话 `fd049-reviewer-20261010-a613`，source event `FD-049-000007-implementation-ready`，审查 HEAD `be852f2`；R1 精确字段白名单已解决，无剩余阻断。Dual 报告：`docs/features/reviews/FD-049-review-r2.md`。未重跑编译、测试或服务请求，运行兼容性仍未验证。
- 独立 Reviewer r1（会话 `fd049-reviewer-20261010-a613`，source event `FD-049-000005-implementation-ready`）已静态审查 `f653509..e72fc27`，请求修改 R1：Chat 字段需精确白名单，拒绝大小写变体与逻辑重复字段。Dual 报告：`docs/features/reviews/FD-049-review-r1.md`；未运行任何测试或编译。
- 静态依据已检查：`internal/say/openai.go` 的真实请求/响应读取，`server.go` 的策略/执行/SSE 路径，`protocol.go` 的严格 JSON/输入大小限制，`app_runner.go` 的 quota/取消/结果，`observation.go` 的 route 枚举和载入校验。
- 初次实施运行一次 `python -B scripts/compile.py`（cwd=`program/agent-gateway`）；R1 源码修复后按预算重跑同一命令一次。已检查脚本：Windows/Linux amd64 compile-only，GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local，输出到系统空设备，不保留发布产物。
- Worker 已在专用 worktree 运行上述命令两次，退出码均为 0；R1 修复后的 Windows/Linux amd64 compile-only 通过。未执行真实 App Server、Say 或 HTTP 请求，接口运行行为尚未验证。
- 实现证据：`chat.go` 的严格请求/文本映射和 completion/usage 对象，`server.go` 的共享 inference 调用链，`observation.go` 的新增 route；初版报告为 `docs/features/reports/FD-049-implementation-r1.md`，R1 修复后的当前 Dual 报告为 `docs/features/reports/FD-049-implementation-r2.md`。
- Worker 已修复 R1：对顶层及各消息对象的原始 JSON 键名进行精确白名单校验，大小写别名及其覆盖组合在执行前拒绝；Responses 解码契约未修改。修复已由独立 Reviewer r2 静态复核通过。
- 不新增/运行测试、完整构建、SDK/App Server 或实际翻译请求；运行兼容性尚未验证。独立 Reviewer 按本 FD 的静态验收范围审查。
- 用户已于 2026-10-10 明确授权分别提交 FD-048 设计与本 FD 计划，创建 `feature/FD-049`、`.wt/FD-049`，完成本地提交、独立审查通过后的合并与归档；不包含测试、下载、推送或部署。实施分支/worktree 与记录一致，父分支为 develop。

## TODO

- PM 交付记录：已核对 revision 8 的 Reviewer receipt 与 FD 内容摘要一致，父分支干净且匹配登记基点；`aiw git wt local-merge FD-049` 已生成 develop squash 提交 `7a3ff43`，并清理 feature/FD-049 和 .wt/FD-049。随后 `aiw fd close FD-049 Complete` 成功归档。共两次独立审查；未构建/安装发布二进制或运行真实请求。

- [x] 收敛 Say 所需接口范围，固定映射、授权/执行复用与验证计划。
- [x] 授权后提交准备文档，创建专用 worktree 并完成实现。
- [x] 更新实际 compile/static 证据及 Dual Worker 报告。
- [x] 独立 Reviewer 审查通过。
- [x] 审查通过后的本地交付/归档。

## Sources

- Issue: none

- 用户当前会话“优先实现 /v1/chat/completions 接口”的指示。
- `docs/features/FD-048_AGENT_GATEWAY_OPENAI_API_CODEX.md`：完整代理模式设计及原 1.9。
- `internal/say/openai.go`
- `program/agent-gateway/server.go`、`protocol.go`、`app_runner.go`、`observation.go`、`scripts/compile.py`
- `openspec/specs/agent-proxy/spec.md`

**Completed:** 2026-10-09
