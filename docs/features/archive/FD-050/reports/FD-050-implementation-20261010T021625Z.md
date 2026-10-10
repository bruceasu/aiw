# FD-050 Worker 修正报告

<!-- aiw-data: FD-050-implementation-20261010T021625Z.json -->

## 结果

已处理 Reviewer 在 `FD-050-000008-changes-requested` 中提出的两项问题，并准备重新提交独立审查。

1. `src/internal/plugin/discover.go` 现在按调用方提供的目录顺序逐个搜索。首个包含匹配插件的目录会胜出；扩展名优先级只用于该目录内部的候选项。只有所有目录均未命中时才搜索 PATH。因此已安装目录中的插件不会再被后续源码目录里的 `.py` 候选覆盖。
2. `.github/copilot-instructions.md` 的 Go CLI 路由已从根级 `cmd/` 更新为 `src/cmd/`。

## 验证

- 编译型检查：`python scripts/compile.py`，退出码 0；脚本将编译输出写入 NUL，没有保留最终产物。
- 静态检查：检查了修正后的插件搜索顺序和命令树指引差异。
- 未运行测试、构建或安装动作、格式化、lint、vet 或完整验证脚本。

## 关联证据

- 前次独立审查：`docs/features/reviews/FD-050-review-20261010T030000Z.md`
- 前次审查事件：`FD-050-000008-changes-requested`
- 实现提交：待完成本报告和修正内容提交后记录在 FD handoff 中。
