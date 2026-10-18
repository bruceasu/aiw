# 统一包管理与 Package Repository：Issue 草案

状态：用户已于 2026-10-18 明确请求批准、晋级及 fd-workflow auto。
来源：用户本轮七项要求，以及随后明确的命令迁移决定。
当前结论：业务边界已确认，工程细节由 FD-083 设计。

## 目标与已确认范围

用户希望通过统一的包管理入口分发、安装和更新 AIW 主程序、插件及 Skills，
支持 package repository，并由 build.py 生成匹配的独立包和集合包。

已确认：

- `aiw pkg`，支持仓库管理及按包名获取。
- 本次直接删除 `aiw plugin`、`aiw skills` 旧入口，功能统一到 `aiw pkg`。
  这是用户明确要求的 CLI 不兼容变更，不保留兼容别名。
- 用户选择仅支持全新安装：旧用户按重装说明安装到新的干净安装目录，
  本次不开发旧聚合包的自动迁移、文件归属转移或旧安装状态转换。
  新安装之后的正常安装、更新与失败恢复遵守本 Issue 更新后的包管理契约。
- 插件和 Skills 按独立单元发包；集合包只声明成员，不重复装入成员文件。
- 提供 `plugins-core`、`plugins-all`、`skills-core`、`skills-all` 等集合包。
- 用户选择发布时同名同版本禁止覆盖：内容相同提示已存在，内容不同拒绝发布并要求更换版本号，
  不提供强制覆盖同名同版本的参数。
- HTTP/HTTPS 仓库本期只实现查询和下载。HTTP publish/unpublish 仅定义协议规范，
  发布和删除均采用 POST 到管理接口的方式；本期不实现 HTTP 写入调用或仓库服务端。
  本地目录和 S3 的显式发布/删除仍在原定范围内；用户随后确认本期增加 SSH 发布/删除。
- REPO 使用统一接口，local filesystem、HTTP、S3、SSH 均通过 adapter 实现。
  根据配置的仓库路径自动识别类型，不要求用户另行填写 type 或 adapter 参数。
- 用户确认 HTTP 读取 + SSH 发布方案：查询/下载使用 url，发布/删除优先使用 publish_url，
  未配置 publish_url 时使用 url。SSH 支持 .ssh/config Host 别名、IP 和域名，
  复用系统 ssh/scp 工具及现有 SSH 配置；远端运行仓库脚本，复用 filesystem 写入逻辑。
- 用户选择集合移除仅删除集合记录，保留成员包。记录自动/独立安装原因，
  提示通过显式命令移除孤立包，或将成员标记为独立安装；不自动级联删除成员。
- 用户明确调整卸载与备份规则：remove 真正卸载并保留配置，purge 卸载并清理该包配置；
  不保留上一版本备份，不提供 rollback 或其他用户回滚功能。
  本次失败恢复所需的临时事务数据仍保留，成功后清理，不形成可供用户回退的历史版本。
- `agent-templates` 随主程序发布；文档归各自程序或插件。
- 增加 `aiw update`，作为 `aiw pkg update` 的同义入口。默认更新已安装内容的最新版本；
  `--core` 更新或补装 core packages，`--all` 更新或补装本发行版的全部 packages。
- Windows 不引入临时/后台更新器，不自动结束或重启程序。占用阻止替换时，
  提示用户退出相关程序并重新执行安装；本次操作已经替换的受管程序/文件需要恢复。
  用户随后收简范围：不增加专门的占用预检或运行中覆盖机制，提供处理提示即可；
  失败恢复沿用现有安装事务契约，不据此引入新的整组回滚机制。
- 同步调整 build.py，支持包仓库，但实际构建、网络访问和发布不在本轮执行范围。
- build.py 把plugins/skills部分拆出来。每个plugin，skill有单独的build处理脚本，保存在scripts目录，build.py 执行他们。

## 当前实现与证据

1. `src/plugins/aiw-pkg/aiw-pkg.py`：已有本地 `install/remove/purge/pack`。
   按包名安装明确报告 repository 尚未实现；显式本地路径失败不回退到网络。
