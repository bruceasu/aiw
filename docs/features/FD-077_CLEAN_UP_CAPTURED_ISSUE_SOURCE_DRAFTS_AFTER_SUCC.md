# FD-077: Clean up captured Issue source drafts after successful capture

**Status:** Design
**Revision:** 2
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

- [ ] 1.1 完成并独立审查清理设计，实施共享草稿生命周期和 Capture 警告载体。规模小到中等、难度中等；无前置实现依赖；完成条件：正确的保存顺序、路径范围、隔离核对与无覆盖恢复。
- [ ] 1.2 接入直接/聊天警告并同步 requirement 稳定规格和 Issue 帮助。规模小、难度低；依赖 1.1；完成条件：成功退出兼容、stderr 警告、文档边界一致。
- [ ] 1.3 完成 compile-only 与静态核对、双格式实现报告和独立 Reviewer 交接。规模小、难度低；依赖 1.2；完成条件：实际证据和未运行项如实记录。

## Acceptance

1. 两类项目草稿 capture 成功后，正式工件与元数据已写入，当前未改变的源文件被删除；其他文件不变。
2. 工件或元数据保存失败，不清理源。
3. 外部普通来源保持原样；不能清理路径穿越、链接、目录、被替换或改变的文件。
4. 清理失败保留正式结果和可恢复草稿，警告包含源或恢复路径；不提示盲目重复 capture。
5. 直接 CLI/聊天都呈现警告；公开函数签名、工件摘要、正常 stdout 和成功退出状态兼容。
6. 原正式工件、批准日志、其他历史草稿不被清理。

## TODO

- 进行一次必要的独立设计审查，再 emit design-ready。
- 在 develop 聚焦提交 ready plan；父工作区干净后用 feature/FD-077、.wt/FD-077 实现。
- 按项提交，独立实现审查通过后 local-merge、确认清理并 Complete 归档。
- 原父工作区门槛已获授权：ISSUE-00008 单独提交为 a732b98；阻塞反馈保留并更新解除事实。

## Verification

仅静态核对和一次 compile-only。预定环境 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local 后执行 python scripts/compile.py；该脚本仅 go build -o NUL 编译 cmd/aiw 与 cmd/aiw-req，不保留发行产物。未授权测试，不创建或运行 Go 测试，不运行 vet/格式化/构建/联网。

设计引入删除和失败恢复边界，因此进行一名独立代理的静态设计审查；它不替代独立 Reviewer 实现审查。当前 compile 和运行行为尚未验证。

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
