# FD-014 Planner 测试授权，第 4 轮

<!-- aiw-data: FD-014-test-authorization-r4.json -->

## 审核结论

批准独立 Tester 在仓库根目录执行一次
`python -B -m unittest tests.test_fd014_blackbox -v`。这是一组仅覆盖 FD-014
公开 CLI 契约的 15 个黑盒用例，正常预计 120 秒内完成。

Planner 检查了测试文件、临时 Git 夹具和被调用 CLI 的归档路径。测试从
仓库读取 CLI 入口及 FD 模板，只在各自的临时目录中创建 Git 仓库、FD、
收据、中文报告及 JSON，并在结束时清理临时目录。测试移除了角色运行器
环境变量，不下载依赖、不访问网络或凭据、不提权、不写真实 FD 证据，
也不产生最终构建产物。因此按低风险命令自动授权。

本授权只适用于当前 FD 修订版、Tester 会话和上述精确命令；不包括
覆盖率工具、扩大范围或重复执行。机器字段见同名 JSON。
