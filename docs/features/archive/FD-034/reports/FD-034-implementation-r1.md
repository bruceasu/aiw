# FD-034 实现报告 r1

<!-- aiw-data: FD-034-implementation-r1.json -->

Worker 会话：`fd034-host-20261008-5c8b71`。来源事件：`FD-034-000005-design-ready`。

## 已实现

- 新增 `aiw git patch create FILE.patch [--staged | --worktree]`：导出已跟踪改动，拒绝空差异和覆盖，并写同名 Markdown，内含范围、改动统计、应用步骤和失败建议。
- 新增 `aiw git patch apply FILE.patch`：先执行 `git apply --check`，失败时显示 Git 错误并通过反向预检提示可能已应用；成功后只改工作区。
- 更新使用文档、CLI 稳定规格及 FD 的 Work Items、TODO 和 Verification。

## 证据和限制

静态查看了参数分发、Git 调用、二进制数据传递、文件写入、预检与错误路径，以及文档契约。`python -m py_compile plugins/aiw-git/git-patch.py` 在最终代码上通过；编辑过程中还执行过一次相同检查。

未运行测试、实际 Git patch 命令、AI 调用、最终构建、格式化或 lint。创建与应用行为、Windows 路径和错误提示尚无运行证据；需要独立 Tester 报告未覆盖场景或在获得具体命令授权后执行聚焦验证。
