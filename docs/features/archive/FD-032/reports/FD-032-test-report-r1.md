# FD-032 独立测试报告，第 1 轮

<!-- aiw-data: FD-032-test-report-r1.json -->

## 结论

本轮两次获 Planner 精确授权的聚焦命令均运行了 3 个 unittest 方法，均在黑箱夹具的前置步骤失败。第一次使用了仅有 Markdown 的临时 Planner 授权记录，Dual evidence 校验拒绝 `test-report-ready`；修正为 Markdown+JSON 后，第二次已通过 `test-report-ready`，但夹具误认领不可由 agent 认领的 PM 事件。两次都没有进入多数决策或 Reviewer 身份校验。本报告不把这些夹具失败写作实现失败或业务测试通过。

目标行为场景共 13 个，通过 0 个，需求场景覆盖率 0/13（0%）。本轮没有业务行为测试完成，业务代码分支覆盖率未测量。第二次运行后的夹具错误已静态删除，修正后的用例未执行。

## 场景与证据

| ID | 可观察行为 | 状态 | 证据/限制 |
| --- | --- | --- | --- |
| S01 | 三票中两票接纳时，带失败测试和 0% 覆盖率的报告仍进入 Reviewer | 阻塞 | PM 决策步骤未到达 |
| S02 | 接纳决定不改写 Tester Markdown 与 JSON 中的失败事实 | 阻塞 | PM 决策步骤未到达 |
| S03 | Reviewer 拒绝 Worker、Tester、三位 assessor session，接受独立 session | 阻塞 | Reviewer 事件未产生 |
| S04 | 三票中仅一票接纳时退回 Worker | 阻塞 | PM 决策步骤未到达 |
| S05 | PM 事件与票数不符时拒绝决定 | 阻塞 | PM 决策步骤未到达 |
| S06 | 缺少评估时拒绝决定 | 阻塞 | PM 决策步骤未到达 |
| S07 | 重复评估路径时拒绝决定 | 阻塞 | PM 决策步骤未到达 |
| S08 | 评估 FD digest 过时时拒绝决定 | 阻塞 | PM 决策步骤未到达 |
| S09 | 两名评估者共用 session 时拒绝决定 | 阻塞 | PM 决策步骤未到达 |
| S10 | 评估者与 Worker 共用 session 时拒绝决定 | 阻塞 | PM 决策步骤未到达 |
| S11 | 评估者与 Tester 共用 session 时拒绝决定 | 阻塞 | PM 决策步骤未到达 |
| S12 | 评估者与 PM 共用 session 时拒绝决定 | 阻塞 | PM 决策步骤未到达 |
| S13 | 评估来源事件伪造时拒绝决定 | 阻塞 | PM 决策步骤未到达 |

两次运行各报告 `Ran 3 tests ... FAILED (failures=3)`。第一次三个失败均为 `fd: Dual evidence report requires one aiw-data JSON reference`；第二次三个失败均为 `fd: human and PM handoffs cannot be claimed by an agent session`。两处均是测试夹具的输入或操作问题。既有 FD 的历史报告读取兼容、文档与稳定规格一致性属于静态审查范围，本轮未作行为测试。

## 命令与风险

- 工作目录：`C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-032`
- 第一次：`python -B -m unittest tests.test_fd032_risk_decision_blackbox -v`；退出码 1；授权：`docs/features/reports/FD-032-test-authorization-r1.md`。
- 临时授权夹具改成 Dual evidence 后第二次运行同一命令；退出码 1；授权：`docs/features/reports/FD-032-test-authorization-r2.md`。
- 测试文件：`tests/test_fd032_risk_decision_blackbox.py`，使用 `tests/test_fd014_blackbox.py` 的临时 Git 仓库夹具；没有触碰真实 `.ai` 回执。
- 剩余风险：核心多数票和身份边界尚无运行时证据；没有分支覆盖率。建议 PM 评估这一证据缺口并决定修复/再测或带风险交付，不把夹具失败当成实现缺陷。
