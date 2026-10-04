# FD-024 独立测试报告，第 2 轮

<!-- aiw-data: FD-024-test-report-r2.json -->

## 结论

Tester session：`fd024-tester-r2-20261004-0e8de67e`。来源事件：`FD-024-000008-implementation-ready`；FD revision 8；digest `88137748eb4d6bf3c6bf7ef7a43f06beb83947f5dbc046de232327c74f760f76`。第 2 轮精确授权见 `docs/features/reports/FD-024-test-authorization-r2.md`。Worker 报告本轮仅修复中文证据文字，行为代码未变。

本轮独立运行 4 项黑盒测试，**4 项全部通过**。14 个可观察验收场景中，9 项由这些测试覆盖并通过，需求场景覆盖率 **64.3%**；其余 5 项未运行。业务代码分支覆盖率未测量，PM 如接受本轮报告，仍须针对低于 70% 和分支覆盖率不可得记录明确例外与剩余风险。第 1 轮夹具顺序失败及修复记录保留在第 1 轮报告，本轮没有发生该失败。

## 场景与证据

| ID | 可观察行为 | 状态 | 本轮证据或未覆盖原因 |
| --- | --- | --- | --- |
| S01 | 过期 pending Tester 交接刷新为 PM 产生的新 Tester 事件，绑定当前报告 | 通过 | `test_stale_refresh_preserves_provenance_and_tester_can_report` |
| S02 | 刷新递增 FD revision，保留 Pending Test 与 Work Items | 通过 | 同上 |
| S03 | 新事件保留 Worker session 和原 implementation event | 通过 | 同上 |
| S04 | 旧事件取消，双方 receipt 记录 supersedes/superseded_by | 通过 | 同上 |
| S05 | 独立 Tester 可认领新事件，并以其为来源提交双份报告至 PM | 通过 | 同上 |
| S06 | 当前交接拒绝刷新且 receipt 不变 | 通过 | `test_current_handoff_is_rejected_without_change` |
| S07 | 已认领的过期交接拒绝刷新且 FD/receipt 不变 | 通过 | `test_claimed_handoff_is_rejected_without_change` |
| S08 | 缺失实施报告拒绝刷新且 FD/receipt 不变 | 通过 | `test_missing_report_is_rejected_without_change` |
| S09 | 新事件保持 Tester 目标角色与原报告路径 | 通过 | `test_stale_refresh_preserves_provenance_and_tester_can_report` |
| S10 | 错误 FD 状态或非 Tester 角色拒绝 | 未覆盖 | 未准备对应临时夹具 |
| S11 | launching/dispatched 中的交接拒绝 | 未覆盖 | 未准备对应临时夹具 |
| S12 | 缺失 Worker 身份或无效 Dual sidecar 拒绝 | 未覆盖 | 未准备对应临时夹具 |
| S13 | 写入失败后回滚 FD、旧 receipt 和索引，无可认领孤儿 | 未覆盖 | 未注入写入故障 |
| S14 | 旧测试授权不能用于新事件 | 未覆盖 | 未构造旧授权及提交拒绝场景 |

测试文件为 `tests/test_fd024_refresh_tester_blackbox.py`，复用公开 CLI 临时仓库夹具 `tests/test_fd014_blackbox.py`；未导入或读取实现模块。CLI 帮助、补全及文档文字属于非行为性验收，未计入行为场景分母。

## 命令与风险

- 实际命令：`python -B -m unittest tests.test_fd024_refresh_tester_blackbox -v`；工作目录 `D:\03_projects\AI-tools\aiw`；运行一次。
- 原始输出摘要：四个具名测试均为 `ok`；`Ran 4 tests in 3.334s`，`OK`，退出码 0。命令输出见本轮 Tester 会话的运行记录。
- 测试仅在系统临时目录创建并清理测试仓库、FD 和回执；未使用网络、外部服务、下载、权限提升或发布产物。未运行覆盖率命令，业务代码分支覆盖率没有原始数据。
- 剩余风险：状态/角色和在途拒绝、Worker 身份与 sidecar 校验、故障回滚、旧授权拒绝尚无本轮运行证据；分支覆盖率未知。
