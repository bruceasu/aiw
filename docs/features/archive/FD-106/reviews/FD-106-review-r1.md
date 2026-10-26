# FD-106 独立静态审查

- 结论：需要修改（changes-requested）。
- Reviewer 会话：`fd106-reviewer-20261026-4d9c2b`。
- 来源事件：`FD-106-000002-review-requested`，FD revision 2。
- 审查对象：当前主工作区相对 HEAD 的 FD-106 Python 改动与对应工作区/交付合同；未重做 FD-105 的审查。
- 授权边界：普通主工作区请求；不实际创建 worktree，不测试、编译、提交、合并、归档或联网。

## 发现

### P1：归档后的未完成交付会被入口和 resume 提前终止

位置：`src/skills/fd-workflow/SKILL.md:28`，`src/plugins/aiw-fd.py:2124`；关联 `resolve_fd`（194 行）和新的 archive-before-delivery 顺序。

新流程要求在 worktree 内先 close/archive 并提交，再执行最终 local-merge 和清理。归档不再代表整个本地生命周期结束。但 SKILL 仍无条件写明 archived Complete FD is done，resume 也在检测到归档 FD 时直接报错。

静态可达情形：auto 已在工作树内归档并提交，尚未交付时会话中断；workspace.json 与工作树仍存在。恢复时，技能入口会把它视作已完成；即使在正确工作树调用 resume，也被拒绝。若从父工作区恢复，新建隔离 FD 在首次交付前尚不存在于父工作区，resolve_fd 还会先报 found 0，无法到达工作区恢复判断。

这不满足本 FD 对 auto 全过程隔离、保留未完成交付状态和可继续既有工作树的要求。请在判断归档完成前检查记录的工作区/交付状态，使用已验证的坐标引导或继续尚未完成的交付/清理；不要重新启动实现、生成重复 squash，或静默转到父工作区编写。真正没有待交付工作区的已归档 FD 才可按完成处理。

### P2：工作树中的旧平铺归档不再被 Git 工作树命令识别

位置：`src/plugins/aiw-git/git-wt.py:127`。

workspace() 从父工作区活动 FD 改为读取工作树中的活动/归档 FD，是新流程所需；但新归档回退仅枚举 archive/<FD-ID>/<FD-ID>_*.md，没有保留 archive/<FD-ID>_*.md 旧平铺归档。

已存在的 worktree 若已按旧版平铺布局归档，活动设计不存在，新回退也找不到文件，因而在 132 行报错。status、commit 和 local-merge 等共享 workspace() 的命令都无法继续。稳定规格仍明确保留旧平铺设计可读（`openspec/specs/fd-workflow/spec.md:1054`），本 FD 也要求既有 FD/worktree 兼容。

请把旧平铺归档纳入工作树内的受检候选集合，同时保留唯一文件、真实文件、路径边界与坐标检查。不要通过回退读取父工作区计划来绕开 auto 的隔离要求。

## 已检查的证据和限制

- 读取并追踪了 git_workspace、非 Git runtime/evidence、new --worktree 的 ID 分配和 bootstrap、git-wt add --prepare、工作树坐标检查、archive-inclusive local-merge/cleanup，以及相应合同和稳定规格。
- 当前 bootstrap 仍要求清洁父工作区、忽略管理目录并核对工作树/分支坐标；未发现此次修改放宽原单父 squash、FD-Source、已交付检查和安全清理约束。
- 一次静态差异命令：`git diff -- src/plugins/aiw-fd.py src/plugins/aiw-git/git-wt.py`。其余为必要源码/文档读取与行号定位，以及真实 Reviewer 认领和结果交接。
- 编译通过仅引用 Worker 已记录的内存编译证据；Reviewer 未编译，未运行真实场景、测试、Git 写操作、构建、验证脚本或网络。
- 修复后只需复查这两个发现及其直接影响，不增加完整审查范围。
