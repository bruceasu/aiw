# FD-080: Separate compile build install and local package lifecycle

**Status:** Complete
**Revision:** 20
**Priority:** Medium
**Evidence policy:** Dual

## Problem

现有 build.py 混合编译、发布暂存和安装：build_main、gateway 和
http-openai-proxy 会直接写安装目录，plugins/skills/docs 则使用不同复制路径。
无参数执行也创建安装目录，Linux 的默认安装路径仍是 Windows 路径。
需要拆分职责，并建立 AIW 自有的包格式及本地安装、升级、历史版本和卸载能力，
不依赖 Debian 工具链。统一远程分发及 skills/plugin 命令整合作为后续工作。

## Options and decision

1. 仅重命名现有动作：改动小，但仍混合发布和安装，也无法统一包记录。
2. 直接使用 DEB/dpkg：有成熟体系，但不符合 Windows 和不依赖 Debian 的要求。
3. 分离构建阶段，采用 ZIP 包和 AIW 安装引擎：选择此方向。

借用 DEB 的元信息、载荷、生命周期脚本和仓库概念，不宣称兼容 .deb。
build.py 负责源码工程的编译、组包及本地产物安装；本 FD 实现本地安装引擎和安装状态。
用户已确认本 FD 提供 aiw pkg install/remove/purge；仓库获取与发布由后续 FD 实现。
skills 保留发现、共享资源物化、项目/用户级安装语义，plugin 保留发现、
元信息和启动语义。后续整合时二者复用本地包引擎，不各自实现下载、历史和回退。
本 FD 仍为 plugin/skill 生成包并通过 build.py 安装；不改其现有管理命令。
用户在 grill-with-docs 讨论中确认按此范围拆分；当前文档不代表接口已经实现。

## Solution

### 构建阶段

建议入口：`python build.py <compile|build|install> [options] <target>`。
target 表示组件，保留 all 聚合含义；取消 windows/linux 平台 target，不要求用户选择平台。
无参数打印帮助，避免意外安装。用户已确认不保留旧动作入口或兼容别名。

| 动作 | 输入 | 输出与副作用 |
| --- | --- | --- |
| compile | 源码及已有本地依赖 | 仅 bin/；Go 可调用 go build；不安装、不组包 |
| build | 源码、已有本地依赖及源码资源 | 先调用所需 compile 写 bin/，再组装到 dist/；不安装 |
| install | dist/ 中选定的本机产物和包 | 仅写 --dest 指定目录；不编译、不组包、不下载 |

用户已确认 build 自动串联 compile：每次 build 先调用所选 target 所需的编译步骤，
成功后再组装发布文件；纯资源 target 无需编译。由编译器处理增量编译，
不能仅因 bin/ 已有同名文件就跳过编译。编译失败立即返回非零，不进入该次发布组装，
不得使用历史 bin/ 产物冒充本次构建结果。compile 和 build 复用同一编译实现。

发布目录遵循用户约定：Go 产物 dist/bin/；插件
dist/plugins/<version>/<package>-<version>.zip；Skills
dist/skills/<version>/<package>-<version>.zip；其他文件 dist/mist/。
保留 mist 拼写。编译与发布按平台/架构区分，安装只选当前平台，避免同时安装两套二进制。
默认目标目录：Windows `C:\green\aiw`；Linux `~/apps/aiw`。
建议优先级：显式 --dest > AIW_INSTALL_DIR > 平台默认值。
build.py install all 覆盖项目的所有可发布 target，不表示安装仓库内所有包。
保留 gateway.json、Say aiw.toml 和 cz 配置的现有保护规则。
旧 `build.py windows`、`build.py linux` 等顶层动作入口移除，不提供兼容别名；
无效旧语法应返回非零并提示新用法，不执行编译、构建或安装。
用户已确认：Go 组件的 compile 一次编译 Windows/Linux 两套产物；build 自动调用
该双平台编译流程，再组装包含两套平台产物的发布内容，不要求手动选择平台。
install 自动识别当前平台并只安装匹配产物，缺少匹配产物时明确报错，不回退到另一平台。
所选 Go 组件任一平台编译失败，整个 compile/build 返回非零，不进入发布组装。
这不表示 bin/ 内已生成的另一平台产物会被自动回滚；部分编译产物不能视为完整构建成功。

