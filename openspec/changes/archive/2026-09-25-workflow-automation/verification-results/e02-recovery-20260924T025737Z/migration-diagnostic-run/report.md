# E02 迁移内部诊断运行

Task：workflow-automation；wi-0012 / 2.2。用户明确批准本命令一次；本目录保留 authorization.json、approved-plan.md、inputs.json、run.json、stdout.jsonl 和 stderr.txt。

## 实际结果

2026-09-24T07:02:27Z 至 07:02:32Z，在既有 worktree 执行：

```text
go test ./internal/workflow -run ^(TestDurableUnknownAndStopRetainWriter|TestDurableResultReplayConsumesOnceAndKeepsAttempt|TestDurablePendingTailRecoveryKeepsRevision)$ -count=1 -timeout=60s -vet=off -json
```

命令、GOPROXY=off/GOSUMDB=off/GOTOOLCHAIN=local 和工作区缓存与首跑相同。退出码 1；三个测试均失败；stderr 为空；登记的输入在执行期间未变化。

三个用例均返回相同位置的错误：

```text
migrate fixture execution: migration: read activation artifact "reports/protocol/b962cf4bfbc038f4de128a03dd9e20b7f58670c7c4e8a4292c02d2912e252ced.json": Access is denied.
```

这证明失败发生于 MigrateDurableExecution 的 ReadExecutionArtifact 调用，早于 VerifyActivation、迁移来源保存、预算迁移和迁移状态/事件提交。fixture 此前成功创建了该激活工件，但不能据此推定后续路径解析或读取必然成功。

## 静态分析与限制

ReadExecutionArtifact 先调用 confinedFile，对 Task 根目录和目标执行 filepath.EvalSymlinks，再检查归属并读取正文。本机 C:/green/go/src/path/filepath/symlink_windows.go 中，EvalSymlinks 经 toNorm/normBase 逐级规范化名称，normBase 的 FindFirstFile 失败会直接返回系统错误，符合本次不带路径的 Access is denied。正常 os.ReadFile 的文件错误带 PathError 上下文。

因此路径规范化/目录查询权限是当前有依据的疑点；尚未通过运行证据确定根目录还是目标路径解析失败、具体被拒绝的路径或权限来源。不将该推断写成已证实的沙箱缺陷，也不按猜测替换安全路径校验、放宽 ACL 或改变持久化协议。

本轮没有源码修改，没有重复编译或再次测试；未更换 TMP/TEMP、shell、运行用户或沙箱，也未提权、联网或操作真实 Task schema。静态读取及文档更新不解除现有 Gate。

%% BLOCKED: 三个 E02 用例仍未通过。下一步需要获授权的运行环境提供路径查询权限诊断，或在能满足既有路径安全检查的正常终端取证；不继续无变化地重跑同一命令，不跳过 EvalSymlinks。2.2/3.1/3.2 保持未完成。
