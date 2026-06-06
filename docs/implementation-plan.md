# aiw CLI 重构与 `cz` 重建 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将当前根目录命令实现重构为 `main -> cmd/* -> internal/*` 结构，并在新结构下重建顶级 `aiw cz` 命令，为后续高还原度 TUI 和 AI 体验打下稳定基础。

**Architecture:** 先做结构重组，把 `git`、`wt`、`tcc`、openspec-like 命令迁到 `cmd/*`，把通用能力迁到 `internal/*`；再建立 `cmd/cz` 的模块化骨架，用线性流程先跑通 `aiw cz`，最后引入高还原度 TUI 组件替换旧交互流。

**Tech Stack:** Go CLI、标准库、Git 子进程调用、TOML 解析（现有轻量解析逻辑）、后续 Go TUI 库封装、自有 `internal/*` 辅助模块。

---

## 文件结构映射

### 现有文件到目标结构

- 保留并重写职责：`main.go`
- 迁移：`gitcmd.go` -> `cmd/git/*`
- 迁移：`gitcz.go` -> `cmd/cz/*`
- 迁移：`task.go` -> `cmd/task/*`
- 迁移：`worktree.go` -> `cmd/wt/*`
- 迁移：`tcc.go` -> `cmd/tcc/*`
- 迁移：`init.go` -> `cmd/task/init.go`
- 迁移：`prompts.go` -> `cmd/task/prompts.go`
- 迁移：`registry.go` -> `cmd/task/registry.go`
- 迁移：`envloader.go` -> `internal/envx/*`
- 拆分：`util.go` -> `internal/fsx/*` / `internal/textx/*` / `internal/cmdx/*`
- 保留测试并重组：`gitcz_test.go` / `tcc_test.go`

### 预期新增目录

- `cmd/git`
- `cmd/wt`
- `cmd/tcc`
- `cmd/task`
- `cmd/cz`
- `internal/envx`
- `internal/gitx`
- `internal/fsx`
- `internal/textx`
- `internal/cmdx`

---

### Task 1: 搭建 `cmd/*` 与 `internal/*` 骨架

**Files:**
- Create: `cmd/git/command.go`
- Create: `cmd/wt/command.go`
- Create: `cmd/tcc/command.go`
- Create: `cmd/task/command.go`
- Create: `cmd/cz/command.go`
- Create: `internal/envx/doc.go`
- Create: `internal/gitx/doc.go`
- Create: `internal/fsx/doc.go`
- Create: `internal/textx/doc.go`
- Create: `internal/cmdx/doc.go`

- [ ] **Step 1: 创建目录和骨架文件**

使用 `apply_patch` 新增上述目录和最小文件，文件内容先提供包声明和占位注释，例如：

```go
package git

func Dispatch(args []string) error {
    panic("not implemented")
}
```

- [ ] **Step 2: 更新 `main.go` 引入新包但保持旧行为可编译**

在 [main.go](D:/03_projects/suk/aiw/main.go) 中先引入新包别名，例如：

```go
import (
    "fmt"
    "os"

    czcmd "aiw/cmd/cz"
    gitcmd "aiw/cmd/git"
    taskcmd "aiw/cmd/task"
    tcccmd "aiw/cmd/tcc"
    wtcmd "aiw/cmd/wt"
)
```

并临时保留旧 `switch` 逻辑不改动分支调用，只确保新包能被编译引用。

- [ ] **Step 3: 运行编译验证骨架可用**

Run: `go build ./...`  
Expected: 编译通过，若因占位 `panic` 不影响构建则继续。

- [ ] **Step 4: 提交骨架**

```bash
git add main.go cmd internal
git commit -m "refactor: scaffold cmd and internal packages"
```

---

### Task 2: 抽取公共辅助代码到 `internal/*`

**Files:**
- Modify: `envloader.go`
- Modify: `util.go`
- Create: `internal/envx/loader.go`
- Create: `internal/fsx/path.go`
- Create: `internal/textx/text.go`
- Create: `internal/cmdx/commandline.go`

- [ ] **Step 1: 识别 `envloader.go` 的可迁移能力**

