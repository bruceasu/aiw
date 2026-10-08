# FD-035 Planner 测试授权：第 3 轮

<!-- aiw-data: FD-035-test-authorization-r3.json -->

## Planner 决定

批准 Tester `fd035-tester-20261008-a7d92e` 在 `C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-035` 执行以下聚焦黑盒测试命令一次：

```powershell
python -B -m unittest tests.test_fd032_risk_decision_blackbox -v
```

授权绑定实现事件 `FD-035-000011-implementation-ready`、FD 修订 11、摘要 `359ffa2cb9c8e6e4307a4189549da31a66bb69d6f9aca12bcb736fd6492b7017` 和上述 Tester session。范围为新增 S20–S25 与同模块原有用例，预计少于 1 分钟。

已检查测试模块及其调用的 `tests.test_fd014_blackbox`：夹具将 CLI 与模板复制到各自的系统临时目录，初始化临时 Git 仓库并在结束时清理；命令和子进程不写 Python 字节码。预期工作区外无持久写入，不访问网络或外部服务，不下载依赖、不提权、不生成发布产物。风险为异常中断可能留下临时目录。此命令聚焦、离线且副作用可检查，Planner 批准，无需人工批准。

本授权不包含分支覆盖率命令；当前 Python 环境没有 coverage.py，后续覆盖率命令必须单独授权。
