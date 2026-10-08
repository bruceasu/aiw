# FD-034: Git 补丁生成、应用与恢复建议

**Status:** Complete
**Revision:** 17
**Priority:** Medium
**Test policy:** Independent
**Evidence policy:** Dual

## Problem

用户经常忘记生成及应用 patch 的 Git 命令，也需要在应用失败时得到下一步建议。现有 `aiw git` 没有对应入口，手动命令不会随补丁附带说明。

## Options and decision

1. 只在 `guide` 增加命令说明：易维护，但仍需手动选择命令，也不能针对失败给建议。
2. 用 `git format-patch` / `git am`：适合传递提交，改变提交历史及作者信息，不符合当前工作区改动的常见用途。
3. 增加 `aiw git patch create/apply`，封装 `git diff --binary` 与 `git apply`，生成同名 Markdown 说明并在应用检查失败时诊断。**选择此方案**。说明为确定性文本，不依赖 AI 服务。

## Solution

`create FILE.patch` 默认导出 HEAD 到当前工作区的所有已跟踪改动（暂存和未暂存）；`--staged` 仅导出暂存区，`--worktree` 仅导出未暂存区。使用二进制安全的 Git diff；不自动暂存未跟踪文件。拒绝空 diff、既有补丁或说明文件，避免覆盖。成功后写 `FILE.patch` 及 `FILE.patch.md`，说明来源范围、改动统计、未跟踪文件限制、推荐的检查/应用命令和失败恢复建议。

用户补充跨设备使用场景后，`create FILE.patch --from A --to B` 允许比较两个 commit ID 或分支，语义为 `git diff A B`，不隐式使用 merge-base。两个参数必须同时提供，不能与 `--staged` 或 `--worktree` 混用。先将两端解析为提交 ID，再用 ID 生成补丁；说明文件同时记录原始 ref 和解析出的 ID，提示目标设备应有 A 对应的文件版本。无需两台设备共享 ref 名称。

`apply FILE.patch` 先用 `git apply --check` 检查，再执行 `git apply`，默认只改变工作区，不自动暂存或提交。检查失败时保留仓库状态；用只读的反向检查辅助识别已应用补丁，展示 Git 错误及分情况建议。不自动启用 `--3way`、`--reject` 或回滚，避免意外冲突/部分应用。拒绝空或不存在的文件。应用调用自身失败时也返回非零并建议查看当前差异。两个子命令均不启动 AI 或网络调用。

## Scope

包含新 `patch` 子命令、两个 ref 的显式比较、帮助、相邻使用文档和稳定 CLI 规格。保留现有 `aiw git` 分发与其他命令行为。不包括提交邮件补丁、自动三方合并或 AI 生成说明。

## Work items

- [x] 1.1 实现二进制安全的补丁生成及同名说明；小，中难度，无依赖；完成标准：三种来源范围准确，空内容及既有路径不覆盖，说明包含应用与限制。
- [x] 1.2 实现补丁预检、应用和失败建议；小，中难度，依赖 1.1；完成标准：先检查再应用，失败非零，已应用与一般不匹配给出不同建议，不自动修改失败现场。
- [x] 1.3 更新帮助、使用文档与稳定规格；小，低难度，依赖 1.1、1.2；完成标准：示例说明默认范围、输出、失败处理和不自动暂存。
- [x] 1.4 增加两个 ref 的显式比较并同步说明与规格；小，中难度，依赖 1.1、1.3；完成标准：commit ID/branch 均可解析，成对参数和互斥规则清楚，说明记录解析 ID，错误不写输出。

## TODO

- [x] 完成实现、帮助与文档。
- [x] 提交 Worker 实现与双份证据并交接独立测试；未执行的行为不记为通过。
- [x] 按用户新增的跨设备场景完成 1.4，并更新 Worker 证据。
- [x] 刷新待认领 Tester handoff，使其指向新增 ref 范围与 r2 报告。
- [x] 接收独立测试报告及后续审查决定；12/13 场景有通过证据，PM 以 3/3 风险接纳票同意继续，S13 未覆盖。

## Acceptance

- 默认创建的补丁含已跟踪文件的暂存及未暂存差异；`--staged` 与 `--worktree` 各只导出对应差异；二进制变化可保留；同名说明列出改动统计，未跟踪文件不被误称为已包含。
- `--from A --to B` 对两个可解析为提交的 ref 生成 A 到 B 的补丁；说明列出两端 ref 与固定提交 ID；单独使用一个参数、与工作区范围参数混用或无效 ref 时非零退出且不创建文件。
- 空差异、Git 失败、目标 patch 或说明已存在时非零退出，不覆盖既有文件；成功时两个文件均存在。
- `apply` 成功时只向工作区应用；失败时返回非零、保留 Git 错误和建议，预检失败不执行实际应用；反向预检成功时指出可能已应用。
- 帮助和文档可让用户不查 Git 参数就完成常见补丁创建与应用，且说明拒绝自动三方合并和部分应用的原因。

## Verification

- 已静态检查参数、Git 调用、写文件顺序、错误路径与文档一致性。
- `python -m py_compile plugins/aiw-git/git-patch.py` 在新增 ref 范围后的实现上通过。
- 独立 Tester 报告 `docs/features/archive/FD-034/reports/FD-034-test-report-r1.md`：授权的聚焦命令最终 9/9 通过，13 个场景中 12 个有通过证据；S13 未覆盖，业务代码分支覆盖率未测。三位评估者均投风险接纳，PM 决策见 `docs/features/archive/FD-034/reports/FD-034-test-decision-r1.md`。
- 独立 Reviewer 静态审查见 `docs/features/archive/FD-034/reviews/FD-034-review-r1.md`，结论 `verification-passed`；S13 仍只有静态证据。Reviewer 未另行运行测试、实际 Git patch 命令、AI 调用、构建或 lint。

## Sources

- Issue: none

- `plugins/aiw-git/aiw-git.py` 与相邻 Python 子命令
- `openspec/specs/cli-and-plugins/spec.md`
- 用户本次需求。

**Completed:** 2026-10-08
