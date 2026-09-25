# E05 模型与保存共享恢复额度验证

Task workflow-automation；wi-0012 / 2.2；实现来源 E05 / wi-0007。状态：新增三个测试及单次取证入口，尚未运行。

依据 task-memory/spec.md 的 SW19、R1 和 R3，补充 auxiliary_save_recovery_windows_test.go。复用已有临时 Store、模型观察工件和服务替身，先持久保存有效结果，再构造“发布预约已提交、后续输出/完成提交中断”的状态。生产 Store 没有可控磁盘故障注入接口，本组明确测试保存后的恢复状态，不声称真实 I/O 错误或进程终止已发生。

| 用例 | 断言 |
| --- | --- |
| TestAuxiliarySaveRecoverySurvivesReopen | 首次保存预约保留但输出缺失，重开 Store/其他消费者复用原账；重存只耗一次额外恢复，保持原输出，不新增模型调用，完成重放不新增提交 |
| TestAuxiliaryModelRecoveryLeavesNoSaveRetry | 已用模型恢复且首次保存未完成时，不再分配保存重试；状态为 unavailable、保留两次模型账与失败理由，关闭队列但不改变开发/Gate/交付 |
| TestAuxiliaryStoredOutputReconcilesWithoutExtraRecovery | 输出已落盘但完成记录尚未提交，重开 Store 复用准确内容寻址工件；即使模型恢复已用，也可对账完成，不新增保存次数或调用 |

## 执行

正常终端一次执行，预计 1–3 分钟：

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e05-save-recovery-terminal.py
```

精确命令：

```powershell
go test ./internal/workflow -run '^(TestAuxiliarySaveRecoverySurvivesReopen|TestAuxiliaryModelRecoveryLeavesNoSaveRetry|TestAuxiliaryStoredOutputReconcilesWithoutExtraRecovery)$' -count=1 -timeout=60s -vet=off -json
```

GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local；仅本组三项，不联网、不调用模型或后台宿主、不改实际 Task 账本。脚本保存独立计划快照、输入清单、环境、工具链、stdout/stderr 和退出码；不自动重试。相关修复后才允许原范围重跑一次，不扩大至全包。

## TODO 与 Verification

2026-09-25 离线执行 `python scripts/compile.py`，退出 0；临时产物已清理。仅覆盖生产程序，不编译新增 `_test.go`。本组三项尚未执行；未运行 formatter/lint/vet、最终制品构建、模型、网络或 Git 写入。

- [x] 编写三个恢复边界测试及单次取证入口。
- [ ] 执行并核对本组证据；此前其他组授权不自动扩展。

%% REMAINING: 本组仅使用 memory 操作检查共用队列实现；知识提取/汇总和 Verifier 各自保存故障、真实磁盘错误、真实进程崩溃、赞助者 Stop 和全部资源边界仍待覆盖。构造中断快照不证明实际错误发生时能正确到达该状态。
