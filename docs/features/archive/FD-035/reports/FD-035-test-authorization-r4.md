# FD-035 Planner 测试重试授权 r4

<!-- aiw-data: FD-035-test-authorization-r4.json -->

## 授权范围

授权 Tester `fd035-tester-20261008-a7d92e` 在 `C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-035` 对修复后的 S20 测试运行一次：

```powershell
python -B -m unittest tests.test_fd032_risk_decision_blackbox -v
```

本次授权绑定实现事件 `FD-035-000011-implementation-ready`、FD revision 11、digest `359ffa2cb9c8e6e4307a4189549da31a66bb69d6f9aca12bcb736fd6492b7017` 及 Tester session。此前同一命令首次运行有 2 个 S20 子场景因测试夹具重复创建 handoff 而失败；Tester 仅调整测试夹具为复用一个 PM handoff，并在每个子场景还原决策文件。审阅确认重试仍只运行该离线黑盒测试模块，文件副作用限于测试自建临时仓库及文件，正常结束自动清理。预计少于 1 分钟。此次是相关测试代码修复后唯一重试；不包含覆盖率命令或依赖安装。
