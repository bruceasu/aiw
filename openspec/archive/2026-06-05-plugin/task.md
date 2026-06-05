---
title: plugin
status: draft
authors: []
---

# Task: plugin 子命令扩展支持

## 概要
为 `aiw` 增加插件（plugin）扩展机制，使用户可以通过 `aiw <plugin-name> [args...]` 调用外部子命令。当内置命令不存在时，自动在约定位置搜索并执行对应的插件程序或脚本。

## 目标
- 实现插件发现（discovery）与加载（launch）逻辑
- 支持多个平台（Windows / Linux / macOS）的可执行文件与脚本
- 注入必要的环境变量，将 `aiw` 的运行上下文传递给插件
- 明确搜索路径、命名规则与执行优先级

## 非目标
- 不在本任务中实现插件的权限沙箱或复杂的安全隔离（但在风险中列出应对建议）

## 要求

1. 触发条件
   - 当用户运行 `aiw <subcommand> ...`，且 `<subcommand>` 不是内置命令时，作为回退行为触发插件查找与执行。

2. 搜索空间（按优先级）
   1. 当前程序目录下的 `plugins` 子目录（即可执行 `aiw` 的同目录下的 `plugins`）
   2. `$HOME/.config/aiw/plugins`
   3. 系统 `$PATH`（仅限文件，不递归目录）

3. 发现规则
   - 在 `plugins` 目录中：
     - 若项为文件，基础文件名为 `aiw-<plugin-name>`；扩展名允许：`.exe`, `.py`, `.sh`, `.bat`, `.cmd`, `.ps1`, `.js` 或无扩展（Linux ELF 或带 shebang 的脚本）。
     - 若项为目录，则在下一层目录中继续查找满足上述命名规则的文件。
   - 在 `$PATH` 中：仅考虑文件，基础文件名为 `aiw-<plugin-name>`，扩展名同上。

4. 执行/解析器映射
   - 扩展名与执行器：
     - `.exe` / 无扩展（且为二进制）由操作系统直接执行。
     - `.sh`：读取 shebang 决定解析器，默认 `bash`。
     - `.py`：由 `python` 解释器执行（系统环境中可用）。
     - `.bat` / `.cmd`：由 `cmd.exe` 执行（Windows）。
     - `.ps1`：由 PowerShell (`pwsh`/`powershell`) 执行（Windows 或 PowerShell 可用平台）。
     - `.js`：由 `bun` 或 `node` 执行（实现方可选择，建议提供用 `.bat`/`.sh` 启动的包装脚本以降低依赖）。
   - 无扩展的文本文件：若第一行为 `#!`（shebang），以 shebang 指定解析器执行。

5. 优先级规则（当同名插件存在多种形式时）
   - 首选：`.bat` / `.cmd` / `.sh`
   - 其次：`.py`
   - 然后：无扩展（文本脚本带 shebang）
   - 最后：原生二进制（ELF / .exe）

6. 环境变量注入（最小集合，供实现参考）
   - `AIW_PLUGIN_NAME`：被调用的插件名称（`<plugin-name>`）
   - `AIW_PLUGIN_PATH`：被执行的插件可执行文件绝对路径
   - `AIW_CMDLINE`：原始命令行（`aiw <plugin-name> ...`）或仅传递参数的序列化表示
   - `AIW_HOME` / `AIW_ROOT`：aiw 可用的根路径或配置目录（如 `$HOME/.config/aiw`）
   - 另外，保留 `PATH`、`HOME` 等原有环境变量供脚本使用

7. 调用行为
   - 以子进程方式启动插件，传递命令行参数和注入环境变量
   - 转发插件的 stdout/stderr 到父进程终端，并将插件退出码作为 `aiw` 的退出码返回（当 plugin 被执行时）

## 验收标准
- 在 `plugins` 目录放置示例脚本/程序，执行 `aiw <plugin-name>` 能被正确发现并运行
- 支持并测试以下情形：
  - `aiw-hello.sh`（带或不带 shebang）
  - `aiw-hello.py`
  - `aiw-hello`（Linux ELF 可执行或脚本带 shebang）
  - Windows 下 `aiw-hello.bat` / `aiw-hello.ps1`
- 当同名插件在多个搜索路径存在时，遵循搜索顺序和扩展优先级
- 插件运行期间能接收到 `AIW_PLUGIN_*` 环境变量

## 实现计划（建议分步）
1. 设计一个 `internal/plugin` 或 `pkg/plugin` 模块，包含：
   - 路径发现（discover）函数
   - 名称匹配与优先级选择逻辑
   - 可执行检测（包括可执行位、shebang 解析、Windows 可执行扩展识别）
   - 启动/管理子进程的封装（含环境注入、超时/取消控制）
2. 在 CLI 路径解析中加入回退：内置命令不存在时调用 plugin 发现函数
3. 实现跨平台启动器，处理 Windows 与 POSIX 差异（.cmd/.bat/.ps1 vs shebang、执行位）
4. 编写单元测试：发现逻辑、优先级决策、shebang 解析
5. 编写集成测试（简单脚本放在临时 `plugins` 目录），验证 end-to-end 行为
6. 文档：更新 README 或 usage 文档，说明 `plugins` 目录与命名规则

## 测试/验证步骤
1. 在仓库根目录（与 `aiw` 可执行相同目录）创建 `plugins`，添加示例：
   - `aiw-hello.sh`（打印接收到的环境变量）
   - `aiw-hello.py`（打印环境变量）
2. 运行：`aiw hello arg1 arg2`，验证打印并退出码
3. 在 `$HOME/.config/aiw/plugins` 添加同名插件，验证优先级（本地 `plugins` 优先）
4. 在 `$PATH` 放置 `aiw-hello` 可执行，验证被发现并按优先级选中

## 风险与缓解
- 执行任意插件存在安全风险：建议在文档中提醒用户仅放置可信脚本，并考虑未来加入白名单/签名机制
- 跨平台行为差异（PowerShell、cmd、shebang 与可执行位）：通过单元测试、CI 上的 Windows/Linux 测试矩阵缓解
- 依赖外部解释器（python/node/bun）：提供包装脚本或在文档中列出可选依赖，并优雅处理找不到解释器的错误

## 受影响文件（建议）
- `main.go`：在命令解析回退点调用 plugin 发现与执行
- 新增：`internal/plugin/discover.go`、`internal/plugin/exec.go`、`internal/plugin/exec_test.go`
- 文档：`README.md` 或 `docs/usage/*`

## 验证命令示例
```bash
aiw hello world
```

---

%% Notes: 基于 openspec/changes/plugin/notes.md 摘要并形成实现任务。
# Goal
Describe the goal.
# Scope
Included:
-
Out of scope:
-
# Constraints
- Do not refactor unrelated modules.
- Preserve backward compatibility.
# Context
Relevant modules:
-
# TODO
- [X] implement
- [X] tests
- [X] verification
# Verification
- [X] tests pass
- [X] no unrelated changes
# Notes
%% AI notes go here