### 软件包与工作区

包工作区：`<package>/data/`，同级放置 info 和可选生命周期脚本。
外层 `<package>-<version>.zip`，根目录为 data.zip、info、可选 preinst/postinst。
用户已确认：preinst/postinst 内容统一为 Python，文件名保留包格式约定，
不依赖 shebang、可执行权限或 Bash/PowerShell；由 Python 解释器显式调用。
存在脚本时，必须在修改安装目录前确认解释器可用；Python 版本下限及解释器选择
在实现前依据现有 Python 入口约定确定并写入帮助，不自动下载解释器。
data.zip 内路径相对于安装根目录，不包含机器绝对路径。
建议 info 使用 UTF-8 TOML，包含 format_version、name、version、kind、支持的平台/架构及其文件映射、
文件清单、载荷 SHA-256、描述和配置文件清单。安装记录保存包来源及整包校验值。
格式版本必须显式声明；包名/版本禁止路径分隔符。
用户已确认：每个 Go 插件每版本仅生成一个 `<package>-<version>.zip`，
其中 data.zip 同时包含 Windows/Linux 两套二进制和公共资源；不按平台拆分外层包，
不在外层包名追加平台后缀。包内平台文件使用不同路径，并在 info 中明确
源路径到安装目标路径的映射，公共资源单独声明；具体路径随 2.2 固定。
安装引擎先校验整个载荷，再只发布公共资源和当前平台文件，另一平台文件不进入安装目标。
安装文件归属清单仅记录实际发布的文件；唯一上一版备份保留对应的原始双平台包
及恢复所需的受管文件快照与记录，不累积多版历史包。
元信息不能用单一 os 值误把整个双平台包声明为单平台包。

### 本地包命令

用户已确认：`aiw pkg install <包文件或包名>`，直接输入包文件即可本地安装；
只有输入包名才从仓库获取。`aiw pkg remove <包名>` 与 `aiw pkg purge <包名>`
处理当前安装根中的包记录，复用相同本地安装引擎。build.py install 也复用该引擎。
本 FD 实现包文件安装，以及 remove/purge；包名解析后访问仓库的分支留给后续 FD。
仓库功能尚未实现时，输入包名必须明确报错，不能声称已获取或安装。
文件路径与包名的识别应确定：已存在的普通文件作为包文件；显式路径或 .zip 文件名
不存在时报告本地文件错误，不能因路径拼错而回退到仓库请求。
已有但格式无效的包文件在本地报错，同样不能回退为包名。
安装版本以 info 为准并验证包文件与元信息一致；remove/purge 以包记录中的名字定位。
用户已确认重复安装规则：同包名、同版本且包内容相同，提示已安装并成功返回，
不执行 preinst/postinst，不改动安装文件、版本记录或唯一备份。
同版本但内容不同（包括 dev）按重新安装处理，执行正常预检、脚本及恢复流程；
成功后安装前的内容成为唯一备份，失败则保留原有备份并恢复安装前状态。
内容比较包含有效元信息、载荷及生命周期脚本，不能只比较版本或文件名；
2.2 定义内容指纹，避免 ZIP 时间戳等容器元数据变化被误认为内容变化。
同版本的当前内容和备份内容须能通过各自指纹区分，remove 按备份记录恢复，
不能因版本字符串相同而跳过恢复。
用户提供的本地包文件属于输入，不因安装完成而删除。

### 后续 FD 候选：仓库、索引与命令整合

本节保留后续需求与方案线索，不属于 FD-080 的实现、验收或完成门槛；
尚未创建后续 FD，不视为已批准的完整接口设计。
范围包括本地仓库定位、索引、HTTP/S3 获取与发布、适配协议，以及
skills/plugin 管理命令接入。目录安装与直接读取本地包仍属于 FD-080，
不需要先实现本地仓库。

