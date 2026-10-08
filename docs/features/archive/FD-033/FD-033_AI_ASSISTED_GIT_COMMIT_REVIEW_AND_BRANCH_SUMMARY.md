# FD-033: AI 辅助 Git 提交、审查与分支摘要

**Status:** Complete
**Revision:** 8
**Priority:** Medium
**Effort:** Medium (1-4 hours)
**Impact:** 在 `aiw git` 中提供 AI 生成提交信息、审查暂存差异和概述分支内容的快捷命令。
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

用户希望在 `aiw git` 中提供三个 AI 辅助操作：暂存所有文件并生成 Conventional Commits 格式的提交信息；审查暂存差异；根据分支提交历史生成描述。当前 `aiw-cz` 已有提交信息生成能力，但它是交互式提交向导，要求人工预览和确认后直接创建提交。其 provider 适配器和 CZ 专属配置可复用；候选生成、交互和提交流程不能直接满足这三个命令。

## Options and decision

1. 让用户配置的 Git alias 直接调用 `codex exec`：改动小，但绕过 AIW 插件接口、CZ provider fallback 和现有模型配置。
2. 让 `aiw git` 直接调用 `aiw cz`：会进入交互向导并提交，无法作为只输出审查/摘要的通用调用接口。
3. 在 `aiw-cz` 暴露通用文本生成入口，并让 `aiw-git` 复用其配置与 provider 调用；`aiw-git` 自己构建 Git 上下文并负责提交或只读输出。**选择此方案**，保留 CZ 配置与 provider 顺序，避免重复 provider 代码及绕过安全调用约束。

## Solution

在 `aiw-cz` 抽取可由任意提示词调用的文本生成函数，沿用 CZ 现有配置、provider fallback 和受限 Codex CLI 调用；现有 `cz` 候选生成行为及交互确认保持不变。`aiw-git` 通过本地插件目录加载共享入口，不要求安装新的依赖。

添加 `aiw git aic`、`aiw git air`、`aiw git aib`：

- `aic` 先执行 `git add -A`，将暂存差异交给模型生成简洁 Conventional Commit 信息，再以 `git commit -F -` 创建提交。AI 或 Git 失败时返回非零，不伪造成功；AI 失败时保留已暂存内容，不自动提交。
- `air` 仅读取暂存差异并输出审查意见；无暂存差异时明确报错，不修改工作区或提交。
- `aib [--base REF]` 默认使用 `main`，对比 `REF..HEAD` 的单行提交历史并输出摘要；缺少基准 ref 或无提交时清晰报错。允许指定 base，不执行 Git 写操作。
- 模型输出只从 stdout 返回给调用流程；调试信息/错误发往 stderr，避免污染提交消息。
- 三个命令通过 CZ 现有 LLM 配置及 provider fallback；不新增配置体系，不调用 shell 拼接命令。

## Scope

包含通用 CZ 文本生成 seam、三个 `aiw-git` 子命令、命令帮助和相邻用户文档。

不包含更改 `aiw cz` 的交互/提交契约、改动全局 AI/Go gateway、自动修复审查意见、推送/发布或额外 Git alias 安装。

## Work items

- [x] 1.1 抽取 CZ 通用文本生成接口并保持现有候选生成路径的行为；小，低难度，无依赖；完成标准：候选路径仍调用同一 provider fallback，新增接口接收提示词并返回模型文本/错误。
- [x] 1.2 实现 `aiw git aic`，按 add、生成、commit 顺序执行并传播失败；中，低难度，依赖 1.1；完成标准：只在模型成功后创建提交，使用 `-F -` 传递完整消息。
- [x] 1.3 实现只读的 `aiw git air` 和 `aiw git aib [--base REF]`；中，低难度，依赖 1.1；完成标准：严格使用暂存差异/指定提交区间，错误不执行写操作。
- [x] 1.4 更新 `aiw git` 帮助、CLI 稳定规格和 Git 使用文档；小，低难度，依赖 1.2、1.3；完成标准：帮助列出命令、参数、默认基准及数据范围。

## Acceptance

- `aic` 暂存 tracked、untracked 和删除变更；模型生成消息符合 Conventional Commits，完整多行消息经 stdin 传给 Git。模型失败、缺少暂存差异或 Git 提交失败均以非零退出；模型失败不运行 commit。
- `air` 仅向模型发送 staged diff，向 stdout 输出审查结果；空 staged diff 返回明确错误，不改变索引、工作区或历史。
- `aib` 默认概述 `main..HEAD`，可通过 `--base` 指定基准；只把该区间单行提交历史传给模型；无效基准、空历史或 provider 失败均明确失败，不修改仓库。
- AI 调用使用 CZ 配置和 provider fallback，不把命令参数交给 shell，不把日志混入 AI 文本或提交信息。
- 现有 `aiw cz` 候选、预览、编辑/取消及提交流程保持不变。

## Verification

- Static review covered provider behavior, plugin loading, Git argument construction, output streams, and failure handling; the final source diff was inspected.
- Python compile-only check passed with python -m py_compile for changed Python files. After correcting base-ref handling, the permitted focused compile retry passed.
- Tests and AI/Git runtime scenarios were not run under the repository's default test authorization. Independent Tester must report uncovered scenarios and residual risk without claiming a pass.
- PM explicitly waived the test-acceptance gate after recording three repair votes; no `test-accepted` event was created. Reviewer claim was rejected by the CLI, with all 14 runtime scenarios still uncovered.
- 2026-10-08：用户明确授权跳过独立审查并接受缺少审查证据的风险。PM 直接接受本 FD；Reviewer 未认领 handoff、未出具报告，也没有 `verification-passed` 事件。14 个运行场景仍未验证。

## Sources

- Issue: none
- `plugins/aiw-cz/aiw-cz.py`, `cz_llm.py`, `cz_core.py`, `cz_providers.py`
- `plugins/aiw-git/aiw-git.py`, `aiw-git-core.py`
- `openspec/specs/cz/spec.md`
- 用户提出的 `aic`、`air`、`aib` 示例。

**Completed:** 2026-10-08
**Disposition reason:** 用户明确授权跳过独立审查并接受缺少审查证据的风险；PM 直接接受，未生成 `verification-passed` 事件。
