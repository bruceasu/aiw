# AI Worker Skills 使用指南

这里维护当前 AIW 的 27 个 canonical Skills。每个技能的 `README.md` 面向使用者，
说明用途和选择方法；`SKILL.md` 是 Agent 执行时的行为合同。二者发生差异时，以
仓库指令和 `SKILL.md` 为准。README 本身不会增加执行、联网或发布权限。

## 从哪里开始

```text
多来源反馈 → 按需 triage → issue-management → fd-workflow
                                          → 独立 fd-review
                                          → 本地交付及归档
```

明确的工程请求可以直接创建 FD。Issue 保留用户目标、软件规则和验收依据；FD 是唯一
工程设计与 Work Item 计划；OpenSpec 管理稳定能力规则，只有显式请求才创建 change。
大型不确定需求在原 Issue／FD 内记录决策依赖，由已有 grilling 方法澄清，不另建地图。

编号 FD 的完整实现请求默认继续独立评审、条件满足后的本地合并和归档，不需要再次
发送 `auto`。只规划、只实现、只审查、部分工作项和不合并等明确限制优先。

需要测试时，可以说“带测试”“需要测试支持”或 `with test`。流程按语义选择合适的
核心规则 TDD 和实现后独立 fd-test；没有这一请求时，不增加默认测试。测试命令仍
遵守仓库授权和有限预算，持续失败后只做静态诊断并继续独立工作；验收缺陷仍阻止交付。

## 技能目录

链接进入独立说明；目录只列当前 `src/skills/<name>/SKILL.md` 实际存在的条目。

### 需求、领域与设计

| Skill | 用途与选择时机 |
| --- | --- |
| [triage](triage/README.md) | 整理多来源用户诉求、信息缺口、相关请求和下一步，不替代软件需求。 |
| [issue-management](issue-management/README.md) | 澄清单个问题、价值、软件规则与验收，批准后交给 FD。 |
| [domain-modeling](domain-modeling/README.md) | 澄清术语、概念关系、状态和不变量，提供正常／边界例子。 |
| [grill-with-docs](grill-with-docs/README.md) | 在关键讨论节点结合资料追问，提高需求和设计结论质量。 |
| [grilling](grilling/README.md) | 沿决策依赖追问，检验计划与假设，等待用户实际作答。 |
| [grill-me](grill-me/README.md) | 显式进入 grilling 访谈的简短入口。 |
| [codebase-design](codebase-design/README.md) | 用小接口、深模块和稳定边界讨论模块设计。 |
| [improve-codebase-architecture](improve-codebase-architecture/README.md) | 查找架构摩擦，生成候选 HTML 报告，再讨论选中的方向。 |
| [to-spec](to-spec/README.md) | 把已确认规则写入稳定规格；不替代 FD 工程计划。 |

### 工程执行、诊断与评审

| Skill | 用途与选择时机 |
| --- | --- |
| [ask-asu](ask-asu/README.md) | 只读判断当前阶段和下一步，不执行建议操作。 |
| [fd-workflow](fd-workflow/README.md) | 编号 FD 的设计、实现、独立评审、本地交付和归档主入口。 |
| [fd-review](fd-review/README.md) | 独立检查编号 FD 的实际实现、验收与证据，并记录正式结论。 |
| [code-review](code-review/README.md) | 审查指定分支、PR 或 diff，也适合没有 FD 编号的工作。 |
| [tdd](tdd/README.md) | 实现期间用有效 Red／Green 推动明确行为，保留回归保护。 |
| [fd-test](fd-test/README.md) | 从 FD 规则独立推导黑盒场景，在授权范围内编写／执行测试。 |
| [diagnosing-bugs](diagnosing-bugs/README.md) | 诊断难复现故障和性能问题，记录假设、证据及复现限制。 |
| [resolving-merge-conflicts](resolving-merge-conflicts/README.md) | 根据 FD 和双方历史解决明确可解的冲突，无法安全解决则保留现场。 |

### 学习、写作、探索与集成

| Skill | 用途与选择时机 |
| --- | --- |
| [teach](teach/README.md) | 围绕学习目标组织跨会话课程、练习、参考材料和学习记录。 |
| [edit-article](edit-article/README.md) | 调整文章结构与信息顺序，逐段改善清晰度和表达。 |
| [research](research/README.md) | 委派资料调查，基于一手来源形成有引用的 Markdown 研究记录。 |
| [prototype](prototype/README.md) | 用一次性逻辑或 UI 原型回答设计问题，结论回到原 Issue／FD。 |
| [handoff](handoff/README.md) | 为另一个会话准备可继续工作的交接文档，不自动派发 Agent。 |
| [setup-project](setup-project/README.md) | 提议并配置项目级 AIW 约定与 Agent 指令。 |
| [fd-release-gate](fd-release-gate/README.md) | 跨领域评估真实发布的候选版本、适用证据和风险，不要求每个FD执行。 |
| [github-issues-management](github-issues-management/README.md) | 通过 aiw-github 查询和管理远程 GitHub Issue。 |
| [publish-github-issue](publish-github-issue/README.md) | 将显式指定的 OpenSpec change 发布为 GitHub Issue 投影。 |
| [writing-great-skills](writing-great-skills/README.md) | 创建、改善、审查或适配 Agent Skills。 |

## 四个辅助技能如何配合工程工作

现在也可直接使用内置 `aiw research`、`aiw prototype`、`aiw edit-article` 和
`aiw teach` 启动辅助对话，不创建 Issue／FD。命令模式读取技能适配引用，采用只读
模型轮次、host 保存新成果；teach 支持多课程与暂停／继续。research 命令首版只用
显式本地资料，不自动启动后台调查。直接 Agent Skill 调用仍遵守各自原始方法。
详细参数与恢复见仓库 `docs/usage/aiw-auxiliary.md`。

