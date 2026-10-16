# FD-066 实现报告 r1

<!-- aiw-data: FD-066-implementation-r1.json -->

- Worker：fd066-worker-20261017-b391；来源事件：FD-066-000003-design-ready。
- 用户批准整体统一 Issue、FD、规格、实现审查、发布检查，并删除不再使用的旧 Task 内容。

## 实现

- 新增 canonical workflow-methods.md，按段落提供拆分原则、规格编写与真实实现审查方法。Issue、fd-workflow、to-spec、fd-review、code-review 均在入口声明，并仅按阶段读取；不再通过相互调用技能复制方法或额外派 Reviewer。
- Planner 加载 planning.md，负责设计、工程拆分、规格协作、就绪和变更一致性；重要设计/明确要求时才进行独立设计反馈。设计报告不产生实现 verification 事件，不豁免高风险决策和授权。此 FD 不涉及重要系统设计或公共接口，未额外启动设计审查。
- fd-review 包含真实代码/调用路径、Correctness、Standards、Spec 和证据；独立 code-review 保留分支/PR用途与显式双轴审查边界，共享方法不授权额外 Agent、轮次或执行检查。文档/配置改动按真实契约审查。
- to-spec 保留独立规格请求，Planner 使用同一方法；区分拟议规则与已实现行为，FD 保持唯一工程计划。只有明确要求才创建 OpenSpec change，不额外维护 tasks.md。
- finance-release-gate 只在发布计划及运维证据场景使用；不作为普通本地 merge/archive Gate，GO 不授权部署。
- 删除 to-tickets 的源/项目共六个文件及项目索引项；清理当前技能、共享契约和 root AGENTS 的旧 Task 指令，保留一般人工待办/宿主工具术语。当前关联技能引用统一到 FD，不动历史 FD、报告、旧 CLI 实现或用户级安装。
- 安装器仅在 SHARED_SKILL_REFERENCES 新增 skills/workflow-methods.md 一项，复用原有入口扫描和共享资源复制；没有参数、依赖、schema 或其它 CLI 行为变更。

## 实际命令和证据

- 编译命令：python -c "from pathlib import Path; p=Path('src/plugins/aiw-skills/aiw-skills.py'); compile(p.read_bytes(), str(p), 'exec'); print('Installer compile-only: PASS (no execution or artifact)')"，退出 0。仅编译，没有执行安装器或生成产物。
- 一次静态批次：git diff cf6163f 检查安装器一行差异；只读 Python 检查 25 组变更技能源/项目文件、29 份 work-management、22 份 reviewed-skill-contract、5 份 workflow-methods，全部一致。
- 静态核对五个入口声明共享方法且项目资源存在；索引移除 to-tickets，两个 Skill 入口均不存在；当前技能扫描只剩“Attempt reviewed local delivery”普通英文动词，无旧 Task 操作指令。
- 执行 FD new/claim/emit、worktree add 和聚焦 Git 提交等流程命令。审查、交付与归档由后续角色完成。
- 未运行测试、最终构建、格式化、lint、验证脚本、安装同步、外部发布、网络或宿主行为评估。
- 风险：安装物化和阶段行为只静态检查；已有用户级安装可能仍有旧入口，未自行修改。现有 CLI/历史规格仍可能含旧模型，删除 CLI 不在本 FD 范围内。