2. `src/plugins/aiw-pkg/archive.py:104`：现有 info 严格校验 format_version=1，
   仅接受 program/plugin/skill/resource，且 files 必须非空。
   因此集合包不是简单传入空 files 就能实现，需要明确新的元信息规则；
   旧版安装状态转换不在本次范围。
3. `build.py:56`、`build.py:61`、`build.py:135`：目录式插件和 Skill 已分别打包，
   但根目录 Python 插件集中在 `aiw-plugin-tools`；根级 Skill Markdown 集中在
   `aiw-skills-support`；docs 分支同时收集 usage 和 agent-templates。
   现有聚合 target 是构建清单，不是可由 pkg 独立解析的集合包。
4. `src/plugins/aiw-plugin.py`：旧入口主要提供 `list --json` 与入口元信息。
5. `src/plugins/aiw-skills/aiw-skills.py`：已有 list/discover/adopt/sync/install，
   project/user scope、受管记录、内容摘要和共享资源补齐。这些不能随着旧入口一并丢弃。
6. `src/cmd/aiw/main.go`：入口通过插件分派；当前无内建 update。
7. `openspec/specs/aiw-packages/spec.md`：本地安装器、文件归属、配置保护、单备份、
   逐包失败停止、下载临时文件清理等为现有稳定契约；当前不包含仓库和命令整合。
   用户本轮明确覆盖其中的单备份和 remove 回退规则；相关 FD 实施时需要更新稳定规范，
   当前实现与历史 FD 记录不代表新需求已经落地。
8. `docs/features/archive/FD-080/FD-080_SEPARATE_COMPILE_BUILD_INSTALL_AND_UNIFY_PACKAGE.md`
   的“后续 FD 候选”记录仓库线索：本地/HTTP/S3、统一布局、索引和 publish。
   该节自身明确不是已批准的后续实现设计；本草案沿用为候选方案，不冒充批准结果。
9. 补充静态检查：`src/internal/plugin/exec.go` 用 cmd.Run 等待插件结束，
   因而经 aiw 启动 pkg 时，aiw 主进程并未先退出。
   `manager.py:109` 的 preflight 尚未查询 Windows 文件占用；
   `manager.py:164` 的 restore 对所有快照文件逐一写回，未跳过内容/mode 均未变的文件。
   因此未变化但被占用的文件也可能阻断后续恢复，不能声称现有恢复已经满足新要求。

## 建议的包模型

以下为待正式设计的方案，不是现有接口承诺。

| 类型 | 内容与边界 |
| --- | --- |
| program | 一个主程序及其资源、文档；AIW 主程序包括 agent-templates，src\skills\*.md 几个通用的文件 |
| plugin | 一个插件的入口、程序、必要资源和文档 |
| skill | 一个 Skill 的 SKILL.md、references 和自身资源 |
| resource | 可共享 Markdown 或运行资源；按明确归属避免跨包重复拥有文件 |
| collection | 只声明精确成员与版本，不携带成员载荷；允许嵌套集合 |

命名建议沿用现有独立包名称，避免仅为统一外观而重命名已有独立包。
集合名采用用户给出的 plugins-core/plugins-all/skills-core/skills-all；
主集合建议命名 `aiw-core`，由 `update --core` 选择；
完整集合建议命名 `aiw-all`，由 `update --all` 选择。

共享资源只允许一个受管所有者。独立 Skill 安装时应能自动解析必需共享资源，
不能要求用户先安装整个 skills 集合。安装到项目/用户作用域时继续物化所引用的
共享 Markdown，使 Skill 在 AIW 仓库以外仍可使用。

### Core 清单：按用户输入保留
src\plugins\plugin-meta.toml 中记录core pulgins 和 extension plugins
src\skills\skills-meta.toml 中记录core pulgins 和 extension plugins

`plugins-core` 的直接成员：
- aiw-git
- aiw-pkg
- aiw-req
- aiw-fd
- aiw-file
- aiw-setup-project
- aiw-patch
- aiw-notify

`skills-core` 的直接成员：

- ask-asu
- fd-review
- fd-test
- fd-workflow
- grill-me
- grill-with-docs
- grilling
- issue-management
- `src/skills/*.md` 的共享资源，包括当前的 README.md、reviewed-skill-contract.md、
  testing-methods.md、work-management.md、workflow-methods.md。

