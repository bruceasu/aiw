# E02 恢复聚焦测试通过

Task：workflow-automation；Work Item：wi-0012 / 2.2。用户在正常终端执行同一取证脚本，并提供 Evidence 目录及退出码。本目录的 run.json、inputs.json、stdout.jsonl、stderr.txt 为实际运行记录，未重写。

2026-09-24T07:44:41Z 至 07:44:43Z；Go 1.25.1 windows/amd64；worktree 工作目录；GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local。退出码 0，stderr 为空，运行期间输入未变。

```text
go test ./internal/workflow -run ^(TestDurableUnknownAndStopRetainWriter|TestDurableResultReplayConsumesOnceAndKeepsAttempt|TestDurablePendingTailRecoveryKeepsRevision)$ -count=1 -timeout=60s -vet=off -json
```

| 用例 | 结果 | 用时 |
| --- | --- | --- |
| TestDurableUnknownAndStopRetainWriter | passed | 0.08 秒 |
| TestDurableResultReplayConsumesOnceAndKeepsAttempt | passed | 0.08 秒 |
| TestDurablePendingTailRecoveryKeepsRevision | passed | 0.04 秒 |

包执行时间 0.242 秒，含编译的命令总耗时约 2.02 秒。输入清单摘要：3075ff815d869b8ac06faaf35d7016efe82c3188ccf73306ae1f2048453d8241。

## 结论与边界

冻结输入与请求共用模型选择的 fixture 修复已在上述运行验证。三个用例分别取得未知派发/Stop 写权保留、结果消费去重且不提前完成整项、截断事件尾部恢复和重复恢复幂等的局部证据。此前失败结果独立保留，不覆盖或改成成功。

这些测试使用临时 Store 和服务替身，不证明真实模型、生产联合启用、断电可靠性或完整 AC/AX 验收。工具环境的路径权限限制也没有被本次正常终端结果修复；安装版 AIW 未更新。

本轮只核对用户运行的原始证据并更新文档，没有重新运行测试、编译、安装、网络或 Git 交付。会话/终端授权不伪装为产品 Runner grant；正式 2.2/3.1/3.2 与整体授权 Gate 保留，继续列出其余验收缺口。
