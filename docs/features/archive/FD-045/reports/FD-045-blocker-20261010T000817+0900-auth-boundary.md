# FD-045 自动流程阻塞反馈：App Server 认证与配置边界

<!-- aiw-data: FD-045-blocker-20261010T000817+0900-auth-boundary.json -->

## 定位

- FD：`FD-045`
- 阶段与角色：设计决策 / Planner
- 关联交接事件：`FD-045-000003-design-requested`（已由当前 Planner claim）
- 记录时间（含时区）：2026-10-10 00:08:17 Asia/Tokyo
- 记录者／会话：`fd045-planner-20261010-000600-5d24`

## 阻塞事实

- 观察到的表现及停止位置：App Server 迁移需要确定 Codex 认证材料、`CODEX_HOME` 配置及 MCP/插件禁用边界。当前 Gateway 的 `childEnv()` 会传递 `CODEX_HOME`、HOME/USERPROFILE；已查阅的 Codex feature flags 说明多个本地功能默认启用，而旧 Runner 只覆盖部分 feature flags。FD 已加入 `%% NEEDS_INPUT`，尚未发出 `design-ready`。
- 已确认的根因：现有 FD 和代码未指定 App Server 是否复用用户 Codex home，亦没有证明不同隔离策略如何保留认证并关闭所有外部工具配置。该选择影响凭据副本、配置继承、MCP/插件权限和清理责任。
- 已尝试的恢复及结果：读取 Agent Gateway `childEnv`、Runner 和 README，以及 Codex `rust-v0.160.1` 的 thread、turn、RPC 和 feature flag 官方源码；确认 App Server 支持 ephemeral thread、turn output schema，且 `dynamicTools` 是实验性字段。没有改动认证实现，没有复制凭据，没有执行 App Server。
- 当前状态：已解决。
- 是否需要人工决策：否。用户选择复用现有 `CODEX_HOME`，不复制认证材料。

## 结果与改进

- 实际解决方案：用户选择复用现有 `CODEX_HOME` 与登录态。FD 已记录不复制凭据；通过显式关闭 Codex 内置工具、apps、plugins、hooks、web、子 agent，并使用 `config/read` 与 `mcpServerStatus/list` 做启动期检查；若有效配置存在启用或状态不明的 MCP server，拒绝启动 turn 并关闭进程槽，不修改用户配置文件。
- 解决时间（含时区）：2026-10-10 00:11:40 Asia/Tokyo。
- 剩余风险或下一步：此安全 preflight 尚未实现或对目标二进制运行验证。实现必须在 turn/start 前失败关闭；若无法静态确认有效配置状态的 wire shape，应停止并记录新 Gate。运行时 SDK/App Server 验证仍需独立授权。
- 可复用的流程改进建议：none（当前为 FD-045 特有的 App Server 认证隔离决策，尚无通用 workflow 问题证据）。
- 建议处理状态：用户决策已记录，Planner 更新设计后可继续；运行时验证依旧未授权。
