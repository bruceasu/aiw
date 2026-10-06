# FD-027 测试授权（第 3 条，R2 修复验证）

<!-- aiw-data: FD-027-test-authorization-r3.json -->

## 审核结论

Planner 已阅读 R2 测试差异。新增 3 例仅改变系统临时仓库的
`.gitignore`，分别构造 `.wt/` 与 `.ai/` 都未忽略、仅 `.wt/` 未忽略、
仅 `.ai/` 未忽略，并断言 `wt add` 在未建分支、worktree 或 metadata
之前拒绝且 parent 保持干净。测试文件 SHA-256 为
`9867222CF909D97DB7026D541EA0C0671749FAE0FBA75DC521EBAF1BCE243D69`。

仅准许 Tester 会话 `fd027-tester-20261006-r2-5f8c1d` 在
`D:\03_projects\AI-tools\aiw\.wt\FD-027` 执行一次精确命令
`python -B -m unittest tests.test_fd027_wt_blackbox -v`，预计 15 秒以内。
该命令只在系统临时目录创建和清理隔离 Git 仓库及 worktree；Git 配置、
模板和 hooks 隔离，离线、无下载、提权或发布产物，`-B` 不写 pycache。
授权绑定 `FD-027-000008-implementation-ready`、revision 8、其 digest
及上述 Tester 会话。测试代码或命令变化需重新审核。
