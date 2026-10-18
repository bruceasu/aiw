# 本地包与构建阶段

## Purpose

FD-080 定义基础构建和本地安装；FD-083 扩展统一包仓库、集合、更新和 Skill 管理，
只支持干净安装，不转换旧聚合包状态。

## Requirements

### Requirement: 显式构建阶段

build.py MUST 提供 compile/build/install，target 为组件，不接受 windows/linux target
或旧顶层动作。无参数 MUST 显示帮助且不安装。
compile MUST 只写 bin/，Go 组件 MUST 一次生成 Windows/Linux amd64 两套产物。
build MUST 复用已验证的固定 snapshot 编译产物；缺少所需产物时 MUST 自动 compile snapshot，再组装 dist；编译失败 MUST 不进入组装。
install MUST 只消费显式发布清单，不隐式编译、组包或下载。
Go 原始程序 MUST 发布到 dist/bin，plugin/skill 包 MUST 位于 dist/plugins 或
dist/skills 的版本子目录，其他发布文件 MUST 位于 dist/<category>/<version>。
一个 Go 插件包 MUST 包含双平台产物，安装 MUST 自动选当前平台。

### Requirement: Fresh local publication and build configuration

build.py MUST support `publish <package> <version>` and `publish-all <version>`.
Publication MUST consume existing ZIPs at the requested version and their exact
metadata-declared dependency/member closure through the existing Repo engine.
It MUST NOT compile, package or select a different version through latest manifests.
AIW_PACKAGE_REPO MUST name a local repository; missing or remote destinations
MUST fail before compilation or packaging. Relative paths MUST use the checkout root.
Existing immutable publication rules apply; a batch MUST stop on error and MUST NOT
claim atomicity across packages. Existing compile/build/install/export syntax remains valid.

The root .env MUST load for build.py and independent package scripts without
overriding process environment variables, executing commands or interpolating values.
AIW_PACKAGE_REPO, AIW_VERSION, AIW_GO, AIW_NODE and AIW_INSTALL_DIR MUST provide
local defaults for their respective repository, version, tools and installation settings.
Explicit command arguments MUST take precedence over defaults. Publication version
MUST be explicit. Go packages MUST retain Windows/Linux amd64 support.

The target catalog MUST distinguish programs, plugins, Skills, resources,
collections and compatibility groups. Existing short targets and bin/plugins/skills/all
MUST remain valid; canonical package names MUST also be accepted.
build all MUST include docs and all catalog packages with dependencies deduplicated.
publish-all MUST select existing versioned release artifacts, independently of current source membership.
The docs target and scripts/packages/docs.py MUST produce aiw-docs as a resource
package, with usage files under docs/reference/usage and README.md under docs/reference.
It MUST NOT own individual programs' docs/usage paths or include development reports/config.

#### Scenario: Publish a fresh named package

- WHEN `publish aiw-git 1.2.3` has a configured local repository
- THEN publish the existing aiw-git 1.2.3 ZIP and its exact declared dependency ZIPs
- AND do not consume any previous aiw-git manifest

#### Scenario: Missing local publish configuration

- WHEN AIW_PACKAGE_REPO is unset or is a remote URL
- THEN fail without compiling, packaging or publishing

Publication selection MUST be confined to the version directories under the five
standard dist categories. Named publication MUST select its canonical root and
declared closure; publish-all MUST enumerate all existing ZIPs for that version,
including packages removed from source later. All root filename/version identities,
duplicates (even identical content), safe paths and closure entry hashes MUST be
validated and original fetch paths fully materialized before the first repository
write. Missing/ambiguous/mismatched/unsafe artifacts MUST fail with a build-first
error and no implicit build or alternate version. Snapshot records, package ZIPs
and build manifests MUST NOT be changed by publication. Existing Repo immutability
and partial multi-package publication semantics remain in effect.

### Requirement: Fixed snapshot compilation and package versions

