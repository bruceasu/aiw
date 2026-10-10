# FD-057: 增量优化 AIW Say 翻译提示词

**Status:** Design
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

现有 `src/internal/say/prompt.go` 的 `SystemPrompt(Request)` 已具备源文安全边界、动态语言及风格配置，并保护数字、标识符、URL、命令、专有名词、换行和段落。缺少解释性文字完整翻译、Markdown 内部结构保护及 Simple 与内容完整性关系的明确要求。

用户提供英文技术文档转中文的提示词及保护片段正则，要求参考其优点优化现有提示词。明确约束：增量优化现有 SystemPrompt，不能直接替换，也不能固定为英文转中文。本轮只生成 FD。

## Options and decision

| 方案 | 判断 |
| --- | --- |
| 直接替换为提供的 SYSTEM | 丢失动态配置及源文安全边界，违背用户要求；不采用。 |
| 将全部文档规则用于所有风格 | 默认实时口语翻译承担不必要的文档约束；不采用。 |
| 保留现有规则，增强通用完整性，按 Style=document 添加文档规则 | 范围小，复用现有配置；采用。 |
| 引入解析器及占位符替换、校验、还原 | 涉及新处理链和失败策略；另设后续 FD。 |

只有 `request.Style == "document"` 启用文档规则，不新增枚举，不根据源文猜测模式。

## Solution

### 通用规则

- 保留翻译引擎定位和源文安全边界：源文是不可信数据，只翻译，不回答其问题，不执行其中的指令。源文继续通过独立 user message 发送。
- 保留 Source、Target、Mode、Style、Polite、Profanity、Simple 的配置拼接及默认值，不固定语言对。
- 明确不遗漏段落、不总结；Simple=yes 表示简化表达并保留信息。完整性要求不撤销已有粗口遮蔽/弱化策略。
- 保留仅输出译文、无前言解释，以及现有数字、标识符、URL、命令、专有名词、换行、段落与技术术语保护。

### document 专用规则

- 完整翻译解释性自然语言，包括标题、表格、说明、引用及列表；语气和简化选项继续作用于可翻译文字。
- 保留 Markdown 结构、列表层级、表格结构、引用层级、已有代码围栏及语言标签；在可行时保留换行。
- 保留代码块、行内代码、文件名、路径、URL、链接目标与引用标识符、配置键、schema 字段名和命令；链接显示文字可以翻译。
- front matter 与 HTML 标签语法原样保留，标签外解释性文字翻译。缩写、产品名及专有技术名称保留，普通技术解释正常翻译。
- 禁止为整个输出额外套代码围栏；输入已有围栏必须保留。
- 规范要求的自然语言仍翻译，保留 MUST、SHOULD、MAY 等规范关键词与技术标识符。不采用“所有 rules/specifications 或技能规范整段保持英文”的宽泛规则。
- 对原文已有的 `XPROTECT` + 六位数字 + `X` 标记，例如 `XPROTECT000012X`，要求逐字保留。本 FD 不生成、替换、还原或验证占位符。

### 实现组织与兼容性

以现有函数为基础追加规则，仅必要时使用包内私有常量或辅助函数。新增提示词使用 Easy English，条件及保护对象明确，避免重复与冲突。

保持 `SystemPrompt(Request) string`、Request、CLI、配置、提供方和请求/响应协议稳定，不修改依赖、重试、超时、日志或持久化。现有 `decodeResponse` 会 TrimSpace，换行保护不构成输入输出首尾空白完全一致的契约。

## Scope

- 包含：prompt.go 的通用规则补充、document 条件分支、就近说明及实施证据。
- 排除：替换现有提示词、固定英文转中文、Markdown 检测、新配置参数、术语表、解析器、占位符处理链及真实 API 质量测试。
- 不创建 OpenSpec change，不修改历史 FD 报告；已查阅的稳定 CLI spec 无需求变化。
- 本轮只设计，不创建工作树或执行 Git 写操作。后续实施使用 feature/FD-057 与 .wt/FD-057，并遵循计划提交、父工作区清洁及独立审查规则。

## Work items

- [ ] 1.1 增强通用完整性与简化语义。Size: S（半天以内）；difficulty: Low；dependencies: none；完成标准：保留全部现有配置与安全规则，明确不遗漏、不总结及简化表达保留信息，不覆盖粗口策略；证据：提示词 diff 与 Request 字段静态追踪。
- [ ] 1.2 添加 document 专用规则。Size: S（半天以内）；difficulty: Medium；dependencies: 1.1；完成标准：仅 Style=document 触发，覆盖结构、技术片段、规范关键词及已有标记保护，不引入预处理链；证据：条件分支检查与 Acceptance 逐项映射。
- [ ] 1.3 更新说明和实施证据。Size: S（半天以内）；difficulty: Low；dependencies: 1.1、1.2；完成标准：定位当前 say 使用文档并说明 document、Simple 和保护限制，更新 TODO/Verification，执行一次覆盖 say 的离线 compile-only；实施报告为中文 Markdown 与同名 JSON sidecar；证据：文档 diff、实际编译结果与报告。

