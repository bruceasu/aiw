# FD-046 独立审查（第一轮）

<!-- aiw-data: FD-046-review-r1.json -->

- Reviewer 会话：`codex-reviewer-20261009-fd046-r1`
- Source event：`FD-046-000004-implementation-ready`
- 审查基准：`1fefd74..ed8a40b`（当前 HEAD `ed8a40b`；实现交接收据已由本 Reviewer 领取）
- 结果：`changes-requested`

## 发现

### R1（中）：强制归档后留下过期的 pending 收据

[plugins/aiw-fd.py](/D:/03_projects/AI-tools/aiw/plugins/aiw-fd.py:2070) 的 `close_locked` 在强制 Complete 分支跳过所有收据写入。若最新收据是 pending（例如等待 Reviewer 领取的 `implementation-ready`），归档会增加 FD revision 并改变摘要，但收据仍显示 pending 且指向旧 revision/摘要。该交接因内容不匹配无法正常领取，却仍以待处理状态留在工作流记录中。

FD-046 的 Solution 要求把 pending receipt 与 FD、索引、审计及证据移动纳入可回滚事务；计划也只禁止修改或伪造 `verification-passed` 收据。请在同一事务中明确处理非运行中的 pending 收据（例如标为 cancelled 并纳入回滚），保持 Reviewer 通过收据不变；或修订计划并给出 pending 收据保留仍满足工作流契约的依据。

## 审查范围与证据

- 静态阅读强制与普通 close 分支、证据选择、审计字段、索引更新和回滚逻辑，并对照 FD-046、FD 工作流稳定规格、CLI 用法及 Worker 报告。
- `python plugins/aiw-fd.py claim FD-046 FD-046-000004-implementation-ready --session codex-reviewer-20261009-fd046-r1` 成功领取了源交接。
- 只运行了 CLI 帮助/交接操作以及静态读取命令；未运行测试、编译、最终构建、lint 或故障注入。Worker 报告中的 compile-only 通过结果仅作为已提交报告记录，本 Reviewer 未重跑。
- 实现所在 HEAD 是一个包含大量其他文件变更的聚合提交，且当前没有 `.wt/FD-046`；本次仅评估 FD-046 相关代码与文档，没有把其他 FD 的变更作为本 FD 的缺陷。

## 剩余风险

未运行事务故障注入，因此回滚行为只有静态证据。强制归档不能代表 Reviewer 验证通过。
