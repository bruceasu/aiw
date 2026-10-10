# FD-052: 为 AI 分支摘要增加有上限的 diff 上下文

**Status:** Open
**Revision:** 4
**Priority:** Medium  
**Evidence policy:** Dual

## Problem

`aiw git aib` 目前只将 `BASE..HEAD` 的单行提交记录交给模型。提交标题可能含糊，模型因此缺少判断分支实际改动内容的证据。与此同时，直接加入完整 diff 会让大分支产生过长输入。

`aic` 和 `air` 已经将暂存区 diff 交给模型；本项不改变它们的输入。

## Options and decision

1. 保持只用提交记录：改动最小，但不能解决提交信息不足的问题。
2. 对所有分支发送完整 diff：信息充分，但输入大小无上限。
3. 在提交记录之外加入受 8,000 字符上限约束的文件清单、汇总统计和最多 12,000 字符的分支 diff；超限时明确标记省略，并要求模型只依据已提供证据总结。**选择此方案**：为模型增加内容线索，同时限制新增上下文的大小。目录名可由文件路径归纳，不再单独重复发送。

## Solution

`aib` 保留现有提交历史上下文，并从 `merge-base(BASE, HEAD)..HEAD` 收集文件状态、简短汇总统计和受限文件清单，再附最多 12,000 个字符的 diff。文件清单最多占 8,000 字符；超限时列出可容纳的路径并说明还有多少路径省略。若 diff 超过上限，截断并明确说明省略了内容。提示模型不得推断被省略的文件或实现细节。Git 上下文收集失败时沿用现有错误处理，不调用模型。命令仍只读。

`aic` 和 `air` 继续使用原有暂存区 diff，不添加重复文件清单。

## Scope

包含 `aib` 上下文收集与提示词、对应稳定规范和用户说明。

不包含更改 `aic`/`air`、新增配置项、修改 provider、依赖、仓库状态或测试策略。

## Work items

- [ ] 1.1 为 `aib` 加入最多 8,000 字符的文件清单和汇总统计，并加入总长不超过 12,000 字符且带省略说明的 diff 上下文；保留只读行为与错误处理。尺寸：小；难度：低；依赖：无。完成标准：模型提示明确列出这些证据及截断状态，生成仅在 Git 上下文成功后发生。
- [ ] 1.2 更新 AI Git 稳定规范和用户说明，记录 `aib` 的上下文范围及上限，并确认 `aic`/`air` 输入约定不变。尺寸：小；难度：低；依赖：1.1。完成标准：文档描述与实现一致。

## Acceptance

- `aib` 同时提供提交历史、受 8,000 字符上限约束的文件名/状态清单、简短变更统计和受 12,000 字符上限约束的 diff。
- 超出 diff 上限时，提示中可见省略事实；模型被要求仅总结可见证据，不得声称掌握省略内容。
- 无提交或 Git 上下文读取失败时保持明确失败行为；不调用 provider、不修改 Git 状态。
- `aic` 仍使用暂存 diff 生成 Conventional Commit；`air` 仍只审查暂存 diff，二者行为不变。

## Verification

- 静态检查 Git 参数、路径状态/统计/diff 的收集顺序、截断边界、截断提示和失败路径；检查最终 diff 及规范一致性。
- 按仓库规则运行一次 Python compile-only 检查。测试、真实 provider 调用和 Git 运行场景不在本项授权内，不运行。

## Sources

- Issue：无。
- `src/plugins/aiw-git/git-aib.py`、`git-aic.py`、`git-air.py`。
- `openspec/specs/cli-and-plugins/spec.md`，AI-assisted Git commands。
- `docs/usage/aiw-git-ai.md`。
