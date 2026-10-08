# FD-036 测试风险评估 r5，评估者 B

<!-- aiw-data: FD-036-test-risk-assessment-r5-b.json -->

## 独立结论

投票：`repair`。本轮 21 项行为测试中，报告记录 20 项通过、1 项 `ERROR`，且缺少 traceback、unittest 最终汇总和可用退出码。S04 的缺失默认配置场景尚未得到可判定结果。现有证据不能证明产品实现错误，也不足以将此失败作为可接受的残余风险；应先取得可诊断的执行结果，再决定是否修复代码或夹具。

## 影响与时间

影响范围是缺少默认 `aiw.toml` 时使用默认配置、并在无模型设置时明确报错的验收路径。若实现行为错误，用户可能在未配置模型时得到不符合预期的错误；若是夹具或执行环境问题，则生产行为未必受影响。根因未知，修复时间及交付影响均为 `unknown`：需要先有 traceback 和最终测试结果才能估算。本评估不建议为未知原因直接改生产代码。

## 证据与风险

- 测试事件：`FD-036-000030-test-report-ready`；FD revision 30，digest `8fc52ef381ac87a4932b0784c36a338ac540785cb732fd479e3d719ec7e05510`。
- 被测实现事件：`FD-036-000029-implementation-ready`；revision 29，digest `64a3b0ec060f0d1d492f6ac3e8acf7d4c3ad20184449064c0b6962596051f522`。
- Tester 报告：`docs/features/reports/FD-036-test-report-r5.md` 与同名 JSON。摘要记录 21 项执行、20 项通过、1 项失败；S04 阻塞；完整场景覆盖 13/18（72.22%），分支覆盖未测量。
- Planner 授权：`docs/features/reports/FD-036-test-authorization-r7.md` 与同名 JSON，批准对指定实现版本执行一次聚焦命令。报告称该执行的可见输出包含一个 `ERROR`，但没有保留 traceback、汇总或退出码；没有证据表明授权命令未按范围执行。
- 静态查看 `tests/test_fd036_say_blackbox.py` 的 `test_missing_default_config_uses_defaults_and_requires_explicit_model`：用例复制可执行文件到临时隔离目录，再断言非零退出、stderr 含 `model is required`、stdout 为空且 mock API 无请求。fixture 的环境设置提供测试 API key 和 loopback endpoint。静态材料不能还原运行时错误，因此不据此臆断根因；未运行测试或 coverage。
- 其余 S08、S09、S14、S15 仍未完整覆盖。r5 新增 TOML 非法转义与未知数组键行为在报告中被记为通过，但 S04 的失败和缺失诊断使本轮整体无法给出干净的验收证据。实际 API、跨平台路径、翻译质量及分支覆盖仍无证据。

接纳风险：若此时通过，会把一个未解释的行为测试错误和 S04 未验证状态带入 Reviewer，无法区分产品回归与测试基础设施问题。建议为新的独立 Tester 轮次取得新的精确授权，捕获完整 stdout/stderr、traceback、unittest 汇总及退出码；若诊断指向测试夹具，修正后以新实现/测试版本重走授权；若指向实现，则先由 Worker 修复并再由 Tester 验证。不得沿用 r7 的一次性授权重跑。