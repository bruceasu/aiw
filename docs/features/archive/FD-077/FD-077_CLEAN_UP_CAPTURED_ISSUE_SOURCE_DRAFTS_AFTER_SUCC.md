# FD-077: Clean up captured Issue source drafts after successful capture

**Status:** Complete
**Revision:** 7
**Priority:** Medium
**Evidence policy:** Dual

## Problem

成功 capture 只写入正式工件和元数据，消费后的临时源草稿仍残留。ISSUE-00009 已批准，用户要求完整自动本地生命周期，并另行授权独立提交 ISSUE-00008 文档。

## Options and decision

- 不清理：保持现状，但不能解决已消费草稿堆积。
- 成功后直接按路径删除：简单，但检查后路径替换可能误删新文件。
- 采用受限草稿根和暂存核对：仅在正式工件与元数据保存成功后，将候选源原子移动到草稿根内随机私有目录，核对文件身份和内容，再删除；若不一致则不删除，并无覆盖地恢复或保留恢复路径。选择此方案降低误删风险。

清理不是 capture 的持久化事务。保存失败不进入清理；清理失败不回滚正式结果、不改变成功退出状态，只在 stderr 明确报告警告和残留/恢复位置。

## Solution

- Capture 公开签名保持不变。Artifact 增加仅本次调用使用的 CleanupWarning 字符串，标记 json:"-"，不写入 TOML，也不加入摘要/批准源 JSON。既有工件摘要语义保持兼容。
- Capture 读取内容时，为项目草稿取得文件身份快照。仅接受当前项目 .ai/issues/drafts 和兼容 .ai/requirements/drafts 内的普通文件；外部来源仍可 capture，但不清理。
- 清理根使用 os.Root；拒绝根以下任意目录/文件的符号链接。草稿根是可信的指定存储位置；项目 .ai 允许仓库管理的 worktree 共享存储链接，但不沿着源提供的任意别名推断清理资格。根本身被替换、非法路径或身份变化时保留并警告。
- 路径先做项目相对定位，拒绝原始 .. 分量、Windows ADS、根目录或非普通文件；允许当前项目内等价绝对源路径。只匹配两个确切根，不用字符串前缀模糊匹配。
- 快照保存初始 FileInfo，内容与 Capture 实际保存的 bytes 对应；打开后 SameFile 再核对，不能将新身份当作旧快照。
- 在成功 Write(meta) 之后，再验证路径与快照仍匹配，通过草稿根 os.Root 在随机 0700 私有子目录隔离当前源，再读取核对 SameFile、字节内容与文件属性。只删除匹配对象；不将内容一致但身份不同的替换文件当成同一源。
- 不一致或清理失败时，用无覆盖的硬链接恢复原路径并移除隔离名；原路径已被其他文件占用或恢复不可用时保留隔离文件，警告给出恢复路径。不得覆盖新的源路径。对无法安全执行的平台/文件系统降级为保留及警告。
- 私有目录只删除自身已清空目录，不递归删除。捕获过程中可见的源变化应保留。不是对同一用户恶意篡改私有隔离目录的跨进程锁协议；不新增后台工作或并发机制。
- 直接 capture CLI 和聊天 Capture 后立即将 CleanupWarning 写 stderr，正常成功文本/返回值不变。聊天后续事实记录失败仍按现有语义报告；源清理在正式工件/元数据保存后，不以 Session 事实记录为提交条件。

## Scope

改动集中于 src/internal/issue 的草稿清理 helper 与 Capture 接入、src/cmd/aiw-req 两个调用点、稳定 requirement 规格与 aiw-issue 使用说明。历史草稿批量清理、额外依赖、迁移、公开 CLI 参数变更、测试和最终构建均不在范围。

## Work items

- [x] 1.1 完成并独立审查清理设计，实施共享草稿生命周期和 Capture 警告载体。规模小到中等、难度中等；无前置实现依赖；证据：capture_draft.go、Capture 在 Write(meta) 后清理，CleanupWarning 排除 JSON；隔离核对与无覆盖恢复均已实现，运行效果未验证。
- [x] 1.2 接入直接/聊天警告并同步 requirement 稳定规格和 Issue 帮助。规模小、难度低；依赖 1.1；证据：warnCapturedDraft 两处调用、capture help、稳定规格及使用文档；成功退出保持兼容。
- [x] 1.3 完成 compile-only 与静态核对、双格式实现报告并准备独立 Reviewer 交接。规模小、难度低；依赖 1.2；证据：reports/FD-077-implementation.md 和同名 JSON，compile-only 退出 0；独立 Reviewer 结果由后续真实事件记录。