统一相对布局：`<repo>/<package>/version/<package>-<version>.zip`。
最初的 tar.gz 示例视为仓库定位示意，首版采用后续明确的 ZIP 格式。
HTTP、本地文件系统及挂载网络盘使用相同定位协议；S3 由适配插件实现，
不把 AWS SDK 或凭据逻辑加入包核心。认证失败与下载中断不得影响已安装版本。
仓库访问只由显式 pkg 操作触发，build.py 保持离线。
建议仓库根 index.json 记录包元信息、版本、平台、相对路径、大小和 SHA-256；
精确版本可直接定位，latest 必须依赖索引中的明确版本，不靠目录顺序推断。
校验值提供完整性检查，不能代替可信来源或签名。
用户补充：从仓库获取的包使用后自动清理，不默认形成长期下载缓存；下载及
后续解压、预检均在该次操作专有的临时目录中进行。后续仓库 FD 必须继承此约定。
获取失败、中断或安装结束后均清理不再需要的下载文件；清理失败明确报告，
不能反向声称已经清理或把已经完成的安装回滚为未安装。
为恢复而正式保存的历史包与备份是安装状态，不属于下载临时文件，不被该清理规则删除。
publish 指令发布到仓库，可以通过命令行指定仓库地址的API，覆盖配置文件的设定。
同样支持多种保存仓库类型，包括local, http, s3. http 使用 post, s3 采用已经安装的awscli, 如果没有可以考虑用Go实现一个tiny s3 client 来 put, get, list.

### 安装状态与恢复

用户已确认安装状态固定保存在 `<安装目录>/.aiw/packages/`，统一记录当前版本、
文件归属、配置清单、历史备份、恢复日志及未完成操作。每个 --dest 使用其自身状态，
不能读取或修改另一个安装根的包记录。此目录是引擎保留区域，包载荷不得覆盖其中内容。
本地包与未来仓库包的解压、预检使用操作专有临时目录，不在用户当前目录、
源码目录或正式安装文件目录内展开。不先复制完整包到安装目标中再解压。
成功后清理操作临时目录；失败时清理无恢复用途的临时文件，日志和必要恢复材料
须保存到安装状态区或独立诊断位置后再清理，报告实际保留的位置。
预检外层/内层 ZIP，拒绝绝对路径、..、越界链接、重复路径和文件归属冲突；
校验平台、元信息和载荷后，再执行脚本或写目标文件。
升级顺序：预检/暂存 → 备份当前包文件及记录 → preinst → 发布载荷 → postinst
→ 提交当前版本记录。脚本失败必须返回非零，保留诊断和恢复材料。
用户已确认：升级期间 postinst 失败时，自动恢复升级前的受管文件和版本记录，
移除本次新增且仍由该操作拥有的非配置文件，保留用户配置、失败日志及必要暂存材料。
自动恢复成功后旧版本仍为当前版本，安装命令仍返回非零，不把失败的新版本加入成功历史。
若恢复本身失败，则记录为未完成恢复，报告失败步骤和恢复材料位置，保留现场，
不能声称旧版本已完整恢复或安装成功。
只能承诺引擎管理文件的恢复，脚本写外部系统产生的副作用不能自动回滚。
禁止自动提升权限；脚本只能在明确的包安装动作中执行。
脚本建议接收 name/version/action/old_version/dest，环境提供暂存目录与安装根，
不沿用示例中的 Tangram 全局配置、GET_CMD 或自动换行转换。

