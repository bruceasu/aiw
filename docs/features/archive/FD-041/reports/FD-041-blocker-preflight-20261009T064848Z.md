# FD-041 自动流程阻塞记录：Preflight

<!-- aiw-data: FD-041-blocker-preflight-20261009T064848Z.json -->

## 已观察事实

- FD：`FD-041`
- 阶段与角色：Preflight / PM
- 关联事件：无（`source_event: null`）。`.ai/fd/` 下没有 `FD-041` 目录或事件文件。
- 当前 FD 状态为 `Open`，修订版为 2；工作区在 `office-dev`，开始检查时干净。
- `aiw fd resume FD-041` 返回：`FD has no event; use emit for a deliberate handoff`。
- `aiw fd show FD-041` 因当前控制台 `cp932` 无法编码 FD 中文标题而失败。

## 阻塞原因

已确认当前共享运行态中没有 FD-041 的 handoff receipt；缺失原因未知。FD 正文只保留旧 Planner handoff 的文字来源，不能替代 Worker handoff receipt。`emit` 支持的事件类型也没有可用于当前 `Open` 状态的 Worker 请求事件，因此不能安全认领或重建原 handoff。

## 已尝试恢复

- 检查 `.ai/fd/`：目录中存在 FD-038、FD-037 等收据目录，没有 FD-041。
- 运行 `aiw fd resume FD-041`：确认没有事件。
- 查看 `aiw fd emit --help`：可用事件类型不包含当前状态所需的 Worker 请求；未伪造事件。
- 查看 `aiw fd show FD-041`：受 cp932 输出编码问题阻断，未能读取其收据摘要。

## 人工决定与状态

- 用户决定：用户要求重建 FD-041 的 handoff。
- PM 处理：不伪造旧 receipt；通过受支持的 `design-requested` → Planner `design-ready` 流程重新审阅当前设计，由 AIW 创建新的真实 Worker handoff。
- 恢复证据：`FD-041-000003-design-requested` 已由 Planner session `fd041-planner-20261009-70ab91` 认领并完成；AIW 创建了新的 Worker handoff `FD-041-000004-design-ready`，当前为 pending。
- 状态：已解决。旧 receipt 的缺失原因仍未知；新 handoff 使用新的事件 ID，没有声称恢复原事件。
- 后续流程：Worker 必须认领 `FD-041-000004-design-ready`，然后在 FD 专属 worktree 实现。
- 风险：重走 Planner 阶段会递增 FD 修订版，但现有范围和 Work Item 保持不变。
- 可复用改进：跨分支交付 FD 文件时，应确认运行态 handoff receipts 对该 FD 可用；`aiw fd show` 的 cp932 中文输出问题也应单独记录并修复。
