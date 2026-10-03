# FD-026 独立测试报告，第 1 轮

<!-- aiw-data: FD-026-test-report-r1.json -->

## 结论

本轮没有执行行为测试，所有 4 个适用场景均未覆盖。需求场景覆盖率为 0/4（0%）；代码分支覆盖率未测量。建议暂缓测试接受，待获得明确的测试执行授权后再验证。

## 场景与证据

| ID | 可观察行为 | 状态 | 结果/未覆盖原因 |
| --- | --- | --- | --- |
| GW-01 | `aiw gw start` 能启动插件目录中的网关入口，并使用约定的默认配置位置。 | uncovered | 未运行；未获测试执行授权。 |
| GW-02 | 新安装布局将入口和 Windows/Linux 网关二进制放在 `plugins/aiw-gw/`，运行时能找到入口及默认配置。 | uncovered | 未运行；未获测试执行授权。 |
| GW-03 | `aiw gw stop` 能通过短命令停止网关。 | uncovered | 未运行；未获测试执行授权。 |
| GW-04 | `aiw agent-gateway` 不再作为可用命令或文档用法提供。 | uncovered | 未运行；需要通过 CLI 和当前用户文档观察确认。 |

静态审阅仅依据 FD-026 的 Acceptance、Worker 报告和事件收据拆解场景；没有读取实现源码，没有将静态描述计为测试证据。Worker 报告记载未运行行为测试、安装或部署。

## 命令与风险

- Tester session：`fd026-tester-20261004-a7e31c`
- 实现事件：`FD-026-000005-implementation-ready`
- FD revision/digest：`5` / `c10172399ac3efbe66a194de734bfecc62a81e8e4ce2eceb35827ccfb79bffab`
- 实际执行的测试命令：无。
- 授权记录：无；handoff 本身不构成测试授权。
- 未执行测试文件：无。
- 覆盖率原始证据：无。需求场景覆盖率为 0%，分支覆盖率未测量；无法产生代码覆盖率数据，因为未运行测试/覆盖率工具。
- 剩余风险：短命令的启动/停止行为、Windows/Linux 安装布局及旧命令移除均未通过运行时行为验证。Worker 报告中的 compile-only 结果未确认，不作为测试证据。
