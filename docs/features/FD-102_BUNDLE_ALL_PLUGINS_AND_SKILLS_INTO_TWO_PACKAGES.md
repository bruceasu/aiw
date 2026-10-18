# FD-102: Bundle all plugins and skills into two packages

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户要求全部 plugins 合并为 aiw-plugins、全部 skills 合并为 aiw-skills，并明确无须兼容。
当前每个插件、技能、共享技能文档分别发包，带来过多产物及精确版本关联。
本 FD 切换新发行体系，不修改已发布 ZIP、旧仓库索引或现有安装目录。

## Options and decision

采用两个包含真实载荷的聚合包，而不是只引用独立包的 collection。
不保留 aiw-git、aiw-skill-* 等旧构建目标/别名，不支持原位迁移旧安装记录。
新包使用干净安装根，标准所有权冲突检查继续阻止覆盖旧独立包。
主程序/两聚合包不互加同版本精确依赖，避免换一种形式再次造成版本联动。
需要完整能力时使用 aiw-core/aiw-all 清单，不把技能文本引用的 CLI 当成包版本约束。

## Solution

- 新发行包：aiw-plugins（kind=plugin，dist/plugins）、aiw-skills（kind=skill，dist/skills）。
  程序 aiw-main/gateway/http-openai-proxy 和文档包 aiw-docs 保留其范围与名称。
- plugins 包含所有可发现插件的入口、manifest、内部模块、双平台原生可执行文件、
  配置/资源与插件使用文档。复用已校验的单插件载荷选择规则后合并文件列表，不生成中间 ZIP。
  file/patch/notify 保持目录布局，排除开发 shim 后携带真正私有辅助文件。
- skills 包含完整 canonical 技能树、支持文件及根共享 Markdown；原 aiw-resource-*、
  单技能包和 plugin-runtime 不再是发行目标。根共享文档唯一归属 aiw-skills，main 不重复携带。
- 构建/编译分离：expand 只返回实体包/collection；compile selection 对 plugins 展开为
  所有源码插件编译组件，确保 req/say/packer/file-convert 等双平台输出和 provenance 检查完整。
  compile/build/latest-compile 使用一致展开，复用、缺失检测和写前失效规则不放松。
- plugins/skills 从旧兼容组变为单实体目标。install 必须只选聚合包为显式 manual 根，
  不能把每个内容当依赖/根。all/bin 保留组合目标；aiw-core 含 main+plugins+skills，
  aiw-all 再含独立 gateway/proxy；可读 docs 继续单独选择或通过 build all 生成。
- 删除旧 plugins-core/all、skills-core/all 及 core/extension 元数据和单组件包装脚本。
  scripts/packages 只保留聚合 plugins/skills 和 docs 入口；其它程序由共享定义构建。
- 通用包仓库、archive schema、依赖/集合检查、事务/配置保留和路径归属保护不改。
  generic pkg install 仍是通用包管理接口，不对历史包名增加特殊屏蔽或新别名。
- aiw pkg skill list/install/sync 仍按 skills/<name> 操作技能，包级 install aiw-skills 与
  agent 级 skill install <name>/--all 是两个步骤；不改变单技能使用/选择名称。
- 使用文档、CLI 参考、技能开发说明及稳定规范同步新规则；历史 FD/Issue 报告不改。

## Scope

范围：build.py、scripts/package_definitions.py、packages 包装脚本/废弃清单、当前相关文档和规范。
排除：旧包迁移、旧别名兼容、安装状态/仓库变更、生产发布、运行能力代码重构、依赖下载、
网络、测试新增/运行、最终包构建。原生源码和插件/技能功能接口不改。
无关联 Issue 或独立 OpenSpec change。

## Work items

- [ ] 1.1 新发行目录/展开与编译组件分离；规模 small、难度 medium、无前置。完成证据：聚合 aliases、ordinary/all/core 闭包、compile provenance/native inventory、单根安装静态路径。
- [ ] 1.2 两实体载荷与新脚本，删除旧产物生产入口；规模 small、难度 medium、依赖 1.1。完成证据：全插件/技能覆盖、helper/配置/native 文件、无重复目标或依赖环、旧脚本/清单移除。
- [ ] 1.3 文档和稳定规范对齐新体系；规模 small、难度 low、依赖 1.1/1.2。完成证据：新命令/干净根/技能materialize/版本独立和旧语义清理。
- [ ] 1.4 compile-only、最终静态证据及独立审查交接；规模 small、难度 medium、依赖 1.1/1.2/1.3。完成证据：一次离线 compile、一次最终只读 diff/status、双格式报告与真实未运行限制。

## TODO

- 独立设计审查通过后提交 ready 计划并创建 feature/FD-102、.wt/FD-102。
- 完成全部项后独立实现审查、本地 squash 交付、worktree 清理、Complete 归档。

## Acceptance

1. build plugins/aiw-plugins 只生成一个载荷插件 ZIP，skills/aiw-skills 只生成一个技能 ZIP，
   不产生单插件/单技能/共享文档/legacy runtime 子包；不依赖这些旧包。
2. 全部发现的 core/extension 插件与技能均覆盖，原插件命令及技能名/agent 安装目录不变。
3. plugins 编译覆盖所有原生插件及平台，snapshot inventory 校验/缺失处理/失效保持真实。
4. 安装实体包仅记录一个 manual 根，支持直接本地 ZIP 和 build manifest；清单仍用原 graph。
5. 旧独立构建目标/脚本不继续生产旧包，旧根文件所有权冲突正常失败，文档要求干净安装根。
6. main/plugins/skills 版本不互锁，共享 Markdown 唯一在 skills；core/all 明确选取整套版本。
7. 不放松路径、文件哈希、配置、平台、事务或 512MiB 输入限制。最终包大小/运行行为未构建验证。

## Verification

- 用户明确允许不兼容；新布局/公共构建命令/包归属需要一轮必要独立静态设计审查。
- 静态追踪 package catalog/expand、source inventory、native compile/provenance、载荷collector、
  LocalPackages/source ZIP 保护、Operation graph/单根原因、skills source_root 与共享文档。
- 一次离线 compile-only：内存 compile() 编译变更 Python，再 scripts/compile.py；
  GOPROXY=off/GOSUMDB=off/GOTOOLCHAIN=local，Go 输出 NUL，不保留最终发行二进制。
- 一次最终只读 diff/status，独立 Reviewer 审查实现；不新增/运行测试、不执行功能/规划模拟、
  最终构建、lint/format/vet、验证脚本、网络、发布安装或清理旧输出/安装目录。

## Sources

- 用户本轮合包要求与“无须考虑兼容”。
- build.py、scripts/package_definitions.py、scripts/local_packages.py。
- src/plugins/aiw-pkg/{archive,operations,manager,skill_management}.py。
- docs/usage/aiw-pkg.md、openspec/specs/aiw-packages/spec.md。
