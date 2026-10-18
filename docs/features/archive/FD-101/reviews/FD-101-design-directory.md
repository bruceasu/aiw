# FD-101 目录式自包含插件设计审查

<!-- aiw-data: FD-101-design-directory.json -->

结论：**ready**。本轮针对修订 3 及用户明确追加的三个插件自包含、目录式迁移范围；未发现阻止实施的设计问题。修订 2 的原审查保留历史，本报告替代其当前范围判断。这里不是实现验收。

来源事件：`FD-101-000002-design-requested`（Planner 提供）；独立会话：`fd101-design-independent`。

## 关键判断

- Go manifest 顶层仅允许 `schema` 和 `plugins`；插件条目允许 `name`、`description`、`help`、`entrypoint`、`startup`。设计要求 schema=1、三个原命令名和显式 Python 入口，与现有校验和发现路径相容，不需要修改 loader。不要将入口里的 META 扩展字段写进 manifest。
- `build.py plugin_targets` 和包内 `entries.py` 均已有 manifest 目录识别。移动而非保留顶层入口，可避免构建目标重复和包条目列表双入口；Go 列表本身也按命令名去重。各目录 manifest 仅声明该插件自身命令。
- 补充核对当前修订 3 的 entries 元数据兼容方案：现有 `_load_meta` 已用 AST 与 `literal_eval` 静态获取 META，可复用在选定的 Python 入口，随后由 manifest 覆盖同名字段并计算最终 path。这样保留 patch 的 `requiresConfirmation` 和三个插件的 commands/readOnly/mutatesFiles/outputFormat，避免迁移后静默丢失提示信息。只读取存在的 Python 文件，非 Python 或无法取得 META 时保持既有目录行为，不导入或执行入口；manifest 仍不接受这些扩展字段。该必要兼容调整已进入工作项 1.1 和验收 2，结论保持 ready。
- 开发 shim 与发行载荷分工明确。codec shim 按绝对路径加载顶层唯一实现并导出原使用符号，避免同名 import 回到自身。Teams shim 用 `runpy` 的 `__main__` 语义转交，必须保持原 argv，使 `--managed-json` 分支继续命中。现有 codec 无依赖自身模块名的复杂类型注册，当前设计无需额外注册机制。
- `tree_entries` 已支持按相对文件名排除。先排除 shim，再从顶层 canonical 文件为私有目标生成一次载荷，可确保发行包不携带源码路径转交逻辑。三个目标目录不同，不与旧 runtime 根目录载荷重叠；无需维护多份共享实现。
- notify 以自身 `__file__` 定位相邻脚本，适配源码 shim 和安装包的真实私有实现。它保持 `-I`、`--managed-json`、输入输出协议和原有环境约束；console 路径仍在 Teams 子进程路径之前返回。此项是路径调整，不扩展通信或重试策略。
- Manager 所有权检查忽略当前包自身，禁止其他包目标及父子路径冲突；升级删除当前包旧记录中不在新目标集合的非配置文件。旧 `plugins/aiw-file.py` 与新 `plugins/aiw-file/...` 的路径并无文件/目录同名冲突，另外两插件同理，因此现有清理足以移除受管理的旧入口。保留根级 runtime 载荷不影响旧版本插件混装。
- 三个插件自包含后移除所有插件的默认 runtime 依赖合理，pkg 不再触发 runtime 替换。显式 legacy runtime 目标保留，旧包和安装记录不迁移，全局精确依赖及反向依赖校验不放松。用户已授权真实消费者契约调整，不再把修订 2 的保留 runtime 要求当成当前 Gate。

## 工作项与边界

1.1 负责目录迁移与源码可调用性；1.2 负责唯一私有载荷规则及依赖解耦；1.3 负责文档、稳定规范和真实证据。依赖次序及验收覆盖合理，新增范围有明确用户来源。无需新增兼容版本范围、解析器、Go schema 或其他插件迁移。

实施审查应重点核对 shim 真正被排除、每个私有目标只写入一次、manifest 无未知字段，以及顶层三个旧入口已删除。源码直接调用路径变化须按 FD 记录在文档中。尚未实际安装，不宣称升级、回滚或 Teams 通知运行成功。

## 实际证据与限制

仅通过 PowerShell `Get-Content`、`rg`、`Select-Object` 静态读取修订 3（含 entries META 合并补充）、Go manifest/discovery、目录 entries 及 `_load_meta`、构建目录枚举/tree_entries、Manager ownership/obsolete 清理和共享实现。未执行测试、编译、规划模拟、构建、网络、安装、状态变更或 Git 写入。后续实现仍须 compile-only 和独立实现审查。
