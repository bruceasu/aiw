# FD-089 设计审阅

结论：changes-needed。审阅 FD Revision 1 和当前源码；这是设计意见，不是实现验收。

<!-- aiw-data: FD-089-design.json -->

## 需要修改

1. **安装根必须与 manifest 闭包分离。** `build.py` 当前对非集合 manifest 的每个 ZIP 单独调用 `Operation.install(package)`；`operations.py` 会把每次传入的包作为 root 并标为 manual。目前单组件 manifest 仅含自身；新增完整闭包后直接沿用该循环会把其依赖提升为 manual，改变 autoremove 语义。FD 虽要求保留 root/manual，但需写明实现选择：单组件仅其自身为根（`aiw` 对应 `aiw-main`），集合仅集合自身为根，分组保留当前显式根语义；闭包只提供解析目录。`expand(target)` 已包含依赖，不能直接拿来替代单组件根选择。
2. **拆分 Work Item 1.1。** 当前标为 medium，并同时覆盖来源适配、闭包构造、manifest 和安装接线。规划约定要求每项 small；建议拆为来源目录及校验、构建 manifest/安装接线两项，明确依赖关系，各自具备静态完成证据。
3. **所有载荷目标必须对所有原始 ZIP 交叉检查。** `Operation.preflight()` 当前仅比较当前包的目标和它自己的 ZIP。来源改成原始 dist ZIP 后，用户将 dest 设为 dist 或其祖先时，一个包可能覆盖另一个依赖的输入 ZIP；提前取得字节只保护当前执行，不能保护之后的构建产物。适配器在 Manager 构造前，按 Manager 相同的 dest 物理根规则，检查全部待安装包的当前平台载荷目标与闭包全部源 ZIP，拒绝重合或会占用其祖先路径的文件目标。源路径检查也应拒绝链接绕行。该预检可放在 build 本地接线，不必扩张共享安装引擎接口。

## 可行性与边界

- `Operation` 仅依赖来源的 `index()` 和 `fetch(entry, folder)`，适配器返回原 ZIP 路径即可；可保持共享安装引擎及显式 Repo 发布行为。
- `entry_for()` 同时读取真实包 metadata、大小、SHA-256 和 fingerprint；`fetch` 应再次检查受限源路径及完整 entry 相等。路径映射应与 entry 的规范仓库路径分开保存，不能拿 `entry.path` 直接当 dist 相对路径。
- `safe_path()` 会检查根的祖先以及叶子的链接/reparse point。清单输入和五类 exact fallback 均应使用此路径约束；缺失或多个候选必须在调用 Manager 安装前失败，同名同版本的冲突内容不可任意取一个。
- 递归闭包应同时处理 dependencies 和 collection members，以确切 name/version 为身份；不得利用已有仓库或 latest 补齐。新 component manifest 包含闭包，旧单包 manifest 可用确切标准路径补齐，兼容方案成立。
- `Manager.__init__` 当前不写文件，首次 `locked()` 才创建状态目录与锁。先完成全部本地目录、metadata 与闭包检查，再创建 Manager，能满足设计要求；图版本/循环检查仍由 Operation 执行。Operation 的平台及目标 preflight 在锁内，不能把“所有错误均不写任何状态文件”写成承诺。
- Operation 使用 `read_package()` 提前取得 payload/script 字节，Manager.install 接收 prepared，事务备份与解包仍有必要。根包经 `install(package)` 直接读入而不走 fetch，接线应先通过适配器同等验证根源路径及 entry，避免根校验弱于依赖。

## 实际检查与限制

执行四批只读 PowerShell 检查：Get-Content/Select-Object 阅读 FD、build.py、package_definitions.py、repository.py、operations.py、manager.py、archive.py、规划与资源约束；rg 定位入口与相关符号。第四批用于确认 roots 与 expand 的具体关系。历史 FD-088-design.json 路径不存在，未重试，未依赖该文件作判断。

未运行测试、构建、安装、网络、编译或 Git 写入；仅新增本报告和同名 JSON。未创建工作流事件：此为主代理直接委派的设计审阅，无独立来源事件，source_event 为 null。

Planner 已确认单组件与集合仅自身为根、分组保留现有全部 manifest roots 语义，拟拆为 adapter、wiring、docs 三个 small 项，并补充全部源 ZIP 交叉预检；本报告针对 Revision 1，待这些决定实际纳入 FD 后可继续设计就绪判定。实现阶段仍需独立静态审阅。实际运行安装与 Windows 路径行为未验证。
