# FD-027 测试授权（第 2 轮）

<!-- aiw-data: FD-027-test-authorization-r2.json -->

## 审核结论

首次执行同一聚焦命令得到 9 项中 7 项通过、2 项失败。失败原因是
临时仓库夹具未忽略 `.ai/` 和 `.wt/`，使待验证命令看到夹具造成的
未跟踪目录。Tester 仅在 `setUp` 增加并提交临时仓库的 `.gitignore`；
测试场景和执行命令未改变。

Planner 已阅读修订处，并确认测试文件新 SHA-256：
`594DB3EAFC9E223286597516D34CD7F8362494D1E68DF71F0AB7FACD1A4933FC`。
准许在 `D:\03_projects\AI-tools\aiw\.wt\FD-027` **仅重跑一次**
`python -B -m unittest tests.test_fd027_wt_blackbox -v`，预计 10 秒以内。
测试仍只写系统临时目录，隔离 Git 配置、模板和 hooks，离线且没有
提权或发布产物；原 7/9 结果必须在 Tester 报告中保留。
