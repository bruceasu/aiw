# FD-103: Folder based feature records and conflict free concurrent FD delivery

**Status:** Open
**Revision:** 3
**Priority:** High
**Evidence policy:** Dual

## Problem

FD 状态变化重写共享 FEATURE_INDEX.md，独立 worktree 在同一表格中修改状态或追加记录，导致可避免的合并冲突。活动设计和证据分散在三个共享目录。旧 shell 补全漏扫 archive/FD-ID 子目录。

## Options and decision

1. 仅移动 FD 文件：仍会重写共享表格，不能解决冲突。
2. 使用自定义 Git merge driver：增加本机配置依赖，也可能掩盖真实冲突。
3. 采用每个 FD 的目录和局部索引，全局只保留固定导航：选择此方案。不同 FD 只修改各自的受跟踪文件，继续使用 Git 标准冲突检测。

## Solution

- 活动目录为 docs/features/FD-ID/，设计保留 FD-ID_SLUG.md 名称，index.md 为局部导航，reports/ 和 reviews/ 属于同一 FD。
- 归档保留 docs/features/archive/FD-ID/，增加局部 index.md。历史设计和报告正文不改写，重复历史编号不合并也不覆盖。
- FEATURE_INDEX.md 改为固定导航，动态状态从 aiw fd list 获取。局部 index.md 只列设计与证据目录，不复制状态。常规状态操作不修改任何索引内容；不在每次事件扫描和重写其他 FD 的索引。
- 兼容读取旧平铺活动设计、旧平铺归档和旧 reports/reviews；创建、重新打开和重新审阅写入目录布局。证据清单同时支持新旧目录和 Git 历史。迁移时保持历史正文与 JSON 原样；历史路径作为证据溯源，不自动改写已签发事件或摘要。
- new/reopen/request-review/close 维护所属局部导航；archive 仍保留原事务回滚和冲突拒绝。用户给定旧 artifact 路径若已迁移且无原文件，可安全映射到唯一同 FD 新位置，存在歧义则报错。
- 更新 shell 补全、技能原始资源、模板、稳定规格和使用文档。安装副本及历史报告不改写。
- 不引入 merge driver、不对真实冲突选择 ours/theirs。不同 FD 的实现若修改相同源码，原有冲突恢复及审阅门禁继续生效。

## Scope

FD CLI、shell 补全、功能目录局部索引迁移及其当前文档契约。当前仓库只有 FD-103 需要从平铺活动路径迁移；已有历史归档不移动，正文及原相对路径不改变。第三方旧布局只兼容读取和操作，不隐式迁移。CLI 旧 artifact 路径映射不承诺浏览器书签重定向，物理路径变更需文档说明。局部索引归档时快照 active/archive 两处，允许更新生成索引，保留重复历史设计链接；失败还原两处。无依赖变更、网络、发布或安装。独立审阅通过后本地 squash 交付、清理并归档。

## Work items

- [ ] 1.1 实现新旧路径发现、创建和证据定位兼容，包括 git-wt 的 workspace/add 两处扫描。Small / Medium；无依赖；静态追踪创建和查询调用路径。
- [ ] 1.2 实现局部导航及归档/恢复兼容，消除共享状态表重写。Small / Medium；依赖 1.1；保留原事务回滚和真实冲突门禁。
- [ ] 1.3 迁移仓库功能索引与补全及当前文档契约。Small / Medium；依赖 1.2；历史正文保持原样，映射断链风险清楚记录。
- [ ] 1.4 完成编译、静态证据和独立审阅及本地交付。Small / Medium；依赖以上；如实记录未执行测试。

## Acceptance

- 每个功能有独立目录，局部 index.md 链接自己的设计和相关证据。
- CLI 可发现并操作新布局，同时读取旧布局；新建和重新打开使用新布局。
- 两个不同 FD 的状态变化不修改共享索引，不因索引状态或相邻表格行自动产生合并冲突。
- 真实源码冲突仍通过原 Git 合并/恢复路径报告，重复 FD 或证据目标碰撞仍拒绝覆盖。
- 当前文档和 shell 补全支持目录布局；历史证据原文和运行时已签发记录保留。

## TODO

完成独立设计反馈后发出 design-ready，在专用 worktree 实施。

## Verification

- 预算：不运行测试、最终构建、格式化、lint、网络或依赖下载。
- 实施后一次 compile-only：Python compile() 不留 pyc，并对变更的 Go completion 包运行离线 go build（包编译不留可分发文件）。
- 静态检查路径发现、artifact 解析、归档回滚、局部导航及标准 Git 冲突处理。实施后一次只读 diff 检查。
- 独立设计审阅和独立实现审阅分别记录真实事件与版本。并发运行场景未执行，以静态证据说明限制。

## Sources

- Issue: none
- 用户本轮迁移及并发 FD 执行请求。
- openspec/specs/fd-workflow/spec.md
- src/plugins/aiw-fd.py
- src/plugins/aiw-git/git-wt.py
