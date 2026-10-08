# FD-036 PM 测试决策 r4

<!-- aiw-data: FD-036-test-decision-r4.json -->

## 决定

本决策对应 Tester handoff `FD-036-000024-test-report-ready`，FD revision 24，digest `c79fbda3eb41d1443a492f6d8b5e89011d36e34ab631993fe20f66aa32aca8d1`。该报告针对实现快照 revision 23 / digest `ec7adbbb1e50c75d8e95834321a3f5d71f0765299d3934ddcee80694851f1c27`。经 Planner r6 授权的命令运行 21 项行为测试，21 通过、0 失败；18 个适用场景中 14 个完整覆盖，需求覆盖率 77.78%，branch coverage 未测量。

本轮采用 `adaptive-v1/escalated`。A、B、C 三份独立评估均投 `accept-with-risk`，达到至少两票接受风险的门槛。决定为 **accepted**，进入独立 Reviewer。

本轮对用户澄清的配置行为已有直接测试证据：S05 验证损坏 TOML 报错，S17 验证无效字段值回退和 profile 保留低优先级有效值，S18 验证未知键忽略。仍有四项覆盖不足：S08 完整四层优先级矩阵、S09 所有枚举、S14 所有未实现选项、S15 跨平台与复制/不覆盖行为。接受进入 Reviewer 时明确保留这些验证例外；未覆盖场景不视为通过。branch coverage、真实 API/模型行为与翻译质量也未验证。

## 评估来源

- Tester：`FD-036-test-report-r4.md/json`，session `fd036-tester-20261008-r4-independent`，授权 `FD-036-test-authorization-r6.md/json`。
- A：`FD-036-test-risk-assessment-r4-a.md`，`accept-with-risk`。
- B：`FD-036-test-risk-assessment-r4-b.md`，`accept-with-risk`。
- C：`FD-036-test-risk-assessment-r4-c.md`，`accept-with-risk`。

## Reviewer 关注

请核对当前实现与新增配置用例、r4 测试证据和三份评估的一致性；确认四个明确保留的覆盖缺口与其他运行限制均有披露。不要仅因上述已接受的覆盖缺口拒绝交接。
