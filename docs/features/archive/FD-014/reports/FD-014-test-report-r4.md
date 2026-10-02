# FD-014 独立测试报告，第 4 轮

<!-- aiw-data: FD-014-test-report-r4.json -->

## 结论

本轮对应 `FD-014-000020-implementation-ready`，FD 修订号 20，摘要为
`f3b6924dfdbb7c8230e17f7aae47cd4df274009743d624b1778c25566c51e9ad`。
Tester 会话为 `fd014-tester-20261002-f7f15d14`，与 Worker 不同。
按本轮两份 Planner 双份授权，精确命令执行了两次。首跑 15 项中 13 项通过、
2 项失败；Worker 修正 JSON 字段映射并加强负例断言后，获批重跑 15 项全部
通过。最终 34 个适用的可观察场景中 26 个有可归因的通过证据，需求场景
覆盖率为 `26/34 = 76.47058823529412%`。业务代码分支覆盖率未测，也没有
原始分支报告；这项缺口须由 PM 明确决定，本报告建议暂缓判定。

## 场景与证据

以下按独立可观察行为计数。一个宽泛验收项的局部用例不覆盖其他行为；
没有执行的旧版测试结果不计入本轮。机器统计以同名 JSON 的 `scenarios`
数组为准。首跑两个 Dual 负例仅断言非零，未计覆盖；重跑时增加了 CLI
错误原因断言，正向交接也通过，故本轮最终可归因覆盖 E32/E33。
JSON 中每个标为 `passed` 的场景都用 `test_cases` 列出本轮实际执行并通过的
方法；E01/E02 的共用夹具由这些方法调用，不把夹具自身当作独立测试方法。

| 场景 | 可观察行为 | 对应黑盒用例或缺口 | 当前 |
| --- | --- | --- | --- |
| E01 | 新 FD 产生 Planner handoff | 所有用例的 `ready_worker` 夹具 | 通过 |
| E02 | Planner 设计完成交给 Worker | `ready_worker` | 通过 |
| E03 | 独立测试 FD 的 Worker 完成后交给 Tester | `test_independent_policy_routes_through_tester_and_pm` | 通过 |
| E04 | Worker 会话不能领取 Tester handoff | 同 E03 | 通过 |
| E05 | Tester 报告交给 PM | 同 E03 | 通过 |
| E06 | PM 接受后交给 Reviewer | 同 E03 | 通过 |
| E07 | Tester 会话不能领取 Reviewer handoff | 同 E03 | 通过 |
| E08 | PM 拒绝后退回 Worker | `test_pm_rejection_returns_to_worker_and_next_implementation_retests` | 通过 |
| E09 | Worker 再实现后重启 Tester 轮次 | 同 E08 | 通过 |
| E10 | 旧 FD 无独立测试标记时直达 Reviewer | `test_legacy_fd_routes_directly_to_reviewer` | 通过 |
| E11 | Pending Test 不能用 request-review 绕过 | `test_reviewer_cannot_bypass_pending_tester` | 通过 |
| E12 | Pending Test 不能直接发 verification-passed | 同 E11 | 通过 |
| E13 | Tester 报告缺会话字段被拒 | `test_report_requires_labelled_tester_session` | 通过 |
| E14 | 已执行测试缺授权被拒 | `test_executed_evidence_requires_authorization` | 通过 |
| E15 | 授权绑定其他实现事件被拒 | `test_authorization_for_other_implementation_is_rejected` | 通过 |
| E16 | 人工授权写 denied 被拒 | `test_human_approval_denied_is_rejected` | 通过 |
| E17 | 人工授权写 pending 被拒 | `test_human_approval_pending_is_rejected` | 通过 |
| E18 | 肯定、可追溯的人工授权引用通过格式门槛 | `test_affirmative_human_approval_reference_is_accepted`；仅为临时夹具引用 | 通过 |
| E19 | PM 不能接受实际失败的行为用例 | `test_pm_cannot_accept_failed_executed_case` | 通过 |
| E20 | PM 明确记录例外可接受低于门槛或未测覆盖率 | `test_independent_policy_routes_through_tester_and_pm` | 通过 |
| E21 | PM 未记录必须的覆盖率例外时拒绝接受 | 未准备用例 | 未覆盖 |
| E22 | PM 决策绑定过期 FD 修订或摘要时拒绝 | 未准备用例 | 未覆盖 |
| E23 | 场景数、通过数与覆盖率不一致的报告被拒 | 未准备用例 | 未覆盖 |
| E24 | 授权 FD 修订或摘要不一致被拒 | 未准备用例；E15 仅测事件不同 | 未覆盖 |
| E25 | 授权的精确命令不一致被拒 | 未准备用例 | 未覆盖 |
| E26 | 分支覆盖率不可用时有原因且不混同场景覆盖率 | `test_independent_policy_routes_through_tester_and_pm` | 通过 |
| E27 | 业务代码分支覆盖率排除生成/测试代码并注明不可达分支 | 尚无授权的独立覆盖率工具 | 未覆盖 |
| E28 | 过期待领 Worker handoff 可由 refresh-worker 替换 | FD-015 历史证据；本轮不重复 | 本轮未覆盖 |
| E29 | 已领取或在途 Worker handoff 不能被 refresh-worker 替换 | FD-015 历史证据；本轮不重复 | 本轮未覆盖 |
| E30 | 当前 Reviewer 通过后 Complete 可归档 FD 证据 | `test_dual_complete_archives_markdown_and_json` | 通过；临时 FD 完成归档 |
| E31 | 中文 Markdown 与同名 JSON 匹配时 Tester 可交接 | `test_dual_report_handoff_with_chinese_markdown_and_json` | 通过 |
| E32 | 缺 JSON 阻止 Tester handoff | `test_dual_report_missing_json_blocks_handoff` 核对缺文件错误 | 通过 |
| E33 | JSON 来源事件错配阻止 Tester handoff | `test_dual_report_mismatched_json_event_blocks_handoff` 核对来源事件错误 | 通过 |
| E34 | Complete 把报告与审查的 Markdown/JSON 成对归档 | `test_dual_complete_archives_markdown_and_json` | 通过；两种文件均存在于临时归档 |

