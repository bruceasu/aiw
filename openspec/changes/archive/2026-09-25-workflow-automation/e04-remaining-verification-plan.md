# E04 剩余预算与路由聚焦验证计划

Task workflow-automation；登记 wi-0012 / 2.2；实现来源 E04 / wi-0006。状态：用户正常终端运行后三项及两个子场景全部 passed，退出 0，见 [原始结果与报告](verification-results/e04-remaining-terminal-20260924T093831058329Z/report.md)。此前两个预算用例结果保留，本组补齐另外三项的局部运行证据。

依据 design.md、specs/ai-routing/spec.md 的 SW15–SW17，以及 E04 实施记录。

| 用例 | 本轮断言 |
| --- | --- |
| TestGenerationBudgetSharedSixRepairsAndSuccessfulSixth | Coder/Tester 共用六次修复；第六次成功被消费，后续验证仍可执行；成功或失败均不能派发第七次修复；JSON 往返不刷新预算 |
| TestGenerationRoutingFallbackAliasesAndRecoverySnapshot | 路由回调失败仅调用一次并回落配置；实际模型别名去重；恢复使用旧快照；Compiler 不参与模型路由 |
| TestGenerationBudgetLegacyUnknownAndFutureVersion | 旧来源不明预算保持 unknown；未来版本拒绝写入 |

使用内存 RuntimeState、JSON 往返及路由替身回调，不调用真实模型、网络、Git 或真实 Task，不迁移生产预算。不能以 JSON 往返证明真实崩溃恢复或实际 provider 路由行为。

## 命令

正常终端显式执行一次，预计含编译 1–3 分钟：

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e04-remaining-terminal.py
```

脚本只运行以下固定命令，无范围扩展参数：

```powershell
go test ./internal/workflow -run '^(TestGenerationBudgetSharedSixRepairsAndSuccessfulSixth|TestGenerationRoutingFallbackAliasesAndRecoverySnapshot|TestGenerationBudgetLegacyUnknownAndFutureVersion)$' -count=1 -timeout=60s -vet=off -json
```

GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local、worktree 缓存；测试阶段限时 60 秒。Go 会编译同包测试源码，但只执行三项。首次一次，有相关修复后最多同范围重跑一次，不自动提权、扩包或联网。此前批准不自动扩展到本组；操作者显式运行才执行。

保存独立结果目录、计划快照、源码/模块/脚本摘要、工具链、argv/cwd、限定环境、时间、stdout/stderr、退出码及输入变化。脚本不自动重试或生成产品 Runner grant；当前脚本仅静态审阅，尚未执行。

%% REMAINING: 本组已通过，只补 SW15–SW17/AC15–AC17 的局部证据，生产联合启用、真实模型能力和完整验收仍有缺口。2.2/3.1/3.2 与 Gate 保留。