根级 Markdown 如何形成小资源包，应由 FD 确定；不再把所有 Skills 的载荷合为一个包。
`plugins-all` 包含 core 及本版本发布清单中的 extension 插件；
`skills-all` 包含 core 及本版本发布清单中的 extension Skills。
all 不是任意第三方仓库的全部内容，也不在安装时重新扫描源码决定成员。

`aiw-core` 包含主程序包、plugins-core、skills-core。
`aiw-all` 包含主程序包、plugins-all、skills-all，以及本发行版明确列出的其他程序/资源包。
显式 core 成员与传递运行依赖应分别展示，不暗中扩大用户指定的 core 清单。

## 建议命令边界

示例用于表达能力边界，参数拼写在 FD 中固定。
aiw.toml.example 中配置 REPO地址：
[repo]
# local fs
url=/the/local/path
# http
# url=http://the-remote-path
# url=https://the-remote-path
# s3
# url=s3://the-bucket/the-s3-path

HTTP 读取、SSH 发布的配置示例：

```toml
[repo]
url = "https://packages.example.com/aiw"
publish_url = "ssh://repo-host/srv/packages/aiw"
```

repo-host 可以是 `~/.ssh/config` 中的 Host 别名；也支持 IP 或域名，
例如 `ssh://deploy@192.168.1.100/srv/packages/aiw`。
认证使用系统 SSH 配置、密钥及 agent，不把私钥内容放入 aiw.toml。


# local install directory
# for windows
# local_install_dir = "c:\\green\\aiw"
# for linux, 需要展开 ~ 为当前用户的HOME目录
# local_install_dir = ~/apps/aiw

```text
aiw pkg list [--all] # REPO 有哪些可用的package,名字+基本描述, 最新版。 参数--all，表示所有的版本。
aiw pkg install <package-or-zip>  --version <version> [--dest <root>]
aiw pkg remove <package> [--dest <root>]
aiw pkg purge <package> [--dest <root>]
aiw pkg autoremove [--dest <root>] [--dry-run]
aiw pkg mark manual <package> [--dest <root>]

aiw pkg index <local-repository>
aiw pkg search <query>  # search some package in REPO
aiw pkg info <package> # show the package info in REPO
aiw pkg show <package> # show the information of a package that installed. 
aiw pkg list --installed --kind plugin|skill|program|resource|collection -- installed # 查看已经安装的packages

aiw pkg update [--dest <root>] [--dry-run] [--core | --all]
aiw update [--dest <root>] [--dry-run] [--core | --all]
aiw pkg publish <zip> <version> # 按 publish_url（未配置时按 url）选择本地/S3/SSH；HTTP 只定义协议
aiw pkg unpublish <name> <version> # 按相同写入地址删除；HTTP 写入本期不实现

```

### update 语义（用户本轮明确调整）

| 调用 | 目标范围 | 是否安装尚未安装的目标包 |
| --- | --- | --- |
| `aiw pkg update` / `aiw update` | 指定安装根中已安装内容的最新版本，包含已安装的 extension | 不额外选入未安装的集合成员 |
| `aiw pkg update --core` / `aiw update --core` | 最新 core 集合：主程序、core plugins、core skills 及必需资源 | 是，补装缺少的 core 成员 |
| `aiw pkg update --all` / `aiw update --all` | 最新完整发行集合：主程序、core 和 extension，以及发行清单内其他包 | 是，补装缺少的发行成员 |

- `--core` 与 `--all` 互斥；无参数不再默认展开 core 集合。
- “全部”沿用本发行版明确清单的范围，不遍历任意第三方仓库安装所有包。
- 默认 update 不因某个已安装集合的新版本新增成员而悄悄补装这些成员；
  需要安装/补齐集合成员时显式选择 `--core`、`--all` 或安装该集合。
  必需运行依赖如何解析以及集合记录如何保持真实，由 FD 明确，不可伪称未安装成员已安装。
- 范围参数不隐含卸载：`--core` 不删除或顺带更新范围外的 extension；
  `--all` 不删除清单外的已有包。
- `--dry-run` 展示将更新和将补装的目标，不修改安装文件及安装记录。

### 集合移除、孤立包及安装原因（用户选择 A 并补充）