compile <target> MUST always refresh one fixed snapshot output slot and MUST NOT
accept --version or use AIW_VERSION. Versioned native executables MUST embed the
literal snapshot. Compilation MUST retain Windows/Linux amd64 output and write
only bin, including successful provenance records. Failed compilation MUST NOT
leave reusable overlapping records.

build MUST package the current validated snapshot regardless of its package version.
Without covering snapshot provenance or when ordinary native files are missing,
build MUST automatically compile snapshot before packaging. It MUST NOT accept
--compile-version. Publish MUST consume existing packages independently of snapshot
availability. Explicit build package versions MUST NOT force native
recompilation. Legacy non-snapshot compile records remain readable but are not reusable.

Package identity MUST use build --version, then nonempty AIW_VERSION, then one UTC
snapshot-YYYYMMDD-HHMMSSffffffZ per invocation. Publish keeps its explicit positional
package version. All ZIP paths, metadata and exact dependencies/members MUST use
that package version. Native files retain snapshot identity. New component/target/group
build manifests MUST record compile_version=snapshot while older manifests remain readable.
Independent package scripts keep their existing package-version argument/env contract.

Reuse MUST compare complete native file sets and SHA-256 values. Missing ordinary
files MAY trigger automatic compilation only after every remaining artifact has
been checked for changes, additions, unsafe links and filesystem errors. Modified
or extra files, malformed records, links and permission failures MUST fail clearly
rather than be hidden by automatic recompilation. Source freshness is not implied.
Compilation/build/clean assume one writer. A forced compile refreshes source edits.

latest-compile [target] MUST remain a read-only compatible query, validating the
current covering snapshot and printing snapshot or failing clearly. It MUST NOT
compile automatically. latest [target] MUST print only the latest successful local
package version in chronological completion order rather than repository SemVer order.
New package manifests MUST be written only after successful packaging,
with completion time. Historical manifests use file mtime if
no completion time exists. Selected invalid records or missing packages MUST fail.
Compile-only operations MUST NOT change package-build records.

clean MUST delete only checkout bin/ and dist/, including their records and staging
repository. It MUST preserve external repositories, installed files, source and workflow
state. Before deletion it MUST verify exact resolved targets and preflight both trees,
rejecting links/junctions and non-regular files. Missing directories are harmless;
filesystem cleanup failure MUST report actual partial completion without claiming atomicity.

#### Scenario: Snapshot becomes a release package

- WHEN compile aiw succeeds and build aiw --version 1.0.1 follows
- THEN reuse the unchanged snapshot binaries and package dependencies at 1.0.1
- AND do not run the compiler again or change its embedded snapshot identity

#### Scenario: Snapshot missing

- WHEN build has no covering snapshot provenance or required native files are absent
- THEN validate remaining output and automatically compile snapshot before packaging
- WHEN another remaining artifact is modified, extra, unsafe or unreadable
- THEN fail clearly without masking the error through recompilation

#### Scenario: Package version omitted

- WHEN build has neither --version nor nonempty AIW_VERSION
- THEN generate one UTC timestamp package version while native identity stays snapshot

#### Scenario: Failed build

- WHEN compilation or packaging fails
- THEN do not replace successful package-build manifests

#### Scenario: Safe clean

- WHEN a link/junction exists anywhere in either output tree
- THEN fail before starting deletion
- WHEN both output trees are absent
- THEN succeed without touching external repositories

### Requirement: Build-local installation without duplicate repositories

build MUST only assemble packages and successful release manifests; it MUST NOT
publish, copy ZIPs to dist/repository, write repository indexes or emit Published.
Only explicit publish/publish-all/export MUST write a requested repository.
Existing dist/repository output MUST NOT be deleted automatically or used for install.

build.py install MUST resolve exact dependencies/members directly from original
dist ZIPs using the shared Operation engine and a read-only in-memory catalog.
Its fetch MUST return the original validated source path, not copy a ZIP or download.
New component manifests MUST include the dependency/member closure. Catalog entries
MUST remain separate from roots: a single component/collection selects only its
requested canonical root, while compatibility groups retain existing root behavior.
Dependencies MUST retain the shared engine's auto/manual semantics.

