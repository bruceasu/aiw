# E01 输入与报告聚焦验证计划

Task：workflow-automation；登记工作项 wi-0012 / 2.2；实现来源 E01 / wi-0001。设计 FD_APPLIED，当前只补验收证据，不重开已完成实现项。状态：用户正常终端执行后，五个指定用例全部 passed，退出 0，见 [原始结果与报告](verification-results/e01-terminal-20260924T092824207444Z/report.md)。

## 范围与断言

依据 design.md、specs/verification/spec.md 的 SW03/SW04，以及 E01 实施记录；使用 internal/workflow/execution_inputs_test.go 的五个已有用例。

| 用例 | 本轮验证内容 |
| --- | --- |
| TestValidationInputsTrackContentAndDiscovery | 同文件等长内容变化使旧证据失效；新增/删除 fixture 被发现；运行日志不改变代码摘要；恢复相同文件集合时摘要确定 |
| TestReportSectionsDistinguishMissingEmptyAndUnknown | 缺失或含糊字段拒绝；有理由的空、未知、未验证明确区分 |
| TestImplementationReportBindsGenerationAndChangedBytes | 有效报告接受；错误 turn 或旧内容摘要拒绝 |
| TestLegacyEvidenceDoesNotCreateValidationOrAuthorization | 旧未知证据须对账；缺验证授权暂停；只缺报告可补报告；下游须已接受；工具链变化使证据失效 |
| TestUnknownScriptEnvironmentCannotReusePassedEvidence | 不完整工具链/脚本环境身份不能作为可复用通过 |

提供 SW01/SW03/SW04 及相关 AC 的部分证据，不证明逐轮报告持久恢复、一次补交预算、真实模型输入注入、完整下游链或生产联合启用。

## 执行方式

两个文件用例创建和删除自身临时 fixture，经过 CaptureValidationInputs/ValidateImplementationReport 的 confinedFile 路径校验。当前工具环境已观察到 Windows EvalSymlinks 访问拒绝，正常终端的 E02 对照已越过该限制；本轮准备正常终端入口，不在受限环境试跑或绕过路径安全检查。

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e01-terminal.py
```

脚本只执行以下固定命令，无选择参数。Go 会编译包内其他测试源码，但只运行以下五项。

```powershell
go test ./internal/workflow -run '^(TestValidationInputsTrackContentAndDiscovery|TestReportSectionsDistinguishMissingEmptyAndUnknown|TestImplementationReportBindsGenerationAndChangedBytes|TestLegacyEvidenceDoesNotCreateValidationOrAuthorization|TestUnknownScriptEnvironmentCannotReusePassedEvidence)$' -count=1 -timeout=60s -vet=off -json
```

工作目录固定现有 worktree；GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local；使用 worktree Go 缓存。预计含编译 1–3 分钟，测试阶段限时 60 秒。无依赖下载、模型/Teams、Git 写入、真实 Task 或 grant 修改。不提权、不调整 ACL、不重设系统 TMP/TEMP。

操作者显式调用脚本即执行本组一次；脚本不自动重试。相关源码/环境修复后才允许同范围的一次重跑，其他用例、全包或网络需另行批准。原 E02/E03 批准不自动扩大到本组。

## 证据与限制

脚本保存独立证据目录、执行前计划快照、Go 源码/模块/脚本摘要、实际 argv/cwd、工具链、环境、时间、stdout/stderr、退出码与输入变化。终端输出 Evidence 和 Exit code。不得把 invocation.json 当作产品 Runner grant；不自动创建 Attempt、改 Gate 或接受整个 2.2。

用户已运行本组一次且全部通过；无需修复或重跑。助手核对结果并更新覆盖记录，没有重复执行测试、编译或修改生产代码。

%% REMAINING: 完整 AC/AX 与生产联合启用仍缺证据，正式 2.2/3.1/3.2 保持未完成。