- 用户直接安装的包或集合标为 manual（独立安装）；仅因集合或运行依赖带入的包标为 auto。
  已经是 manual 的包加入集合时不能被降为 auto。
- `update --core` / `update --all` 将显式选择的根集合记为 manual，
  新补装的成员及传递依赖记为 auto；更新已有包不改变其安装原因。
- 移除集合只移除该集合的记录及相应成员关系，不级联卸载成员。
  清理集合本身的记录也不等于 purge 成员。
- 孤立包是：已安装、标为 auto，且不再被任何保留的 manual 包/集合通过依赖关系需要的包。
  同时考虑嵌套集合与运行依赖；被其他集合或独立包需要的成员不是孤立包。
- 移除集合后显示孤立包清单，并提示 `aiw pkg autoremove` 和
  `aiw pkg mark manual <package>`；提示不会自动执行清理。
- `autoremove --dry-run` 仅显示拟清理清单；`autoremove` 显式卸载当前安装根里的孤立包，
  保留用户配置，不默认执行 purge，不操作无归属文件或项目/用户作用域的 Skill 副本。
- `mark manual` 只修改指定已安装包的安装原因，不下载、重装或更换版本。
  用户也可直接安装某个已自动安装的包，将它保留为独立安装；即使同版本同内容，
  也应更新安装原因，包文件无需重新部署。
- 成员提升为 manual 后，其自身及其必需依赖不属于孤立清理对象。

### 卸载与事务恢复（用户明确调整）

- `remove` 真正卸载包的受管非配置文件，保留配置及供后续 purge 使用的必要归属记录。
  不恢复上一版；集合的成员仍按前述规则保留。
- `purge` 卸载包并清理其登记配置及相关状态；不删除无归属的用户文件，不级联 purge 集合成员。
- `autoremove` 对孤立包采用 remove 的卸载语义，保留配置。
- 安装或更新成功后不保存 previous、上一版 ZIP 或供用户回滚的文件快照，
  不提供 rollback 命令，也不提供通过 remove 隐式回退的能力。
- 对正在进行的操作保留临时事务快照及必要状态，以便失败时恢复本次已变更的受管文件。
  成功后清理临时数据；失败诊断及未完成恢复必需材料可以保留，但不是历史版本备份，
  不提供用户任选版本回退能力。仍不保证脚本外部副作用可恢复。
- 普通升级的配置保护、文件归属冲突检查与失败恢复继续有效；
  取消历史备份不能作为破坏这些保护的理由。

建议将原 skills 领域操作放到 `aiw pkg skill` 下：

```text
aiw pkg skill-list 
aiw pkg skill-discover --scope project|user
aiw pkg skill-adopt --scope project|user
aiw pkg skill-install <skill-or-local-source> --scope project|user --dry-run
aiw pkg skill-sync  <skill-or-local-source> --scope project|user --dry-run
```

`pkg install` 管理 AIW 安装根里的 canonical 包；`pkg skill install` 将 Skill 安装到
项目或用户的 .agents/skills。两种动作、路径和受管记录不得混淆。
项目/用户作用域的 .aiw-skills.json 受管记录及本地修改保护沿用现有能力；
不据此增加旧版 AIW 安装根的包状态转换。
旧 plugin list 的入口信息应由 pkg 查询提供，包括路径、命令及只读/写文件等元信息，
不能只显示仓库索引而遗漏可用入口。
直接删除旧入口时同时更新帮助、Skills、共享工作流文档、安装说明和代码内调用。

## 仓库与构建建议

### 统一 Repo 接口与 adapter（用户明确要求）

- 包命令通过统一 Repo 接口进行查询、下载、发布和删除；由 local filesystem、HTTP、S3、SSH
  adapter 负责各自的索引获取及存储访问。具体方法签名由 FD 确定。
- 查询/下载根据 `[repo].url` 自动选择 adapter；发布/删除根据可选的 `[repo].publish_url`
  选择 adapter，未配置时使用 url。不增加必须配置的 type 字段或命令参数。

| 配置路径示例 | adapter |
| --- | --- |
| `/srv/aiw-repo`、`./repo`、`~/repo` | local filesystem |
| `C:/aiw-repo`、Windows UNC 路径 | local filesystem |
| `http://host/repo`、`https://host/repo` | HTTP |
| `s3://bucket/prefix` | S3 |
| `ssh://repo-host/srv/packages/aiw`、`ssh://user@host/path` | SSH，用于本期发布/删除 |