把 `.env` 读取逻辑搬到 `internal/envx/loader.go`，导出一个明确 API，例如：

```go
package envx

type Loader struct {
    Env map[string]string
}

func (l *Loader) ParseFile(path string) error
```

- [ ] **Step 2: 把 `util.go` 按职责拆开**

优先只迁移这几类确定可复用的能力：

```go
package fsx

func Exists(path string) bool
```

```go
package textx

func EnsureTrailingNewline(s string) string
func NormalizePipeMultiline(s string) string
```

```go
package cmdx

func SplitCommandLine(s string) ([]string, error)
```

- [ ] **Step 3: 让旧代码先改用 `internal/*`，不立即删除根目录文件**

先在现有根目录调用方中把直接实现替换成 `internal/*` 调用，降低后续迁移难度。

- [ ] **Step 4: 运行定向编译和测试**

Run: `go build ./...`  
Expected: 编译通过。

Run: `go test ./...`  
Expected: 现有测试保持通过，若失败则记录失败点并修复导入。

- [ ] **Step 5: 提交公共能力迁移**

```bash
git add envloader.go util.go internal
git commit -m "refactor: move shared helpers into internal packages"
```

---

### Task 3: 迁移 `wt` 到 `cmd/wt`

**Files:**
- Modify: `worktree.go`
- Modify: `main.go`
- Create: `cmd/wt/add.go`
- Create: `cmd/wt/remove.go`
- Create: `cmd/wt/list.go`
- Create: `cmd/wt/lock.go`
- Create: `cmd/wt/repair.go`
- Create: `cmd/wt/command.go`

- [ ] **Step 1: 将 `worktree.go` 中分发逻辑移入 `cmd/wt/command.go`**

目标 API：

```go
package wt

func Dispatch(args []string) error
```

- [ ] **Step 2: 按子命令拆分实现**

把 `add/rm/list/prune/lock/unlock/repair/ignore` 按职责拆到对应文件，避免继续保留一个超大文件。

- [ ] **Step 3: 修改 `main.go` 的 `wt` 分支**

替换为：

```go
case "wt":
    err = wtcmd.Dispatch(os.Args[2:])
```

- [ ] **Step 4: 运行定向验证**

Run: `go build ./...`  
Expected: 编译通过。

Run: `go test ./...`  
Expected: 不引入回归。

- [ ] **Step 5: 提交 `wt` 迁移**

```bash
git add main.go worktree.go cmd/wt
git commit -m "refactor: move worktree commands into cmd/wt"
```

---

### Task 4: 迁移 `tcc` 到 `cmd/tcc`

**Files:**
- Modify: `tcc.go`
- Modify: `tcc_test.go`
- Modify: `main.go`
- Create: `cmd/tcc/command.go`
- Create: `cmd/tcc/build.go`
- Create: `cmd/tcc/help.go`

- [ ] **Step 1: 抽出 `Dispatch` 入口到 `cmd/tcc/command.go`**

目标 API：

```go
package tcc

func Dispatch(args []string) error
```

- [ ] **Step 2: 拆分构建和帮助逻辑**

将参数组装和帮助输出分别放到 `build.go` 和 `help.go`，保证 `command.go` 只负责分发。

- [ ] **Step 3: 更新测试包路径**

把 [tcc_test.go](D:/03_projects/suk/aiw/tcc_test.go) 对应迁移为 `cmd/tcc/*_test.go` 或调整为调用 `tcccmd.Dispatch`。

- [ ] **Step 4: 更新 `main.go` 分支**

```go
case "tcc":
    err = tcccmd.Dispatch(os.Args[2:])
```

- [ ] **Step 5: 运行定向验证**

Run: `go test ./...`  
Expected: TCC 现有测试继续通过。

- [ ] **Step 6: 提交 `tcc` 迁移**

```bash
git add main.go tcc.go tcc_test.go cmd/tcc
git commit -m "refactor: move tcc wrapper into cmd/tcc"
```

---

### Task 5: 迁移 openspec-like 命令到 `cmd/task`

