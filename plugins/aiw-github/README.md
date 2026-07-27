# aiw-github

常用 GitHub 命令行插件，使用 `GITHUB_TOKEN` 环境变量鉴权。

## 支持命令

- `create-issue [repo] --title ... [--body ...]`
- `create-pr [repo] --title ... --head ... --base ... [--body ...]`
- `merge-pr [repo] <number> [--message ...]`
- `list-issue [repo] [--state open|closed|all] [--per-page N]`
- `list-pr [repo] [--state open|closed|all] [--per-page N]`
- `get-issue [repo] <number>`
- `get-pr [repo] <number>`
- `update-issue [repo] <number> [--title ...] [--body ... | --body-file PATH]`
- `issue-comment [repo] <number> --body ...`
- `issue-label-add [repo] <number> <label> [label ...]`
- `issue-close [repo] <number>`
- `repo-info [repo]`

如果省略 `repo`，插件会优先通过当前目录的 git `origin` remote 自动解析 `owner/repo`。如果无法解析，会直接报错，要求显式传入。

## 示例

```bash
# 省略 repo，自动从当前 git 仓库发现
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py --json list-issue

# 指定 repo 创建 issue
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py create-issue owner/repo --title "Issue title" --body "desc"

# create or update a long Markdown body without shell escaping
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py create-issue owner/repo --title "Issue title" --body-file body.md
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py update-issue owner/repo 123 --body-file body.md

# 创建 PR
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py create-pr owner/repo --title "My PR" --head feature --base main

# 查询单个 issue / PR
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py get-issue 123
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py get-pr 42

# 给 issue 评论并加标签
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py issue-comment 123 --body "Looks good"
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py issue-label-add 123 bug needs-triage

# 关闭 issue
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py issue-close 123

# 查看仓库信息
GITHUB_TOKEN=... python plugins/aiw-github/aiw-github.py repo-info
```

## 配置

- 推荐使用环境变量 `GITHUB_TOKEN`
- `config.example.toml` 仅作说明；插件当前不自动读取该文件
- 安装 TUI 依赖：`python -m pip install -r plugins/aiw-github/requirements.txt`
- 需要机器可读输出时，使用 `--json`

## 说明

- `list-issue` / `list-pr` 目前只做基础列表输出
- `issue-label-add` 支持一次添加多个标签
- 如果环境中安装了 `rich`，脚本会自动用更友好的表格 / 面板渲染输出；未安装时会回退到 JSON
- `list-issue` / `list-pr` 会显示更清晰的状态、标题、负责人 / 分支
- `get-issue` / `get-pr` / `repo-info` 会以卡片方式展示核心字段
- `--json` 必须放在子命令之前，例如 `aiw-github --json get-issue 123`
- `--body-file -` 从标准输入读取 UTF-8 Markdown
