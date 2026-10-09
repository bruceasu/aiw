# FD-045 自动流程阻塞反馈：协议证据 Gate

<!-- aiw-data: FD-045-blocker-20261009T234346+0900-protocol-gate.json -->

## 定位

- FD：`FD-045`
- 阶段与角色：实施前设计核对 / PM
- 关联交接事件：`FD-045-000002-design-requested`（该 Planner handoff 已取消；当前无 pending handoff）
- 记录时间（含时区）：2026-10-09 23:43:46 Asia/Tokyo
- 记录者会话：`fd045-pm-20261009-234346-c19a`

## 阻塞事实

- 观察到的表现及停止位置：FD 状态为 `Design`，工作区在 `develop` 且干净；最新 Planner handoff 因 FD revision 不匹配已由先前 PM 取消。FD 的 Work Item 1.1–1.2 和验收依赖目标版本的 App Server 握手、`turn/start`、输出 schema、item/event、usage、interrupt、错误及 rollout 正文留存契约。当前没有可核对这些字段的本地一手资料。
- 已确认的原因：仓库资源预算将网络调用设为 0；FD 也要求对 SDK/backend 的运行验证先取得明确授权。`auto` 未授予网络或运行验证授权，不能据此补齐协议事实或将设计标为 ready。
- 已尝试的恢复及结果：检查 FD-045、`agent-proxy` 稳定规格、Gateway 当前实现路径、FD-045 现存收据与本地 workflow 规则；发现 FD 明确把协议和留存证据列为实现前置条件。没有联网、执行 SDK/App Server、测试或编译；未创建 handoff 或 worktree。
- 当前状态：已解决（官方来源证据 Gate）；运行时 Gate 仍未授权。
- 是否需要人工决策：否（本次范围）。用户已确认只查阅官方来源。后续如需执行 SDK/App Server 验证，仍须在给出确切命令、范围、预计时长和风险后另行决定。

## 结果与改进

- 实际解决方案：用户确认只查阅官方来源后，核对 OpenAI Codex `rust-v0.160.1` 和 OpenAI Agents SDK `v0.18.0` 的版本化协议源码。证据确认 `initialize` 实验能力协商、`thread/start.ephemeral`、实验性 `dynamicTools`、服务端 `item/tool/call` 请求、`turn/start` 的输入/工具输出/输出 schema、turn 用量和 `turn/interrupt` 字段。来源包括：
  - [Codex App Server thread protocol at rust-v0.160.1](https://github.com/openai/codex/blob/rust-v0.160.1/codex-rs/app-server-protocol/src/protocol/v2/thread.rs)
  - [Codex App Server turn protocol at rust-v0.160.1](https://github.com/openai/codex/blob/rust-v0.160.1/codex-rs/app-server-protocol/src/protocol/v2/turn.rs)
  - [Codex App Server RPC methods at rust-v0.160.1](https://github.com/openai/codex/blob/rust-v0.160.1/codex-rs/app-server-protocol/src/protocol/common.rs)
  - [Codex feature flags at rust-v0.160.1](https://github.com/openai/codex/blob/rust-v0.160.1/codex-rs/features/src/lib.rs)
  - [Agents SDK Responses model at v0.18.0](https://github.com/openai/openai-agents-js/blob/v0.18.0/packages/agents-openai/src/openaiResponsesModel.ts)
- 解决时间（含时区）：2026-10-09 23:51:34 Asia/Tokyo。
- 剩余风险或下一步：这是静态源码证据，不是 SDK/App Server 运行证据。`dynamicTools` 在该 Codex 版本中标记为 experimental；设计须明确启用该 capability、将外部函数调用映射到待处理的 `item/tool/call` 请求，并限制等待时长及线程/槽位寿命。使用 ephemeral thread 可避免将该 thread 保存为可恢复会话，但仍须隔离 Codex home 并清理临时文件。运行时兼容性验证未执行，也未获授权；实现按仓库规则仅做静态审查和 compile-only。
- 可复用的流程改进建议：none（当前证据只表明本 FD 的协议前置条件尚未验证，未确认通用流程缺陷）。
- 建议处理状态：可继续 Planner 设计；运行时验证仍等待单独授权