## Acceptance

1. 两类项目草稿 capture 成功后，正式工件与元数据已写入，当前未改变的源文件被删除；其他文件不变。
2. 工件或元数据保存失败，不清理源。
3. 外部普通来源保持原样；不能清理路径穿越、链接、目录、被替换或改变的文件。
4. 清理失败保留正式结果和可恢复草稿，警告包含源或恢复路径；不提示盲目重复 capture。
5. 直接 CLI/聊天都呈现警告；公开函数签名、工件摘要、正常 stdout 和成功退出状态兼容。
6. 原正式工件、批准日志、其他历史草稿不被清理。

## TODO

- 独立设计审查已完成，结论 ready：reviews/FD-077-design-review.md 和同名 JSON；无阻塞发现。
- ready plan 已提交 193f0de；已在 feature/FD-077、.wt/FD-077 完成实现。
- 按项提交已完成；R1 已修复且通过聚焦独立复核，源修改后的同一离线 compile-only 再执行通过。后续由 PM 执行交付门槛检查、local-merge、确认清理并 Complete 归档。
- 原父工作区门槛已获授权：ISSUE-00008 单独提交为 a732b98；阻塞反馈保留并更新解除事实。

## Verification

已执行一次静态 diff 核对，以及环境 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local 下的 python scripts/compile.py，退出码 0；该脚本仅 go build -o NUL 编译 cmd/aiw 与 cmd/aiw-req，不保留发行产物。未授权测试，未创建或运行 Go 测试，未运行 vet/格式化/最终构建/联网。详见 reports/FD-077-implementation.md 和同名 JSON。

设计引入删除和失败恢复边界，因此进行一名独立代理的静态设计审查；它不替代独立 Reviewer 实现审查。独立设计审查已记录 ready，compile-only 已通过；实际删除/恢复运行行为未验证。

独立 Reviewer fd077-reviewer-20261018-c9d283 已审查基线 193f0de 至 a6e3db9 的实际实现，结论 changes-requested。R1：草稿根 Lstat 身份未与 OpenRoot 身份绑定，根在两者之间被替换为链接时可能扩大删除范围，阻塞 Acceptance 3。详见 reviews/FD-077-implementation-review.md 及同名 JSON；未运行测试、编译或运行验证。%% 修复 R1 并取得聚焦独立复核后再继续交付。

Worker 已按 R1 修复：保存预检的 family/root 身份，打开后及删除前均核对非链接目录身份和固定句柄；同一离线 compile-only 在源修改后再执行，退出 0。详见 reports/FD-077-root-identity-repair.md 和同名 JSON；本说明不替代独立复核 verdict。

当前独立 Verdict：verification-passed。Reviewer fd077-reviewer-20261018-c9d283 对 a6e3db9 至 313e100 的 R1 修复完成有界聚焦静态复核，确认原始 family/root 身份与固定根句柄绑定，打开后和清理前拒绝链接或替换目录；R1 已解决。报告为 reviews/FD-077-implementation-review-r2.md 和同名 JSON，来源事件 FD-077-000006-implementation-ready。结合前轮完整静态审查，其余验收结论继续有效；本轮未运行测试、编译或运行验证。前述 R1 待修复说明保留为历史，当前无未解决阻塞发现。

## Sources

- Issue: ISSUE-00009
- docs/issues/ISSUE-00009/issue-plan.md
- openspec/specs/requirement/spec.md
- src/internal/issue/store.go、context_sources.go
- src/cmd/aiw-req/issue-capture.go、issue-chat.go
- scripts/compile.py
- reports/FD-077-blocker-20261018-parent-dirty.md 及同名 JSON
- Planner source event: FD-077-000002-design-requested
- Host session: fd077-host-20261018-53b7ac

**Completed:** 2026-10-18
