# FD-036 PM 测试决策 r3

<!-- aiw-data: FD-036-test-decision-r3.json -->

## 决定

本决策对应 `FD-036-000021-test-report-ready`，FD revision 21，digest `bf8ed80fa087ac34af85881240879d51aa9be42a8ba5617d6601690f7316b8b7`。Tester r3 报告记录 19/19 个行为测试通过、0 个失败；16 个适用场景中 11 个完整覆盖，需求覆盖率 68.75%，branch coverage 未测量。

本轮采用 `adaptive-v1/escalated`。三份有效独立评估为：A2 `repair`、B2 `repair`、C `accept-with-risk`。A2 与 B2 指出 S05、S08、S09 的配置错误行为、完整优先级和非法枚举证据不足；C 认为可带风险进入 Reviewer。多数票为 `repair`，因此本次决定为 **rejected**，返回 Worker。

用户在本决策期间明确要求：合法 TOML 中配置项的值错误时回退到较低优先级的有效值，最终使用内置默认值；整份 TOML 语法损坏时报错退出；未知键忽略。当前 FD 仍写明无效配置报错；Worker 应先把此要求同步到 FD 与实现。现有 Tester 报告是在此行为澄清前形成，不能作为新行为的验证证据。

S08 四层优先级、S14 未实现选项、S15 跨平台与复制行为仍按 Tester 报告如实记录为不完整；未覆盖项不计为通过。本次不接受覆盖率例外。

## 评估来源

- A：`FD-036-test-risk-assessment-r3-a2.md`，独立替代 session，`repair`。原 r3-a JSON 无法解析，故由 A2 替代，不计入投票。
- B：`FD-036-test-risk-assessment-r3-b2.md`，独立 session `fd036-assessor-r3-b2-20261008-085110`，`repair`。原 r3-b 报告因评估员误读历史报告而由 B2 替代，不计入投票。
- C：`FD-036-test-risk-assessment-r3-c.md`，独立 session `fd036-assessor-c-20261008-r3-ops-9d4f`，`accept-with-risk`。

下一步：更新 FD 的配置行为与 Verification，刷新 Worker handoff，修复后由独立 Tester 基于新 revision 验证，再重新进行 PM 决策。