用户已确认：remove 移除当前版本，有历史版本时恢复上一版，保留用户配置；
无历史版本时移除当前受管包文件、保留用户配置，不凭空生成历史版本。
用户已确认每个包只保留一个上一版本备份：升级成功后，升级前的当前版本成为
唯一备份，更早的备份删除；升级失败保留原有备份，不提前删除或覆盖。
升级中的临时恢复快照不计为正式历史版本；它必须在升级完成或恢复完成前保留，
不能为满足单备份限制而破坏失败恢复能力。
remove 使用唯一备份恢复成功后删除该备份，不把被移除的版本反向备份；
若恢复失败则保留备份和恢复材料，明确报告未完成状态。
例如 v1 → v2 → v3 后为当前 v3、备份 v2；remove 成功后为当前 v2、无备份；
再次 remove 卸载 v2，保留用户配置。
用户已确认首次安装失败时，撤销本次新增的受管非配置文件，保留用户配置、
失败日志和必要恢复材料，命令返回非零；不能删除安装前已有文件或生成成功安装记录。
若撤销未能完成，记录未完成状态及剩余文件，保留恢复材料并明确报告。
purge 删除当前版受管文件、唯一上一版备份及登记为该包的配置，不恢复旧版本；
保留用户额外创建且不归属该包的文件，不递归删除整个安装目录。
帮助须明确 remove/purge 的配置保留差异和单备份规则。
恢复历史版本仍需满足文件归属约束，不能覆盖其他包或无归属的用户文件。
安装状态目录、首次安装失败撤销新增受管文件及升级 postinst 失败自动恢复均已确认。

## Scope

用户已确认分阶段交付：FD-080 覆盖 compile/build/install 拆分、ZIP 包格式、
包工作区、本地产物安装、Python 生命周期脚本、升级历史、remove 恢复与 purge。
HTTP/S3、仓库索引、本地仓库管理、publish 和 skills/plugin 管理命令整合
保留为后续 FD 候选，不阻塞本 FD 完成，也不能被标记为已实现。
首版不加入依赖求解、系统服务托管、发行版集成或隐式 sudo。
不在本次会话安装到默认系统目录，也不连接 HTTP/S3 或下载依赖。

## Work items

### 本轮实现决策（Planner）

- 共享引擎放在 src/plugins/aiw-pkg/，build.py 显式导入同一模块，
  aiw pkg 使用既有 plugin manifest/python 启动机制；不修改 Go 命令分派。
- Python 最低 3.11，使用标准库 tomllib、zipfile、hashlib、tempfile；不增加依赖。
- info 的 schema 为 format_version=1、name/version/kind、data_sha256，
  [[files]] 声明 source、target、platform（all/windows/linux）、sha256、config、executable。
  data.zip 的 source 为 common/...、windows-amd64/... 或 linux-amd64/...；
  target 相对于安装根。公共文件和选中平台文件映射后的目标必须唯一。
- 保留现有 amd64 范围，非 Windows/Linux 或非 amd64 安装提前报错；纯资源包 platform=all。
- 内容指纹覆盖解析并规范化的 info（不含 data_sha256）、每个载荷文件字节和 preinst/postinst
  字节，不受外层/内层 ZIP 时间戳影响。脚本、元信息与载荷均有固定大小/数量上限。
- 原始包中的声明文件与实际文件必须完全匹配；拒绝特殊文件、链接、Windows 保留名、
  盘符、反斜杠、大小写折叠目标冲突，以及已有目标路径中的链接或重解析点。
- .aiw/packages/<name>/ 保存 current.json、唯一 previous/、事务快照和诊断；
  引擎全局互斥锁拒绝同时写入，同步操作不重试，不自动接管失效锁。
  未完成恢复的事务禁止新修改，保留材料并报告；无需新增后台并发或自动修复机制。
- aiw pkg 的 install/remove/purge 均支持 --dest，默认解析与 build.py 一致。
  不覆盖无归属非配置文件；既有配置只登记并保留，升级、恢复不覆盖其内容。
  非配置文件若安装后被用户修改，删除/替换前报错并保留，不静默丢弃修改。
- remove 的版本恢复不执行安装脚本，只恢复受管非配置文件与版本记录；
  脚本外部副作用不属文件快照回滚承诺。purge 只删除该包记录中的文件与配置。
- 每包事务独立；install all 中某包失败即停止，已成功包保留，不宣称整批原子性。
  文件回滚涵盖常规异常；断电/强杀后保留事务材料并阻止继续修改，不宣称崩溃原子性。
