# FD-083: 统一包管理与 Package Repository

**Status:** Complete
**Revision:** 6
**Priority:** Medium
**Evidence policy:** Dual

## Problem

ISSUE-00011 要求统一包查询、仓库获取、安装、更新、发布与 Skill 作用域管理。
现有实现只有本地 ZIP，聚合包缺乏独立身份，remove 会恢复历史版本。用户已批准删除旧入口、只支持全新安装、取消历史备份。

## Options and decision

拒绝继续扩展旧聚合包及迁移：与确认范围冲突。采用 Python 标准库统一引擎、精确依赖、固定发行清单、Repo adapter。
保留 v1 独立包读取；v2 支持集合及依赖。S3/SSH 复用已安装工具，不新增依赖、服务端、后台进程或整组原子回滚。

## Solution

### 包与安装

- v2 在 v1 基础上增加 dependencies/members 数组，每项只有 name/version；collection 无载荷和脚本，允许嵌套。普通包禁止 members。
- DFS 在写受管文件前获取并校验全部包，拒绝缺失、循环、同名版本冲突、平台不匹配及目标冲突；逐包事务，失败停止并报告已完成部分。
- current.json 保存 kind、dependencies、members、manual/auto；集合只有记录。已有 manual 不降级；同内容显式安装仍提升为 manual。
- remove 真正卸载、保留配置登记；purge 清该包登记配置，不级联。取消 previous，但保留本次 before 事务及失败恢复材料，成功清理。
- 旧 previous 状态明确拒绝并要求干净重装，不接管其他包归属。共享文档唯一归属，独立 Skill 显式依赖共享资源包。
- autoremove 从 manual 根遍历已安装成员和依赖，remove 不可达 auto 包；mark manual 只改原因。移除被其他包依赖的包拒绝并显示引用。
- 默认 update 只选已安装包，集合新成员不补装；保留真实的已安装成员子集。必要运行依赖可补装并展示。--core/--all 展开固定根集合，根 manual，其余保留旧原因或新增 auto。不删除范围外包。
- dry-run 只输出计划，不创建安装状态、不执行 hook。Windows 占用错误提示具体路径、退出重试；主程序需提供不经 EXE 的独立 Python 入口。恢复失败非零并报告 transaction 路径。

### Repo 与协议

- Repo 统一 index/fetch/publish/unpublish。--repo 显式覆盖读取与写入两端；未指定时按配置 url/publish_url 分离，向上查最近 aiw.toml/.aiw.toml，再查 AIW_ROOT；相对路径以配置目录为基准，--repo 以 cwd 为基准。
- 写入取 [repo].publish_url，缺省 url。按操作地址识别 local/HTTP(S)/S3/SSH；盘符、UNC、~ 作为本地路径，未知 scheme 拒绝，无探测/回退。
- 规范布局 name/version/name-version.zip，根 index.json（schema=1、packages 数组）。条目包括身份、kind、规范 path、size、sha256、fingerprint、dependencies、members、platforms。下载有界，校验包内身份和完整性，临时文件用后清理。
- latest 按 SemVer major.minor.patch，稳定版高于预发布，dev 最低；其他历史版本仅可显式选取。同优先级不同标识拒绝歧义。
- local 写入独占锁目录串行化，持锁读取最新索引；先验证暂存包，再写包和原子索引。相同身份同 fingerprint 不修改包或索引，不同内容拒绝，无 force。失败清理未索引的新包；删除先移除索引，再删包并如实报告清理失败。
- HTTP 只 GET。HTTP 写入仅定义 POST /api/packages：publish/unpublish 操作、JSON/base64 ZIP、Bearer 认证、幂等及错误码；本期不实现写入客户端/服务端。
- S3 调用 aws s3/s3api，条件创建锁对象（If-None-Match）协调写入，持锁拉取最新索引；不自动夺锁或安装工具。系统凭据不进入包/索引。
- SSH 使用 ssh/scp 暂存上传，远端 Python 调用共同 filesystem 写入逻辑与锁。远端需预装 pkg 脚本；ssh://[user@]host/absolute/path 支持 Host 别名。正确引用路径、拒绝选项注入；不从 HTTP URL 推测远端路径。

### 命令与构建