For historical incomplete manifests, only exact declared name/version ZIPs under
dist/mist, plugins, skills, resources and collections version directories MAY be
resolved. No repository, latest-version or network fallback is allowed. Missing,
mismatched, conflicting duplicate or ambiguous artifacts MUST fail before destination
writes. Fetch MUST revalidate identity, metadata, size and hash. Shared graph/cycle,
platform, target ownership, exact-version and transaction protections MUST remain.

Before Manager construction, host-selected payload targets MUST be checked against
all original source ZIPs. A file target equal to a source ZIP or an ancestor of any
source ZIP MUST be rejected before writes, including across different packages.
Source links/path escapes MUST fail, and allowed destination root aliases retain
the existing Manager resolution policy. Necessary extraction/transaction staging
is distinct from duplicate repository or download archives and MUST remain intact.

#### Scenario: Build then install without publication

- WHEN build all at 1.0.0 succeeds
- THEN versioned ZIPs and closed manifests exist without dist/repository publication
- WHEN install aiw-core follows
- THEN install the collection and exact members from original dist ZIPs
- AND do not create another repository or ZIP copy

#### Scenario: Original archive protection

- WHEN one package's payload would overwrite another source ZIP or its parent
- THEN reject installation before any destination writes

### Requirement: 可移植包契约

包 MUST 命名为 name-version.zip，内含 info、data.zip 及可选 Python preinst/postinst。
info MUST 为 format_version=1 或 2 的 UTF-8 TOML，声明文件来源、目标、平台、SHA-256、
配置及可执行标记。解包 MUST 在操作专有临时目录完成；载荷与声明 MUST 完全对应。
安装前 MUST 完成完整性、平台、路径、类型及目标冲突预检；MUST 拒绝链接、
越界路径、重复目标、特殊文件以及覆盖 .aiw 状态区。
包文件错误 MUST 不回退为仓库查询。本地包输入 MUST 不因使用完成而删除。
安装选中的任一目标与本地输入 ZIP 真实路径相同时 MUST 在写状态或执行脚本前拒绝。
未来仓库下载的临时包 MUST 用后清理；正式备份及失败恢复材料不属下载缓存。

### Requirement: 共同安装状态与本次事务恢复

aiw pkg install/remove/purge 与 build.py install MUST 使用相同本地引擎。
--dest MUST 优先于 AIW_INSTALL_DIR 和平台默认值（Windows C:\green\aiw；Linux ~/apps/aiw）。
状态 MUST 位于 DEST/.aiw/packages/，不同根独立；MUST 阻止互相冲突的写入及未完成恢复后的新修改。
普通 install MUST 在备份后自动接管当前包声明的无归属普通文件；
MUST 保留既有配置、无关文件，并拒绝其他包归属冲突。
没有已安装记录时，接管 MUST 保留本次事务真实 before 快照用于失败恢复，成功后清理。
MUST NOT 将旧聚合包文件自动转移归属；旧安装用户 MUST 获得新目录重装说明。
用户指定的安装根 MUST 先解析为真实绝对路径，允许根及其祖先的链接别名；
根内部的路径及状态目录 MUST 继续拒绝链接/重解析点。
成功安装 MUST NOT 保留上一版备份或提供 rollback；旧 previous 状态 MUST 要求干净重装。
同版本同内容 MUST 跳过载荷和脚本；显式安装仍 MUST 将原因提升为 manual。
同版本不同内容的本地安装 MUST 正常重装，仓库发布另遵守不可覆盖规则。
内容指纹 MUST 忽略 ZIP 容器时间戳，覆盖实际元信息、载荷及脚本。

#### Scenario: 卸载而不回退

- WHEN v1 → v2 → v3 均安装成功
- THEN 当前为 v3，没有历史备份
- WHEN 执行 remove
- THEN 卸载程序文件，保留配置及其登记供 purge 使用，不恢复 v2

### Requirement: 失败恢复与清理

