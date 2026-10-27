# FD-107 独立静态审查 r1

结论：**changes-requested**。本周期第 1 次失败，累计 1/3。
Reviewer：`fd107-reviewer-20261027-73a9c4`；实际认领 `FD-107-000002-review-requested`。
审查基准：HEAD `61a9040008b47763b1513069bb5dbfcf75ffe300` 与当前明确限定的未提交 diff；本轮未生成提交。

## Findings

- **R1 / P2 — 稳定规格仍强制整个 parent 干净，与本次用户规则及实现冲突。**
  `openspec/specs/fd-workflow/spec.md:407` 的正常生命周期仍要求 `clean workspaces`；
  `:660` 的 `new --worktree` / `add --prepare` 路径明确写 `require ... clean parent`。
  `:353` 的交付场景以 parent clean 为前提，`:724` 的内容冲突路径仍未区分 dirty-parent 必须先停。
  但本次 `worktree_add` 已允许无关修改，`local_merge` 为不碰交付路径的 parent 修改提供独立 index 交付，
  canonical `work-management.md` 与 `preflight.md` 也已放宽。稳定规格是后续 FD 的能力契约，
  保留相互冲突的硬要求会继续触发旧门槛，违背用户本次明确要求和仓库“决策改变时同步稳定规格”的规则。
  最小修复：在该稳定规格中同步“仅相关修改要求选择”“不冲突 parent 修改保留内容与 staging”
  以及“dirty parent 遇分支内容冲突时，在 parent mutation 前停止；clean 后才用既有恢复流程”，
  并补充对应场景。无须新建 OpenSpec change，也不改历史归档 FD/报告。

## 静态检查范围与判断

- 实际阅读 `git-wt.py` 的创建、相关路径识别、`--from-head`、状态解析、独立 index 的 read-tree/commit、
  真实 index 定点刷新、已有交付检测、FD-Source/单父提交检查和 cleanup 路径；对照规范、技能、使用文档和根回归用例。
- 现有代码将同一交付路径上的本地修改也先拦截；文档已明确，且保持 Git 不覆盖本地文件的边界。
  本轮不将其理解为必须新增 stash 式同文件不同 hunk 自动重放，也不追加该范围。
- 对 ignored 目录替换仅存在未证实疑问；`read-tree -m -u` 自身有工作区保护，未在授权内运行场景，
  因而不把该猜测写成确定代码缺陷。实际保留行为仍属于运行未验证项。
- `.agents/**` 的预存修改/删除以及预存 `build`、`install` 未跟踪内容均不属于本次 diff，未改动、未纳入结论。

## 实际命令和证据限制

实际执行：`aiw fd resume FD-107`、本 Reviewer 的真实 claim；分批 `cat`/`sed`/`rg`/`nl`/
`git diff`/`git rev-parse HEAD` 静态读取；读取本机 `git-read-tree` man 文档。
首次读取 aiw 仓库 `.agents/prompts/core` 发现目录已不存在，随后读取已知 customer-profile core 合同作为资源约束参考。
一次 `/usr/share/doc/git*` glob 在本机无匹配；没有据此更换 shell 或运行验证，也未重复同一失败命令。

Host 说明 Python 内存 compile 已明确通过；Go compile 脚本已执行且无错误输出，
但两段命令的最后 exit0 不等于 Go 脚本独立退出码，不能将该部分报告为已确认通过。
本 Reviewer 未执行业务 import、compile、测试、Git 临时仓库场景、网络、安装、发布、lint/format/typecheck/build。
新增和修改的用例只静态读取，未执行。

残余风险：Git 版本与平台、混合 staged/unstaged、ignored 路径、hook/故障恢复行为尚未实际运行验证；
source 修改尚未同步已安装分发与 customer-profile 技能。修复 R1 后可由本 Reviewer 聚焦复核稳定契约与邻近边界。
