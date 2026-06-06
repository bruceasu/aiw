# aiw CLI 重构与 `cz` 能力重建设计

## 背景

当前仓库中的命令实现主要散落在根目录多个 `.go` 文件中，`git cz` 也以内嵌方式挂接在 Git 快捷命令集合里。随着 `gitcz` 计划向 `cz-git` 参考项目靠拢，现有结构已经不足以承载更复杂的交互、配置和 AI 能力。

本次设计目标不是照搬 Node.js 项目的内部实现，而是在 Go 中重建等价的用户能力，并顺手完成 CLI 命令结构整理。

## 当前结论

### 总体方案

采用“功能全覆盖，但使用更符合 Go 的内部结构”的方案：

- 用户侧能力尽量向参考项目对齐
- 不追求兼容参考项目的内部模块划分
- 不追求兼容参考项目的配置文件格式
- CLI 入口统一重组为 `main -> cmd/* -> internal/*`

### 新增约束

以下约束已确认，后续设计和实现必须遵守：

1. `cz` 代码移动到 `cmd/cz` 目录，并拆分子模块
2. `cz` 不再挂接到 `gitcmd.go`，直接由 `main.go` 分发
3. `git`、`wt`、`tcc` 分别迁移到 `cmd/git`、`cmd/wt`、`cmd/tcc`
4. openspec-like 命令迁移到 `cmd/task`
5. 公共代码迁移到 `internal`
6. `cz` 第二阶段只支持本项目自己的配置文件格式，无需兼容 `cz-git` / `czg` 配置
7. 不保留过渡兼容命令 `aiw git cz`
8. TUI 交互方式尽可能贴近参考项目，这是产品卖点之一

## 目录设计

建议目录结构如下：

```text
aiw/
├─ main.go
├─ cmd/
│  ├─ cz/
│  │  ├─ command.go
│  │  ├─ config.go
│  │  ├─ tui.go
│  │  ├─ ai.go
│  │  ├─ draft.go
│  │  ├─ commit.go
│  │  └─ help.go
│  ├─ git/
│  │  ├─ command.go
│  │  └─ ...
│  ├─ wt/
│  │  ├─ command.go
│  │  └─ ...
│  ├─ tcc/
│  │  ├─ command.go
│  │  └─ ...
│  └─ task/
│     ├─ command.go
│     └─ ...
└─ internal/
   ├─ gitx/
   ├─ envx/
   ├─ fsx/
   ├─ textx/
   ├─ termx/
   └─ ...
```

## 命令边界

### `main.go`

- 只负责一级命令分发
- 不包含具体命令业务逻辑

### `cmd/cz`

- 顶级命令入口为 `aiw cz`
- 负责配置读取、交互流程、AI 生成、消息组装、提交
- 不依赖 `cmd/git` 提供入口

### `cmd/git`

- 保留 Git 快捷命令集合
- 不再承载 `cz` 功能

### `cmd/wt`

- 承载 worktree 管理子命令

### `cmd/tcc`

- 承载 TCC 包装器命令

### `cmd/task`

- 承载 openspec-like 任务工作流命令
- 包括 `init/new/list/show/status/done/archive/context/decision/spec/registry/prompts`

### `internal/*`

- 放置跨命令共享能力
- 例如 Git 命令执行、环境变量/配置加载、编辑器启动、文本包装、终端能力探测

## `cmd/cz` 功能设计

`cmd/cz` 的用户感知能力分为五层：

1. 命令与参数层
2. 配置层
3. TUI 交互层
4. AI 生成层
5. commit message 组装与提交层

### 1. 命令与参数层

建议顶级命令形态：

```text
aiw cz
aiw cz ai
aiw cz emoji
aiw cz break
aiw cz checkbox
```

建议支持的会话级选项：

- `--config`
- `-N`, `--ai-num`
- `-M`, `--ai-model`
- `--api-key`
- `--api-endpoint`
- `--api-proxy`
- `--no-ai`
- `-r`, `--retry`
- `-h`, `--help`

### 2. 配置层

