# GitHub Issues 管理

使用 `github-issues-management` Skill，通过自然语言描述 Issue 目标。Skill 会先解析 `owner/repo`，对已有 Issue 先读取当前远程状态，再准备写入动作。

所有创建、更新、关闭、评论和标签操作都需要用户明确回复 `confirm` 或“确认”后才执行。

常见操作由 `aiw-github` 提供：

```text
list-issue / get-issue
create-issue
update-issue
issue-close
issue-comment
issue-label-add
repo-info
```

示例对话：

```text
用户：把 Issue 12 的标题改成“修复报表时区问题”
AI：我先读取 owner/repo 的 Issue 12 当前标题、正文、状态和标签，然后展示更新摘要并等待确认。
用户：确认
AI：调用 aiw-github update-issue，并返回 Issue URL。
```

如果无法从 Git `origin` 唯一解析仓库，必须提供明确的 `owner/repo`；不能猜测目标仓库。GitHub 请求需要 `GITHUB_TOKEN`。
