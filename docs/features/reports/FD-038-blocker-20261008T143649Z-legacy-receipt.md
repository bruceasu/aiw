# FD-038 自动流程阻塞反馈

<!-- aiw-data: FD-038-blocker-20261008T143649Z-legacy-receipt.json -->

## 定位

- FD：`FD-038`
- 阶段与角色：preflight；PM/Planner
- 关联交接事件：`FD-038-000002-design-requested`
- 记录时间（含时区）：2026-10-08T14:36:49+00:00
- 记录者／会话：当前主持 Agent；`fd038-pm-20261008-7b2f6a`

## 阻塞事实

- 观察到的表现及停止位置：FD-038 的待处理 Planner 收据存在于 `.ai/fd/FD-038/000002-design-requested.json`，其状态为 `pending`，但当前 CLI 的事件读取路径只枚举 `.ai/fd/FD-038/events/*.json`。该目录不存在。运行 `aiw fd resume FD-038` 返回“FD has no event; use emit for a deliberate handoff”，因此无法按规定领取收据；流程停在 Planner 领取之前，尚未修改 FD 或创建 worktree。
- 已确认的根因：当前 `plugins/aiw-fd.py` 的 `event_paths` 仅扫描 `events/` 子目录，而 FD-038 收据保留在旧的根目录布局。该收据为何仍处于旧布局尚不明确。
- 已尝试的恢复及结果：读取收据确认事件 ID、角色、状态、Revision 和摘要；运行 `aiw fd show FD-038` 与 `aiw fd resume FD-038`；检查当前 CLI 的 `runtime_dir`、`event_paths`、`latest_event` 和 `claim` 实现；查看 `aiw fd --help` 并比较其他 FD 的收据布局。确认没有可见的 CLI 迁移命令。最初未尝试 claim，也未移动或改写旧收据。用户随后选择由 PM 新发交接并保留旧收据。
- 当前状态：已解决。原始阻塞记录于 2026-10-08T14:36:49+00:00 时未解决；以下恢复在 2026-10-08T14:38:54+00:00 完成。
- 是否需要人工决策：否。用户已决定由 PM 新发 design-requested 事件，保留旧收据为历史记录。

## 结果与改进

- 实际解决方案：用户选择保留旧收据并由 PM 新发交接。执行 `aiw fd emit FD-038 design-requested --producer pm --artifact docs/features/FD-038_IMPROVE_FD_REPORT_REVIEW_AND_STATUS_INSPECTION.md`，CLI 创建待处理事件 `FD-038-000003-design-requested`（Revision 3，target role 为 planner）；原收据 `FD-038-000002-design-requested` 仍保留在根目录，没有手动迁移或修改。
- 解决时间（含时区）：2026-10-08T14:38:54+00:00
- 剩余风险或下一步：新交接可由当前 CLI 读取；下一步核对最新事件并由 Planner 领取。旧收据继续留在 CLI 不扫描的旧路径，作为历史记录保留。现有证据只确认 FD-038 使用旧布局，不能推断其它历史 FD 状态。
- 可复用的流程改进建议：评估增加 CLI 管理的旧版收据迁移/恢复入口，并在迁移时校验事件 ID、FD ID、Revision 和摘要；决定该能力后再更新相应稳定规范和 Auto preflight 说明。目前没有修改这些规则。
- 建议处理状态：待评估；该建议不授权手动改写收据或绕过领取 Gate。