- 本地路径按当前平台支持范围处理，展开本地路径中的 `~`。
  Windows 盘符不能被误判为 URI scheme；未知 URI scheme 或非法路径明确报错，
  不通过尝试联网来猜测类型，不静默回退到其他 adapter。
- 不同 adapter 共享包名/版本、索引、摘要及重复发布的业务规则，避免各自定义不同包语义。
  HTTP adapter 本期支持查询和下载，publish/unpublish 返回明确的不支持结果；
  统一接口不意味着本期所有 adapter 都实现写操作。
- adapter 不另建安装状态、依赖解析或安装事务；获取包后继续使用共同安装引擎。
  普通 build.py build/install 仍保持离线，不因配置的是远程仓库而自动联网。

### SSH 发布（用户确认纳入本期）

- SSH adapter 复用已有 ssh/scp 工具，不实现新的 SSH 协议栈，不自动安装缺失工具。
  地址中的 host 支持 SSH Host 别名、IP 或域名，连接参数交由系统 SSH 配置解析。
- 通过 SSH 执行远端仓库脚本，复用 local filesystem adapter 的发布/删除及索引逻辑。
  远端需提供 Python 和该脚本；无需常驻仓库管理服务。
- 发布顺序为临时上传、校验、同名同版本检查、发布包、更新索引；
  上传未完成或校验失败的内容不能进入可查询索引。同一仓库的写入需串行化，
  具体协调方式由 FD 明确，不能因同时发布而覆盖其他包的索引项。
- SSH unpublish 使用相同远端写入逻辑，同时处理包及索引；不经 HTTP POST。
- 读取 URL 与发布路径由用户配置为同一逻辑仓库，不从 HTTP URL 猜测远端文件系统路径。
  配置了 SSH publish_url 时，HTTP 仓库可通过 SSH 完成写入；未配置且 url 为 HTTP 时，
  发布/删除仍明确报 HTTP 写入不支持。
- 本段是 SSH 发布能力，不改变 HTTP POST 写入仅交付协议规范的边界。

### 索引、发布与构建边界

- 候选布局：`<repo>/<package>/<version>/<package>-<version>.zip`，根 index.json。
  原 FD 线索写作 package/version，其中 version 按具体版本解释，正式设计应给出实值示例。
- 索引包括包名、版本、类型、路径、大小、SHA-256、成员/依赖、平台能力。
  集合成员版本应固定；latest 由明确的版本策略和索引决定，不按目录顺序猜测。
- publish 的唯一发布标识为包名与版本。目标已存在时，内容相同则成功返回并提示已存在，
  不覆盖包或改写其索引；内容不同则拒绝发布并要求更换版本号，保留原包及索引。
  不提供 --force 等绕过参数，本规则适用于程序、插件、Skill、资源和集合包。
  “内容相同”按包的规范化内容指纹判断，包含元信息、载荷及脚本，
  不将 ZIP 时间戳等容器差异视为新内容；下载完整性仍按发布 ZIP 的 SHA-256 校验。
  这是仓库发布规则，不将本地文件安装隐式变为仓库发布。
- 支持本地目录、HTTP/HTTPS 查询与下载及 S3 适配。
  本地目录/S3/SSH 实现显式 publish/unpublish；HTTP/HTTPS adapter 本期只实现读取。
  S3 可复用已安装 awscli。不得自动安装依赖或把凭据写进包/索引。
- HTTP publish/unpublish 的交付物仅为协议规范：发布和删除均向管理接口发送 POST，
  不使用 GET 修改仓库。接口路径、操作区分、上传格式、包名/版本/摘要字段、
  响应与错误码、认证传递及重复发布规则由后续 FD 写入规范；
  不因此增加 HTTP 写入客户端、HTTP 服务端、部署或认证系统实现。
  写入地址选择到 HTTP 时，本期明确报告不支持，不发送 HTTP 写入请求；
  显式配置 SSH publish_url 时使用 SSH adapter。不静默换仓库、不把仅定义协议报告为已实现。