- build 的编译阶段全部成功后才开始组装；dist 写入以临时文件替换包，不清空整个 dist。
  每个 target 的发布清单记录本次版本和包路径；install 只消费该清单，不扫描猜测 latest。
- aiw 主程序与 Gateway 同时保留 dist/bin 的两套原始程序；本地安装使用
  dist/mist 中带映射的对应包，目标程序名维持既有安装布局。
  proxy 的编译 JS 放 bin/http-openai-proxy/，发布包在 dist/mist/。
- plugins 按现有插件目录分别组包，根目录脚本和公共辅助文件作为 aiw-plugin-tools 包；
  req/say/packer/file-convert 的 Go 程序纳入双平台编译及对应插件包。
  skills 按含 SKILL.md 的目录逐个组包，根共享资源另成 aiw-skills-support 包；
  全部安装仍保留 canonical skills/ 布局，现有项目/用户级 skills 命令不改。
- Windows 正在运行的程序若拒绝替换，安装明确失败并按事务恢复，不增加自更新守护进程。


- [x] 1.1 明确安装状态目录；小/中；无依赖；状态目录为 <安装目录>/.aiw/packages/，首次安装失败撤销、升级 postinst 失败自动恢复、本地生命周期 CLI、双平台构建、安装自动选平台、不保留旧入口、Python 脚本和 remove 规则均已有用户确认；仅完成此决策项，不代表设计审查或实现完成。
- [x] 1.2 完成一次独立设计审查；小/中；依赖 1.1；结论 ready，无阻塞，证据 reviews/FD-080-design-r1.md 及 JSON；实现须先安装 skills 共享资源，remove 后保留配置登记供 purge 使用。
- [x] 2.1 build.py compile_components 统一组件入口生成 Windows/Linux amd64 两套产物；无平台 target、任一失败即异常退出；无安装路径。
- [x] 2.2 定义 info 和 ZIP 包读写、工作区规则；小/中；依赖 1.2；archive.py 实现 TOML 映射、校验与规范化指纹、双层 ZIP 安全读取及工作区组包；静态读取写入字段一致，待最终 compile-only。
- [x] 2.3 build 先完成所有选中组件 compile 再 package_component；主程序/Gateway 发布双平台原始程序及映射包，proxy/docs 等发布到 mist；发布清单供 install 使用。
- [x] 2.4 build 按插件目录及 SKILL.md 目录组包，Go 插件一个 ZIP 包含双平台产物；skills-support 排在 skills 前，保留 manifest 和共享资源。
- [x] 3.1 archive.read_package 全量预检并在专有临时目录展开；只安装匹配平台，不删除本地输入包。
- [x] 3.2 Manager 在各安装根 .aiw/packages/ 记录文件归属；全局互斥、拒绝状态覆盖和其他包冲突；配置保留登记。
- [x] 3.3 Manager.hook 通过当前 Python 显式调用，校验解释器后才写安装根；参数/环境及失败日志已实现。
- [x] 3.4 Manager 保存 before 快照、成功后轮换唯一 previous；异常自动恢复且保留诊断，恢复失败保留 transaction 并阻止后续写操作。
- [x] 3.4.1 首次安装失败恢复 before 的缺失文件状态，保留配置登记及诊断；不产生成功安装记录。
- [x] 3.4.2 版本与规范化指纹均相同则跳过；同版本不同内容正常重装，previous 使用快照和旧记录恢复。
- [x] 3.5 Manager.remove 统一 remove/purge；恢复成功才消费备份，配置保留或按 purge 清除；无独立 rollback 命令。
- [x] 3.6 build.py install 消费本次 target 发布清单并调用共享 Manager；不编译或扫描 latest，all 逐包处理。
- [x] 3.7 aiw-pkg.py 提供本地 install/remove/purge/pack；文件错误不回退仓库，包名入口明确报告仓库能力未实现。
- [-] 4.1 index 格式和本地仓库定位：用户确认移至后续 FD 候选，未实现。
- [-] 4.2 HTTP 获取和缓存完整性：用户确认移至后续 FD 候选，未实现。
- [-] 4.3 仓库适配协议：用户确认移至后续 FD 候选，未实现。
- [-] 4.4 S3 适配：用户确认移至后续 FD 候选，保留 awscli/tiny client 方案讨论，未实现。
- [-] 4.5 plugin 管理命令接入共同包引擎：用户确认移至后续 FD 候选，未实现。
- [-] 4.6 skills 管理命令接入共同包引擎：用户确认移至后续 FD 候选，未实现。
- [x] 5.1 pkg manifest/CLI 帮助、docs/usage/aiw-pkg.md、既有 cli-reference/cz 文档及 aiw-packages/cli-and-plugins 稳定规范已更新，说明命令迁移、路径、单备份及恢复限制。
- [-] 5.2 原合并项混合实现验证与后续生命周期，取消并拆为 5.2.1；独立审查/交付/归档由 TODO 跟踪，不在实现阶段假报完成。
- [x] 5.2.1 compile-only 与静态实现证据；小/低；依赖全部实现项；四个 Python 文件内存 compile 通过，git diff --check 通过，见 reports/FD-080-implementation-r1.md 及 JSON。

