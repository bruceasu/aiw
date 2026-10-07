# FD-030 Planner 测试授权，第 1 轮

<!-- aiw-data: FD-030-test-authorization-r1.json -->

## 审核结论

批准 Tester 对以下单条命令执行一次测试。批准绑定 FD-030 revision 10、implementation event `FD-030-000010-implementation-ready`、digest `339471324719e2421b025959ec3d25acf0193aab2f741e353329f0e89be9e4ec` 和 Tester session `fd030-tester-20261008-a71c9e`。不得扩大测试文件或命令参数。

```powershell
python -c "import unittest; names=['test_fd027_wt_blackbox','test_fd029_wt_squash','test_fd030_conflict_recovery','test_fd030_worktree_blackbox','test_fd030_junction_cleanup']; suite=unittest.TestSuite(unittest.defaultTestLoader.loadTestsFromName(n) for n in names); result=unittest.TextTestRunner(verbosity=2).run(suite); raise SystemExit(not result.wasSuccessful())"
```

- 工作目录：`D:\03_projects\AI-tools\aiw\.wt\FD-030`
- 范围：五个列明的 FD-027/FD-029/FD-030 CLI 黑盒测试模块；覆盖 FD worktree 创建、校验、squash 交付、冲突恢复、清理和 junction 证据保留场景。
- 预计时长：约 1–5 分钟。内含一次离线 Go CLI 构建（超时 180 秒）；测试调用本地 Git/CLI。
- 已检查副作用：测试中的 Git 仓库、worktree、branch、Go cache 和可执行文件均建于系统临时目录或 `.wt/FD-030` 内的自动清理临时目录；命令环境关闭 Go module 下载及 checksum 网络访问，禁用 Git 系统/全局配置并使用空 hooks 目录。无网络、仓库源文件修改或发布产物保留。失败中断时，可能留下 `.fd030-*` 临时目录，需由 Tester 报告。
- 风险审核：命令范围明确、离线且可检查；所有被测交付/删除操作都针对各测试新建的 disposable Git repository。新 junction 测试在当前平台未创建链接时会报告 skipped，不把跳过计为通过覆盖。
- 人工批准：不需要；按低风险、离线且限定临时目录的 Planner 授权执行。

此授权仅允许上述精确命令运行一次。它不授权重复运行、其他测试/覆盖率工具、构建发布物或网络操作。
