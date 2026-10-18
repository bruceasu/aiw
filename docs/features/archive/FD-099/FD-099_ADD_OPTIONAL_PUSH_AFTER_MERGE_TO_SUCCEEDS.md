# FD-099: Add optional push after merge-to succeeds

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户要求 `src/plugins/aiw-git/git-merge-to.py` 新增 `--push`：合并成功后推送目标分支。
现有命令只进行隔离 worktree 合并，帮助与稳定规范声明不进行 push。

## Options and decision

采用可选开关，默认行为完全保留。未指定远程参数，沿用 Git 插件的 origin 默认约定，
明确推送目标分支到 origin 同名分支；不推送调用方当前分支、不依赖 push.default。
不采用无参数 git push（可能选错调用方分支）或自动强推（超出需求）。

## Solution

- 参数解析接受 `--push` 放在普通模式或 --new 模式的任意位置，并拒绝重复开关和未知选项。
  移除 --push 后保留原 --new 位置/分支个数约束。内部返回值增加 push 标志。
- 所有源合并成功且临时 worktree 成功清理后，才进行一次
  `git push --no-follow-tags --recurse-submodules=no -- origin refs/heads/<target>:refs/heads/<target>`。
  显式 refspec 只推送目标同名分支，不设置 upstream，不使用 force，不 fetch。
- 无 --push、任一合并失败、worktree 创建/清理失败时不 push；--new 单源创建目标
  也属于本地成功，可推送新目标。origin 缺失、权限/网络问题或非快进拒绝由 Git 报错。
- push 失败返回该 Git 退出码，明确本地合并已保留，不重试、不回滚、不重新合并；
  异常/中断继续返回原退出约定并保留本地合并成功信息。
- 保留调用方分支、文件、索引、源解析和所有失败恢复路径；更新帮助、CLI 文档及稳定规范。
- 用户授权的是实现可选网络行为，不是在本轮实际执行 push 或访问远程。

## Scope

仅 merge-to 参数、合并后的可选 push 和对应文档规范。不增加 --remote、不变更其他 Git
命令、FD 交付机制或安装配置；不执行实际 merge-to/push、测试、构建、发布安装。
无关联 Issue 或独立 OpenSpec change。

## Work items

- [x] 1.1 实现 --push 参数和成功后推送、失败退出语义；规模 small、难度 medium、无前置。完成证据：parse_args 移除唯一 --push 后沿用原参数规则；main 仅在全部合并及清理成功后显式推送 origin 同名目标，失败返回 Git 退出码且保留本地合并。
- [x] 1.2 同步帮助、CLI 文档和稳定规范，完成双格式证据；规模 small、难度 low、依赖 1.1。完成证据：META/CLI 文档/稳定规范列出普通和 --new 用法、origin 同名目标、精确成功边界和失败保留；实现报告与 JSON 已写。

## TODO

- 无剩余实现项。审查事件 `FD-099-000005-verification-passed` 后已交付 develop：`149f123`，来源 `bbe61fe`；CLI 已校验并清理 worktree/分支，`aiw fd close FD-099 Complete` 已归档。以下阶段说明作为历史记录保留。

- 必要独立设计审查 ready，见 `docs/features/reviews/FD-099-design.md`；Worker 领取 `FD-099-000003-design-ready`，专用 worktree/分支已创建并核对 parent=develop。
- 离线 compile-only 和一次最终只读 diff 检查已完成；待独立实现审查、本地交付清理和归档。

- 独立实现审查通过：`docs/features/reviews/FD-099-review.md`；待 PM 完成本地交付、清理及归档。

## Acceptance

1. 原有普通模式/--new 不带 --push 的行为保留，绝不调用 push。
2. `merge-to main feature --push`、`merge-to --push main feature` 和
   `merge-to --new integration feature/a feature/b --push` 接受开关，默认源仍为当前分支。
3. 全部合并和清理成功后仅调用一次 push，完整 refspec 从目标分支推到 origin 同名分支。
4. 任一合并、worktree 创建或清理失败不推送；原恢复指令和本地状态保留策略不变。
5. push 失败返回其非零退出码，已成功的本地合并保留；不强推、自动重试、回滚或推其他引用。
6. 重复 --push、未知开关、错误分支个数在变更 Git 前拒绝；帮助明确开关及 origin 约定。

## Verification

- 本地交付：`aiw git wt local-merge FD-099` 成功，生成单父 squash `149f123`，验证来源 `bbe61fe` 后移除专用 worktree/分支；`aiw fd close FD-099 Complete` 成功。未执行实际 push、网络、最终构建、发布或安装。

- 一次 compile-only 批次：Python 内置 compile() 对变更脚本内存语法编译，随后
  `scripts/compile.py`，环境 GOPROXY=off/GOSUMDB=off/GOTOOLCHAIN=local；不保留最终二进制。
- 一次最终只读 diff/status 检查，追踪 parse_args/main/git 清理和退出路径。
- 添加 CLI 可选网络行为，执行一轮必要的独立静态设计审查；最终实现由独立 Reviewer 审查。
- 不新增/运行测试，不执行脚本功能/真实 push、格式化、lint、验证脚本、最终构建或网络请求。
- 实际编译批次：设置 GOPROXY=off/GOSUMDB=off/GOTOOLCHAIN=local，执行 `python -c "from pathlib import Path; import runpy; p = Path('src/plugins/aiw-git/git-merge-to.py'); compile(p.read_text(encoding='utf-8'), str(p), 'exec'); print('Python syntax compile: OK'); runpy.run_path('scripts/compile.py', run_name='__main__')"`，退出码 0；无最终二进制或 pyc。
- 独立设计审查 ready：`docs/features/reviews/FD-099-design.md`；实现证据：`docs/features/reports/FD-099-implementation.md` 及同名 JSON。
- 一次只读 diff/status 批次：`git diff 01606f6 -- <实现/文档/FD/设计报告路径>`、`git diff --check`、`git status --short`，无空白错误及范围外改动。随后仅澄清 META 帮助文本区分合并失败保留 worktree 与 push 失败已清理，不改变执行逻辑；此帮助字符串静态复核，未重复编译。

- 独立 Reviewer `fd099-review-20261019-b7319d` 已领取 `FD-099-000004-implementation-ready`，审查 `01606f6..0ad456e` 的实际源码、CLI 文档、稳定规范和实现证据，未发现阻塞缺陷；报告：`docs/features/reviews/FD-099-review.md` 及同名 JSON。最后 META 字符串静态覆盖，未重复编译；未执行测试、功能脚本、网络或 push。

## Sources

- 用户本次明确需求。
- `src/plugins/aiw-git/git-merge-to.py`
- `openspec/specs/cli-and-plugins/spec.md` 的 Git merge-to requirement
- `docs/usage/cli-reference.md` 的 aiw git merge-to 小节

**Completed:** 2026-10-18
