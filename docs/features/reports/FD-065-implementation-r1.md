# FD-065 实现报告 r1

<!-- aiw-data: FD-065-implementation-r1.json -->

- Worker：fd065-worker-20261017-6b72；来源事件：FD-065-000003-design-ready。
- 已按用户要求移除独立 implement 入口，不保留兼容薄入口。

## 变更

- 实现方法迁入 fd-workflow/references/implementation.md，入口及 Worker 角色按需加载。阶段路由仍由 fd-workflow、角色与共享契约管理，参考文件不反向发起技能调用。
- 合并入口的 Worker 步骤改为加载参考文档，避免重复维护；保留主工作区无 Worker handoff 的 managed review 路径、旧 Task/Core 兼容、准备和逐项变更、真实证据及阶段判断。
- 明确自然语言调用例子：正常完整实现、只实现、仅某个工作项、实现并审查但不合并；不声称新增 CLI 参数。完整实现仍自动继续独立审查及本地交付归档，显式限制优先。
- 更新 ask-asu、Issue、setup-project、TDD、技能目录/安装示例、稳定规格和八组项目副本；删除源入口两个文件及项目入口四个文件，只从项目安装索引删除 implement 一项。
- 未修改 CLI、共享生命周期契约、其它技能入口、历史 FD/报告、用户级全局安装或其它安装索引条目。引用资源由现有整目录安装机制携带，无需新增安装代码。

## 静态证据和实际命令

- 一次静态检查批次：git diff a88b62d 检查路由/边界变更；只读 Python 比较八组 source/project 文件、六个退役文件、当前引用和安装索引，均通过；CLI source 无改动。
- 当前维护技能和稳定规格中没有旧技能调用、旧路径或薄入口；历史材料保留旧名称属于历史事实。
- 执行了 FD new/claim/emit、worktree add、聚焦 Git 提交等流程命令。本报告后交接独立 Reviewer；尚未执行交付与归档。
- 纯 Markdown/YAML/JSON 元数据，无适用编译目标；未运行编译、测试、构建、格式化、lint、验证脚本、安装同步或网络。
- 风险：旧调用名称不再兼容是用户明确要求的变更；用户级安装仍可能存在旧技能，未自动删除。未运行宿主行为评估或安装验证，静态检查不代表所有宿主已刷新技能目录。
