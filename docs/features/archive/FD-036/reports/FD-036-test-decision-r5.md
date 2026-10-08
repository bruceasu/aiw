# FD-036 PM 测试报告决策，第 5 轮

<!-- aiw-data: FD-036-test-decision-r5.json -->

## 决策

**拒绝本轮测试验收，交回 Worker。** 本决策绑定测试报告交接 `FD-036-000030-test-report-ready`、FD revision 30、digest `8fc52ef381ac87a4932b0784c36a338ac540785cb732fd479e3d719ec7e05510`。

Tester r5 按 Planner r7 授权命令执行一次：可见结果为 20 项通过、1 项 `ERROR`；`test_missing_default_config_uses_defaults_and_requires_explicit_model` 的 traceback、unittest 最终摘要和退出码均不可用。报告将 S04 标为 blocked，需求场景覆盖为 13/18（72.22%），branch coverage 未测量。当前不能判定错误来源，也不能将 S04 计为通过。

三份独立风险评估均投 `repair`：A `docs/features/reports/FD-036-test-risk-assessment-r5-a.md`、B `...-r5-b.md`、C `...-r5-c.md`。A 从验收影响审查，B 核对执行证据与技术诊断，C 审查交付与运维风险；三者均未猜测根因。

采用 `adaptive-v1/escalated`，因存在行为测试错误且诊断证据不完整而升级为三评估流程。票数为 `accept-with-risk` 0、`repair` 3，未达到接纳门槛，因此发出 `test-rejected`。此前 PM r4 接受的 S08/S09/S14/S15 和 branch coverage 缺口仍是明确风险，但不能覆盖本轮 S04 错误。

## 后续

需要先解决本次执行诊断缺失或由相关实现/环境修正后重新取得精确授权，再对当前实现证据进行测试。不得将本次 ERROR 归因为产品或测试夹具，也不得在没有新的授权时重跑。

剩余风险包括 S04 结果不明、S08/S09/S14/S15 未完整覆盖、branch coverage 未测量、真实 API/模型行为、跨平台 profile 路径及真实翻译质量未验证。