**Files:**
- Modify: `task.go`
- Modify: `init.go`
- Modify: `prompts.go`
- Modify: `registry.go`
- Modify: `main.go`
- Create: `cmd/task/command.go`
- Create: `cmd/task/init.go`
- Create: `cmd/task/task.go`
- Create: `cmd/task/status.go`
- Create: `cmd/task/archive.go`
- Create: `cmd/task/context.go`
- Create: `cmd/task/decision.go`
- Create: `cmd/task/spec.go`
- Create: `cmd/task/prompts.go`
- Create: `cmd/task/registry.go`

- [ ] **Step 1: 设计 `cmd/task` 顶层分发入口**

目标 API：

```go
package task

func DispatchTopLevel(name string, args []string) error
```

让顶级命令 `init/new/list/show/status/done/archive/context/decision/spec/registry/prompts` 都走同一入口。

- [ ] **Step 2: 拆分任务工作流逻辑**

把根目录不同文件里的实现按职责落到 `cmd/task/*`，并把共享常量留在一个统一位置，例如：

```go
const (
    openspecDir  = "openspec"
    changesDir   = "openspec/changes"
)
```

- [ ] **Step 3: 更新 `main.go` 的顶级命令分发**

例如：

```go
case "init", "new", "list", "show", "status", "done", "archive", "context", "decision", "spec", "registry", "prompts":
    err = taskcmd.DispatchTopLevel(os.Args[1], os.Args[2:])
```

- [ ] **Step 4: 运行编译验证**

Run: `go build ./...`  
Expected: 顶级任务命令仍可编译。

- [ ] **Step 5: 人工抽样回归**

Run: `go test ./...`  
Expected: 无新增回归。

人工检查命令帮助：
- `aiw init`
- `aiw new demo-task`
- `aiw prompts`

- [ ] **Step 6: 提交 `task` 迁移**

```bash
git add main.go task.go init.go prompts.go registry.go cmd/task
git commit -m "refactor: move task workflow commands into cmd/task"
```

---

### Task 6: 迁移 Git 快捷命令到 `cmd/git`

**Files:**
- Modify: `gitcmd.go`
- Modify: `main.go`
- Create: `cmd/git/command.go`
- Create: `cmd/git/help.go`
- Create: `cmd/git/snapshot.go`
- Create: `cmd/git/history.go`
- Create: `cmd/git/remote.go`
- Create: `cmd/git/branch.go`
- Create: `cmd/git/file.go`
- Create: `cmd/git/recovery.go`

- [ ] **Step 1: 建立 `cmd/git` 分发入口**

目标 API：

```go
package git

func Dispatch(args []string) error
```

- [ ] **Step 2: 按命令组拆分 [gitcmd.go](D:/03_projects/suk/aiw/gitcmd.go)**

按现有帮助分组拆到各文件：

- Snapshot & Commit
- History & Status
- Sync & Remote
- Branch
- File
- Recovery
- Guides

- [ ] **Step 3: 明确删除 `git cz` 挂接**

迁移过程中删除 `git cz` 入口，不做兼容壳。

- [ ] **Step 4: 更新 `main.go`**

```go
case "git":
    err = gitcmd.Dispatch(os.Args[2:])
case "cz":
    err = czcmd.Dispatch(os.Args[2:])
```

- [ ] **Step 5: 运行定向编译**

Run: `go build ./...`  
Expected: Git 快捷命令仍可编译。

- [ ] **Step 6: 提交 `git` 迁移**

```bash
git add main.go gitcmd.go cmd/git
git commit -m "refactor: move git shortcuts into cmd/git"
```

---

### Task 7: 建立 `cmd/cz` 最小可运行骨架

**Files:**
- Create: `cmd/cz/command.go`
- Create: `cmd/cz/options.go`
- Create: `cmd/cz/config.go`
- Create: `cmd/cz/session.go`
- Create: `cmd/cz/flow.go`
- Create: `cmd/cz/draft.go`
- Create: `cmd/cz/message.go`
- Create: `cmd/cz/commit.go`
- Create: `cmd/cz/editor.go`
- Create: `cmd/cz/ai.go`
- Modify: `main.go`

