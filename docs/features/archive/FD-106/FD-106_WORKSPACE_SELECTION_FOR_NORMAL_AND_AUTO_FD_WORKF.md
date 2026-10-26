# FD-106: Workspace selection for normal and auto FD workflows

**Status:** Complete
**Revision:** 5
**Priority:** Medium
**Review policy:** Independent
**Evidence policy:** Markdown

## Problem

用户要求工作区规则统一为：非 Git 不创建 worktree；Git 普通请求默认原工作区，仅明确要求时使用 worktree；Git 显式 auto 从规划到归档都在 worktree 中完成。旧规则强制先在父分支提交计划、归档留到 worktree 删除后，无法满足 auto 全过程隔离。

## Solution

普通请求在当前工作区执行；仅显式 auto 或明确 worktree 意图选择隔离。非 Git 使用本地 FD/证据且不初始化 Git。新增 new --worktree 入口，分配 FD 后先通过受管 worktree bootstrap 创建分支，再在其中写计划。已有 FD 使用原 add 路径。worktree 内完成计划、实现、审查、归档及提交后，一次 squash 交付并验证清理；父分支只接收工具管理的最终交付。

## Scope

同步根指令、fd-workflow、共享合同和稳定规格；修改 aiw-fd.py 的非 Git 支持与隔离新建，修改 git-wt.py 的 bootstrap 和 worktree 内归档定位。保持历史普通 add、事件/审查策略和交付安全检查。本次为普通请求，延续主工作区修改、不自动提交；不实际创建 worktree，不测试、联网、打包或发布。

## Work items

- [x] 1.1 统一三条选择规则及 auto 规划到归档的工作区与交付顺序。
- [x] 1.2 支持非 Git FD 工作区与新 FD 隔离创建，保持编号和收据边界。
- [x] 1.3 支持创建 worktree 后再写 FD，并在归档后从 worktree 安全交付/清理，兼容原 add。
- [x] 1.4 完成一次针对修改 Python 文件的内存编译及静态检查，记录实际证据。

## Acceptance

- 非 Git 不初始化 Git 或创建 worktree，可保存 FD 和证据。
- Git 普通请求默认原工作区；显式 worktree/auto 才建立隔离。
- auto 在新 FD 计划写入前创建 worktree，计划、实现、审查和归档均写入 worktree；归档提交包含在最终 squash 中。
- 既有 FD 和 worktree 可继续；父分支清洁、坐标校验、冲突恢复与验证后清理不被放宽。
- CLI 无法执行必要步骤时保留状态，不静默退回父工作区执行 auto。

## TODO

- 独立静态审查及针对两项发现的限定复查已完成；本次不提交、不合并、不归档。

## Verification

- 本次涉及工作区和交付 CLI 契约，选 Independent，单次独立静态审查；如有发现只限定复查修复影响。
- 已运行两次内存编译（初版及审查修复后），均通过：`python -c "from pathlib import Path; paths=('src/plugins/aiw-fd.py','src/plugins/aiw-git/git-wt.py'); [compile(Path(p).read_text(encoding='utf-8'), p, 'exec') for p in paths]; print('Python compile-only passed: aiw-fd.py, git-wt.py (no artifacts)')"`。第二次覆盖最终两个 Python 源码，未生成 pyc 或可分发产物。
- 已运行 `git diff -- src/plugins/aiw-fd.py src/plugins/aiw-git/git-wt.py`，静态检查选择规则、非 Git 本地根、编号分配、隔离 bootstrap、归档后的交付定位；这些结论不是运行测试结果。
- 不运行测试、真实 worktree/Git 写入、构建、验证脚本或网络；这些路径只做静态判断。
- 首轮独立审查要求修改：归档后恢复错误结束，以及旧平铺归档查找缺失。报告 `reviews/FD-106-review-r1.md`。
- 已修复：resume 优先定位记录的 worktree；归档 Complete 但仍有工作区记录时提示继续交付/清理，异常元数据阻塞恢复，Closed/Deferred 继续使用 reopen；入口和共享合同同步。
- 已修复：worktree 定位同时支持原活动布局、规范归档与旧平铺归档，保留链接、目录边界和 Git 坐标检查。
- 限定独立复查通过：`reviews/FD-106-review-r2.md`；来源 `FD-106-000004-implementation-ready`。仅复查首轮发现和直接影响，未新增完整审查范围，未声称运行命令链已测试。
- 独立静态审查要求修改：归档后的未交付状态被入口/resume 提前视作完成，Git 工作树定位遗漏旧平铺归档。报告：`reviews/FD-106-review-r1.md`；来源 `FD-106-000002-review-requested`。修复后仅复查发现和直接影响。

## Sources

- Issue: none
- 用户当前确认的三条 worktree 选择规则；此前授权主工作区修改且不自动提交。
- FD-105 轻量流程及 openspec/specs/fd-workflow/spec.md。

**Completed:** 2026-10-26
**Disposition reason:** Done
