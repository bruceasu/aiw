# FD-105 独立静态审查（第 1 轮）

- 结论：需要修改（changes-requested）。
- Reviewer 会话：`fd105-reviewer-20261026-6ab3f5`。
- 来源事件：`FD-105-000004-implementation-ready`。
- 审查对象：FD revision 4，以及当前主工作区相对 HEAD 的实现与相关合同差异；未绑定实现提交。
- 审查范围：Self/Independent 默认行为、角色交接恢复、历史兼容及稳定规格和交付规则的一致性。

## 发现

### P1：刷新 Worker 交接后可丢失既有独立复审要求

位置：`src/plugins/aiw-fd.py:1895`，相关路径 `refresh_worker`（1370）、`recover_worker`（1433 附近）及 Self 归档检查（2193 附近）。

新保护只在最新事件恰好为 `changes-requested` 时阻止 Self implementation-ready。一次正常的过期交接恢复会把该最新事件换成 `work-requested`，从而绕开检查。

静态可达事件链：

1. Independent Reviewer 发出 `changes-requested`，生成待处理 Worker 交接，FD 为 In Progress。
2. FD 内容变动，同时 Review policy 被改成 Self；现有 Worker 交接的内容摘要过期。
3. 按正常恢复路径调用 `refresh-worker`，旧事件取消并被新的 `work-requested` 取代。
4. Worker 认领新事件并发出 `implementation-ready`。当前代码只看最新事件类型，因此按 Self 路由到 Complete/PM。
5. Self close 检查看到最新事件为发往 PM 的 `implementation-ready`，允许正常归档，期间没有任何独立复审通过事件。

这违反本 FD 验收及稳定规格中“已要求的独立审查和已有问题不得静默降级”的约定。需要把当前周期内已要求的独立审查约束沿正常 refresh/recover 链保留下来，并在 Self 路由和归档时应用；不能只检查最新事件的类型。结论来自代码分支追踪，未运行复现命令。

### P2：稳定规格和恢复说明残留无条件重流程要求

当前新默认规则与以下仍有效的 MUST 要求冲突：

- `openspec/specs/fd-workflow/spec.md:291` 的 Persistent automatic workflow blocker feedback 仍要求每个 Gate 单独生成 Markdown/JSON 反馈对（294 行起），与普通阻塞只记 FD 相冲突。
- 同一文件 316 行仍要求正常隔离生命周期保留独立实现和审查；334、354 行把恢复后的新独立审查写成无条件要求。
- 同一文件 371 行仍要求正常完整 FD 实现 MUST retain independent review。
- `src/skills/fd-workflow/references/delivery.md:80` 仍要求恢复后的正常交付必须有当前 independent pass，没有限定为 Independent FD。

执行者遵循稳定规格时仍会为普通 Self FD增加独立审查或双格式阻塞报告，无法可靠获得用户已确认的轻量默认路径。请把这些段落与新的条件审查、FD 内记录及 Self 恢复自查策略对齐，保留 Independent 路径的有效通过门槛。

## 已检查的证据与限制

- 阅读了当前 FD、fd-review 技能、共享 work-management 合同、实际 CLI 修改和上述稳定规格/交付说明。
- 静态追踪 new 默认创建、Self/Independent implementation-ready、直接 request-review、Self archive 后再请求 review、refresh-worker 以及 close 分支。
- 编译证据仅引用 Worker 的既有记录；Reviewer 未运行编译，未声称该记录覆盖后续改动。
- 实际命令：一次限定范围 `git diff -- ...`；必要的 `Get-Content`/`rg -n` 源文件读取与定位；认领本 Reviewer 事件和提交本次审查结果。
- 未运行测试、构建、quick_validate、其他验证脚本或网络；未改实现、提交 Git 或归档。
- 修复后仅需针对上述问题及其影响范围复查；无须重新执行整套审查。
