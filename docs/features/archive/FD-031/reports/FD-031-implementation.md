# FD-031 Worker 实现报告

<!-- aiw-data: FD-031-implementation.json -->

## 结果

恢复 `aiw issue promote <id>` 与 `aiw req promote <id>`。命令要求正式批准，使用 Issue 标题和 ID 调用 `aiw fd new`；FD 插件负责创建编号 FD、拒绝重复关联并生成 Planner handoff。该路径不创建 Task，也不更新 `[promotion]` 元数据。

## 改动范围

- `cmd/aiw-req/`：增加 promote dispatch、用法说明、批准校验和 FD CLI 调用；更新聊天指引。
- `docs/usage/`、`skills/issue-management/SKILL.md` 和 `openspec/specs/requirement/spec.md`：统一说明 promote 创建 FD 和旧元数据保留规则。
- `docs/features/FEATURE_INDEX.md`：保留 FD-031 索引项，撤销 FD 创建器同时引入的重复历史行。

## 验证与风险

- `git diff --check` 通过；静态复核命令接线、批准状态检查、标题与 ID 参数传递、子进程错误传播和元数据不变更路径。
- `go build -o NUL ./cmd/aiw-req` 编译通过，未保留构建产物。
- 未运行测试。没有运行时执行 `aiw issue promote`；重复关联、CLI 定位和子进程失败路径仍需 Tester 验证。

## 来源

- Source event：`FD-031-000003-design-ready`。
