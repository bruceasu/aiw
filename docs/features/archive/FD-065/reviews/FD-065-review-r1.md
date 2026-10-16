# FD-065 独立审查 r1

<!-- aiw-data: FD-065-review-r1.json -->

- 结论：changes-requested。
- Reviewer：fd065-reviewer-20261017-94ac；来源事件：FD-065-000004-implementation-ready，已独立 claim。
- 审查范围：a88b62d..d9650bea752a1df9bdff596fad1ae12d8d997c58；FD Revision 4，claim 前摘要与事件一致。

## 发现

1. **R1，中：实现方法迁入后的共享依赖不能随安装携带。** `src/skills/fd-workflow/references/implementation.md:3` 要求读取 `../skills/reviewed-skill-contract.md`；但 `src/skills/fd-workflow/SKILL.md:8` 只声明 work-management。安装器 `src/plugins/aiw-skills/aiw-skills.py:181` 的 `shared_skill_references` 仅检查入口 SKILL.md，`materialize_skill_source` 虽复制整个技能目录，也仅按入口声明补充共享文件。因此新安装 fd-workflow 不会补齐 reviewed-skill-contract，当前项目 `.agents/skills/fd-workflow/skills/reviewed-skill-contract.md` 也不存在，Worker 的必读依赖断链，旧实现入口附带的规则未完整迁移。请在 fd-workflow 入口显式声明该共享依赖，同步项目入口并加入正确的项目共享副本；无需改变安装器。

## 已核对的其它验收

- 实现参考无 Skill frontmatter，不反向调用旧入口；入口与 Worker 按阶段加载。只实现、部分项、不合并的限制先于派发，完整范围才可 implementation-ready；设计讨论不启动实现。
- 共享生命周期仍拥有授权、独立审查、默认交付归档和冲突恢复；现有 entry 保留处理，未增加 CLI 参数或权限。
- 八组 source/project 文件逐字节一致；六个退役文件均不存在；项目安装索引与 base 比较，仅删除 implement 条目，其它项完整保留。
- 已检查当前路由/安装说明/TDD/Issue/spec 的真实 diff；无旧入口调用。未改 CLI 或全局安装；历史报告保留。

## 实际命令与限制

- 静态读取技能/角色/共享契约、FD/报告、receipt、Git diff 和安装器相关函数；最初尝试读取 events.jsonl 路径不存在，改读实际 events/000004-implementation-ready.json。
- 执行 aiw fd claim；一次有界静态核对批次使用只读 Python 比较副本、索引、删除文件、共享依赖和 FD 摘要，配合 rg、git status、git rev-parse。
- 未运行测试、编译（纯文档无编译目标）、构建、lint、格式化、验证脚本、安装/同步、网络或宿主行为评估。
- 静态证据不能证明宿主实际自动加载和安装运行结果；修复依赖后需重新独立审查该修订。