## TODO

- 创建时索引阻塞和过期 Planner 交接均已通过受支持 CLI 恢复，原临时文件已清理，详见两份 blocker 的恢复结果。
- 1.1 所列决策均已确认；设计审查、实现及独立实现复查通过，待 PM squash 交付和 Complete 归档。
- 每包只保留一个上一版本备份已确认；remove 恢复成功后删除备份、不反向备份，purge 清除当前版、备份及登记配置。
- 重复安装规则已确认：同版本同内容跳过，同版本不同内容重装；内容指纹纳入 2.2 的包契约设计。
- 解压使用专有临时目录；未来仓库下载包用后清理，正式历史与必要恢复材料单独保留。
- 范围拆分已确认；后续 FD 候选尚未分配编号，不为其宣称进度或建立实现依赖。
- build 自动串联 compile 已确认；install 保持只消费已有发布产物。
- Go 插件每版本一个 ZIP 包含 Windows/Linux 两套产物已确认；2.2 明确包内平台与公共文件映射。
- Worker 已在 feature/FD-080、.wt/FD-080 完成实现；父分支 develop，独立复查已通过，交付/归档尚未完成。

## Acceptance

compile 不触及 dist/ 或安装目录；build 先调用所需 compile，再写 dist/，不写安装目录；install 不隐式构建。
修改 Go 源码后直接 build all 会先编译；即使 bin/ 有旧产物，编译失败也返回非零且不进入发布组装。
纯资源 target 可直接组包，不要求不存在的编译步骤。
Go 组件的 compile/build 一次生成 Windows/Linux 两套产物，无需用户选择平台。
任一平台编译失败即返回非零，不进入发布组装；不能以另一平台成功代替整体成功。
install 自动选择当前平台，缺少匹配产物时报错，不能安装另一平台产物或触发编译补齐。
旧动作语法及 windows/linux target 返回非零，不保留兼容别名且不产生构建或安装副作用。
版本化 plugin/skill 包具有规定的双层 ZIP 与元信息。所有 target 可单独安装，all 全部安装。
Go 插件包名保持 `<package>-<version>.zip`，单包包含 Windows/Linux 两套产物；
安装同一个包时，Windows 只发布 Windows 文件和公共资源，Linux 只发布 Linux 文件和公共资源。
Windows/Linux 默认目录和显式 --dest 生效。已有用户配置保留。
安装记录、历史备份和恢复日志保存于 <安装目录>/.aiw/packages/；不同 --dest 状态独立，包载荷不能覆盖该区域。
损坏、路径越界、平台不匹配和文件归属冲突在发布前失败；脚本失败不得记为成功。
每包仅保留一个上一版本备份；升级成功才替换旧备份，升级失败保留原备份。
同版本同内容重复安装不运行脚本、不写安装文件/记录、不替换备份，提示已安装并成功返回。
同版本不同内容执行重新安装，成功后安装前内容成为唯一备份；失败保留原备份并恢复安装前状态。
版本字符串相同不妨碍 remove 恢复备份中的旧内容；内容判断不能因 ZIP 时间戳改变而触发重装。
remove 有备份时恢复上一版，恢复成功后删除备份且不备份被移除版本；无备份时卸载当前版，均保留用户配置。
remove 恢复失败保留备份及诊断；purge 删除当前版、备份及登记配置，保留无归属用户文件。
升级 postinst 失败后自动恢复旧受管文件与版本记录、保留用户配置，命令返回非零。
恢复失败明确记录未完成状态及材料位置，不能虚报恢复成功。
首次安装失败时撤销本次新增受管非配置文件，保留用户配置和诊断，返回非零；
已有文件不删除、不产生成功安装记录；撤销失败明确报告剩余状态。
包解压和预检在操作专有临时目录进行；成功后清理，失败所需诊断/恢复材料先保存再清理。
本地输入包不删除；后续仓库获取的包使用后自动清理，正式历史副本不随下载临时文件删除。
preinst/postinst 统一由 Python 解释器调用，不能隐式按 Bash/PowerShell 执行。
本地安装直接消费 dist 产物与本地 ZIP 包，不依赖仓库索引或网络。
aiw pkg install 接收本地包文件；显式路径不存在或文件无效时本地报错，不触发仓库获取。
包名才进入仓库获取分支；本 FD 未实现该能力时明确报错且不修改安装状态。
aiw pkg remove/purge 和 build.py install 使用共同本地引擎，行为与文件归属记录一致。
HTTP/S3、索引、publish 及现有 skills/plugin 管理命令整合不属于本 FD 验收范围。

