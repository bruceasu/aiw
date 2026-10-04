# FD-023 独立测试报告，第 1 轮

<!-- aiw-data: FD-023-test-report-r1.json -->

**Implementation event:** FD-023-000005-implementation-ready
**Tested FD revision:** 5
**Tested FD digest:** ab5b7b1897ec96ec1441100adcde6afa5e06c87bb2f0d696faad8d86008fc642
**Tester session:** fd023-tester-20261004-a9e4bd
**Applicable scenarios:** 22
**Covered scenarios:** 10
**Executed behavior tests:** 10
**Passed behavior tests:** 10
**Failed behavior tests:** 0
**Unrun behavior tests:** 12
**Requirements coverage:** 45.5%
**Branch coverage:** not measured
**Raw coverage evidence:** not measured; see FD-023-test-results-r1.json and FD-023-test-results-r1-retry.json
**Coverage unavailable reason:** 唯一获批命令没有对业务代码插桩。
**Test files:** tests/fd023_client_blackbox.mjs
**Commands:** ["node tests/fd023_client_blackbox.mjs", "node tests/fd023_client_blackbox.mjs"]
**Authorization records:** ["docs/features/reports/FD-023-test-authorization-r1.md", "docs/features/reports/FD-023-test-authorization-r2.md"]
**Recommendation:** pass
**Residual risk:** Gateway 实际默认与取消清理、长期限、HTTPS 与输出边界未运行；PM 需对 45.5% 覆盖率作例外决定。

## 结论

Tester 会话 `fd023-tester-20261004-a9e4bd` 认领 `FD-023-000005-implementation-ready`；测试对应 FD 修订 5、摘要 `ab5b7b1897ec96ec1441100adcde6afa5e06c87bb2f0d696faad8d86008fc642`。修正测试 fixture 后，当前版本 10 个离线客户端黑盒场景全部通过，0 个行为失败。22 个适用场景覆盖 10 个，需求场景覆盖率 **45.5%**；业务代码分支覆盖率**未测量**。建议 PM 对低于 70% 的覆盖率作明确例外决定，不能据此宣布真实 Gateway 或长请求已通过。

## 场景与证据

获批脚本 `tests/fd023_client_blackbox.mjs` 只启动本地 Node 客户端子进程和 `127.0.0.1` 临时 HTTP mock。第二轮原始证据为 `FD-023-test-results-r1-retry.json`。各场景的机器可统计记录位于同名 JSON 的 `data.scenarios`。

| ID | 可观察场景 | 结果 |
| --- | --- | --- |
| T01 | 帮助显示 660 秒与参数，且不请求 | 通过 |
| T02–T06 | 缺值、重复、零值、3661、非整数均在请求前退出 2 | 逐项通过 |
| T07 | 300 参数不进入 Prompt/Responses JSON；认证、一次 POST、文本输出 | 通过 |
| T08 | `--` 后参数保留为字面 Prompt | 通过 |
| T09 | 请求头前 1 秒超时并关闭本地连接 | 通过，约 1.2 秒 |
| T10 | 响应体未完成时 1 秒超时并关闭本地连接 | 通过，约 1.2 秒 |
| U01–U02 | Gateway 600 秒运行默认值、显式配置范围与重启生效 | 未覆盖 |
| U03–U04 | 客户端实际 660 秒默认及 300 秒显式期限 | 未覆盖 |
| U05–U10 | 3660 上界、响应大小、JSON、verbose、HTTP 错误与不重试、HTTPS | 未覆盖 |
| U11–U12 | 真实 Gateway 取消清理、模型调用与部署长请求 | 未覆盖 |

帮助文本只能证明声明默认值；1 秒超时只能证明短期限及完整响应读取取消，不能替代 300/660 秒真实等待。示例 JSON 字面值和文档内容属于静态材料，没有计作运行场景通过。

## 命令与风险

Planner 授权为 `FD-023-test-authorization-r1.md` 与 `FD-023-test-authorization-r2.md`，均绑定上述事件、修订、摘要及 Tester 会话。实际在仓库根目录运行相同命令两次：

1. `node tests/fd023_client_blackbox.mjs`：退出 1；8/10 通过，T07/T08 因测试 mock 缺标准 Response 的 `object: "response"` 字段而失败。原始证据 `FD-023-test-results-r1.json` 已保留。这是测试 fixture 错误，不能归咎于客户端实现。
2. 修正 fixture 后按 Planner 的一次重试授权运行同一命令：退出 0；10/10 通过。原始证据 `FD-023-test-results-r1-retry.json`。

未运行真实 Gateway、真实模型、外部网络、长期等待、覆盖率插桩、最终构建或部署。未测量分支覆盖率，因为唯一获批命令未插桩业务代码。Gateway 的实际默认与显式配置、服务端执行上限、客户端取消后服务端清理，以及未覆盖的输出和协议边界，仍是验收风险。
