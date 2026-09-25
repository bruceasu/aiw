## Context

Session Resolve 在确认不存在合法位置的记录时返回 ErrSessionNotFound，该 sentinel 包装 os.ErrNotExist。Workflow 请求准备与 repair 仍用 os.IsNotExist 判断，无法识别 fmt.Errorf 包装。已观测任务的工作树创建成功，但没有 Session、Attempt、prepared request 或写租约，supervisor 停在 runner-paused。

## Design Readiness

FD_NOT_REQUIRED：问题与兼容边界已明确，复用现有 Session sentinel、创建及恢复接口，属于局部可逆修复，不改变数据模型或并发协议。用户已确认主要测试边界，本轮不加载独立 FD。

## Goals / Non-Goals

**Goals:** 首次启动自动创建合法缺失 Session；恢复入口使用相同错误分类；既有工作树和审计记录保留。

**Non-Goals:** 不新增公共接口或命令，不批量替换所有文件缺失判断，不自动修复损坏 Session，不修改既有恢复状态机。

## Decisions

1. 在消费 Session Store.Load 错误的位置用 errors.Is(err, session.ErrSessionNotFound) 判断。保留领域 sentinel，不改为字符串匹配，也不广泛匹配 os.ErrNotExist：后者可能把已存在 Session 内部文件缺失误判为整个记录不存在。
2. 请求准备保留 meta.Session == Task ID 的自动创建条件。只在真正缺失时调用既有 createTaskSession，再重新 Load；创建失败返回原错误，不创建 Attempt 或 prepared request。异名绑定缺失继续报错，不能猜测创建另一个 Session。
3. 既有 repairWorkflowState 只在其原有条件（写租约存在、无 prepared request、Attempt Session 匹配）下识别缺失并调用 RepairMissingSessionAttempt；不扩大恢复资格、不删除 Attempt，不对损坏或归档 Session 执行缺失恢复。
4. 检查 Session Load 的相关调用点，不机械替换处理普通文件的 os.IsNotExist。正常已有 Session 继续复用，归档 Session 的现有不可运行检查保持有效；未修复前不建议单纯重试 supervisor。

## Testing Decisions

用户已确认以现有 Workflow 命令入口为主要测试边界。优先复用 workflow_runner_test.go、workflow_commands_test.go 和 workflow_supervisor_test.go 的临时目录及请求准备夹具，使用真实 Session Store 的错误返回；避免只测试 errors.Is 本身。

- 首次准备：无 Session、合法同名绑定，准备成功且只有一份 Session、一个合法 Attempt/request；重复准备复用。
- 拒绝路径：异名绑定缺失、已有目录但 status 缺失/损坏、身份冲突、归档 Session；不得覆盖数据或错误创建活动 Session。
- 恢复路径：已有缺失 Session Attempt 通过原修复入口恢复并保留历史；非缺失错误不改变 Attempt。
- 创建失败：不产生新 Attempt/request，保存可诊断错误。权限错误优先用已有可控故障边界，不依赖 Windows ACL 或管理员权限。
- 不启动真实模型、不进行网络调用，不用真实 supervisor 长循环作为测试前提。当前轮仅编写规格，测试运行另行授权。

## Risks / Trade-offs

- 部分调用点仍使用旧判断 → 定向审查 Session Load 消费点，以启动和恢复入口的行为测试覆盖。
- 已存在但损坏的 Session 被误建 → 只匹配领域 sentinel，负向测试比较原记录未被覆盖。
- 旧已安装程序仍包含 bug → 恢复时使用含修复的可执行程序；主工作区 go run 与旧 aiw 不混用。

## Migration Plan

无需数据迁移。修复验收后，对 fix-task-metadata-lifecycle-sync 先 diagnose/status，确认无活动 supervisor 且租约允许后再显式 start，复用原 worktree。该操作由用户另行执行或授权，本 change 创建时不启动。回滚代码不会删除已有 Session、Attempt 或工作树。

## Open Questions

%% 本次根因来自用户日志和静态源码，尚未运行回归；实现阶段应按授权执行聚焦验证。没有阻止实现的设计问题。