第二阶段只支持本项目自己的配置文件格式：

- `aiw.toml`
- `.aiw.toml`

不兼容以下格式：

- `.czrc`
- `cz.config.json`
- `cz.config.js`
- `commitlint` 配置自动导入

建议保留并逐步扩展的自有配置项：

- `types`
- `scopes`
- `scopeOverrides`
- `enableMultipleScopes`
- `scopeEnumSeparator`
- `allowCustomScopes`
- `allowEmptyScopes`
- `markBreakingChangeMode`
- `allowBreakingChanges`
- `useAI`
- `aiModel`
- `aiNumber`
- `aiDiffIgnore`
- `upperCaseSubject`
- `maxHeaderLength`
- `maxSubjectLength`
- `minSubjectLength`
- `issuePrefixes`
- `allowCustomIssuePrefix`
- `allowEmptyIssuePrefix`
- `emoji`
- `messages`

### 3. TUI 交互层

TUI 是卖点，目标是尽可能接近参考项目当前体验，而不是退化成普通问答式输入。

至少需要支持：

- 单选列表
- 可搜索列表
- checkbox 多选
- 空选项和自定义项
- 普通文本输入
- 多行输入
- AI 候选选择
- commit message 预览
- 外部编辑器编辑

优先追求：

- 交互顺滑
- 键盘操作体验统一
- 视觉反馈接近参考项目

### 4. AI 生成层

建议 AI 仍以 staged diff 为输入，支持：

- 模型切换
- API endpoint / proxy / key
- 候选数量
- 忽略指定文件的 diff
- 选择候选后继续人工编辑

第一阶段建议重点生成：

- `subject`
- 或候选 header

不优先让 AI 自动完整生成 `body`、`breaking`、`footer`，以降低幻觉和误判风险。

### 5. 消息组装与提交层

建议从交互和 AI 中解耦，成为纯业务逻辑模块，负责：

- header 生成
- emoji 映射
- scope 拼接
- breaking 标记
- body/footer 拼装
- 长度校验提示
- 最终提交
- retry 最近一次消息

## 阶段拆分

### Phase 1: CLI 结构重构

- `main.go` 改为顶级命令分发
- `git`、`wt`、`tcc`、`task` 拆到 `cmd/*`
- 公共逻辑下沉到 `internal/*`
- 现有 `gitcz` 从根目录逻辑中抽离，建立 `cmd/cz` 基础骨架

### Phase 2: `cmd/cz` 基础迁移

- 把现有 `gitcz.go` 的能力迁入 `cmd/cz`
- 使用本项目自己的配置格式
- 去掉 `aiw git cz` 入口
- 保持现有可用能力不回退

### Phase 3: TUI 重建

- 实现接近参考项目的选择式 UI
- 支持搜索、checkbox、多候选 AI、预览与编辑
- 这是 `cz` 产品价值的重点阶段

### Phase 4: `cz` 能力增强

- 补齐 issue prefix、breaking 模式、长度提示、retry 等体验
- 完善配置项覆盖面
- 根据需要补细节和兼容行为

## 风险与约束

### 风险

- TUI 若不引入合适的 Go 终端交互能力，可能难以达到参考项目体验
- 命令重构涉及入口迁移，回归面比较大
- 当前仓库尚未初始化 `openspec/` 目录，和仓库工作流说明存在偏差

### 暂不做

- 不兼容 `cz-git` / `czg` 配置文件
- 不执行 JS 配置文件
- 不保留 `aiw git cz` 兼容入口

## 待继续设计

下一阶段需要继续细化：

1. 根目录现有文件到 `cmd/*` / `internal/*` 的逐文件迁移方案
2. `cmd/cz` 的 TUI 技术选型与实现边界
3. `main.go` 的新分发结构与兼容策略
4. 测试重组策略

## `cmd/cz` TUI 详细设计

### 目标

`cmd/cz` 的 TUI 不是辅助功能，而是产品卖点。设计目标不是“能选”，而是尽可能复刻参考项目当前交互方式的核心体验：

