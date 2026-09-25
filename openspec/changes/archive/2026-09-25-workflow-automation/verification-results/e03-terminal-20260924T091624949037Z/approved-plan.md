# E03 授权日志聚焦验证计划

Task：workflow-automation；登记 Work Item：wi-0012 / 2.2；来源实现项 E03 / wi-0005。状态：已完成静态核对和取证准备，等待本组测试的单独授权。3.1/3.2 仍依赖完整 2.2；不重开已经完成的实现项或按历史 wi-0003 错绑派发 E03。

## 范围

依据 design.md 的 FD_APPLIED、r2-authorization-design.md、specs/verification/spec.md；测试源码为 internal/workflow/grant_log_test.go，实现为 grant_log.go。

| 用例 | 实际断言 | 局限 |
| --- | --- | --- |
| TestGrantLogRejectsAmbiguousOrMissingDecisionFields | 有效日志可读；重复键、未知/缺失字段、null、尾随内容、多区块、未来版本、跨 Task 被拒绝 | 不证明日志持久化或截断恢复 |
| TestGrantDenialInvalidatesOldApprovalAndRequiresExplicitReplacement | 后续拒绝使旧批准失效；新批准缺少拒绝记录的显式替代仍无效；完整替代才允许 | 内存日志，不证明并发撤销或真实决策身份核验 |
| TestGrantScopeChangeCannotReuseApproval | 真实目标变化后不能复用原批准 | 本用例未覆盖所有其他范围字段或链接竞态 |

这些是 SW11/SW12 及 AC11/AC12/AX03 的部分授权边界证据，不能替代 E03 独立 Tester、宿主隔离、接受条件或整个测试闭环的运行验收。三个用例只使用内存结构和 JSON/Markdown 字节，不创建真实 Task 或 grant 文件，不调用模型、Teams、Git 或网络。

## 精确命令

工作目录为既有 .wt/workflow-automation；Go 仍会编译包内其他测试源码。

```powershell
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:GOTOOLCHAIN = 'local'
go test ./internal/workflow -run '^(TestGrantLogRejectsAmbiguousOrMissingDecisionFields|TestGrantDenialInvalidatesOldApprovalAndRequiresExplicitReplacement|TestGrantScopeChangeCannotReuseApproval)$' -count=1 -timeout=60s -vet=off -json
```

预计含编译 1–3 分钟，测试阶段限时 60 秒。首次运行一次；有相关修复或环境变化后最多重跑一次。权限失败停止相应路径，不自动换环境、提权或扩大范围。此次授权不覆盖其他用例或全包测试。

## 取证

已准备 run-e03-terminal.py，可在用户明确批准后由工具环境执行，或由操作者在正常终端显式执行。它固定上述范围，无命令参数扩展；一次运行保存独立目录、计划快照、脚本及 Go 源码/模块摘要、实际 argv/cwd、工具链、限定环境、时间、完整 stdout/stderr、退出码及输入前后比较，不自动重试。

脚本不申请或生成产品 Runner grant、不改变 Gate 或 Core。真实会话授权另按实际来源登记；运行脚本自身不能使旧批准变成新批准。

## 后续

获授权后运行、修复真实失败并保留原始结果，再更新覆盖记录。本轮没有执行测试、模型、网络、Git 写入、生产迁移或最终制品构建。脚本仅静态审阅；无需因验证工件变更重复编译生产程序。

%% REMAINING: E03 的宿主隔离、受管计划/执行清单、持久 grant 锚点恢复、真实 Tester 与接受链仍有运行证据缺口；本组即使通过也不单独关闭 2.2 或整体 Gate。
