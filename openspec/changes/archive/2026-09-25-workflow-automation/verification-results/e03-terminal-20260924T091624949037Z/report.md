# E03 授权边界：首跑通过

Task：workflow-automation；登记 Work Item：wi-0012 / 2.2；对应 E03 实现项 wi-0005。用户回复 confirm 批准计划中的三个指定用例后，助手在工具环境执行 run-e03-terminal.py 一次。目录名沿用脚本命名，不表示本次由用户终端执行。

## 结果

2026-09-24T09:16:25Z 至 09:16:26Z；Go 1.25.1 windows/amd64；退出码 0；stderr 为空；三个顶层用例全部 passed，日志解析用例的七个子场景全部 passed。包运行 0.131 秒，命令含编译约 1.17 秒。

```text
go test ./internal/workflow -run ^(TestGrantLogRejectsAmbiguousOrMissingDecisionFields|TestGrantDenialInvalidatesOldApprovalAndRequiresExplicitReplacement|TestGrantScopeChangeCannotReuseApproval)$ -count=1 -timeout=60s -vet=off -json
```

工作目录为既有 worktree，GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local，使用 worktree 缓存。完整 argv、环境、时间、工具链和摘要保存在 run.json；原始结果为 stdout.jsonl/stderr.txt；运行前计划快照为 approved-plan.md，输入清单为 inputs.json。授权发生在运行前，会话确认的独立文件 authorization.json 在结果归档时补录，明确不作为产品 Runner grant。

## 覆盖与边界

- 有效授权日志可读，歧义、缺字段、未知格式或跨 Task 输入被拒绝。
- 后续拒绝使旧批准失效；新批准必须显式完整替代冲突记录。
- 实际目标变化不能沿用原授权。

它们提供 SW11/SW12、AC11/AC12/AX03 的部分边界证据。用例只处理内存日志，不证明持久 grant 锚点恢复、并发撤销、真实授权身份核验、链接竞态、独立 Tester、宿主隔离或接受链。

本次无需源码修复或重跑；未执行额外编译、网络、最终制品构建、Git 写入或真实 Task 迁移。只更新验证记录及受管清单；完整 2.2/3.1/3.2 和整体 Gate 保留。
