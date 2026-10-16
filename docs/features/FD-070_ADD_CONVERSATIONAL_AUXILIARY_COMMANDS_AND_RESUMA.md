# FD-070: Add conversational auxiliary commands and resumable teaching courses

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户希望 AIW 提供 research、prototype、edit-article、teach 对话入口。先理解任务，再按对应 Skill 完成；教学需多个课程、暂停和继续。这些是独立辅助工作，不是 Issue/FD 的需求或实施阶段。本 FD 仅用于实现这些产品能力。

## Options and decision

| 方案 | 决定 | 理由 |
| --- | --- | --- |
| 四套独立聊天与存储 | 不采用 | 重复后端、历史和恢复逻辑。 |
| 全部强制进入 Issue/FD | 不采用 | 日常学习、写作与探索不是工程交付。 |
| 共用 Session 的辅助命令适配层 | 采用 | 四个命令共享会话与产物协议，按模式应用 Skill，保留各自交互。 |

## Solution

新增 internal/commands/assist，主 CLI 注册 aiw research、prototype、edit-article、teach。复用 internal/session 的持久化、冻结请求、输出/usage 保存和 provider 路由，不修改 Session 公共 JSON 形状。辅助元数据存为当前 Session artifacts/auxiliary.json；课程是 kind=teach 的独立 Session，各课程资料、目标、历史和课件隔离。所有辅助工作不创建、批准或推广 Issue/FD，不操作 Git。

公共形式：`aiw <kind> [new] [topic] [--source PATH ...] [--provider NAME] [--model NAME]`、`list`、`resume <id>`；prototype 加 `--kind logic|ui`。无参数时先展示该类型可继续项目并选择新建或恢复。交互命令 `/run`（根据已有澄清摘要生成成果）、`/brief`、`/pause`、`/exit`、`/help`；teach 日常问答持续教与练，`/run`生成本节课件，非一次性任务结束。Ctrl+C/EOF 安全暂停；恢复读取已保存目标、对话与产物，不重置为新课程。只查询/list/help不调用模型。

每个普通输入最多一次模型调用。澄清返回结构化 reply/ready/brief；仅用户 `/run` 明确使用最近就绪的 brief 进入成果生成，不再加重复 confirm。新的澄清输入使旧ready失效；生成失败保留 brief 供显式重试，无自动重复模型调用。teach 的问答和课件增量保存，原稿与旧成果不覆盖。

按现有 provider/model 全局配置与显式覆盖解析。使用 ExecuteFrozenTurn、ReadOnly=true、新 provider thread 和冻结 Skill/上下文，不让 model 自行写工作区或使用 workflow 工具。生成阶段也是模型返回文件内容，由 host 以有界 JSON 协议验证后写入独立成果目录；不执行模型命令、不运行原型、不自动联网研究、不改产品源码、不原地覆盖文章。CLI provider不声称拥有网络或全面文件隔离；其只读能力依赖既有 provider 实现。research 首版从用户显式给定的本地资料研究（可含已保存网页资料），缺少外部资料时说明缺口，不伪装已联网或已执行后台调查。其“后台 Agent”方法适配为独立只读模型任务，并明确实际模式。

`--source`允许当前项目内的明确文本文件/目录，拒绝符号链接/越界/敏感凭据，限制文件数、单文件大小、总量和类型；模型收到有路径标识的只读快照。原文是数据不是指令。Skill源按显式AIW_SKILLS_SOURCE_ROOT、项目src/skills、发行skills、项目安装目录解析，选中后冻结入口与该模式必要引用，保存digest；恢复沿用快照。当前任务阶段指令优先适配 Skill 中工程分支/全课程/自动写入等默认动作，不启动第二生命周期。

新产物名称只能是安全单文件名，禁止绝对路径、目录、重复文件名和控制字符，限数量/大小；放在 `artifacts/result-<turn>/`，目录独占创建，无覆盖。元数据和历史用 Session WriteArtifact 原子替换；会话在进程级独占锁下运行，拒绝并发和未知遗留锁。取消模型调用通过 context 传递，保存失败/取消证据后暂停；硬杀造成遗留锁或running态时不盲目接管，报告现场供人工确认恢复。无后台守护服务、自动删除和自动重试。

独立设计反馈D1/D2补充：模型调用前写入pending轮次/输入/phase，完成后保存不可变辅助turn记录、host成果和auxiliary元数据的LastTurn。恢复时要求auxiliary与Session.LastTurn一致且无pending；不一致拒绝新模型调用，提供 `recover <id>` 读取已有冻结输出完成本地提交，不重复调用模型。已有成果只校对同一输出，可补写缺失文件但不覆盖不同内容；无法确认轮次、残留运行态或未知锁则保留现场。Session结果、pending及原始输出都是事实，不因辅助提交失败宣称成功。

完整辅助轮次历史逐轮保存在artifacts，模型上下文只带固定课程目标/当前brief、最近6轮和最近成果索引；总记忆限制32KiB，Skill/资料也各自有界。不静默截断规则或伪称已读取完整历史；超限时要求缩小资料或明确历史范围。`/recall N`显式加入一个较早轮次（仍受预算），已保存完整历史不删除。每课独立，不让其它课程历史进入提示词。