- 搜索式单选
- 搜索式多选
- 实时状态提示
- 分页滚动
- 自定义项与空项插入
- 输入态与选择态的自然切换

### 参考体验提炼

从参考项目当前交互实现中，可以提炼出以下必须保留的体验特征：

1. `type` 使用可搜索列表，而不是纯数字菜单
2. `scope` 在单选模式下使用可搜索列表，在多选模式下使用可搜索 checkbox
3. 搜索结果会随输入实时过滤
4. 交互提示清楚，例如：
   - 单选提示方向键或输入搜索
   - 多选提示 `<space>` 选择、`<enter>` 提交
5. 支持分页和光标循环
6. 自定义项和空项不是另开命令，而是插在选项流中
7. `subject` 输入时有实时长度反馈
8. 最终输出前有明确预览和再次编辑入口

### 交互模型

建议将 TUI 抽象成四类组件：

1. `SearchList`
2. `SearchCheckbox`
3. `CompleteInput`
4. `PreviewConfirm`

#### `SearchList`

使用场景：

- 选择 `type`
- 单选 `scope`
- 选择 issue prefix
- 选择 AI 候选

需要支持：

- 输入即过滤
- 上下方向移动
- `tab` / `down` 向下切换
- `enter` 确认
- 无结果提示
- 初始默认项定位
- 可插入分隔项和 pinned item

#### `SearchCheckbox`

使用场景：

- 多 scope 模式

需要支持：

- 输入即过滤
- `space` 勾选/取消
- `enter` 提交
- 已选结果在过滤切换后保持状态
- 自定义项单独处理
- 多选结果按配置分隔符拼接

#### `CompleteInput`

使用场景：

- 自定义 scope
- subject
- body
- breaking
- footer

需要支持：

- 普通文本输入
- 可回填默认值
- 可定制 transformer
- 支持验证函数
- 支持 subject 实时长度提示

#### `PreviewConfirm`

使用场景：

- 最终 commit 前确认

需要支持：

- 渲染完整 commit message 预览
- 快捷确认提交
- 返回编辑
- 取消退出
- 调起外部编辑器

### 推荐的交互流程

建议 `cmd/cz` 的主流程如下：

1. 检查 staged diff
2. 进入 `type` 搜索单选
3. 进入 `scope` 搜索单选或搜索多选
4. 如命中自定义项，进入 `custom scope` 输入
5. 若启用 AI：
   - 生成候选
   - 进入 AI 候选选择
   - 回填 `subject`
6. 进入 `subject` 输入并显示长度提示
7. 进入 `body`
8. 进入 `breaking` 相关流程
9. 进入 issue prefix / footer 流程
10. 进入预览确认
11. 提交 / 编辑 / 取消

### 技术选型建议

TUI 方案我建议分两层：

#### 方案 A：自研轻量终端交互层

优点：

- 更容易精确贴近参考项目交互方式
- 不会被现成框架的组件模型限制
- 可控性高

缺点：

- 终端输入处理、ANSI 渲染、分页、按键映射都要自己实现
- Windows 兼容性成本更高

适用前提：

- 只实现本项目需要的少量组件
- 控制组件数量，避免演化成一个通用 TUI 框架

#### 方案 B：基于 Go TUI 库封装组件

优点：

- 输入事件、屏幕刷新、样式控制会更省力
- 跨平台稳定性通常更好

缺点：

- 组件交互可能和参考项目不完全一致
- 搜索列表、搜索 checkbox 往往仍需二次封装

我的推荐是：

- **优先选方案 B**
- 但不是直接套现成列表组件，而是在 TUI 库之上封装我们自己的：
  - `SearchList`
  - `SearchCheckbox`
  - `CompleteInput`
  - `PreviewConfirm`

这样既保住交互体验，也减少底层终端兼容工作量。

### 实现边界

第一版 TUI 需要做到：

- 单选搜索
- 多选搜索
- 自定义项
- 空项
- AI 候选选择
- subject 长度提示
- 预览确认
- 外部编辑器编辑

第一版可以暂缓：