Python 脚本 MUST 显式调用解释器并记录退出码/日志，不依赖 Bash 或 Debian。
升级 postinst 失败 MUST 自动恢复旧受管文件与记录，保留配置和诊断并返回非零。
首次安装失败 MUST 撤销新增受管非配置文件，不产生成功记录。
恢复失败 MUST 保留现场并报告未完成，不宣称已恢复或安装成功。
脚本外部副作用及断电/强杀 MUST 不被宣称可原子回滚。
purge MUST 删除本包受管文件、登记配置和相关状态，MUST 保留无归属用户文件及集合成员。
临时清理失败 MUST 报告实际状态，不把已完成安装伪称为未安装。
聚合安装 MUST 逐包处理并在失败时停止，不承诺整批原子性。

### Requirement: 集合、依赖和安装原因

v2 MUST 声明精确 name/version 的 dependencies/members；只有 collection 可含 members，
collection MUST 无载荷和 hook。集合成员 MUST 不重复携带于集合包。
安装 MUST 在文件变更前检查全部包身份、路径、平台、缺失、循环、版本与目标冲突，
并校验最终状态中未更新包的反向依赖。不满足精确版本 MUST 失败，不隐式降级。
安装状态 MUST 记录真实成员、运行依赖及 manual/auto，已有 manual MUST NOT 降级。
集合 remove MUST 保留成员并显示孤立包及显式清理/保留提示。
autoremove MUST 只卸载从所有 manual 根不可达的 auto 包且保留配置；dry-run MUST 不写状态。
mark manual MUST 不重新部署或下载包。保留包仍依赖的包 MUST 不被单独移除。

### Requirement: 更新范围与命令整合

aiw update MUST 与 aiw pkg update 一致。默认 MUST 只更新已安装内容，MUST NOT 补装集合新增成员，
MUST NOT 记录未安装成员；必需运行依赖可补装且 MUST 展示。--core/--all MUST 互斥，
MUST 更新并补装固定发行集合，MUST NOT 卸载范围外包或遍历第三方仓库作为发行清单。
所有 dry-run MUST 不写安装文件/状态或执行 hook。
旧 plugin/skills 命令 MUST 删除；pkg skill MUST 保留原作用域、受管记录及本地修改保护。
pkg entries MUST 保留可用插件入口元信息，与仓库列表和已安装包列表区分。

### Requirement: 统一仓库与不可覆盖发布

local、HTTP(S)、S3、SSH MUST 通过统一 Repo 接口自动按地址选择，无类型探测或失败回退。
盘符和当前平台本地路径 MUST 不被误判为网络协议；未知 scheme MUST 明确失败。
读取 MUST 用 url，写入 MUST 优先 publish_url；相对配置路径 MUST 相对配置文件目录。
规范布局 MUST 为 name/version/name-version.zip 和根 index.json（schema=1、packages 数组）。
索引 MUST 包含 name/version/kind/path/size/sha256/fingerprint/dependencies/members/platforms。
下载 MUST 验证大小、ZIP SHA-256 及包内身份，并清理操作临时文件。
latest MUST 使用 SemVer，稳定版高于同版本预发布；dev 最低，非 SemVer 仅显式指定，歧义拒绝。
发布 MUST 串行协调索引写入；同身份同规范化 fingerprint MUST 成功且不修改已有包/索引，
不同内容 MUST 拒绝并要求新版本，MUST NOT 提供 force。指纹忽略 ZIP 容器时间戳，包含元信息、载荷及 hook。
包 MUST 先校验并上传，再发布索引；失败 MUST 不让半成品出现在索引中。
删除 MUST 先更新索引再清理对象；不确定结果、恢复或清理失败 MUST 如实报告。
S3 MUST 复用已有 aws 工具和条件锁，不自动接管锁；SSH MUST 复用系统 ssh/scp 与远端共同 filesystem 写入脚本。
凭据 MUST NOT 写入包或索引；哈希校验 MUST NOT 被描述为发布者认证。

### Requirement: HTTP POST 写入协议（仅规范，未实现）

