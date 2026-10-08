# FD-030 Planner 测试授权 R2

<!-- aiw-data: FD-030-test-authorization-r2.json -->

## 审核结论

批准 Tester session `fd030-tester-20261007-7c56c2` 在一次针对测试 harness 权限修正后，重跑以下唯一命令：

`python -B -m unittest tests.test_fd030_worktree_blackbox`

工作目录为 `D:\03_projects\AI-tools\aiw\.wt\FD-030`。该命令此前已执行一次，但在 `setUpClass` 创建系统 TEMP 下的 CLI 目录时收到 `WinError 5`，Ran 0 tests。修正版将 Go runtime 和 Git fixture 的临时目录显式放在当前 FD worktree 下；本次授权只允许这一次重跑。

本次范围仍限于三个黑盒 CLI 场景：独立 `aiw wt` 入口不可用、交付失败时保留 worktree/分支、成功交付后清理 worktree/分支并保留收据及 `FD-Source`。预计耗时 2–4 分钟。

Go 二进制、Go build cache、插件副本、Git 仓库、worktree、分支、hooks 和 templates 均位于 `.wt/FD-030` 下的临时目录，并由 `TemporaryDirectory` 清理。测试清除了继承的 `GIT_*` 与 `GO*` 覆盖变量；使用 `-mod=readonly`、`GOWORK=off`、`GOENV=off`、`GOPROXY=off`、`GOSUMDB=off` 和 `GOTOOLCHAIN=local`，只读取已有本机模块缓存，缺依赖时离线失败，不下载。命令不改实现文件或配置，不访问网络，不保留最终构建产物。Windows junction 的有效目标行为会由成功清理场景间接覆盖；未知 reparse 类型/目标、linked-worktree handoff、冲突恢复和来源校验失败仍须报告为未覆盖。

先前系统 TEMP 的清理报错可能留下目录 `C:\Users\suk\AppData\Local\Temp\fd030-aiw-runtime-16wpl8lr`；本授权不包含对该路径的探查或清理。

## 绑定信息

- Implementation event: `FD-030-000006-implementation-ready`
- FD revision / digest: `6` / `5bb6333ca24b65f58d6fb708312166fecd7367108c54ecdcfdce1f8741474793`
- Tester session: `fd030-tester-20261007-7c56c2`
- Planner: `/root`
- 决策时间：2026-10-07 14:19 UTC
