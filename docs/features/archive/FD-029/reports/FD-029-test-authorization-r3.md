# FD-029 Planner 测试授权（第 3 轮）

<!-- aiw-data: FD-029-test-authorization-r3.json -->

用户在本次对话明确批准新增重复交付用例后再运行一次聚焦命令。Planner 检查了 `D:\03_projects\AI-tools\aiw\.wt\FD-029\tests\test_fd029_wt_squash.py`，SHA-256 为 `E7146235127D79DFDC6C020F6874AEF638E8037F4CF41C97EDB49441FA5D2A0F`。六项用例在系统临时目录建立隔离 Git 仓库和 worktree，禁用全局/系统 Git 配置、提交钩子与终端交互；公开 CLI 均在临时仓库运行，清理临时目录。不访问网络、不下载、不提权、不产生发布产物。

准许独立 Tester `fd029-tester-r3-4e7b19c2` 在 `D:\03_projects\AI-tools\aiw\.wt\FD-029` **只执行一次**：

`python -B -m unittest tests.test_fd029_wt_squash -v`

预计 6–20 秒。授权绑定 `FD-029-000015-implementation-ready`、FD revision 15、摘要 `8430f6b55b4dcdc8f41850215880546228dfb3826f77776b6275cccd37672c6b` 和上述测试文件。代码或环境再次改变时不得沿用。Planner：`fd029-worker-r3-root`；时间：`2026-10-06T15:45:12+00:00`。