- 明确仓库选择与同名冲突规则；指定仓库失败不能静默回退到其他来源。
- 获取与安装前校验大小、摘要及包内身份。哈希校验不等于发布者认证。
- 下载只存在于本次操作临时目录，成功/失败均清理不再需要的文件；恢复材料保留。
- 集合先解析全部成员、检测缺失/循环/版本冲突，再开始文件变更。
  沿用现有逐包事务与失败停止规则；失败包恢复其已变更的受管文件，
  报告已完成包及失败包，不增加整个集合的原子回滚或脚本外部副作用恢复机制。
- build.py 保留 compile/build/install 阶段隔离，默认离线；单包及集合使用同一份包定义。
  仓库导出/生成索引作为显式动作，不在普通 build/install 中隐式上传或下载。
- 构建集合时生成成员包与集合清单；输出应可直接作为本地仓库读取，发布复用该索引契约。
- agent-templates 从 docs 包移入主程序包；usage 文档按程序/插件归属，
  仓库开发文档不因“docs”目录存在而整体塞进用户安装根。

## 安装边界与风险

1. 已安装 aiw-plugin-tools/aiw-skills-support/旧 docs 的文件已有包归属。
   用户已选择全新安装，不实现这些旧包的自动迁移或文件归属转移。
   提供备份所需配置/自定义内容、退出相关程序、选择新的干净目录重新安装的说明；
   不要求安装器自动删除旧目录或旧状态，也不允许新小包抢占旧包文件。
2. Windows 文件占用策略已由用户确认：不引入临时/后台更新器；遇到占用提示退出重试，
   不增加专门的占用预检或运行中覆盖机制。失败恢复沿用现有安装事务契约。
   pkg 由 aiw 启动时 aiw.exe 自身仍在运行；若它阻止更新，错误提示必须给出不经该 EXE
   启动的安装方式（例如显式 Python 入口，具体命令由 FD 固定），不能只要求重复同一条命令。
3. pkg 自身是 core 插件；首次安装与 pkg 自身更新需要可用的引导路径，
   不得在更新过程中提前破坏正在使用的 Python 模块或恢复资源。
4. core 直接成员清单已包含 aiw-git；用户补充确认允许显式运行依赖自动带入 aiw-git。
   直接成员和传递运行依赖应分别展示。
5. 旧 CLI 删除和仅支持全新安装均已由用户确认。
   旧安装目录及状态由用户按说明处理，本次不自动删除、转换或接管。
6. 静态检查发现的恢复受占用阻断风险保留为现有风险，本次不扩展为专门的恢复算法改造。
   若现有恢复失败，应如实提示恢复未完成、材料位置及用户处理方式，不宣称已恢复成功。

### Windows 处理范围与查阅依据

- 用户明确不再展开占用检测或运行中覆盖设计，只要求错误时提示如何处理。
- 提示应说明失败文件、退出相关程序后重试的方法，以及适用的独立 Python 安装入口。
  若恢复未完成，同时指出恢复材料位置和当前状态；不让用户误以为可以直接继续安装。
