# FD-084 独立设计审阅

<!-- aiw-data: FD-084-design.json -->

结论：**ready**。本次有界审阅提出的三项问题已合入 FD Revision 2；对修订后的 FD 作了聚焦静态核对，没有开启额外审阅轮次。这是设计证据，不是实现通过或运行验证。

来源为用户直接请求及主代理的有界设计审阅委派，无派发的设计审阅事件，因此 `source_event` 为 null。审阅对象是当前 FD-084、build.py、scripts/package_definitions.py 和 openspec/specs/aiw-packages/spec.md；补充检查包引擎的目标冲突规则。

## 发现与已完成的设计修订

1. **平台配置违反兼容性契约：已解决。** 初稿允许 AIW_BUILD_SYSTEMS 选择 windows/linux；稳定规范要求每个 Go 组件一次生成 Windows/Linux amd64 两套产物，Go 插件包必须同时携带两套产物。仅保留双平台默认值不能保证该契约。Revision 2 删除了 AIW_BUILD_SYSTEMS，固定双平台与 amd64，并通过 AIW_GO、AIW_NODE、AIW_INSTALL_DIR、AIW_VERSION 和 AIW_PACKAGE_REPO 配置本机差异。

2. **docs 安装目标需隔离：已解决。** 既有包定义把所属 usage 文档安装到 docs/usage；若 aiw-docs 再使用相同目标，安装 all 或共存安装时会产生文件所有权冲突。operations.py 对重复目标直接拒绝，manager.py 也拒绝其他包已有归属，即使内容相同也不能共享。Revision 2 保留原包目标，将 usage 映射到 docs/reference/usage、根 README.md 映射到 docs/reference/README.md，两个来源也不再竞争同名 README 目标，并明确排除开发报告、本机配置与构建输出。

3. **Work Items 尚缺 readiness 信息：已解决。** 初稿三项只有任务标题。Revision 2 为每项补充了规模、难度、依赖、完成条件和对应静态或 compile-only 证据；顺序覆盖目录与环境、发布与 docs、规范与交付证据，无须另建工程计划。

## 已有设计可保留的部分

- 新发布命令先构建依赖闭包，再发布本次返回的 ZIP 路径，避免 latest manifest 引用其他版本，符合用户意图。
- 复用 Repo 的同版本不可覆盖规则，并明确多包发布可能部分成功，未引入事务或覆盖语义。
- 发布前检查必需仓库参数并拒绝远程地址；须复用或对齐既有盘符识别规则，不能把 d:/playground/aiw-packages 当作 URL。相对本地路径以 checkout 根解析。
- 统一 target catalog、保留旧组件与 bin/plugins/skills/all 分组、为单包提供规范包名别名是合理方案。需确保 aiw-main 映射 aiw、aiw-plugin-runtime 映射 plugin-runtime；集合名不要机械剥除 aiw-，递归闭包应去重并先构建成员与依赖。
- .env 无依赖、无执行或插值、进程环境优先，配合忽略的本机 .env 和安全示例，范围合理。独立 docs 脚本也必须经过同一加载入口。

## 证据与限制

实际执行的命令仅为 PowerShell Get-Content、Get-ChildItem、rg 及有限输出选择，用于阅读上述文件、规划技能与包引擎冲突点；读取不存在的 FD_REVIEW_DATA_TEMPLATE.json 失败，未重试。报告通过 apply_patch 写入。未运行测试、编译、最终构建、发布、安装、网络或 Git 写入，未发工作流事件。

本报告记录初稿发现及 Revision 2 的处置，不宣称尚未审阅的实现已通过。运行时 .env 边界处理、双平台编译、发布路径和多包部分失败仍待实现后的静态审阅，受现有零测试/构建预算限制。
