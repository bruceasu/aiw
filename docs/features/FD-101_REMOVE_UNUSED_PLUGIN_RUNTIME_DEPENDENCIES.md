# FD-101: Self-contained directory plugins without runtime coupling

**Status:** Complete
**Revision:** 6
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户更新 aiw-pkg 时被迫升级 aiw-plugin-runtime，继而被其他插件的精确依赖阻挡。
根因是 package_definitions.py 对全部插件无条件扩展 runtime，并生成与插件同版本的
精确依赖。aiw-pkg 自带 archive/manager/operations 等模块，并未使用 runtime。
用户明确要求修复这条不必要的耦合。

## Options and decision

1. 仅特殊处理 pkg，或为 file/patch/notify 保留 runtime：用户已明确要求三个插件也自包含，不采用。
2. 删除依赖但继续引用安装根共享文件：干净安装会缺文件，不采用。
3. 三个插件迁为目录式、每包私有辅助文件：采用。用户进一步明确要求由单文件改为目录。
   入口名/命令/协议不变，辅助实现源码仍共用，打包时各包携带完整实现。
4. 全部复制共享实现到三个源码目录：虽可自包含，但产生多份实现维护点，优先使用构建期复制。

## Solution

- 将三个顶层脚本移至 src/plugins/aiw-<name>/aiw-<name>.py，各加 schema=1 的
  plugin.toml，声明原命令 file/patch/notify、显式 entrypoint 和 Python startup。
  使用现有目录发现机制；不修改 Go 发现器或 manifest schema，不向 manifest 增加未知字段。
- 目录内含开发用辅助入口：file/patch 的 aiw_codec.py 通过 importlib 按绝对源码路径
  加载父目录已有唯一 aiw_codec.py，明确导出所需符号；notify 的 send_teams_msg.py
  用 runpy 转交父目录唯一实现，保留 __main__/argv 和 --managed-json 语义。
  这些开发 shim 不进入发行包；不复制源码实现，也不改变源树直接调用入口的能力。
- package_definitions.py 使用唯一的私有文件映射：file/patch -> aiw_codec.py，
  notify -> send_teams_msg.py。tree_entries 排除同名开发 shim，再从顶层唯一实现
  将真实内容写入 plugins/aiw-<name>/<helper>。辅助模块仅归各自包所有，路径互不冲突。
- notify 按 Path(__file__).resolve().with_name('send_teams_msg.py') 调用相邻私有脚本，
  不再从调用方 CWD/plugins 取脚本；配置、通信、凭证、超时、子进程隔离与重试语义不变。
- entries.py 的目录分支静态读取选定 Python 入口的 META 后合并 manifest 字段（manifest
  的命令名/入口/startup 优先），保留 commands/readOnly/mutatesFiles/requiresConfirmation
  等原元数据。不能把这些未知字段写进严格 schema 的 plugin.toml，也不能执行入口来取元数据。
  目录迁移导致元数据的 name 使用 manifest 命令名和新入口路径，与既有目录插件规则一致。
- 移除 component_dependencies 的全插件 runtime 分支和 build_component 的统一引用。
  pkg/file/patch/notify 及其他插件的单目标闭包/元数据不再要求 runtime。
- 两个顶层共享实现与 plugin-runtime 目标保留为旧发行兼容资源；显式 all/plugins
  兼容组原选择可保留，但新插件/集合闭包不因依赖自动拉入它，不主动卸载已有 runtime。
- 旧 ZIP/安装记录不迁移、不覆盖；新版本安装器沿现有 obsolete 文件清理移除该包旧单文件，
  写入同名包目录和 manifest，不触碰 runtime 旧共享路径。新旧插件版本可混装。
- 包格式、精确依赖检查、集合语义、外部命令和协议保持不变。物理入口路径变化是用户
  明确授权的迁移，文档说明直接调用路径变化；不修改无关插件形态。
- 原独立设计报告只覆盖 Revision 2，作为历史保留；本次范围变更需新的有界设计审查。

## Scope

范围：打包定义、三个插件入口/manifest/开发辅助入口、entries.py 静态元数据合并、
包管理使用文档与稳定规范。
排除：其他单文件插件迁移、兼容范围/解析器/Go loader 修改、已发布包重写、安装状态变更、
测试新增或运行、最终构建/发布/安装/网络。无关联 Issue 或 OpenSpec change。

## Work items

