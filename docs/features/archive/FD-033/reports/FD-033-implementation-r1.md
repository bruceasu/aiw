# FD-033 实施报告（第 1 版）

<!-- aiw-data: FD-033-implementation-r1.json -->

## 结果

已完成 FD-033 的四个 Work Item。`aiw git aic` 会暂存全部改动、通过 CZ 配置的 provider 生成 Conventional Commit 消息并提交；`aiw git air` 审查暂存差异；`aiw git aib [--base REF]` 总结指定基准到 HEAD 的单行提交历史，默认基准为 `main`。

我从 `aiw-cz` 抽取了通用文本生成接口。现有 `aiw cz` 继续使用原结构化候选解析、预览和人工确认路径。新增命令由插件元数据自动发现，并已补充 CLI 稳定规格及中文使用文档。

## 验证

- Python 编译检查通过：首次覆盖三个 CZ 模块及四个 Git 模块；修正 `aib` 基准 ref 处理后，对 `git-aib.py` 执行了一次获准的编译重试并通过。
- 已静态检查 Git 参数传递、AI 输出与错误流、失败退出及只读/写入边界，并审阅最终源码差异。
- 未运行测试、真实 AI provider 调用或 Git 提交场景。仓库默认不授权测试和运行验证，因此 Tester 的可观察场景与缺口仍待独立记录，不能视为运行通过。

## 变更路径

- `plugins/aiw-cz/cz_llm.py`
- `plugins/aiw-cz/cz_providers.py`
- `plugins/aiw-cz/cz_openai.py`
- `plugins/aiw-git/aiw-git-ai.py`
- `plugins/aiw-git/git-aic.py`
- `plugins/aiw-git/git-air.py`
- `plugins/aiw-git/git-aib.py`
- `docs/usage/aiw-git-ai.md`
- `openspec/specs/cli-and-plugins/spec.md`
- `docs/features/FD-033_AI_ASSISTED_GIT_COMMIT_REVIEW_AND_BRANCH_SUMMARY.md`

## 残余风险

- Provider API、Codex/Copilot CLI 兼容性及 Git 仓库中的端到端行为尚未运行验证。
- `aiw-git` 通过相邻的 `aiw-cz` 插件目录加载 provider 文件；若实际部署未保持两个插件目录并列，AI 命令会清晰失败，尚未验证各安装布局均满足该约定。