- 复杂颜色主题配置
- 动态动画效果
- 鼠标支持
- 完整的 separator 富渲染
- 像参考项目一样的全部视觉细节

### 状态模型

建议在 `cmd/cz` 中引入显式状态对象，而不是让每个交互函数直接拼字符串：

```go
type SessionState struct {
    Type          string
    Scope         []string
    CustomScope   string
    Subject       string
    Body          string
    Breaking      string
    FooterPrefix  string
    Footer        string
    MarkBreaking  bool
    AICandidates  []DraftCandidate
}
```

作用：

- TUI、AI、提交组装共享同一个状态
- 便于在“AI 回填后继续人工编辑”场景下保持一致
- 测试时可以直接验证状态流转

### 终端兼容策略

这是 Go 版最容易踩坑的点，建议提前定义边界：

- 优先保证 Windows Terminal、PowerShell、常见 Unix shell
- 若检测到不支持交互终端：
  - 降级为线性问答模式
  - 或明确报错并提示改用交互终端

建议不要为了兼容极弱终端而牺牲交互体验主路径。

### 测试策略

TUI 不能只靠人工试。

建议至少分三层测试：

1. 过滤与选择逻辑的纯单元测试
2. 状态流转测试
3. 少量端到端交互快照测试

重点覆盖：

- 搜索过滤
- 默认项定位
- checkbox 勾选保持
- 自定义项分支
- subject 长度校验提示
- AI 候选选择后回填

### 设计结论

`cmd/cz` 的 TUI 应当被当成独立子系统设计，而不是若干 `fmt.Scanln` 的增强版。推荐路径是：

- 用 Go TUI 库承载底层输入输出
- 在其上封装贴近参考项目体验的自有组件
- 明确提供非交互降级路径
- 把交互状态从 commit 组装逻辑中解耦

## `cmd/cz` 内部模块与数据结构设计

### 目标

当前 `gitcz.go` 中的主要问题不是功能少，而是职责混合：

- 配置解析和业务状态混在一起
- TUI 和 AI 逻辑混在一起
- 编辑器、Git、OpenAI 调用都直接耦合在命令文件里
- commit message 组装与交互流程彼此穿插

新的 `cmd/cz` 设计目标是把这些职责拆成可独立理解、可测试、可替换的模块。

### 模块划分

建议 `cmd/cz` 至少拆成以下文件或子模块：

```text
cmd/cz/
├─ command.go
├─ options.go
├─ config.go
├─ session.go
├─ flow.go
├─ tui.go
├─ ai.go
├─ draft.go
├─ message.go
├─ commit.go
├─ editor.go
└─ help.go
```

#### `command.go`

职责：

- `aiw cz` 顶级入口
- 命令分发
- 把参数、配置、依赖装配成运行时对象

不负责：

- 具体交互流程
- AI 请求细节
- commit 拼装细节

#### `options.go`

职责：

- 命令行参数结构定义
- 参数解析
- 参数默认值

建议包含：

- 子命令模式：`ai` / `emoji` / `break` / `checkbox`
- 会话级开关：`--no-ai` / `-N` / `-M`
- API 参数：`--api-key` / `--api-endpoint` / `--api-proxy`
- `--config`
- `--retry`

#### `config.go`

职责：

- 加载 `aiw.toml` / `.aiw.toml`
- 合并默认值和 CLI 覆盖值
- 产出运行时配置对象

建议注意：

- 配置结构分层，不要继续平铺成巨型 struct
- 配置与 session 状态分离

#### `session.go`

职责：

- 定义一次 `cz` 交互会话的状态模型
- 管理流程中不断被填写和修正的数据

这是 `cmd/cz` 的核心对象之一。

#### `flow.go`

职责：

- 编排完整交互流程
- 决定先问什么、什么时候进入 AI、什么时候显示 preview

它是“业务流程层”，不负责具体渲染。

#### `tui.go`

职责：

- 暴露交互接口
- 封装 `SearchList`、`SearchCheckbox`、`CompleteInput`、`PreviewConfirm`

建议这里继续拆成更细文件，但从设计层先以一个逻辑模块讨论。