- 以下链接仅保留前轮查阅依据，不构成 Restart Manager 或其他专门机制的实现要求。
- 官方依据：[CreateFileW 共享规则](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew)、
  [RmGetList 占用查询](https://learn.microsoft.com/en-us/windows/win32/api/restartmanager/nf-restartmanager-rmgetlist)、
  [ReplaceFileW 替换语义](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-replacefilew)。

## 验收示例

- 添加本地仓库后，可查找并安装一个独立插件、一个独立 Skill 和一个固定版本集合。
- HTTP/HTTPS 仓库可查询索引、搜索/查看包并下载用于安装；下载继续校验大小、摘要及包内身份。
- 同一组仓库查询/安装命令仅更换配置路径即可使用 local filesystem、HTTP 或 S3，
  不要求增加类型参数；三类后端均通过统一 Repo 接口接入。
- Windows 盘符、当前平台支持的 UNC/本地路径不被误判为网络协议；
  未知 URI scheme 明确报错，不触发类型探测请求或其他 adapter 的回退。
- HTTP publish/unpublish 具有明确的 POST 协议规范；写入地址为 HTTP 时本期返回不支持，
  不发送 HTTP 写入请求。不要求实现 HTTP 服务端或部署。
  实际发布/删除验收在本地目录、S3 与 SSH 范围内。
- url 为 HTTP、publish_url 为 SSH 时，查询/下载走 HTTP，发布/删除走 SSH，
  写入完成后的包及索引可从对应 HTTP 地址读取；Host 别名、IP 和域名均可配置。
- SSH 发布遵守同名同版本不可覆盖规则；上传或校验失败不发布索引，
  并发写入不丢失其他包的索引项。缺失 SSH 工具或远端脚本时明确提示，不自动安装。
- 同名同版本、同内容再次 publish 时提示已存在并成功返回，不修改仓库中的包及其索引。
- 同名同版本、不同内容再次 publish 时失败并提示使用新版本号，原包及索引保持不变；
  不存在强制覆盖参数。使用新版本号的包可正常发布。
- 安装 plugins-core 只展开指定成员及明确展示的必需依赖；不顺带安装全部 extension。
- 安装 skills-core 后，列出的八个 Skills 和根级共享 Markdown 均可用。
- 独立安装依赖共享资源的 Skill，不要求先安装 skills-all。
- 集合嵌套会去重；循环依赖、同名版本冲突、缺失成员在文件变更前报告。
- 不存在或损坏的本地 ZIP 不触发网络请求；仓库内容校验失败不影响已安装版本。
- 已安装 core 与 aiw-say、未安装 aiw-github 时：无参数 update 更新已有 core 与 say，
  不补装 github；`--core` 更新并补齐 core，不更新 say；`--all` 更新并补齐发行清单，
  若 github 属于该清单则将其安装。
- 只有部分 core 包已安装时：无参数 update 不补装缺失 core；`--core` 补齐。
- 已安装集合的新版本增加成员时：默认 update 不自动补装；显式范围参数才补齐对应集合。
- aiw update 与 aiw pkg update 在相同参数下目标一致；`--core --all` 同时出现时报参数错误。
- update 不隐含卸载清单外的已有包；dry-run 不修改安装文件及安装记录。
- 包 X 仅由集合 A 自动安装时：移除 A 后 X 保留，并提示 X 可通过 autoremove 清理。
- 包 X 同时被集合 A、B 需要时：移除 A 后 X 仍被 B 需要，不进入孤立清单。
- 将 X 标记为 manual 后：即使移除全部引用它的集合，autoremove 也不删除 X 及其必需依赖。
- autoremove --dry-run 不写安装状态；实际清理只卸载孤立包、保留配置，不恢复其上一版。
- 单独直接安装已存在的 auto 包时：该包变为 manual，同版本同内容不重复部署文件。
- 项目/用户 Skill 安装仍保护未接管目录和本地修改，并补齐必需共享文档。
- 删除旧命令后，仓库内活跃文档和调用使用 pkg；历史归档保持原状。
- build.py 的离线仓库导出可被 pkg 消费；普通 build/install 不隐式联网。
- 在干净安装根安装独立小包及集合后，文件归属唯一，正常更新及失败恢复满足本 Issue 的新规则。
- remove 后程序文件被卸载、配置保留，不恢复上一版；随后 purge 可清理所登记的配置。
- 成功安装或更新后无上一版本备份，不存在 rollback 或 remove 隐式回退能力。
- 安装中途失败时仍按本次事务恢复已变更的受管文件；成功后不保留供用户回滚的临时快照。
- 旧用户有明确重装说明；旧包文件归属冲突时拒绝覆盖，不执行自动归属转移，
  不自动删除旧目录、配置或安装状态。
- agent-templates 的安装归属为主程序，插件 usage 文档归对应插件。
- 文件占用导致替换失败时，明确提示退出相关程序及可用的重试方式；
  已发生的变更按现有安装事务规则恢复，不增加占用预检或强制覆盖机制。
- 无法完全恢复时，非零退出并报告未完成状态、材料位置及用户处理方式。
- 全程不自动结束程序，不启动临时/后台更新器，不安排重启后替换。

## 决策依赖与待确认项

| 决策 | 状态 | 影响 |
| --- | --- | --- |
| 删除旧 plugin/skills 命令 | 已由用户明确确认 | 所有活跃调用同步迁移，不设兼容入口 |
| Windows 主程序更新方式 | 已由用户明确收简 | 提示退出及独立入口重试；沿用现有事务，无专门占用检测或更新器 |
| update 默认范围与显式集合 | 已由用户明确调整 | 默认只更新已安装内容；--core/--all 更新并补装指定集合 |
| 同名同版本重复发布 | 已由用户选择 A | 同内容提示已存在，不同内容拒绝并要求换版本；不提供强制覆盖 |
| HTTP 仓库本期范围 | 已由用户选择 A 并补充 | 只实现查询和下载；publish/unpublish 仅定义 POST 协议，不实现写入或服务端 |
| Repo 接口及类型选择 | 已由用户明确要求 | local filesystem/HTTP/S3/SSH 使用统一接口与 adapter；根据操作地址自动识别类型 |
| HTTP 读取与 SSH 发布 | 已由用户 confirm | url 读取、publish_url 写入，缺省写入地址为 url；SSH 支持 Host 别名/IP/域名 |
| 集合移除与孤立包 | 已由用户选择 A 并补充 | 成员保留；显式 autoremove 清理孤立 auto 包，mark manual 转为独立安装 |
| 卸载与历史备份 | 已由用户明确调整 | remove 卸载保留配置，purge 清配置；不保留上一版、不提供回滚，保留本次失败恢复 |
| core 的 aiw-git 运行依赖 | 用户于 2026-10-18 明确允许 | 保留直接成员清单，显式展示运行依赖 |
| 旧版安装迁移 | 已由用户选择 B | 仅支持全新安装并提供重装说明；不开发自动迁移或归属转移 |
| 仓库协议与发布配置 | 待 FD 明确细节 | adapter 与读写地址规则已确定；接口签名、相对路径基准、远端脚本及写入协调需设计 |

已确认：允许 core 集合通过明确声明的运行依赖自动带入 aiw-git。
%% OPEN_DESIGN: 集合格式、版本选择、manual/auto 及依赖记录、卸载与本次事务恢复、仓库配置作用域与 HTTP POST/S3/SSH 适配协议，
%% 独立安装入口的具体提示命令；Windows 专门占用预检、运行中覆盖及整组回滚不在本次范围。

## 规模与交付边界建议

这是仓库与包生命周期的较大变更，跨 CLI、包格式、持久化归属、打包及自更新。
虽然旧版安装迁移已排除，仍不宜把七项要求当作一个小改动。

建议按可验收结果拆分后续 Issue/FD：

1. 仓库获取与查询、显式本地/S3/SSH 发布：复用现有本地包，建立统一索引；HTTP 写入仅定义 POST 协议。
2. 小包、集合和 build.py 仓库产物：包含依赖解析及全新安装说明，支持 core/all；不含旧归属迁移。
3. pkg 命令归并和 aiw update：迁移 Skill 作用域能力，删除旧入口，实现已安装包更新及显式 core/all 更新与补装。

规模评估：大型、中高难度。用户明确要求晋级一个 FD；保留一个统一包管理交付结果，
由 FD-083 拆为小型、低或中等难度 Work Items，不创建或隐式批准子 Issue。第 3 项依赖前两项；
不能在新入口的 Skill 管理能力及主程序更新尚未可用时把整体需求标记完成。
稳定规范只在相关能力决策批准并进入 FD 后更新；不创建额外 OpenSpec change。

## 本轮验证与限制

仅静态阅读源码、稳定规范、FD-080 候选记录和工作区状态。
本轮只新增此草案；未修改产品源码、稳定规范、历史报告或 Git 状态。
初轮未运行测试、编译、构建、安装、网络请求、发布或自动审查。
本次补充只执行 Get-Content/rg 静态阅读与草案编辑，并查阅上述微软官方网页；
未运行测试、编译、构建、安装、占用探测、覆盖实验、发布或 Git 写入。
本文件不是批准记录，也不授权执行草案中的示例命令。

## 2026-10-18 决策补充

用户明确授权批准、晋级及 fd-workflow auto，并确认 aiw-git 运行依赖。
现有 core 直接成员表优先于后文过时备注，本补充消除该矛盾。
采用 synthesis 汇总现有证据；grill-with-docs 本会话未提供，未执行额外访谈。
业务批准不等于工程设计就绪。
