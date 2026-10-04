# FD-024 独立测试报告，第 1 轮

<!-- aiw-data: FD-024-test-report-r1.json -->

## 结论

Tester session：`fd024-tester-20261004-53ffaa61`。来源事件：`FD-024-000004-implementation-ready`；FD revision 4；digest `0e3759c666059029de31d878c5c542fc82fcab50b68e9f7f9c4e95df6c48f52b`。

精确授权见 `docs/features/reports/FD-024-test-authorization-r1.md`。黑盒命令首次运行 4 项，3 通过、1 因夹具先修改 FD 再认领旧回执而失败；此时 CLI 正确拒绝过期认领。修正夹具顺序后，同一命令重试一次，4 项全部通过。没有观察到产品行为失败。

可观察验收场景共 14 项，实际覆盖并通过 9 项，需求场景覆盖率 **64.3%**。其余 5 项未运行；业务代码分支覆盖率未测量。建议 PM 仅在明确记录覆盖率例外及剩余风险后决定是否接受。

## 场景与证据

| ID | 可观察行为 | 状态 | 用例/原因 |
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

测试经 `tests/test_fd024_refresh_tester_blackbox.py` 调用 `tests/test_fd014_blackbox.py` 的公开 CLI 临时仓库夹具；没有导入或读取实现模块。静态排除：CLI 帮助、补全及文档文字属于非行为性验收，本轮没有将它们计入场景分母。

## 命令与风险

- 授权命令（工作目录 `D:\03_projects\AI-tools\aiw`）：`python -B -m unittest tests.test_fd024_refresh_tester_blackbox -v`。
- 首次输出：`Ran 4 tests in 3.277s`，`FAILED (failures=1)`；失败堆栈指向测试夹具 `test_claimed_handoff_is_rejected_without_change` 的认领步骤，CLI 返回 `FD changed since the handoff; reconcile before claiming`。这是夹具先造成过期再认领的顺序错误。
- 修正夹具后同一命令重试：`Ran 4 tests in 3.324s`，`OK`，4 项均通过。
- 命令只在系统临时目录创建并清理测试仓库及回执；未请求网络、外部服务、依赖下载或权限提升。未运行覆盖率命令，因此没有业务代码分支原始数据。
- 剩余风险：状态/角色和在途拒绝、Worker 身份与 sidecar 校验、故障回滚、旧授权拒绝尚未由本轮黑盒运行验证；业务代码分支覆盖率未知。