新CLI/持久化设计先进行有界独立设计评审，确认后才design-ready；不变更已有provider、Session状态机、auth或依赖。FrozenTurn可新增可选SystemPrompt内部字段并透传Request，原调用默认空且Session公共JSON不变。辅助调用显式组装冻结Prompt数据与高优先级阶段指令，不依赖Instructions自动拼接，不使用InvocationObserver（其Codex路径会选择workspace-write）。保留现有命令兼容，帮助/补全/README/规格同步。

## Scope

包含：四命令与共享辅助包；受限文本上下文与Skill快照；共用Session的任务/课程恢复和产物；帮助/补全、独立使用文档、四Skill的辅助CLI适配引用、stable capability规格及双格式证据。

不包含：自动创建Issue/FD、后台守护服务、网页浏览器/联网检索工具、生产代码写入、文章原地覆盖、原型执行、依赖安装、课程自动知识采集、运行时测试/模型请求、全局安装或build发行。teach首版以教学对话与可保存课程材料为主，不强制创建完整教学工程目录。

## Work items

- [ ] 1.1 定义CLI/持久化合同并取得独立设计反馈。Size: M; Difficulty: Medium; Dependencies: none. Completion: 命令、恢复、权限和数据边界明确，重要设计发现解决后就绪。
- [ ] 1.2 辅助会话存储、列表与课程恢复。Size: M; Difficulty: Medium; Dependencies: 1.1. Completion: 每任务/课程独立Session和元数据，类型/项目验证、锁、持久历史、暂停/恢复、不覆盖旧成果。
- [ ] 1.3 有界来源/Skill加载与模型协议。Size: M; Difficulty: Medium; Dependencies: 1.2. Completion: 路径/敏感文件/体积约束、冻结技能与输入、只读模型轮次、结构化结果校验与host产物写入。
- [ ] 1.4 共享对话循环及四模式行为。Size: M; Difficulty: Medium; Dependencies: 1.3. Completion: 澄清/brief/run、teach持续问答、EOF/Ctrl+C保留现场，无隐式执行/自动重试，错误可解释。
- [ ] 1.5 主CLI/帮助/补全接入。Size: S; Difficulty: Low; Dependencies: 1.4. Completion: 四名称可达、帮助/list无需模型，旧命令保留。
- [ ] 1.6 文档/技能适配/规格和交付证据。Size: M; Difficulty: Medium; Dependencies: 1.5. Completion: 新辅助能力可理解且不冒充运行验证，一次离线compile-only与有界静态检查、独立实现评审、正常本地交付归档。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

- 四个aiw命令均支持任务澄清、显式生成和成果位置展示；辅助任务无Issue/FD/Git生命周期副作用。
- teach可创建多个课程，按ID/列表选择继续；目标、材料、对话与课件不混课；暂停/恢复不用重新交代已保存内容。
- 新输入使先前生成就绪状态失效；/run只执行用户已看到的当前任务摘要；未就绪不调用生成模型。普通轮次/生成轮次各一次，无隐式重试。
- 用户source/原稿仅只读，跨项目/符号链接/凭据或过大输入被拒绝或明确说明；冻结Skill来源和原文版本可追溯。
- 模型只返回成果内容，由host约束文件名/大小/数量后保存；原型生成不等于运行，研究来源不足不伪装为联网证据，课程不默认联网获取资料。
- EOF/Ctrl+C在输入或模型调用期间可取消并保存到可恢复状态，硬杀未知现场不自动抢锁；并发同一课程被拒绝。
- 中断或提交失败造成pending/LastTurn不一致时禁止重放模型；recover仅从已保存结果恢复本地材料，冲突不覆盖。长期课程保留完整历史但冻结上下文有限，/recall允许显式回顾旧轮次。
- list/help不调用provider；错误/无效JSON/取消保留原始模型输出，不能记为成果成功或自动循环修复。
- 新命令帮助、补全、README、四种Skill适配和stable规格一致；无新依赖或共享Session/API行为破坏。

## Verification

- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

- 本FD不选择测试支持；不写/跑Go测试，不执行模型/网络/安装/最终构建。
- 实现后使用scripts/compile.py一次compile-only，设置GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local，不保留二进制；不得下载依赖。
- 一批有界静态检查主分发/帮助/补全、状态/错误/取消路径、路径与产物限制、文档一致性；独立设计反馈和实现Reviewer不自审。
- 独立设计反馈第1轮：docs/features/reviews/FD-070-design-r1.md/.json，D1本地提交恢复与D2有限课程上下文已按上述补足；第2轮聚焦复核 docs/features/reviews/FD-070-design-r2.md/.json 建议ready，设计门槛已解决，不是实现通过事件。

## TODO

- [ ] 完成1.1–1.6、静态证据与独立实现评审。
- [ ] 正常本地交付及归档。

## Sources

- Issue: none

- 本会话用户已确认的辅助对话命令设计、多课程/中断/恢复及不属于Issue/FD的决定。
- src/internal/session、src/internal/ai、src/internal/commands/ask、src/cmd/aiw/main.go。
- src/skills/{research,prototype,edit-article,teach}/SKILL.md及引用；openspec/specs/cli-and-plugins、ai-routing和command-help-consistency。
