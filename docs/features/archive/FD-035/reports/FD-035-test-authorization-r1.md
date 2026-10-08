# FD-035 测试授权：第 1 轮

<!-- aiw-data: FD-035-test-authorization-r1.json -->

## Planner 决定

批准独立 Tester `fd035-tester-20261008-8f4ac2` 在 `C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-035` 执行以下命令一次：

```powershell
python -B -m unittest tests.test_fd032_risk_decision_blackbox -v
```

此授权仅绑定实现事件 `FD-035-000004-implementation-ready`、FD 修订 4 和摘要 `031564c0783d1959717aeec8cf1e3e37b085013c3e9a032a75ea11b7f31ddc97`。预计 10–45 秒。范围是该模块的 7 个黑盒测试方法及其子场景，涵盖单份评审、三份升级及互补视角、投票与 Reviewer 身份隔离、旧决策兼容。

已检查 `tests/test_fd032_risk_decision_blackbox.py` 及其调用的 `tests/test_fd014_blackbox.py`：每个用例在系统临时目录复制 CLI 与 FD 模板，初始化临时 Git 仓库，并在其中生成与清理夹具文件；命令和子进程均禁用 Python 字节码写入。预计不修改当前工作区，不访问网络或外部服务，不下载依赖，也不提权。中断时可能残留系统临时目录。风险为低，Planner 可以直接批准；不需要人工批准。仅在相关代码或环境修正后允许按规则重跑一次，扩大范围须另行审批。
