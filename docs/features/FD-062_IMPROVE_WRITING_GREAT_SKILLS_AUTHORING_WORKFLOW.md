# FD-062: 优化 writing-great-skills 的编写流程与跨平台参考

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户要求结合前述讨论优化 writing-great-skills。当前技能偏理论，过度强调过程一致，对调用机制和提示技巧存在绝对化描述，缺少创建、修改、审查及平台适配流程，也未区分静态证据与真实效果。

## Options and decision

选择保留技能名称与显式调用策略，主入口提供有完成条件的轻量流程，通用原则与 Codex/Claude 规则分别放 references，效果评估独立可选。直接拼接官方 creator 会重复规则并引入平台与评估依赖；只修措辞仍无法提供实际工作流程，均不采用。

## Solution

重写 SKILL.md 为创建/修改/只读审查/平台适配四种模式；明确用户授权内的编辑范围、必要输入、结果与完成条件、未知事实处理。常规只做静态检查，运行评估、脚本、多 Agent 对照、网络和打包均遵从宿主授权，不因本技能自动执行。

新增 authoring-principles.md、codex.md、claude.md、evaluation.md、sources.md。吸收已阅读的官方方法并注明来源、核对日期和本地调整；不复制上游脚本、不自动下载或依赖系统绝对路径。主入口可与可用且已授权的原生 creator 协作，不存在时自身仍可用。保留并精简 GLOSSARY.md，纠正调用、可预测性及提示技巧的定义。同步项目副本与界面文案；按现有安装器规则从 canonical 共享契约物化两份项目副本，确保链接可解析。

## Scope

范围：src/skills/writing-great-skills、对应 .agents 项目副本和必要共享契约生成副本、src/skills/README.md、FD/双格式证据。不改技能名称或调用策略、不改代码、依赖、CLI、安装器、稳定能力规格或历史归档，不更新用户级安装。平台文档只采用会话已核对的来源，平台未知处标明需确认，不新增网络。

## Work items

- [ ] 1.1 重写主流程和精简术语，明确任务效果、权限和完成条件。规模小、难度中，无依赖；四种模式可执行，简单改动不必走完整流程。
- [ ] 1.2 新增通用/平台/评估/来源参考，更新 UI/README 并同步项目副本及必要共享引用。规模小、难度中，依赖 1.1；按需读取、自包含、无绝对路径和新增运行依赖。
- [ ] 1.3 一次静态检查并记录双格式证据，交接独立 Reviewer。规模小、难度低，依赖 1.2；如实区分结构检查、行为证据和未验证效果。

## Acceptance

- 主入口描述实际能力，提供触发/非触发、输入、四种模式、完成条件、未知信息与报告边界。
- name、disable-model-invocation 与 allow_implicit_invocation 保持不变。
- 平台发现、隐式选择和明确流程加载分开定义；不把移除 description 或单个平台字段当成通用规则。
- 内容精简、关键词与正向表达是可检验方法，不以词数或过程一致代替效果。
- 评估按授权选择，未运行的行为/对照结果明确未知；无自动网络、脚本、打包或多 Agent 执行。
- 所有新增引用有明确加载时机；项目副本与源一致，共享引用按安装约定可解析。

## TODO

- 完成三个 Work Items 与独立审查；交付合并和归档尚未执行。

## Verification

- 计划一次静态 diff、引用和项目副本一致性检查，独立 Reviewer 检查最终内容。
- 纯 Markdown/YAML，无适用编译目标，不运行无关 Go 编译；不运行验证脚本、测试、效果评估、构建、格式化、网络或安装同步。
- 尚未实施或验证。

## Sources

- Issue: none

- 本轮用户明确要求结合前述讨论优化技能。
- 原技能与 GLOSSARY，基线 8cbec0a。
- 会话已读取的 Codex 系统 skill-creator 和 UI 元数据参考；OpenAI 官方 Build skills 文档。
- 会话已读取的 anthropics/skills 官方 skill-creator。
- src/skills/work-management.md、reviewed-skill-contract.md 及安装器共享引用物化逻辑。
