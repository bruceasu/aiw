# FD-100: Improve package search and show output

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户通过 `aiw pkg search skill-fd-package` 找到完整名称 `aiw-skill-fd-package`
的 1.0.1 / 1.0.2，却使用短名安装失败，并要求输出美观。
静态证据：search 对名称作不区分大小写的片段包含匹配；repository.select 则按
完整 name 精确相等筛选，再选版本。未匹配完整名称时，指定 --version 也无法命中。
这是命令匹配语义不同，不是仓库缺包或版本歧义。当前 show/search 均直接输出 JSON。

历史阶段：用户曾选择“先完成设计，我来清理父工作区”，当时未实现、提交或创建 worktree。
当前用户已明确授权 `FD-100 auto`，恢复正常本地实现/审查/交付/归档流程。
预检中原未跟踪脚本已不再出现在工作区差异内，本次只处理 FD-100 的文件。

## Options and decision

1. 只解释完整包名、维持 JSON：能解释安装错误，但未满足可读输出要求，不采用。
2. 增加名称缩写或模糊安装：可能扩大有副作用命令的解析范围，不在本需求中采用。
3. 复用 list 表格用于 search，新增 show 的对齐详情，提供 --json 保留机器接口：采用。
   不增加依赖、颜色库、终端宽度探测或持久化格式。info 继续原 JSON，不扩展范围。

## Solution

- search 增加 --json；默认复用 format_package_list，显示 NAME / VERSION / KIND，
  每个匹配版本一行，不折叠 latest。空结果显示现有 No packages found.。
- 表格仅作展示排序；search --json 保持原完整 JSON 数组、字段和仓库返回顺序。
  包名列展示完整名称，便于复制到 install，不加入自动别名。
- show 增加 --json；默认使用独立详情 formatter，对齐字段标签：Name、Version、Kind、
  Status、Reason、Dependencies、Members (recipe)、Files、Config files、Fingerprint。
  依赖/成员分行展示完整 name/version，空列表显示 none，缺少的标量显示 -。
  legacy reason 缺失时显示 manual，不改写记录；文件仅显示总数及配置文件数，
  完整文件数组和其他字段由 --json 查看。只显示记录里已有的值，不推断当前成员版本。
- show --json 保持原完整记录对象，包括文件和未知额外字段；不改变原 manager.record
  的查找、状态/路径校验、未记录错误或返回码。show 只看本地记录，不访问仓库；
  未安装包应使用 info 查看仓库信息。
- --json 仅新增到 show/search parser；输出分派需按 action 短路，不能让没有 --json
  参数的 info 等命令无条件读取 args.json。稳定规范原“Other subcommands MUST retain
  their existing output”应只对本次新范围之外的命令继续生效。
- 文档明确 search 使用片段、install/info/show 等按其现有完整名称语义处理。
  给出本例 `aiw pkg install aiw-skill-fd-package --version 1.0.2` 和完整名称搜索/详情示例。
- 这是默认输出兼容性变更，旧脚本需要显式加 --json；不修改 resolver、依赖、集合、
  archive/record schema 或安装状态，不执行用户贴出的安装命令。

## Scope

范围：src/plugins/aiw-pkg/aiw-pkg.py 的 search/show 展示与参数、docs/usage/aiw-pkg.md、
openspec/specs/aiw-packages/spec.md 的可读输出要求（实现阶段才更新稳定行为）。
排除：名称别名、模糊安装、info 格式、仓库/安装修改、新测试、依赖、网络和发布安装。
无关联 Issue 或独立 OpenSpec change。

## Work items

- [ ] 1.1 search 默认表格和 --json 兼容入口；规模 small、难度 low、无前置。完成证据：同名多版本、空结果、表格排序与 JSON 原顺序、完整字段保留的静态调用链。
- [ ] 1.2 show 对齐详情和 --json 兼容入口；规模 small、难度 low、无前置。完成证据：完整/空依赖成员、legacy 缺字段、removed 状态、文件计数和无仓库访问的静态调用链。
- [ ] 1.3 同步文档/稳定规范，记录命名诊断及编译/静态证据；规模 small、难度 low、依赖 1.1/1.2。完成证据：说明完整名称和 --json 迁移，更新 TODO/Verification 和双格式报告。

## TODO

- 设计及必要独立设计审查已完成，结论 ready、无阻断发现；用户已授权 auto，恢复实施。
- 提交仅本 FD 计划并创建 feature/FD-100、.wt/FD-100，完成三个 Work Items。
- 一次离线 compile-only、最终静态检查与独立实现审查通过后，本地交付清理并归档。

## Acceptance

1. search skill-fd-package 默认返回可对齐阅读的两条完整包名/版本/种类记录，空结果有短消息。
2. search --json 保留原 JSON 数组及索引顺序，不遗漏 path/size/hash/依赖/成员/platforms 等字段。
3. show 完整包名默认显示对齐详情；空依赖/成员清晰显示，完整名称与配方版本不截断。
4. show --json 原样完整序列化安装记录，未知额外字段和文件元数据保留；默认摘要不改记录。
5. show 不访问仓库；未安装/未记录包继续原错误语义，info 保持现有输出。
6. install aiw-skill-fd-package --version 1.0.2 按原精确名称与版本解析；
   install skill-fd-package 仍不自动解析，文档清晰解释与 search 片段匹配的差异。
7. 既有 list/entries/其他子命令默认和 --json、全部写操作及返回码保持不变。

## Verification

- 本轮静态诊断已读 main 的 search/show/install 分支、repository.select 和现有 list formatter，
  无运行复现、安装、仓库访问或网络。用户日志中的安装失败是已提供证据，不是本轮执行结果。
- 默认输出影响公共 CLI 兼容性，需要一轮有界独立静态设计审查，写中文 Markdown 和同名 JSON。
- 实现阶段：一次离线 compile-only 批次，内存 compile() 检查改动 Python，然后
  scripts/compile.py 输出 NUL；GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local。
- 实现阶段仅一次最终只读 diff/status 批次，独立 Reviewer 按验收审查；不新增或执行测试，
  不执行脚本功能、格式化/lint/验证脚本、最终包构建、push、网络或安装。
- 历史设计阶段未编译、未声称实现完成；本次用户恢复实施后按上述预算继续。
- 独立设计证据：`docs/features/reviews/FD-100-design.md` / `FD-100-design.json`，
  source_event=`FD-100-000002-design-requested`，审查 Revision 2，ready。
  未运行测试、编译、功能脚本、仓库访问或网络；设计完成不等同实现审查通过。

## Sources

- 用户提供的 search/install 输出及本轮设计-only 指示。
- src/plugins/aiw-pkg/aiw-pkg.py：format_package_list/main
- src/plugins/aiw-pkg/repository.py：select
- docs/usage/aiw-pkg.md；openspec/specs/aiw-packages/spec.md
