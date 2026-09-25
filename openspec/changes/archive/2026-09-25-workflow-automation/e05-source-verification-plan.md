# E05 来源与固定输入聚焦验证

Task workflow-automation；wi-0012 / 2.2；实现来源 E05 / wi-0007。状态：用户 confirm 授权后，2026-09-25 三个测试及四个子场景全部通过，见 [结果](verification-results/e05-source-terminal-20260925T005720742474Z/report.md)。

依据 task-memory/spec.md 的 SW18/SW19、R1 来源账和 R3 完整输入限制；不修改生产行为或启用 schema。新增 internal/workflow/auxiliary_source_test.go，直接验证来源及作业构造契约，不使用真实 Store、后台进程、模型、网络或凭据。

| 用例 | 契约与断言 |
| --- | --- |
| TestAuxiliarySourceTracksMaterialFactsOnly | revision/事件序号及未执行 Session 请求不新增来源；实质工作状态变化产生新版本；仅勾完成但无接受证据仍是进展；有接受引用才成为 completed；旧快照不被后续事实改写 |
| TestAuxiliaryJobIdentitySharedAcrossSponsors | 相同来源/操作/固定输入在不同消费 Task 下拥有相同 key、prompt 和摘要，保持来源所有者与各自 sponsor；指令、操作或来源变化改变作业标识 |
| TestAuxiliaryOversizeInputIsUnavailableWithoutTruncation | 四种辅助操作均对超限完整输入标 unavailable、保留理由与全文摘要；通知、报告补交和诊断不得混入共享辅助恢复账 |

第二项只验证共享来源账使用的确定性标识，不证明跨 Task 持久预约/恢复额度的实际共享。第三项只验证字节上限，不证明模型 token 测量或能力记录。memory 发布、迟到结果、Stop、真实宿主、持久恢复和资源累计仍须其他场景。

## 执行

正常终端显式执行一次，预计 1–3 分钟：

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e05-source-terminal.py
```

脚本固定命令：

```powershell
go test ./internal/workflow -run '^(TestAuxiliarySourceTracksMaterialFactsOnly|TestAuxiliaryJobIdentitySharedAcrossSponsors|TestAuxiliaryOversizeInputIsUnavailableWithoutTruncation)$' -count=1 -timeout=60s -vet=off -json
```

GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local；测试阶段限时 60 秒，Go 编译同包测试源码但只运行三项。不自动重试；相关修复后才允许同范围重跑一次，不提权、联网或扩包。此前其他组批准不自动扩展到本组。

独立目录保留计划快照、源码/模块/脚本摘要、argv/cwd、工具链/环境、时间、stdout/stderr、退出码和输入变化。终端输出 Evidence 与 Exit code；不生成产品 Runner grant 或操作 Core/Gate。

## 当前检查

新增测试后执行离线 python scripts/compile.py，退出 0；脚本清理临时产物。该命令只编译生产程序。2026-09-25 根据用户 confirm 执行上述取证脚本一次，Go 编译同包测试源码并仅运行指定三项及四个子场景，全部通过，退出 0。未执行真实宿主、模型、网络、Git 写入或生产迁移。

%% REMAINING: 局部契约已有通过证据；持久队列、恢复额度、资源累计、Stop/迟到结果及 memory 发布仍待验证，不代表完整 E05、SW09/18/19/27 或 AC/AX 验收。