### 不计入可执行场景的静态要求

- FD 的编号验收项 2、11 要求报告逐项列场景并计算比例；本表与 JSON
  是待审阅的书面证据。E23 单独覆盖 CLI 对计数错误的可执行拒绝行为。
- 根目录 `tests/`、独立黑盒写法以及中文 Markdown 的可读性，可直接检查
  路径和文本；JSON 的交接校验列为 E31 至 E33。
- Planner 对命令副作用的低风险判断与危险操作升级是人工决策过程；
  E14 至 E18、E24 至 E25 列出其可执行的 CLI 授权门槛。
- 源/安装 Skill、稳定规格及使用说明的一致性需静态比对。
- r1 至 r3 旧报告仍在原位，属于历史文件证据；E09 验证新 Tester
  轮次，E34 验证新双份文件归档。

## 命令与风险

两次均在仓库根执行精确命令
`python -B -m unittest tests.test_fd014_blackbox -v`。首次授权为
`FD-014-test-authorization-r4.md` 与同名 JSON；重跑授权为
`FD-014-test-authorization-r4-retry.md` 与同名 JSON。两份授权均绑定
本轮实现事件、摘要、Tester 会话和命令。首跑退出码 1，耗时 13.59 秒；
方法结果如下：

```text
test_affirmative_human_approval_reference_is_accepted ... ok
test_authorization_for_other_implementation_is_rejected ... ok
test_dual_complete_archives_markdown_and_json ... FAIL
test_dual_report_handoff_with_chinese_markdown_and_json ... FAIL
test_dual_report_mismatched_json_event_blocks_handoff ... ok
test_dual_report_missing_json_blocks_handoff ... ok
test_executed_evidence_requires_authorization ... ok
test_human_approval_denied_is_rejected ... ok
test_human_approval_pending_is_rejected ... ok
test_independent_policy_routes_through_tester_and_pm ... ok
test_legacy_fd_routes_directly_to_reviewer ... ok
test_pm_cannot_accept_failed_executed_case ... ok
test_pm_rejection_returns_to_worker_and_next_implementation_retests ... ok
test_report_requires_labelled_tester_session ... ok
test_reviewer_cannot_bypass_pending_tester ... ok
Ran 15 tests in 13.534s
FAILED (failures=2)
```

首跑两个失败均在 `test-report-ready`，CLI 原始错误为
`fd: test evidence requires **Implementation event:**`。Dual Tester 报告的
该字段位于同名 JSON 的 `data.implementation_event`；Worker 随后修正该字段
映射。重跑前，两个 Dual 负例增加了具体 stderr 断言，以防因别的错误造成
假阳性。重跑退出码 0，耗时 13.80 秒，原始方法结果如下：

```text
test_affirmative_human_approval_reference_is_accepted ... ok
test_authorization_for_other_implementation_is_rejected ... ok
test_dual_complete_archives_markdown_and_json ... ok
test_dual_report_handoff_with_chinese_markdown_and_json ... ok
test_dual_report_mismatched_json_event_blocks_handoff ... ok
test_dual_report_missing_json_blocks_handoff ... ok
test_executed_evidence_requires_authorization ... ok
test_human_approval_denied_is_rejected ... ok
test_human_approval_pending_is_rejected ... ok
test_independent_policy_routes_through_tester_and_pm ... ok
test_legacy_fd_routes_directly_to_reviewer ... ok
test_pm_cannot_accept_failed_executed_case ... ok
test_pm_rejection_returns_to_worker_and_next_implementation_retests ... ok
test_report_requires_labelled_tester_session ... ok
test_reviewer_cannot_bypass_pending_tester ... ok
Ran 15 tests in 13.751s
OK
```

未运行覆盖率工具、网络调用或最终构建。用例只读取仓库中的 CLI 入口和 FD
模板，复制到逐案
临时 Git 项目后，通过子进程调用复制的 CLI；文件写入、归档和清理均发生在
临时目录。每个子进程超时 15 秒。Planner 在执行前已审查 CLI 副作用并
写入双份授权记录。

剩余风险：E21 至 E25、E27 至 E29 缺少本轮用例或测量。业务代码分支覆盖率
不可用，PM 需要明确记录例外或退回补测。此前 r1 至 r3 的结果不作为本轮
通过证据；首跑失败也未被改写为通过。