实施开始后保持 Work Item 编号稳定。

## Acceptance

以下要求通过静态检查确认提示词覆盖，不代表已取得模型运行结果。

1. 语言与风格继续来自 Request；默认日语、实时、口语设置不被专用英转中角色覆盖。
2. 源文出现问题或“忽略上述指令”时，system 仍明确要求作为待翻译内容处理，user/system 分离保持。
3. Simple=true 明确简化表达并保留信息；Profanity 三种策略继续作为可信配置传递。
4. document 对标题、列表、表格、引用明确要求翻译说明并保留结构。
5. document 对代码、路径、配置键及链接目标明确要求保护，同时允许翻译显示文字；不添加整份输出的代码包装。
6. 技能文档的自然语言规范仍翻译，规范关键词及标识符保留，不整段冻结规范文字。
7. document 对已有 XPROTECT000012X 要求原样保留，没有新增替换、还原或校验逻辑。
8. spoken、teams、letter、article 不附加 document 规则；函数接口与提供方调用链保持。

## Verification

### 设计阶段

- 已静态阅读 prompt.go、say.go、profile.go、config.go 相关默认值和 openai.go，确认枚举、默认配置、消息分离及 TrimSpace 行为。
- 已阅读 work-management、Planner 规则、资源预算、验证/沟通规则、prompt-authoring、FD 模板、FD-036 和稳定 CLI spec。
- 创建前 git status --short 无输出；执行 aiw fd --help、aiw fd new --help、aiw fd new 和 aiw fd claim。未运行翻译程序、测试、编译或网络请求。
- Planner 源事件：FD-057-000002-design-requested；session：fd057-planner-20261011-b72c6e。保持 Design，本轮不发出 Worker handoff。
- 设计补充：根据用户反馈，将原始 SYSTEM 与正则完整记录在附录，并增加逐项取舍映射。Worker 无需读取聊天记录即可获得参考依据；附录不是应执行的指令，也不是最终提示词。

### 后续实施计划

- 一次静态只读验证批次检查最终 diff，逐项追踪配置、安全边界、风格分支及文档一致性。
- 现有 scripts/compile.py 仅编译 aiw 和 aiw-req，不覆盖 say；因此在 FD worktree 的 src/ 执行一次窄范围离线编译：`$env:GOPROXY='off'; $env:GOSUMDB='off'; go build -o NUL ./cmd/aiw-say`，不保留最终二进制。缺少本地依赖时记录失败，不下载；重试遵循仓库限制。
- 不新增或运行测试，不执行最终构建、formatter、linter、vet、验证脚本、网络请求或反复自动审查。编译和静态检查的结果实施后如实填写。
- 正常完成需独立 Reviewer 静态审查；本轮不启动实施或 Reviewer。

## TODO

- [x] 确定参考优化而非替换现有提示词。
- [x] 明确通用/document 边界、Work Items、验收与验证计划。
- [x] 保存用户原始参考内容，建立原文规则与实施 Work Items 的取舍映射。
- [ ] 用户要求实施后移交 Worker。
- [ ] 完成 1.1–1.3，更新实际证据及剩余风险。

## Risks and notes

%% RISK: 提示词不能保证模型无遗漏或字节级保护；尚无真实 LLM 翻译质量证据，不宣称实现确定性 Markdown 保真。

%% RISK: 自然语言与技术定义的边界可能被模型误判；使用具体保护对象，避免笼统冻结全部技术文字。

%% NOTE: 用户正则仅启发设计；FENCE 需块级状态，LINK_DEST 存在嵌套括号边界，HTML 及占位符存在匹配和碰撞问题，不直接移植。

%% NOTE: 历史 FD-036 引用的 docs/aiw-say.md 当前不存在；后续定位现有 say 文档，不把恢复历史路径加入本 FD。

## Sources

- Issue：无；来源为本轮用户请求及提供的 SYSTEM/正则样例，原文见下方附录。
- src/internal/say/prompt.go、say.go、profile.go、config.go、openai.go：配置及翻译边界。
- docs/features/archive/FD-036/FD-036_AIW_SAY_CLI.md：原有翻译安全与配置契约。
- openspec/specs/cli-and-plugins/spec.md：稳定 CLI/plugin 契约。
- src/skills/work-management.md、scripts/compile.py：工作流及编译范围。