#### `ai.go`

职责：

- staged diff 收集
- prompt 构造
- OpenAI 兼容调用
- 候选解析与过滤

不负责：

- 最终交互选择
- commit message 直接提交

#### `draft.go`

职责：

- `Draft` 数据模型
- 草稿清洗、规范化、校验

这里承接 AI 和人工输入的中间结构。

#### `message.go`

职责：

- header 组装
- body / breaking / footer 拼装
- emoji 注入
- 长度计算和格式规范化

它应是纯函数型模块，便于测试。

#### `commit.go`

职责：

- 调用 Git 提交
- retry 最近一次 message
- 处理 message 文件落盘

#### `editor.go`

职责：

- 外部编辑器解析与启动
- 临时文件管理
- 读取用户编辑结果

#### `help.go`

职责：

- 帮助文本
- 子命令说明

### 运行时依赖对象

建议不要让每个函数自己去拿环境变量或直接跑 Git，而是显式注入运行时依赖。

建议引入：

```go
type Runtime struct {
    Git      GitClient
    AI       AIClient
    UI       UI
    Editor   EditorRunner
    Env      EnvLoader
    Clock    Clock
}
```

这里不一定一开始就定义 Go interface，但至少要有清晰的依赖承载对象。

设计原则：

- 先用具体类型也可以
- 只有当测试替身或实现切换明确需要时，再引入接口

这符合当前仓库“接口最小化”的约束。

### 配置数据结构

建议把配置结构分层，而不是继续一个 `czConfig` 塞所有字段。

例如：

```go
type Config struct {
    Messages MessagesConfig
    Commit   CommitConfig
    Scope    ScopeConfig
    AI       AIConfig
    UI       UIConfig
    Editor   EditorConfig
}
```

细分建议：

```go
type MessagesConfig struct {
    Type          string
    Scope         string
    CustomScope   string
    Subject       string
    Body          string
    Breaking      string
    FooterPrefix  string
    CustomFooter  string
    Footer        string
    Confirm       string
}

type CommitConfig struct {
    Types              []CommitType
    Emoji              bool
    AllowBreakingTypes []string
    MarkBreakingMode   bool
    MaxHeaderLength    int
    MaxSubjectLength   int
    MinSubjectLength   int
}

type ScopeConfig struct {
    Items              []ScopeItem
    ScopeOverrides     map[string][]ScopeItem
    EnableMultiple     bool
    Separator          string
    AllowCustom        bool
    AllowEmpty         bool
}

type AIConfig struct {
    Enabled        bool
    Model          string
    Candidates     int
    APIKey         string
    APIEndpoint    string
    APIProxy       string
    DiffIgnore     []string
    DebugSource    bool
}

type UIConfig struct {
    ThemeColor string
    PageSize   int
}

type EditorConfig struct {
    Command string
}
```

这样做的好处：

- 配置语义更清楚
- 以后加字段不会继续污染一个大结构
- 单元测试可以按主题构造配置

### 会话状态数据结构

会话状态不要直接复用配置或最终消息结构。

建议定义：

```go
type Session struct {
    Mode       SessionMode
    Config     Config
    Staged     StagedContext
    Draft      Draft
    AIResult   AIResult
    Flags      SessionFlags
}
```

配套子结构：

```go
type SessionMode struct {
    EnableAI        bool
    EnableEmoji     bool
    EnableBreakMark bool
    EnableCheckbox  bool
    Retry           bool
}

type SessionFlags struct {
    UsedAI         bool
    UsedEditor     bool
    SelectedCustom bool
}

type StagedContext struct {
    Root        string
    Files       []string
    Diff        string
    RecentLog   []string
    IssueRefs   []string
}

type AIResult struct {
    Prompt      string
    Candidates  []DraftCandidate
    Selected    int
    RawResponse string
}
```

作用：

- 配置是静态输入
- Session 是动态过程
- 最终 Draft 是业务结果

### 草稿与候选结构

建议把“最终草稿”和“AI 候选草稿”分开建模。