- [x] 1.1 三个插件迁为目录式，保留入口行为与源码开发路径；规模 small、难度 medium、无前置。完成证据：旧顶层入口迁入各自目录，三份严格 manifest、开发 shim、notify 相邻脚本路径及 entries 的目录内 Python AST META 合并已完成；未执行功能或网络。
- [x] 1.2 私有辅助文件打包及移除 runtime 依赖；规模 small、难度 medium、依赖 1.1。完成证据：PLUGIN_PRIVATE_FILES 同时驱动 shim 排除和真实模块私有目标写入，三个辅助目标互异且不归旧 runtime；删除规划和元数据的统一 runtime 依赖，普通单目标闭包不再拉入 runtime。
- [x] 1.3 文档规范及双格式证据；规模 small、难度 low、依赖 1.1/1.2。完成证据：使用文档和稳定规范明确目录/私有实现/无 runtime 默认依赖、旧发行兼容边界；离线 compile-only 通过，双格式实现报告齐备。

计划修订时，原 1.1 的依赖规则映射至新 1.2，原 1.2 文档映射至新 1.3；新增 1.1
承接用户明确追加的目录/自包含需求，没有已完成项或历史完成证据被重写。

## TODO

- 修订 3 目录方案独立设计审查 ready，见 FD-101-design-directory.md；计划 033da91 已提交，worktree/分支已创建，父分支 develop，Worker 领取 FD-101-000004-design-ready。
- 三项实现已完成，一次离线 compile-only 和最终只读差异检查通过；独立实现审查通过，待 PM 本地交付归档。

## Acceptance

1. pkg/全部插件的元数据和普通单目标闭包不再无条件要求 runtime；新 pkg 升级不替换 runtime。
2. file/patch/notify 各被发现一次，命令与参数协议不变，入口位于各自目录；entries
   保留 META 的行为/确认提示属性，manifest 名称和入口路径反映新目录形式，不执行源码。
3. 每个三插件发行目录均含真正辅助实现而非开发 shim，不依赖源码树、其他插件或 runtime；
   辅助目标无重复归属，file/patch 的 codec 源码实现仍唯一维护。
4. 源码入口可通过私有开发 shim 调用唯一共享源码，notify -I/--managed-json 的 argv/配置
   和失败契约不变，console 通知不触发 Teams 子进程。
5. 新包可与仍使用共享根文件的旧包混装；旧 runtime 载荷不删除、不换归属，旧单文件仅由
   本插件升级的现有 obsolete 清理处理，不留下两个可发现入口。
6. 包格式/安装器精确检查不变，旧包不修改，未实装前不声称安装或通知成功。

## Verification

- 静态证据：现有共享实现、Go 目录发现/manifest 允许字段、entries.py 目录支持、
  tree_entries exclude、Manager 所有权/obsolete 清理、notify 子进程与配置路径。
- 改变发布依赖兼容性，需要一轮独立静态设计审查；实现后由独立 Reviewer 检查真实源码。
- 一次离线 compile-only：内置 compile() 内存语法编译全部改动 Python（入口、开发 shim、打包定义），随后
  scripts/compile.py；GOPROXY=off/GOSUMDB=off/GOTOOLCHAIN=local，Go 输出 NUL，不留发行二进制。
- 一次最终只读 diff/status；不新增/运行测试，不执行 build.py 规划模拟或功能验证、
  最终包构建、lint/format/vet、验证脚本、网络、发布或安装。
- 当前设计证据：docs/features/reviews/FD-101-design-directory.md / .json，修订 3，ready。
  FD-101-design.md / .json 只保留用户追加范围前修订 2 的历史判断。
- 实际 compile-only：GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，Python 内存 compile()
  检查打包定义、entries 和三个目录的入口/开发 shim 共 8 个文件，再 runpy 执行
  scripts/compile.py（两个 Go 入口输出 NUL），退出码 0；无最终二进制或 pyc。
- 双格式实现证据：docs/features/reports/FD-101-implementation.md / .json。未执行安装或通知。
- 一次最终静态批次：git diff --find-renames 033da91 -- <打包/源码/文档/规范路径>、
  git diff --check、git status --short，无空白错误/无范围外改动；三处入口 rename、
  严格 manifest、codec 导出、Teams argv、私有排除/复制路径和精确依赖保留已静态核对。

- 独立实现审查通过：docs/features/reviews/FD-101-review.md / .json；Reviewer
  fd101-review-independent-20261019，来源 FD-101-000005-implementation-ready，
  基线 033da91、实现 5889483。静态核对目录发现、真实私有载荷替换、所有权/obsolete、
  notify 隔离协议和 AST 元数据；未执行测试、包构建、安装或通知，运行限制保持明确。

## Sources

- 用户明确要求修复，以及 FD-100 发布预检发现的 21 个反向依赖阻挡事实。
- scripts/package_definitions.py、build.py expand/build_packages。
- src/plugins/aiw-file.py、aiw-patch.py、aiw-notify.py、aiw-pkg/。
- docs/usage/aiw-pkg.md、openspec/specs/aiw-packages/spec.md。