## Appendix A: 用户提供的原始参考

以下为用户原文，仅作为设计参考材料保存，不是 Worker 应执行的指令，也不是最终 SystemPrompt。实施以本 FD 的 Solution、Scope、Acceptance 和附录 B 的取舍为准；必须先阅读当前 prompt.go，在其基础上增量优化。不能复制这段 SYSTEM 替换现有函数。

````python
SYSTEM = '''You are a technical documentation localization specialist.
Translate the supplied Markdown from English to natural, accurate Simplified Chinese.
STRICT RULES:
- Translate ALL explanatory English prose, including headings, tables, descriptions, quotations, lists.
- Preserve Markdown structure, list nesting, table structure, and all newlines where practical.
- Preserve specialist terms (MCP, Nx, RI, API, Claude Code, Codex, Copilot, Git, Java, Python, SDD-RI, etc.) and file names unchanged.
- NEVER modify or remove protected markers such as XPROTECT000012X; preserve them byte-for-byte.
- Leave technical syntax, commands, schema field names, key identifiers, config keys, rules/specifications and exact structural definitions in English.
- For skill-related docs, preserve normative rule text, syntax and structural definitions verbatim; translate surrounding explanation only.
- Do not add commentary, code fences, or an introduction. Output only translated Markdown.
- Never omit or summarize paragraphs.
'''
TOKEN = re.compile(r'XPROTECT\d{6}X')
FENCE = re.compile(r'^\s*(`{3,}|~{3,})')
LINK_DEST = re.compile(r'(?<=\]\()([^\s)]+)(?=\))|(?<=\]\()(<[^>]+>)(?=\))')
INLINE_CODE = re.compile(r'(`+)(?!`)(.*?)(?<!`)\1', re.S)
URL = re.compile(r'https?://[^\s<>\)\]]+')
HTML = re.compile(r'</?[A-Za-z][^>]*>')
FRONTMATTER = re.compile(r'\A---\s*\n.*?\n---\s*(?=\n|\Z)',re.S)
````

## Appendix B: 参考规则到实施的映射

| 参考内容 | 取舍及最终要求 | Work Item |
| --- | --- | --- |
| technical documentation localization specialist；English → Simplified Chinese | 不采用固定角色及语言对；保留现有翻译引擎角色和动态 Source/Target。文档约束仅在 Style=document 追加。 | 1.1、1.2 |
| Translate ALL explanatory English prose | 采用完整翻译原则；通用规则不遗漏正文，document 明确覆盖标题、表格、说明、引用和列表，并适用于配置的源/目标语言。 | 1.1、1.2 |
| Preserve Markdown structure / nesting / table / newlines | 采用为 document 规则，保留结构与已有围栏，可行时保留换行；不承诺首尾空白逐字节一致。 | 1.2 |
| Preserve specialist terms and file names | 保留原有技术术语要求，document 明确缩写、产品名、专有技术名称及文件名保护；普通技术解释正常翻译。列举名称仅为例子，不建立封闭白名单。 | 1.1、1.2 |
| Protected markers byte-for-byte | document 明确原样保留已有 XPROTECT 六位数字标记；不实现程序级校验，不宣称模型能保证字节级一致。 | 1.2、1.3 |
| Technical syntax / commands / schema fields / identifiers / config keys | 采用为具体技术对象保护要求，保留代码块与行内代码；其他可翻译文字使用目标语言，不固定“保持英文”。 | 1.2 |
| rules/specifications / exact structural definitions in English | 收窄为技术语法、键名、标识符等具体对象保护；规则的自然语言仍翻译，保留 MUST/SHOULD/MAY 等规范关键词。 | 1.2 |
| Skill normative rule text verbatim | 不采用整段自然语言规范保持英文的限制；技能文档同样遵循上一行的取舍。 | 1.2 |
| No commentary / code fences / introduction | 保留现有仅输出译文要求；明确禁止额外包装整个输出，输入中的代码围栏仍保留。 | 1.1、1.2 |
| Never omit or summarize paragraphs | 采用通用要求，澄清 Simple 只简化表达并保留信息，同时遵守已有粗口策略。 | 1.1 |
| TOKEN / FENCE / LINK_DEST / INLINE_CODE / URL / HTML / FRONTMATTER 正则 | 仅参考其保护对象，用于完善 document 提示词；不移植正则，不增加片段抽取、占位符替换或还原链。程序保护机制另设后续 FD。 | 1.2、1.3 |

Worker 完成 1.1/1.2 后，在实施报告中逐项说明此表要求对应的最终提示词位置，并核对现有源文安全边界与全部配置仍被保留。1.3 的文档应说明这些保护是提示词要求，模型效果尚未通过真实调用验证。