- **teach** 教人：适合持续学习一个概念或能力，产出课程、练习和学习记录；不是普通
  一问一答，也不默认创建 FD。
- **edit-article** 改文章：根据读者和表达目标确认章节结构，再改善行文；不会自动
  核实所有事实、发布文章或启动工程流程。
- **research** 查证据：适合官方文档、接口约束和方案事实调查，产出可追溯来源的
  研究记录；网络和子 Agent 操作仍遵守授权与预算，不等于已验证实现。
- **prototype** 验想法：分别提供逻辑／状态模型和 UI 探索两条路径，产物明确是一次性
  原型；不是生产代码或验收结论。被接受的想法通过原 Issue／FD 进入正式设计与实现。

## 源文件、发行目录与项目安装

### 顶层 Markdown 是什么

| 文件 | 角色 | `aiw skills install` 的处理 |
| --- | --- | --- |
| `README.md` | 人类目录导航与用法 | 不作为 Skill 安装，不默认复制到项目技能根目录。 |
| `reviewed-skill-contract.md` | 技能输入、输出、授权与证据通用合同 | 入口声明白名单路径时，复制到该技能的 `skills/`。 |
| `work-management.md` | Issue／FD 角色、交接、工作区与交付合同 | 同上。 |
| `workflow-methods.md` | 分阶段的领域澄清、决策依赖、分流、拆分、规格和评审方法 | 同上，执行时按需读取章节。 |
| `testing-methods.md` | 可选测试的设计、执行与证据方法 | 同上，测试支持按需加载。 |

任意新增顶层 `.md` 不会自动成为共享依赖。当前白名单是上表四个合同／方法文件，
声明形式如 `skills/work-management.md`，必须出现在入口 `SKILL.md` 中。安装器做
字面路径匹配，不递归扫描 README 或 references 来发现额外共享依赖。

### `build.py skills`：复制发行源

在仓库根目录运行 `python build.py skills`，会把整个 `src/skills` 合并复制到
`AIW_INSTALL_DIR/skills`，包含顶层 Markdown、每个技能的 README、入口及支持资源。
`AIW_INSTALL_DIR` 默认是 `C:\green\aiw`，可用同名环境变量覆盖。

它不是向当前项目 `.agents/skills` 安装技能，也不自动加载文件到 Agent。复制采用
`dirs_exist_ok=True`，会覆盖同名文件，**不会清理源中已删除的旧文件或退役技能目录**。
例如仅从仓库删除技能后再复制，发行目录仍可能保留旧条目；本页说明这个限制，不提供
未经核对的自动删除操作。`build.py all` 还会执行其它构建／安装动作，不能用它替代文档检查。

### `aiw skills install`：安装到 Agent 技能目录

canonical source 默认是插件发行树旁的 `skills` 目录；在仓库插件源码布局下对应
`src/skills`。可通过 `AIW_SKILLS_SOURCE_ROOT` 覆盖。当前目录不会自动决定发行版 CLI
使用哪个源，因此修改仓库文档后，旧发行目录中的同名技能不会自行更新。

另一个细节：白名单共享文件目前仍从插件所在树的 `skills` 读取，
`AIW_SKILLS_SOURCE_ROOT` 只改变技能源的选择，不改变这些共享文件的定位。只切换
技能源不能保证发行版中的旧合同也同步更新，应分别核对技能源和共享资源版本。

canonical `list` 和 `install --all` 只识别直接子目录中含 `SKILL.md` 的技能；目录名
需与 frontmatter 的 `name` 一致。显式目录／ZIP 安装另有收集逻辑，不应混同为任意
顶层 Markdown 都能安装。

安装会整体复制选中技能的目录，所以独立 README、references 和其它本地资源都会
保留；随后补入入口声明的白名单共享文件。默认目标是当前项目 `.agents/skills`，
`--scope user` 则使用用户主目录下 `.agents/skills`。已有 unmanaged 同名目录受到保护；
managed 安装记录和本地改动保护仍按 CLI 规则处理。

README 被携带不等于被宿主自动加载。Agent 以 `SKILL.md` 为入口，支持资料按指令读取。

以下为实际 CLI 用法，供需要安装时选择；本次文档更新不会执行这些操作：

```powershell
aiw skills list
aiw skills install fd-workflow --dry-run
aiw skills install fd-workflow
aiw skills install --all --dry-run
aiw skills discover
aiw skills sync fd-workflow
aiw skills install research --scope user
```

## 调用与维护

独立 README 中的 `$skill-name` 是宿主支持的 Skill 调用形式，不是 `aiw` 子命令。
带 `disable-model-invocation: true` 的入口需要显式调用或已定义的流程选用；不能因名称
出现在 README 就假设自动执行。调用例子是示意，具体宿主语法与授权以实际环境为准。

新增／删除 canonical Skill 时同步本页目录和独立 README。修改用途或边界时先修改
相应行为合同，再同步用户说明；不要通过 README 暗改授权。共享方法只维护规范源，
安装副本由现有机制分发。用户级旧副本和已退役技能残留不属于当前 canonical 清单。

源码核对位置：仓库 `build.py` 的 `copy_tree`／`install_skills`，以及
`src/plugins/aiw-skills/aiw-skills.py` 的 `source_root`、`discover_skills`、
`shared_skill_references`、`materialize_skill_source`。这些处理方式由静态阅读确认，
不表示本页示例中的构建、安装或测试已经执行。