```go
type Draft struct {
    Type          string
    Scope         []string
    CustomScope   string
    Subject       string
    Body          string
    Breaking      string
    FooterPrefix  string
    Footer        string
    MarkBreaking  bool
}

type DraftCandidate struct {
    Type     string `json:"type"`
    Scope    string `json:"scope"`
    Subject  string `json:"subject"`
    Body     string `json:"body"`
    Breaking string `json:"breaking"`
    Footer   string `json:"footer"`
}
```

设计说明：

- `Draft.Scope` 用 `[]string`，兼容 checkbox 模式
- AI 候选可继续维持单字符串 scope，回填时再标准化
- `FooterPrefix` 和 `Footer` 分开，避免后续处理时反复拆字符串

### 流程层建议 API

`flow.go` 建议提供一个高层入口：

```go
func Run(ctx context.Context, rt Runtime, cfg Config, opts Options) error
```

内部再拆成若干步骤：

```go
func prepareSession(ctx context.Context, rt Runtime, cfg Config, opts Options) (*Session, error)
func runTypeStep(ctx context.Context, s *Session, ui UI) error
func runScopeStep(ctx context.Context, s *Session, ui UI) error
func runAIStep(ctx context.Context, s *Session, ai AIClient, ui UI) error
func runSubjectStep(ctx context.Context, s *Session, ui UI) error
func runBodyStep(ctx context.Context, s *Session, ui UI, editor EditorRunner) error
func runBreakingStep(ctx context.Context, s *Session, ui UI, editor EditorRunner) error
func runFooterStep(ctx context.Context, s *Session, ui UI, editor EditorRunner) error
func runPreviewStep(ctx context.Context, s *Session, ui UI, git GitClient) error
```

好处：

- 每一步都能单测
- 某一步要替换 UI 形式时不会影响其它层
- 非交互降级模式也能复用同一套步骤

### TUI 层建议 API

不建议把 UI 做成很多零散函数，建议先定义统一能力面。

```go
type UI interface {
    SearchList(ctx context.Context, req SearchListRequest) (SearchListResult, error)
    SearchCheckbox(ctx context.Context, req SearchCheckboxRequest) (SearchCheckboxResult, error)
    Input(ctx context.Context, req InputRequest) (InputResult, error)
    ConfirmPreview(ctx context.Context, req PreviewRequest) (PreviewDecision, error)
}
```

这不是为了“抽象而抽象”，而是因为：

- TUI 模式和降级线性模式都需要实现同一套交互能力
- 测试时可以用 fake UI 直接喂结果

### AI 层建议 API

建议把 prompt 构造和请求发送拆开：

```go
type PromptBuilder struct{}

func (b PromptBuilder) BuildCandidatesPrompt(cfg Config, staged StagedContext) string

type AIClient interface {
    GenerateCandidates(ctx context.Context, req GenerateCandidatesRequest) (GenerateCandidatesResponse, error)
}
```

其中：

```go
type GenerateCandidatesRequest struct {
    Model      string
    Endpoint   string
    APIKey     string
    Prompt     string
    Candidates int
}

type GenerateCandidatesResponse struct {
    Raw        string
    Candidates []DraftCandidate
}
```

这样可以把以下职责独立出来：

- prompt 内容是否合理
- OpenAI 兼容请求是否正确
- 返回解析是否鲁棒

### Message 层建议 API

建议完全做成纯函数：

```go
func BuildMessage(d Draft, cfg Config) string
func BuildHeader(d Draft, cfg Config) string
func NormalizeDraft(d Draft, cfg Config) Draft
func ValidateDraft(d Draft, cfg Config) error
```

理由：

- 最容易测试
- 最不该依赖终端或外部环境
- 后续也能给 `retry`、`--dry-run`、`preview` 共用

### Git 层依赖边界

`cmd/cz` 不应该自己到处执行 `git`。

建议把所需能力压缩为明确的小集合：

