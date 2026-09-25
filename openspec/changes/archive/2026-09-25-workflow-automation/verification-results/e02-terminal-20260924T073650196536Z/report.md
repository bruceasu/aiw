# E02 正常终端结果与测试数据修复

用户在正常 PowerShell 终端执行 run-e02-terminal.py，并提供本目录与退出码 1。已读取本地原始 run.json、stdout.jsonl 和 stderr.txt。

## 实际结果

Go 1.25.1 windows/amd64；三个指定用例；离线配置；2026-09-24T07:36:50Z 至 07:36:51Z。退出 1，stderr 为空，执行期间输入摘要未变。

| 用例 | 结果 |
| --- | --- |
| TestDurablePendingTailRecoveryKeepsRevision | passed，0.04 秒 |
| TestDurableUnknownAndStopRetainWriter | failed：prepare fixture stage: frozen Actor input differs from the requested generation |
| TestDurableResultReplayConsumesOnceAndKeepsAttempt | failed：同上 |

本次在原默认 Temp 中成功执行迁移及来源读取，不再出现工具环境中的 EvalSymlinks 拒绝；说明该拒绝在本次正常终端环境未复现，不能归结为产品路径校验必然失败。两个剩余失败来自独立的 fixture 数据缺陷。

## 静态根因与修复

preparedStageFixture 的 ExecutionInput 未设置 AISelection，持久化后为 nil；对应 StageRequest.Model 却是非空 test 选择。PrepareStage 按已确认的固定输入契约校验 equalJSON(frozen.AISelection, request.Model)，因此正确拒绝不一致输入。

已仅修改 worktree 的 internal/workflow/protocol_windows_test.go：定义一份 selection，在 PersistExecutionInput 之前写入 input.AISelection，并用于 StageRequest.Model。保持生产校验、测试断言、请求身份和预算逻辑；不将错误改成跳过，也不弱化校验。

修改后离线 python scripts/compile.py 退出 0；该检查只覆盖生产编译，不证明修改后的测试源码通过。未在已知路径权限受限的工具环境重复运行测试。

## 下一步

仍使用已准备的 run-e02-terminal.py，在用户正常终端同命令重跑一次。源码已发生相关修复，脚本自动保存新的目录与摘要，不覆盖本次失败证据；测试范围不扩大、不联网、不提权。

当前仅事件尾部恢复取得该输入版本的通过证据，不代表三个用例全部通过、真实进程崩溃或整体验收完成。Gate 与 2.2/3.1/3.2 保留。

%% PENDING: fixture 修复尚待正常终端运行验证；当前报告不宣称本轮新代码已经测试通过。
