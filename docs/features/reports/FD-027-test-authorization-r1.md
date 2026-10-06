# FD-027 测试授权（第 1 轮）

<!-- aiw-data: FD-027-test-authorization-r1.json -->

## 审核结论

Planner 准许独立 Tester 仅执行下列一条命令：
`python -B -m unittest tests.test_fd027_wt_blackbox -v`。工作目录为
`D:\03_projects\AI-tools\aiw\.wt\FD-027`，预计不超过 10 秒。

已阅读实际调用的 `tests/test_fd027_wt_blackbox.py`，其 SHA-256 为
`3432A6B6256C283462205A6029D1D9859DAB5B0071E3555A2EDB92BD8D38C864`。
测试只在系统临时目录创建独立 Git 仓库、分支、worktree、提交和 FD
metadata，结束时由 `TemporaryDirectory` 清理。子进程隔离 Git 全局及
系统配置、模板和 hooks；`-B` 禁止 Python 字节码输出。没有网络、下载、
提权或发布产物。授权绑定 `FD-027-000004-implementation-ready`、FD
revision 4、其 digest 和 Tester 会话。若测试代码或命令改变，需重新审核。