```go
type GitClient interface {
    ProjectRoot(ctx context.Context) (string, error)
    StagedFiles(ctx context.Context) ([]string, error)
    StagedDiff(ctx context.Context) (string, error)
    RecentCommits(ctx context.Context, n int) ([]string, error)
    CommitFile(ctx context.Context, path string) error
}
```

这样比“通用执行任何 git 命令”的接口更小，也更容易测试。

### 错误模型

建议把用户取消与真正错误区分开：

```go
var ErrUserAbort = errors.New("user aborted")
```

适用场景：

- AI 候选选择时取消
- preview 阶段取消
- 外部编辑器主动放弃

这样 CLI 层可以决定：

- `ErrUserAbort` 输出简短提示并返回 0
- 其它错误输出到 stderr 并返回非 0

### 测试映射

模块拆完以后，测试也应该按模块分布：

- `options.go`：参数解析测试
- `config.go`：配置合并测试
- `draft.go`：草稿规范化测试
- `message.go`：消息拼装测试
- `ai.go`：候选解析和 prompt 构造测试
- `flow.go`：基于 fake UI 和 fake Git 的流程测试
- `editor.go`：编辑器参数构造测试

### 设计结论

`cmd/cz` 应采用“会话状态 + 流程编排 + 可替换交互层 + 纯函数消息层”的结构。核心原则：

- `command` 负责装配
- `flow` 负责过程
- `session` 负责状态
- `tui` 负责交互
- `ai` 负责候选生成
- `message` 负责最终文本
- `commit` 负责落地提交

## 实施计划设计

### 总体策略

本次改造不建议一次性“边搬结构边改体验边上 TUI”。推荐采用两条线分离的方式推进：

1. 先完成 CLI 结构重组
2. 再在新结构内单独重建 `cmd/cz`

原因：

- 结构迁移和 `cz` 重建各自都有回归风险
- 分开做更容易定位问题
- 先把命令边界拉直，后面的 `cz` 实现会明显更顺

---

### 阶段 A：CLI 结构重组

#### 目标

把根目录命令实现迁到 `cmd/*` 和 `internal/*`，但尽量不改变现有对外行为。

#### 范围

- `main.go`
- `gitcmd.go`
- `task.go`
- `worktree.go`
- `tcc.go`
- `init.go`
- `prompts.go`
- `registry.go`
- `envloader.go`
- `util.go`

#### 产物

- `cmd/git`
- `cmd/wt`
- `cmd/tcc`
- `cmd/task`
- `internal/*`

#### 执行顺序

1. 新建 `cmd/git`、`cmd/wt`、`cmd/tcc`、`cmd/task`、`internal/*` 目录骨架
2. 提取当前公共辅助逻辑到 `internal/*`
3. 将 `wt` 迁到 `cmd/wt`
4. 将 `tcc` 迁到 `cmd/tcc`
5. 将 task-like 命令迁到 `cmd/task`
6. 将 Git 快捷命令迁到 `cmd/git`
7. 修改 `main.go` 仅做一级分发

#### 验证重点

- 原有顶级命令仍可用
- `aiw wt ...` 行为不回退
- `aiw tcc ...` 行为不回退
- `aiw git ...` 行为不回退

#### 风险

- 大量符号迁移后编译错误
- 包引用循环
- 原本根目录共享常量散落后找不到统一归属

---

### 阶段 B：建立 `cmd/cz` 新骨架

#### 目标

在不接入完整 TUI 的前提下，先把现有 `gitcz.go` 的基础能力迁入新结构。

#### 范围

- 从根目录抽离 `gitcz.go`
- 新建 `cmd/cz`
- 定义 `options/config/session/flow/message/commit/editor/ai`

#### 执行顺序

1. 新建 `cmd/cz` 目录及文件骨架
2. 把现有 `cz` 的数据结构迁移成新模块
3. 把配置读取迁移到 `config.go`
4. 把编辑器逻辑迁移到 `editor.go`
5. 把消息组装迁移到 `message.go`
6. 把 OpenAI 兼容调用迁移到 `ai.go`
7. 在 `command.go` 中接入最小可运行流程
8. `main.go` 增加 `cz` 顶级命令

#### 这一阶段的策略