## Verification

### 独立实现复查 r2

Reviewer session=fd080-reviewer-20261018-6b2e94，source_event=FD-080-000019-implementation-ready；
审查 e145b40120c71325e490914caab1b7a63ab04167，结论 verification-passed。
报告 reviews/FD-080-review-r2.md 及 JSON；首轮 R1/R2 经有界静态复查已解决，
原 blocker-review-r1 保留原事实并标记 resolved。未执行测试、编译、构建、安装或网络；
通过不表示运行风险消失。待 PM 检查交付及归档门槛，不由 Reviewer 执行交付。

### 首轮审查修复（复查前历史）

首轮 Reviewer 发出 FD-080-000018-changes-requested；Worker 已认领并修复 R1/R2。
R1 区分原备份 rename 未成功与新备份创建，R2 在写状态前独立校验平台支持。
修复后四文件内存 compile 通过；未运行测试/安装/故障注入。
见 reports/FD-080-implementation-r2.md 及 JSON，待第二次独立审查。

### 独立实现审查 r1

Reviewer session=fd080-reviewer-20261018-6b2e94，source_event=FD-080-000017-implementation-ready；
审查 76b563d41c2a4daa1e2afc1d98c284729b12d584，相对 fafde4f，结论 changes-requested。
报告 reviews/FD-080-review-r1.md 及 JSON。阻塞项：备份轮换首个 rename 失败可能删除
原唯一备份；公共资源可能掩盖缺少当前平台程序，导致错误安装成功。
仅静态审查，未运行测试、编译、构建或安装；修复后需当前独立审查结论。

### 首轮 Worker 证据（历史）

source_event=FD-080-000016-design-ready；session=fd080-host-20261018-a19d7f。
Python 内存 compile 检查 build.py 与 aiw-pkg 的三个 Python 文件通过（4 文件，
无字节码或程序产物）；更新聚合发布清单后同命令重新执行一次通过。
`git diff --check fafde4f` 通过；静态追踪目标展开、平台映射、脚本调用、
配置登记、before/previous 轮换和 remove/purge。见 implementation-r1 报告。
未运行测试、Go/Node 编译、发布构建、安装、故障注入、网络或依赖下载。
根 scripts/compile.py 只编译未变更 Go 程序，本次采用更窄的 Python 语言级 compile。
跨平台运行、Windows 文件占用、崩溃恢复和 proxy 运行时依赖未经运行验证。
独立实现审查尚未执行，不把 Worker 静态分析表示为通过审查。

