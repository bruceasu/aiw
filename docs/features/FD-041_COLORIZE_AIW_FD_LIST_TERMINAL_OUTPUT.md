# FD-041: 为 aiw fd list 提供终端表格与颜色

**Status:** Open
**Revision:** 2
**Priority:** Medium  
**Evidence policy:** Dual

## Problem

`aiw fd list` 当前直接打印 Markdown 索引，终端里显示的是 Markdown 表格语法，阅读体验较差。实际调用还遇到 Windows GBK 控制台无法编码归档标题中的 `U+FEFF`，命令以 `UnicodeEncodeError` 退出。用户希望在支持颜色的终端中为列表列着色，同时重定向和不支持颜色的终端仍得到可读的纯文本。

## Options and decision

| 方案 | 判断 |
| --- | --- |
| 继续打印 Markdown 索引，仅增加 ANSI 颜色 | 不采用。终端仍会显示竖线、分隔行等 Markdown 语法，也不能清理标题中的不可见 BOM。 |
| 从 FD 文档生成终端表格，按终端能力选择性着色 | 采用。列表是派生视图，活动和归档 FD 文档仍是状态来源；普通终端、重定向输出和 `NO_COLOR` 环境可使用纯文本。 |

## Solution

将 `aiw fd list` 改为从活动及归档 FD 文档生成分组表格，列为 FD、状态、优先级和标题。标题仅在输出时清理不可见格式字符，并以可读方式处理当前 stdout 编码不支持的字符；不改 FD 正文或索引文件。只在 stdout 为终端、终端未声明 `TERM=dumb` 且未设置 `NO_COLOR` 时输出 ANSI 颜色。状态和优先级使用稳定、可辨识的颜色；颜色不承载唯一信息，纯文本仍保留完整字段。

## Scope

- 只调整 `aiw fd list` 的终端展示与编码兼容，不改 FD 文件、状态、索引格式或事件流程。
- 列表继续覆盖活动和归档 FD，并按当前状态分组。
- 不新增第三方依赖或 CLI 参数，不引入颜色强制开关。
- 不创建 OpenSpec change。

## Work items

- [ ] 1.1 从活动和归档 FD 文档构建紧凑的纯文本分组表格。 Size: S; Difficulty: Low; Dependencies: none. Completion: `aiw fd list` 展示 FD、状态、优先级、标题列；标题渲染安全清理不可见 BOM，当前 stdout 编码无法表达的字符不会导致命令失败；源文件和 Markdown 索引保持不变。
- [ ] 1.2 按终端能力为状态和优先级列着色。 Size: S; Difficulty: Medium; Dependencies: 1.1. Completion: 支持 ANSI 颜色的交互终端显示颜色；`NO_COLOR`、`TERM=dumb`、stdout 重定向或不支持的终端输出无 ANSI 转义序列，字段文本和排序分组不变。
- [ ] 1.3 更新 CLI 使用说明并记录验证结果。 Size: S; Difficulty: Low; Dependencies: 1.1, 1.2. Completion: 文档说明表格列、颜色含义及纯文本回退；FD Verification 与 TODO 反映实际完成和未运行的检查。

## Acceptance

- 终端列表以易读的列展示 FD 标识、状态、优先级和标题，不输出 Markdown 表格分隔语法。
- 活动与归档 FD 均可列出，状态分组、标题和字段值来自 FD 文档。
- ANSI 色彩只用于状态和优先级的冗余提示，不遮盖字段原文；关闭颜色后信息仍完整。
- `NO_COLOR`、`TERM=dumb`、非 TTY 输出和不支持颜色的终端不含 ANSI 控制码。
- 标题中的 BOM 等不可见格式字符不会污染显示；当前 stdout 编码不能表示的标题字符不会使命令失败。
- 命令只读，不修改 FD 文档、索引、receipts 或归档文件；实现不增加第三方依赖。

## Verification

- 静态检查命令路由、活动/归档 FD 枚举、分组/列输出、终端颜色探测、`NO_COLOR` 回退和编码错误处理。
- 实现后按仓库约束执行一次窄范围 Python compile-only 检查；默认不运行测试、smoke、lint 或完整构建。
- 目前未运行实现验证；设计阶段复现了 `aiw fd list` 在 GBK stdout 遇到 `U+FEFF` 时的编码异常。

## TODO

- [ ] 实现基于 FD 文档的分组终端表格和安全标题渲染。
- [ ] 添加支持终端检测、状态/优先级颜色及纯文本回退。
- [ ] 更新 CLI 文档，完成静态审查与 compile-only 检查。

## Sources

- `plugins/aiw-fd.py`：`list` 当前直接 `print()` UTF-8 Markdown 索引；`all_files()` 提供活动和归档 FD 文档，`status()` / `title()` 提供列表字段。
- `docs/features/FEATURE_INDEX.md`：当前派生索引含有归档标题 BOM；直接打印在 GBK stdout 触发 `UnicodeEncodeError`。
- `docs/usage/aiw-fd.md`：当前 CLI 用法文档。
- Planner handoff: `FD-041-000002-design-requested`; session `fd041-planner-20261008-a17c4d`.
