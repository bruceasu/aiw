# 首轮聚焦验证提案

Task: workflow-automation；登记工作项：wi-0012 / 2.2。
状态：用户已回复“confirm, continue”，授权本计划的两个聚焦用例；2026-09-24 首跑通过。完整记录见 [执行结果](verification-results/e04-budget-20260924T010710Z/report.md)。批准时的原计划已另存 approved-plan.md；本文件不是产品 Runner grant，也不代表完整验收通过。

## 范围与命令

先验证 E04 的两个已有预算用例：重启后的缺陷去重，以及预算预约/释放、两次有效失败后的升级和每 Actor 升级上限。此轮只提供对应 SW08/SW15/SW16 的部分单元证据，不代表完整 AC、真实崩溃恢复、联合启用或其他 E 项通过。

工作目录：C:/Users/svictor/workspace/tools/aiw/.wt/workflow-automation。

```powershell
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:GOTOOLCHAIN = 'local'
go test ./internal/workflow -run '^(TestGenerationBudgetDeduplicatesDelayedDefectsAcrossRestart|TestGenerationBudgetReservationsAndEscalationLimits)$' -count=1 -timeout=60s -vet=off -json
```

仅执行上述两项及其子测试；Go 仍会编译该包的测试源码。预计含编译约 1–3 分钟，60 秒限制适用于测试执行阶段，不包括编译。读取本地依赖与工具链，允许普通测试临时目录/缓存和受管证据工件写入；不下载依赖、不调用模型或外部服务，不修改真实 Task 的产品状态、不执行 Git 交付。静态读取所选用例显示它们使用内存 RuntimeState 和 JSON 往返，不调用真实模型或网络；禁用 Go 下载不等价于全局网络沙箱证明。

## 授权与取证

用户明确批准该命令后，保存此次授权来源与精确范围，再冻结本地输入版本：go.mod/go.sum、internal/workflow 的生产及测试源码、相关设计/规格与本计划的内容摘要；登记工具链、宿主、实际 argv、工作目录、环境限制、起止时间、退出码和 JSON 输出引用。若使用产品受管 Runner，必须另满足它实际要求的计划/grant 契约；本提案不能替代该 grant。缺少工具链/依赖或测试源码编译失败即如实登记，不联网补齐、不自动扩大范围。

一次首跑；只有相关修复或环境变化后才允许一次重跑。扩大到整个包、模块、集成、真实通知、模型调用或故障注入时另行取得明确授权。

本轮两个测试通过，已保存会话授权和实际证据；其他 AC01–AC34 / AX01–AX05 的完整场景仍有缺口，不据此勾选 2.2/3.1 或直接豁免授权 Gate。没有扩大测试、重跑、操作 Gate 或启动 Supervisor。