- [ ] **Step 1: 定义 `Options`、`Config`、`Session`、`Draft` 基础结构**

写入最小类型定义，例如：

```go
type Options struct {
    EnableAI    *bool
    EnableEmoji *bool
    Candidates  *int
}
```

```go
type Draft struct {
    Type         string
    Scope        []string
    Subject      string
    Body         string
    Breaking     string
    FooterPrefix string
    Footer       string
    MarkBreaking bool
}
```

- [ ] **Step 2: 把现有 `gitcz.go` 的纯数据逻辑迁入新模块**

优先迁移：

- 默认 type 列表
- 默认 messages
- draft 规范化
- message header / body / footer 组装
- editor 参数拆分

- [ ] **Step 3: 在 `command.go` 中暴露新入口**

```go
package cz

func Dispatch(args []string) error
```

- [ ] **Step 4: 接入 `main.go` 的顶级 `cz`**

```go
case "cz":
    err = czcmd.Dispatch(os.Args[2:])
```

- [ ] **Step 5: 运行编译验证**

Run: `go build ./...`  
Expected: `aiw cz` 命令入口已存在且编译通过。

- [ ] **Step 6: 提交 `cmd/cz` 骨架**

```bash
git add main.go cmd/cz
git commit -m "feat: add top-level cz command skeleton"
```

---

### Task 8: 迁移现有 `gitcz.go` 线性流程到 `cmd/cz`

**Files:**
- Modify: `gitcz.go`
- Modify: `cmd/cz/flow.go`
- Modify: `cmd/cz/config.go`
- Modify: `cmd/cz/editor.go`
- Modify: `cmd/cz/message.go`
- Modify: `cmd/cz/commit.go`
- Modify: `cmd/cz/ai.go`
- Modify: `gitcz_test.go`

- [ ] **Step 1: 迁移配置加载流程**

把以下能力迁入 `cmd/cz/config.go`：

- 默认配置
- 程序目录和项目目录配置查找
- CLI 覆盖配置

- [ ] **Step 2: 迁移线性交互流程**

先在 `flow.go` 保留一个线性版本，例如：

```go
func RunLinear(ctx context.Context, cfg Config, opts Options) error
```

用于在 TUI 完成前保持 `aiw cz` 可用。

- [ ] **Step 3: 迁移 AI 调用与候选解析**

把这些能力迁入 `cmd/cz/ai.go`：

- staged diff 收集
- prompt 构造
- OpenAI 兼容调用
- 候选 JSON / 头部解析
- issue footer 过滤

- [ ] **Step 4: 迁移编辑器与提交逻辑**

把外部编辑器、临时文件和 `git commit -F` 提交流程迁到 `editor.go` 和 `commit.go`。

- [ ] **Step 5: 改写现有测试指向新包**

将 [gitcz_test.go](D:/03_projects/suk/aiw/gitcz_test.go) 中与以下能力相关的测试迁到 `cmd/cz/*_test.go`：

- 参数解析
- LLM 候选解析
- editor 参数构造
- OpenAI 响应提取
- issue ref 过滤

- [ ] **Step 6: 运行定向验证**

Run: `go test ./...`  
Expected: `cmd/cz` 相关现有测试全部通过。

Run: `go build ./...`  
Expected: 顶级 `aiw cz` 可编译。

- [ ] **Step 7: 提交线性版 `cz`**

```bash
git add gitcz.go gitcz_test.go cmd/cz
git commit -m "refactor: move existing cz flow into cmd/cz"
```

---

### Task 9: 引入 TUI 组件并替换线性流程

**Files:**
- Modify: `cmd/cz/tui.go`
- Modify: `cmd/cz/flow.go`
- Modify: `cmd/cz/session.go`
- Modify: `cmd/cz/options.go`
- Test: `cmd/cz/tui_test.go`
- Test: `cmd/cz/flow_test.go`

- [ ] **Step 1: 选定并接入底层 TUI 库**

先在 `cmd/cz/tui.go` 建立统一交互接口：