本轮 Auto：用户授权 force-emit 后成功认领 FD-080-000015-design-requested；
独立设计审查 ready，见 reviews/FD-080-design-r1.md，未跳过实现审查。
以下未执行说明保留为此前设计阶段事实；实际实现证据在完成后追加。

已静态阅读 build.py、根 compile 脚本、plugin 元信息入口、skills manifest，及
cli-and-plugins/aiw-packer 稳定规范。现有 packer 是归档工具，不拥有安装状态。
本次仅形成设计，不主张任何运行行为已验证；没有运行测试、编译、构建或安装。
实现后按变更范围执行一次 compile-only；Python 改动使用不落盘的 compile()，
Go 改动优先采用离线的最小范围编译，不能以 build.py compile 生成发布产物充当验证。
最终静态检查不超过一个命令；独立实现审查另按既有流程，不增加测试或重复审查轮次。
本设计涉及新 CLI 和持久化，设计就绪前需要一次独立设计审查，当前尚未进行。
本轮讨论已同步范围、工作项和验收，保留用户补充的 TOML 元信息与 publish/S3
方案线索；未执行独立审查或实现。取消项不计入本 FD 完成条件。
已依据用户确认移除 build 禁止调用 compile 的旧假设；自动串联仅为行为设计，未运行构建验证。
双平台构建及安装自动选平台已获用户确认，尚未运行交叉编译或安装验证。
单个 Go 插件 ZIP 包含双平台产物已获用户确认；仅更新设计，未生成或安装包。
本地 pkg 命令与按输入类型分流已获用户确认；未执行命令或访问仓库。
已记录用户确认的 postinst 失败自动恢复和下载清理/临时解压规则；未执行故障恢复或清理验证。
网络、运行测试、包安装演练及最终产物构建均未授权，不得表示已通过。

## Sources

- Issue: none

- 用户本轮需求及安装脚本示例。
- 用户确认重复安装：同版本同内容跳过，同版本不同内容重新安装，成功后安装前内容成为唯一备份。
- 用户确认只保留一个上一版备份：成功升级替换旧备份，失败保留；remove 成功恢复后删除备份、不反向备份；purge 删除当前版、备份及登记配置。
- 用户确认 <安装目录>/.aiw/packages/ 保存包记录、历史备份和恢复日志，不同 --dest 独立管理。
- 用户确认首次安装失败撤销本次新增受管文件，保留用户配置和诊断材料并返回失败。
- 用户确认升级 postinst 失败自动恢复旧受管文件和记录，保留配置及诊断材料。
- 用户补充仓库获取的包用后自动清理，解压等操作在临时目录执行。
- 用户确认 aiw pkg install/remove/purge：install 可直接输入包文件，输入包名时才需要仓库获取。
- 用户后续明确选择：统一 Python 脚本；remove 恢复上一版并保留用户配置。
- 用户在 grill-with-docs 讨论中确认：本 FD 先交付构建拆分和本地包生命周期，仓库及命令整合另行规划。
- 用户确认 build 自动串联 compile；compile 可独立执行，install 不隐式构建。
- 用户确认不保留旧入口；Go 程序及 build 覆盖 Windows/Linux，用户无须选择平台。
- 用户明确选择：一次生成 Windows 和 Linux 两套产物，安装时自动选择当前平台。
- 用户确认一个 Go 插件 ZIP 包含双平台产物，保持 `<package>-<version>.zip` 命名。
- build.py；scripts/compile.py；src/plugins/aiw-plugin.py。
- openspec/specs/cli-and-plugins/spec.md。
- openspec/specs/aiw-packer/spec.md。
- src/skills/work-management.md。
- reports/FD-080-blocker-20261018-create.md。