- aiw update 转发 pkg update，主入口拒绝旧 plugin/skills。原 Skill 实现移动到 pkg，保留 pkg skill 子命令及 skill-* 拼写；受管记录、共享资源物化、作用域与本地修改保护保持。
- pkg 提供 list/search/info/show/install/update/remove/purge/mark/autoremove/index/publish/unpublish/pack、entries --json；entries 保留插件入口路径/命令/只读等元信息。list --installed --kind 面向安装记录。
- 每个 plugin/skill 在 scripts 下有独立打包处理脚本，调用共同打包库。plugin-meta.toml、skills-meta.toml 固定 core/extension，core 清单沿用 Issue，显式展示运行依赖。aiw-core/aiw-all 嵌套 plugins-core/all、skills-core/all，集合精确版本。
- build.py 保持 compile/build/install 隔离，显式仓库导出产物可由 pkg 消费；普通动作离线。agent-templates 随主程序，usage 按所属包分配，根共享 Markdown 由独立资源包唯一拥有；主程序按需依赖资源包。不安装整个开发文档目录。
- pkg 更新前导入安装所需模块；运行中不依赖随后被替换的源码。独立主程序安装提示：python <AIW_ROOT>/plugins/aiw-pkg/aiw-pkg.py install <ZIP> --dest <ROOT>。

### 独立设计审阅后的边界补充

- update 先独立选取目标包最新版本，再解精确依赖；同一包有不同精确版本要求时直接拒绝，不隐式降级、选择旧发行版或静默跳过。
  对最终候选状态与所有未更新的已安装包做全图依赖校验，含反向依赖；任何已安装依赖约束被破坏则写入前拒绝。
  默认更新集合时只记录原有且新清单仍声明的成员，新成员不记为已安装；成员所需精确版本与候选不符同样拒绝。
- S3 在持条件锁期间拉取当前索引和同身份已有对象；先写不可覆盖的新包，再提交索引。
  索引提交失败，删除本次新上传且未索引的对象；若远端响应不确定或清理失败，报告路径和未完成状态，不声明完整恢复。
  unpublish 先提交删除条目的索引，再清理对应对象；对象清理失败不伪称包仍可查询，报明确错误。锁不自动接管。
- 所有下载、解包、依赖图、配置和安装所需模块在第一次目标文件变更前准备完成；官方 pkg 包不带安装 hook。
  用户包 hook 外部副作用仍不保证恢复。执行期间不重新导入刚更新的 pkg 模块，不启动后续解释器读取它们。
  真正被锁定的主程序通过独立 Python 入口在用户退出相关 EXE 后更新；不承诺 aiw update 能在运行中替换 aiw.exe。
- 每个 plugin/skill 的独立处理脚本由共享生成器产生短包装文件，路径固定为 scripts/packages/plugin-<name>.py、skill-<name>.py；
  包定义由共同打包模块实现，避免每份脚本重复维护文件清单。构建时缺少独立脚本即报错，不动态回退聚合旧包。

## Scope

覆盖 Issue 全部验收，排除旧聚合安装迁移、HTTP 写入实现、依赖下载、部署、后台更新器、占用预检、强制覆盖及整组回滚。只更新受影响稳定规范，不创建 OpenSpec change。
CLI、持久化和写入协调变化均属于已批准 Issue；实施前独立设计审阅。

## Work items

- [x] 1.1 v2 包元数据（小/中；无依赖）：集合、精确依赖校验及 v1 读取兼容。新增 references，v2 精确 schema，集合禁止载荷及脚本，写包和 pack 同用格式。
- [x] 1.2 Repo 获取与配置（小/中；1.1）：repository.py 统一读取、严格索引身份与路径、SemVer、配置作用域及有界读取；下载与完整包元数据核对。
- [x] 1.3 仓库写入（小/中；1.2）：repository.py 独占锁、不可覆盖和原子索引；S3 条件锁与提交结果核对；SSH 上传并复用远端 filesystem。HTTP 实现明确拒绝写入，POST 规范随 1.8 落盘。
- [x] 1.4 安装与卸载（小/中；1.1）：Manager 取消 previous/current.zip 留存，拒绝旧备份迁移，保留 before 恢复；记录成员/依赖/原因，增加引用保护与 mark/orphans。
- [x] 1.5 集合与更新（小/中；1.2/1.4）：Operation 全图获取/平台/目标/反向依赖预检，根 latest 与精确冲突失败，默认集合成员过滤，显式范围/原因及 dry-run/autoremove。
- [x] 1.6 CLI 迁移（小/中；1.3/1.5）：pkg 全命令与 entries 元数据；Skill 实现移动，skill/skill-* 路由；Go update 转发、拒绝旧入口，更新帮助。
- [x] 1.7 独立打包与发行清单（小/中；1.1/1.3）：固定 core/extension TOML、每包 scripts/packages 入口、共同定义与递归依赖、资源唯一归属、集合及本地仓库导出。
- [x] 1.8 文档与稳定规范（小/低；1.6/1.7）：稳定规范定义新包/依赖/更新/卸载和 HTTP POST；重写包指南及配置模板，迁移活跃 Skill/帮助/调用，历史归档不变。
- [x] 1.9 交接证据（小/低；以上各项）：58 个 Python 文件语法编译及两 Go CLI 离线无产物编译通过；一次静态检查、中文报告与 JSON 已保存，交接独立实施审阅。独立审阅是后续门禁，不属于 Worker 自评。

