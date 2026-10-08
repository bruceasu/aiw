# FD-038 Planner 测试授权，第 1 轮

<!-- aiw-data: FD-038-test-authorization-r1.json -->

## 审核结论

批准仅执行以下精确命令：

`python -m unittest discover -s tests -p test_fd038_cli_blackbox.py -v`

- 绑定事件：`FD-038-000005-implementation-ready`
- FD Revision / 摘要：`5` / `3baf0eafa1afb625b6e3f8a4e59dc76f265c7ae2dc7294e3026cf27169521baf`
- Tester Session：`fd038-tester-20261009-7d32c91e`
- 工作目录：`D:\03_projects\AI-tools\aiw\.wt\FD-038`
- 范围：只运行 `tests/test_fd038_cli_blackbox.py` 中的 8 个黑盒场景；预计 5–15 秒。

已检查测试代码。它只使用 Python 标准库 `unittest`，通过 PATH 中的 `aiw` CLI 调用查询命令；不会访问网络、下载依赖或调用外部服务。测试在 FD worktree 的 `docs/features/reports/` 与 `docs/features/reviews/` 写入带 UUID 的临时 Markdown/JSON fixture，并在 teardown 删除；写入前检查目标路径不存在。`fd show` 场景只读取 FD、索引、workspace 与 receipt，并比较调用前后字节。

副作用限于该 FD worktree。若测试进程被强制终止，teardown 可能来不及清理临时 fixture；若发生中断，应先检查并仅清理带 `FD-038-blackbox-` 唯一标记的遗留文件。本授权不覆盖其他命令、文件、revision、session 或覆盖率工具。

## 决策

- Planner：`fd038-planner-20261008-9c4d21`
- 决策时间：`2026-10-08T15:07:12+00:00`
- 依据：命令聚焦、离线、可检查，所有预期写入均限于指定 worktree 的唯一临时 fixture。