```go
type UI interface {
    SearchList(ctx context.Context, req SearchListRequest) (SearchListResult, error)
    SearchCheckbox(ctx context.Context, req SearchCheckboxRequest) (SearchCheckboxResult, error)
    Input(ctx context.Context, req InputRequest) (InputResult, error)
    ConfirmPreview(ctx context.Context, req PreviewRequest) (PreviewDecision, error)
}
```

- [ ] **Step 2: 实现 `SearchList`**

要求支持：

- 输入即过滤
- 上下导航
- 回车确认
- 默认项定位
- 分页渲染

- [ ] **Step 3: 实现 `SearchCheckbox`**

要求支持：

- 输入即过滤
- 空格勾选
- 已选状态在过滤后保持
- 自定义项分支

- [ ] **Step 4: 实现 `CompleteInput` 和 `PreviewConfirm`**

其中 `subject` 输入必须具备实时长度反馈。

- [ ] **Step 5: 用 TUI 流程替换 `RunLinear` 主路径**

将 `RunLinear` 保留为降级方案，默认主路径改为 TUI：

```go
func Run(ctx context.Context, rt Runtime, cfg Config, opts Options) error
```

- [ ] **Step 6: 增加流程测试**

至少写出 fake UI 驱动的测试，例如：

```go
func TestFlowSelectsAICandidateAndBuildsDraft(t *testing.T) {
    // fake UI + fake AI + fake Git
}
```

- [ ] **Step 7: 运行验证**

Run: `go test ./...`  
Expected: `cmd/cz` 流程和组件测试通过。

Run: `go build ./...`  
Expected: TUI 版本 `aiw cz` 可编译。

- [ ] **Step 8: 提交 TUI 第一版**

```bash
git add cmd/cz
git commit -m "feat: add interactive tui flow for cz"
```

---

### Task 10: 补齐 `cz` 高级能力与文档

**Files:**
- Modify: `cmd/cz/config.go`
- Modify: `cmd/cz/flow.go`
- Modify: `cmd/cz/help.go`
- Modify: `README.md`
- Modify: `docs/design.md`

- [ ] **Step 1: 补齐 issue prefix、breaking mode、retry**

把这些能力纳入新结构：

- issue prefix 选择
- `MarkBreakingMode`
- 最近一次 message retry

- [ ] **Step 2: 扩展 `aiw.toml` 配置项**

至少覆盖：

- types
- scopes
- multiple scopes
- messages
- AI 相关字段
- editor

- [ ] **Step 3: 更新帮助与 README**

在 [README.md](D:/03_projects/suk/aiw/README.md) 中补充：

- 顶级 `aiw cz`
- 参考项目风格 TUI 说明
- 自有配置格式说明

- [ ] **Step 4: 做最终验证**

Run: `go test ./...`  
Expected: 所有测试通过。

Run: `go build ./...`  
Expected: 构建通过。

- [ ] **Step 5: 提交收尾**

```bash
git add cmd/cz README.md docs/design.md
git commit -m "feat: complete cz rebuild on new cli structure"
```

---

## 验证矩阵

### 仓库级验证

- [ ] `go build ./...`
- [ ] `go test ./...`

### 命令级人工抽样

- [ ] `aiw wt list`
- [ ] `aiw tcc help`
- [ ] `aiw git help`
- [ ] `aiw cz --help`
- [ ] `aiw cz`

### `cz` 专项验证

- [ ] 无 staged changes 时给出清晰错误
- [ ] AI 候选选择可用
- [ ] subject 长度提示可用
- [ ] checkbox scope 可用
- [ ] 外部编辑器可用
- [ ] preview 阶段可以提交、编辑、取消

---

## 风险控制

- 结构迁移期间，不要同时大改行为逻辑
- `cmd/cz` 先让线性流程跑通，再切 TUI 主路径
- `internal/*` 只抽稳定公共能力，避免过早抽象
- 不引入 `aiw git cz` 兼容层
- 不引入第三方依赖，除非 TUI 实现确实需要且已评估风险

## 计划自检

- 已覆盖结构迁移、`cz` 骨架、TUI、能力补齐、测试和文档
- 未使用 “TODO / TBD / 后续补” 之类占位步骤
- 每个阶段都给出了目标文件、顺序和验证命令