- 先保留“能用”的线性流程
- 不保留 `aiw git cz`
- 先保证 `aiw cz` 可以独立跑通

#### 验证重点

- `aiw cz` 能跑通基础交互
- `--llm` 或等价 AI 开关能生成候选
- 外部编辑器仍可用
- commit message 拼装结果不回退

#### 风险

- 从根目录迁走后，旧测试可能全部失效
- `cz` 对 `.env`、Git、编辑器的依赖边界不清时容易再耦回去

---

### 阶段 C：TUI 第一版

#### 目标

实现接近参考项目的关键交互卖点。

#### 必做功能

- `SearchList`
- `SearchCheckbox`
- `CompleteInput`
- `PreviewConfirm`
- AI 候选选择
- subject 长度提示

#### 执行顺序

1. 选定 Go TUI 库并完成最小终端事件验证
2. 封装 `SearchList`
3. 封装 `SearchCheckbox`
4. 封装 `CompleteInput`
5. 封装 `PreviewConfirm`
6. 用 `flow.go` 替换线性问答流程
7. 增加非交互终端降级策略

#### 验证重点

- 搜索过滤是否顺滑
- 多选状态是否在过滤后保持
- 自定义项是否能正确进入输入分支
- AI 候选选择后能否回填到 Draft
- preview 阶段编辑/取消/提交是否闭环

#### 风险

- Windows 终端兼容性
- 屏幕刷新闪烁
- 输入法或特殊按键处理

---

### 阶段 D：`cz` 能力补齐

#### 目标

把第一版 `cmd/cz` 从“能用”提升到“够像、够顺手”。

#### 补齐项

- issue prefix 流程
- breaking 模式增强
- retry 最近一次提交消息
- 更丰富的 `aiw.toml` 配置项
- 更细的校验与提示
- help 文案与 README 更新

#### 验证重点

- 多种配置组合下行为稳定
- 不同 type/scope/breaking 组合输出正确
- AI 候选噪声解析鲁棒

---

### 阶段 E：测试与收尾

#### 目标

补齐测试、文档、命令帮助和回归验证。

#### 范围

- `cmd/cz/*_test.go`
- `cmd/task/*_test.go`
- `cmd/tcc/*_test.go`
- `internal/*_test.go`
- README / 文档

#### 执行顺序

1. 先补纯函数模块测试
2. 再补流程测试
3. 再补命令级测试
4. 更新 README 和帮助输出

#### 验证重点

- `go test ./...`
- `go build ./...`
- 关键命令人工回归

---

### 迁移映射清单

#### 根目录到新目录的初步映射

- `main.go` -> 保留根目录，仅做分发
- `gitcmd.go` -> `cmd/git/*`
- `gitcz.go` -> `cmd/cz/*`
- `task.go` -> `cmd/task/*`
- `worktree.go` -> `cmd/wt/*`
- `tcc.go` -> `cmd/tcc/*`
- `init.go` -> `cmd/task/init.go`
- `prompts.go` -> `cmd/task/prompts.go`
- `registry.go` -> `cmd/task/registry.go`
- `envloader.go` -> `internal/envx/*`
- `util.go` -> 按职责拆到 `internal/fsx` / `internal/textx` / `internal/cmdx`

### 建议的实际落地顺序

推荐按以下顺序执行实现：

1. 阶段 A
2. 阶段 B
3. 阶段 C
4. 阶段 D
5. 阶段 E

其中真正的关键里程碑有三个：

1. `main -> cmd/* -> internal/*` 结构跑通
2. `aiw cz` 从新结构独立跑通
3. `cz` TUI 达到参考项目的核心体验

### 不在本轮实现范围内

以下内容当前不建议纳入首轮：

- 兼容 `cz-git` / `czg` 配置文件
- 执行 JS 配置文件
- 保留 `aiw git cz`
- 抽象通用终端 UI 框架给其它命令复用

### 设计结论

实施上应遵守两个原则：

1. 先重构结构，再重建体验
2. `cz` 的 TUI 卖点优先级高于历史兼容层
