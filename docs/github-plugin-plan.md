## Plan: 增加 GitHub 插件

TL;DR - 在现有插件框架下新增一个 `aiw-github` 插件，使用 GitHub REST API 实现常用工作流（创建 issue/PR、合并 PR、查询/触发 workflow、添加标签）。采用 Python 实现以复用现有插件样式，分步骤迭代交付并覆盖测试与文档。

### Steps
1. 研究插件加载/发现机制，确认插件目录和命名约定。
2. 采用 Python 实现（与现有 `plugins/*.py` 保持一致）。
3. 新建 `plugins/aiw-github/`，包含 `aiw-github.py`、`README.md`、`config.example.toml`。
4. 实现基础功能：
   - 配置读取（token、endpoint）
   - 认证与请求封装（支持 PAT，通过 ENV 或配置注入）
   - 命令：`create-issue`、`create-pr`、`merge-pr`（优先实现）、`list-workflows`、`trigger-workflow`、`add-label`
   - dry-run / 确认标志以避免意外变更
5. 确保 `plugin/discover.go` 能发现新插件（按现有发现约定放置即可）。
6. 添加测试：单元测试请求封装/参数解析；集成测试使用测试仓库或请求回放。
7. 文档：在 `plugins/aiw-github/README.md` 提供使用与配置示例，更新 `aiw.toml.example`。
8. 验证：运行仓库验证脚本与单元/集成测试；手动在带 PAT 的隔离环境测试关键命令。
9. 风险与回退：对合并/触发类操作默认 require `--yes` 或 `--confirm`，并提供 `--dry-run`；记录操作日志。

### Decisions / Assumptions
- 使用 GitHub REST API（简洁、易测试；未来可扩展到 MCP）。
- 使用 PAT；优先级 `ENV > config file`。
- 优先实现 `create-issue`、`create-pr`、`merge-pr`。

### Relevant files
- `plugins/` — 插件目录
- `plugin/discover.go` — 插件发现/加载
- `plugins/aiw-git/aiw-git.py` — 参考实现
- `aiw.toml.example` — 配置示例位置

### Verification
1. 单元测试覆盖请求封装与参数解析。
2. 集成验证：使用测试 GitHub 仓库与 PAT 验证关键命令。
3. 运行仓库现有验证脚本并报告结果。