每项预计半天以内，按依赖顺序实现并独立提交；如实现证据显示需再拆，保留编号新增子项。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

沿用 ISSUE-00011 的全部验收示例：仓库与发布→1.2/1.3；独立包/集合/共享资源→1.1/1.7；
卸载/配置/失败恢复→1.4；update/manual/auto/孤立包→1.5；Skill/入口/旧命令→1.6；
归属/构建/重装/Windows 提示→1.7/1.8。依赖/索引错误在受管文件变更前拒绝，dry-run 不写安装状态。
运行结果未实测时只报告静态依据。

## Verification

- 2026-10-18 独立实施审阅：`fd083-reviewer-20261018-92b7ad` 已 claim `FD-083-000005-implementation-ready`，比较 `4e5f2ff..49a544d`（产品 `9e38f3b`）；结论为静态 verification-passed，未确认阻塞缺陷。[Reviewer 报告](reviews/FD-083-implementation-review-20261018.md) 及同名 JSON 保存证据与限制，未运行测试/构建/网络或重新编译。
- 审阅非阻塞决定：PM 明确 `--repo` 显式覆盖读写两端，未提供时读取 url、写入优先 publish_url；原 Solution 只写读取覆盖，交付记录以此补充为准。Windows 恢复占用风险沿用 Issue 已确认边界，不要求本轮重构恢复算法。
- 当前剩余 TODO：PM 检查交付门禁后本地 squash 交付、验证清理、Complete 归档及归档提交。Review 通过不表示运行验收、远端发布或部署通过。

- 2026-10-18：Issue 已批准并创建本 FD；修复来源编码并重新 capture 后，CLI 拒绝对 APPROVED 状态重新审批，旧审批摘要未更新。自动流程停在 Planner，未进入实施。详见 [阻塞反馈](reports/FD-083-blocker-20261018-approval.md)。
- TODO：先通过受支持操作恢复 Issue 审批与当前来源的一致性，再完成设计、独立设计审阅及 design-ready；保留现有 FD 和交接事件。
- 恢复补充：用户 confirm 后备份原记录，显式恢复顶层 DECIDED，再调用正式 approve 成功；保留旧审批历史和本 FD，未重复晋级。
- 当前 TODO：独立设计审阅、design-ready、计划提交、隔离工作树、逐项实施、编译、独立实施审阅、本地交付与归档。
- 当前进度补充：设计独立审阅 ready；design-ready、计划提交、工作树及 Worker claim 完成；1.1–1.9 实施完成。两次同一离线编译命令通过，第二次覆盖恢复提示补充。一次静态批次检查最终 diff 和调用，无运行时验收。
- [实施报告](reports/FD-083-implementation-20261018.md)；产品代码版本 9e38f3b，基线 4e5f2ff。
- 交付结果（2026-10-18）：独立实施审阅通过，事件 FD-083-000006-verification-passed，本周期一次实施审阅；报告见 reviews/FD-083-implementation-review-20261018.md。
- develop squash 提交 8153bf1，FD-Source 为 0d53bfb555a609f5aa3f2aaff36cb28491b7b9db；CLI 验证交付后已删除 .wt/FD-083 和 feature/FD-083，保留运行回执。
- Complete 归档成功；本 FD 范围内 TODO 已完成，归档变更随专门提交保存。测试、最终构建、远端、安装、自更新及发布仍未运行，不视为通过。
- 非阻塞后续：.agents/agents/work-management.md 的旧辅助命令文字可在后续文档维护中更新；Windows 恢复占用风险按 Issue 已确认范围保留。
- 编译计划：离线环境执行一次 python scripts/compile.py；同一次命令对 Python 变更用 compile(bytes, path, 'exec')，不导入执行产品、不保留产物。
- 风险：不运行测试、构建、格式化、lint、vet、网络及发布；远端、Windows 占用、自更新及构建产物仅有静态证据。
- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

## Sources

- Issue: ISSUE-00011
- openspec/specs/aiw-packages/spec.md；src/plugins/aiw-pkg/{archive,manager,aiw-pkg}.py；build.py；src/plugins/aiw-skills/aiw-skills.py。

Link relevant specs and evidence used for decisions.

**Completed:** 2026-10-18