本期 HTTP adapter MUST 仅读取；写入地址为 HTTP 时 MUST 不发送写请求并明确不支持。
未来管理接口为 `POST /api/packages`，Content-Type 为 application/json，认证通过
`Authorization: Bearer <token>` 传递，由服务端验证；MUST NOT 将 token 写入索引。
publish 请求字段为 operation="publish"、name、version、sha256、fingerprint、archive_base64；
archive_base64 为完整 ZIP，name/version MUST 与包内身份一致，摘要 MUST 验证。
unpublish 请求字段为 operation="unpublish"、name、version；MUST NOT 使用 GET 改写仓库。
成功响应为 JSON：status（published/already_exists/unpublished）、name、version。
首次发布返回 201；同内容幂等发布及删除成功返回 200；错误 JSON 含 error/code/message。
400 为非法请求/包，401 为缺少或无效认证，403 为无写权限，404 为包不存在，409 为不同内容同身份或写锁冲突，
413 为载荷过大，500 为存储/索引失败。并发写入及不可覆盖规则与其他 adapter 相同。
此规范 MUST NOT 被报告为已交付 HTTP 客户端、服务端、认证系统或部署。

### Requirement: 独立构建与更新入口

每个插件和 Skill MUST 有独立 scripts 处理入口，共享同一包定义与固定 core/extension 清单。
构建集合 MUST 输出独立成员及集合包，显式本地仓库产物 MUST 可直接读取；普通 build/install MUST 离线。
agent-templates MUST 随主程序；共享根 Markdown MUST 唯一归属且可按独立 Skill 依赖安装。
usage MUST 归所属程序/插件，MUST NOT 整体安装仓库开发文档。
Windows 文件占用失败 MUST 显示文件、退出重试和独立 Python 更新主程序的入口。
MUST NOT 自动终止程序、启动后台更新器、强制覆盖或安排重启替换。
pkg 自更新 MUST 在首次修改前准备所需模块、配置、包与恢复数据；官方 pkg MUST 不带 hook。

### Requirement: Unique program usage document ownership

Program packages MUST select docs/usage documents using the longest matching
package-name prefix, matching either the complete stem or a hyphen-delimited
suffix. A shorter package prefix MUST NOT claim a more specific package's guide.
The existing cz and aiw-issue aliases and explicit main-program documents remain
supported. The aiw-docs resource continues to use docs/reference paths.

#### Scenario: Nested plugin names

- WHEN packaging aiw-file and aiw-file-convert
- THEN docs/usage/aiw-file-convert.md belongs only to aiw-file-convert
- AND docs/usage/aiw-file.md remains with aiw-file.

### Requirement: Shared local installation planning

Build-local installation MUST select all requested roots before expanding a single
deduplicated dependency/member graph. Roots MUST retain manual-install semantics.
Before mutation the complete graph MUST pass integrity, platform, target ownership,
exact-version, cycle and existing per-operation aggregate input size checks.
The 512 MiB aggregate input-ZIP limit applies to the entire selected operation;
large groups MAY require separate explicit targets. Transactions remain per package.

Local index generation MAY reuse validated prepared metadata. Local fetch MAY
avoid repeated decoding only after safe-path, raw size and SHA-256 checks bind
returned bytes to the originally validated entry. Full package validation and
in-memory payload pinning before self-update MUST remain in Operation.

An installed matching version/fingerprint MUST skip temporary payload writes after
the installation lock and incomplete-transaction guard, preserving reason,
dependency and member record updates. No persistent validation cache is introduced.

### Requirement: Readable package lists with explicit JSON

`aiw pkg list` MUST default to an aligned table showing NAME, VERSION and KIND.
`list --installed` MUST additionally show REASON, displaying legacy missing reasons
as manual without changing stored records. Empty selections MUST have readable
messages. Only table rows MAY be sorted for display.

`list --json`, including `--installed --json`, MUST retain the previous complete
JSON arrays, metadata fields and record order. Existing --all, --kind, latest-version,
repository and destination selection MUST remain unchanged. Installed listing MUST
not contact a repository. Other subcommands MUST retain their existing output.
Scripts relying on the previous default JSON MUST explicitly select --json.
