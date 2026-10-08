# FD-031 Planner 测试授权，第 3 轮

<!-- aiw-data: FD-031-test-authorization-r3.json -->

## 审核结论

批准独立 Tester session `fd031-tester-r2-20261008-4433f97b` 在 `C:\Users\svictor\workspace\tools\ai-tools\aiw\.wt\FD-031` 执行且仅执行：

`python -B -m unittest tests.test_fd031_promote_blackbox tests.test_fd031_recover_worker_blackbox -v`

授权绑定新的 `FD-031-000013-implementation-ready`、FD revision 13、digest `7a5783b35df0209e193dbfc330bcb6af6204b00c7f5235465695ef8fdd16464a`。第 2 轮授权与本修订不匹配，不得复用。

已复查两个测试模块及新增 `test_index_decode_failure_rolls_back_all_recovery_writes`。新用例仅在系统临时 Git 项目中放置无效 UTF-8 的另一个编号 FD，触发索引更新失败；逐字节比较目标 FD、旧回执、索引及事件集合，确认没有遗留新事件。其余 12 例仍检查 promote 与恢复命令公开行为。promote 模块从本地 Go 缓存复制依赖并设置 `GOPROXY=off`、`GOSUMDB=off`，所有构建产物、缓存、Git/FD 夹具和故障文件均限于系统临时目录并由夹具清理。无外部服务、仓库业务文件写入或提权。预计 25–45 秒；Planner 按低风险例外批准这一条聚焦离线命令。未授权其他测试、覆盖率或扩大范围。

Planner：`fd031-host-planner-20261008`。决定时间：2026-10-08 02:58:03 UTC。
