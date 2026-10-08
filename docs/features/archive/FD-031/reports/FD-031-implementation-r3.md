# FD-031 Worker 实现报告（第 3 轮）

<!-- aiw-data: FD-031-implementation-r3.json -->

## 交接与范围

- Source event：`FD-031-000008-work-requested`，PM 经用户授权恢复的待认领 Worker 事件。
- Worker session：`fd031-worker-repair-20261008-4e367cb0`。已用此标识认领新事件。旧 `FD-031-000006-test-rejected` 已取消，并记录新事件及停止工作的旧 session；本轮没有冒用旧标识。
- 此前提交 `0555ce47` 修复 Issue 标题读取与黑箱失败夹具；提交 `f14f0c61` 增加受限的 PM Worker 恢复命令、规格与用法。本轮更新 FD TODO、Verification 和双格式证据。

## 静态复核

- Issue 标题由 `%q` 写入，`strconv.Unquote` 还原引号等转义，`promoteIssue` 将读取后的标题和原 Issue ID 作为独立参数传给 `aiw fd new`；该路径不调用旧 promotion 写入函数。
- 黑箱夹具在临时项目中把 `TEMPLATE.md` 路径设为目录。FD 创建器先尝试原子写入模板，因目标是目录而失败；测试还检查无 FD、无 handoff、Issue 元数据不变和无 Task。
- `recover-worker` 限定活动的 Open/In Progress FD、最新 dispatched Worker 事件、精确的事件 ID 与旧 session，检查 revision/digest；成功时取消旧回执并建立有双向关联的新 pending 事件。失败分支尝试恢复 FD、旧回执和索引并移除新事件。未执行失败注入，回滚仍缺运行时证据。

## 已执行命令与结果

- `aiw fd claim FD-031 FD-031-000008-work-requested --session fd031-worker-repair-20261008-4e367cb0`：成功认领。认领前新事件的 revision 8 和 FD 摘要 `4781df702c17db27bd81e6c45dbf80375354cc7cb19d822e97b02bcf8fe9fcf6` 一致。
- `python -c "from pathlib import Path; compile(Path('plugins/aiw-fd.py').read_bytes(), 'plugins/aiw-fd.py', 'exec')"`：通过，只做内存编译，无产物。
- `git diff --check` 与 r3 JSON 静态解析均通过；Git 仅提示本地换行符转换。
- 前一修复轮次的 `go build -o NUL ./cmd/aiw-req` 与 `git diff --check` 已通过；详见 r2 报告。本轮未重新执行 Go 编译。

## 未执行检查与风险

- 本轮没有运行黑箱测试、恢复命令失败注入、构建产物检查、linter 或完整验证。旧 Tester 的 6/8 只适用于修复前版本。
- 独立 Tester 需要基于新的 `implementation-ready` 事件、当前 FD revision/digest 和精确命令重新获得 Planner 授权；PM 随后决定测试报告是否可接纳。`recover-worker` 的错误回滚仍由独立静态审查与后续聚焦验证确认。
