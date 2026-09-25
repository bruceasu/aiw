# E08 Verifier 报告契约与状态隔离验证

Task workflow-automation；wi-0012 / 2.2；实现来源 E08 / wi-0010。状态：三个测试已编写，尚未运行。

依据 verification/spec.md 的 SW31/SW32 及 R3，新增 verifier_contract_windows_test.go。前两组只检查报告结构；第三组用 E05 临时 Store、固定测试快照和明确的服务替身验证发布与持久读取。测试中的允许证据摘要由 fixture 提供，不证明模型评估的依据充分；不调用模型、Git、网络或生产宿主。

| 用例 | 断言 |
| --- | --- |
| TestVerifierCoverageRequiresEvidenceAndConsistentOutcome | 十个子场景：全有据通过、失败优先、缺证据、全不适用、显式排除；空覆盖、重复项、passed 带缺口、failed 无证据、缺适用性必须拒绝 |
| TestVerifierFrozenReportRejectsMismatchedInput | 十个子场景：Task/Work Item/Attempt/快照不匹配、外部引用、漏项/未知项、结论矛盾、旧 schema、空覆盖均拒绝；正确报告可解析，空报告不通过 |
| TestVerifierNegativePublicationPreservesDevelopmentState | failed/inconclusive 两个子场景：迟到合法负面报告作为 valid 输出保存、重开 Store 可读、仍绑定旧快照、不增加模型恢复调用、不改变现有 Work Items/Attempts/Gates/写入权及 planning/delivery |

执行一次，预计 1–3 分钟：

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e08-verifier-terminal.py
```

固定命令：

```powershell
go test ./internal/workflow -run '^(TestVerifierCoverageRequiresEvidenceAndConsistentOutcome|TestVerifierFrozenReportRejectsMismatchedInput|TestVerifierNegativePublicationPreservesDevelopmentState)$' -count=1 -timeout=60s -vet=off -json
```

GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local；编译同包测试，仅运行本组三项。不自动重试、不复用 E06 授权、不生成产品 Runner grant。保留冻结计划、输入、环境、工具链和原始结果。

## TODO 与 Verification

2026-09-25 离线执行 `python scripts/compile.py`，退出 0；临时产物由脚本清理，仅覆盖生产程序，不编译新增测试文件。新增测试/脚本仅静态核查，尚未执行本组。沿用 aiw patch 内部 Git apply 不可用时的直接补丁回退，没有 Git 写入、网络或模型调用。

- [x] 编写三项测试与单次取证入口。
- [ ] 取得本组授权及运行结果。

%% REMAINING: 测试没有经生产接受路径建立已完成 Task，不能证明接受/Task 完成端到端隔离；真实 Git diff/需求/receipt 封存、首次接受来源登记、独立模型派发及语义审查、保存故障和完整资源边界仍待验证。不宣称完整 AC31/AC32 或联合启用。
