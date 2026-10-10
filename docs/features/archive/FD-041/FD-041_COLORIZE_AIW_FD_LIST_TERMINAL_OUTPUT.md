# FD-041: 为 aiw fd list 提供终端表格与颜色

**Status:** Complete
**Revision:** 18
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

将 `aiw fd list` 改为从活动及归档 FD 文档生成分组表格，列为 FD、状态、优先级和标题。标题仅在输出时清理不可见格式字符，并以可读方式处理当前 stdout 编码不支持的字符；不改 FD 正文或索引文件。只在 stdout 为终端、终端未声明 `TERM=dumb`、未设置 `NO_COLOR`，且至少识别到 TERM 前缀、TERM_PROGRAM 或 Windows ANSI 能力标记之一时输出 ANSI 颜色；没有识别到能力信号的终端输出纯文本。状态和优先级使用稳定、可辨识的颜色；颜色不承载唯一信息，纯文本仍保留完整字段。

## Scope

- 只调整 `aiw fd list` 的终端展示与编码兼容，不改 FD 文件、状态、索引格式或事件流程。
- 列表继续覆盖活动和归档 FD，并按当前状态分组。
- 不新增第三方依赖或 CLI 参数，不引入颜色强制开关。
- 不创建 OpenSpec change。

## Work items

- [x] 1.1 从活动和归档 FD 文档构建紧凑的纯文本分组表格。 Size: S; Difficulty: Low; Dependencies: none. Completion: `aiw fd list` 展示 FD、状态、优先级、标题列；标题渲染安全清理不可见 BOM，当前 stdout 编码无法表达的字符不会导致命令失败；源文件和 Markdown 索引保持不变。
- [x] 1.2 按终端能力为状态和优先级列着色。 Size: S; Difficulty: Medium; Dependencies: 1.1. Completion: 支持 ANSI 颜色的交互终端显示颜色；`NO_COLOR`、`TERM=dumb`、stdout 重定向或不支持的终端输出无 ANSI 转义序列，字段文本和排序分组不变。
- [x] 1.3 更新 CLI 使用说明并记录验证结果。 Size: S; Difficulty: Low; Dependencies: 1.1, 1.2. Completion: 文档说明表格列、颜色含义及纯文本回退；FD Verification 与 TODO 反映实际完成和未运行的检查。

## Acceptance

- 终端列表以易读的列展示 FD 标识、状态、优先级和标题，不输出 Markdown 表格分隔语法。
- 活动与归档 FD 均可列出，状态分组、标题和字段值来自 FD 文档。
- ANSI 色彩只用于状态和优先级的冗余提示，不遮盖字段原文；关闭颜色后信息仍完整。
- `NO_COLOR`、`TERM=dumb`、非 TTY 输出和不支持颜色的终端不含 ANSI 控制码。
- 标题中的 BOM 等不可见格式字符不会污染显示；当前 stdout 编码不能表示的标题字符不会使命令失败。
- 命令只读，不修改 FD 文档、索引、receipts 或归档文件；实现不增加第三方依赖。

## Verification

- Reviewer R1 的两项 finding 已落实修复：未知/不支持的终端能力默认关闭 ANSI；列宽按完整转义序列的实际长度补偿。原始 findings 和审查证据保留在 docs/features/reviews/FD-041-review-r1.md。
- Reviewer outcome：changes-requested；source event FD-041-000008-implementation-ready；session fd041-reviewer-20261009-ce272b；审查基线 4a445b27de935b39d45cfb1bd4276258abf33d36，HEAD 4e8bdacb1d9886b495189daed8c82f2800a144d6。
- Worker 修复提交 `2151d091` 收紧颜色能力检测，但 R2 发现任意包含 `256color` 或后缀 `-color` 的未知 TERM 仍会启用 ANSI，详见 docs/features/reviews/FD-041-review-r2.md。最终修复只依据已识别的终端前缀、明确的 TERM_PROGRAM 或 Windows ANSI 环境标记；未知 TERM 使用纯文本。ANSI 列宽按完整转义字符串实际长度计算。
- Worker 报告：docs/features/reports/FD-041-implementation-r2.md；source event FD-041-000009-changes-requested；session fd041-worker-20261009-3c5b1a。
- Reviewer R1/R2 的 changes-requested 均作为历史结果保留；修复后待本周期最后一次独立复审。针对终端能力 fail-closed 修复的 compile-only 已通过。未运行测试、smoke、lint、完整构建或运行时 CLI 验证。
- 设计阶段曾复现 aiw fd list 在 GBK stdout 遇到 U+FEFF 时的编码异常。
- Reviewer R2 结果：changes-requested；source event `FD-041-000012-implementation-ready`；session `fd041-reviewer-r2-20261009-6e42cd`；审查基线 `4a445b27de935b39d45cfb1bd4276258abf33d36`，HEAD `fd70cc64f2e736c62fd86de694f370b0893fb281`。发现未知 TERM 名称匹配 `256color` 或 `-color` 后缀仍可启用 ANSI；报告见 `docs/features/reviews/FD-041-review-r2.md`。ANSI 列宽补偿经静态检查与实际转义序列长度相符。未运行测试、smoke、lint、完整构建、运行时终端验证或 Reviewer compile-only 检查。
- R2 finding 已修复：移除未知 TERM 后缀匹配，只对明确识别的终端前缀、TERM_PROGRAM 和 Windows ANSI 标记启用颜色；未知能力关闭颜色。针对修复运行的 compile-only 检查通过。Worker r3 报告待提交。
- Reviewer R3 outcome：changes-requested；source event `FD-041-000015-implementation-ready`；session `fd041-reviewer-r3-20261009-b7d24c`；审查基线 `4a445b27de935b39d45cfb1bd4276258abf33d36`，HEAD `371f2bfe2d571b1476fdb91f4c2450d0dcfd29ba`。实现静态通过本轮终端能力和 ANSI 列宽审查；finding 是 `docs/usage/aiw-fd.md` 对 TTY 着色条件的说明宽于实现。报告见 `docs/features/reviews/FD-041-review-r3.md`。未运行 Reviewer 测试、build、compile-only 或运行时检查。

- 人工恢复决定：用户明确要求“1. 收紧说明 2. 继续”。PM 据此对三次失败复审的停止门禁作单次 override，允许修正 R3 文档 finding 后进行一次额外独立复审；保留前三次结果与计数，不免除 Reviewer。CLI 文档和 Solution 已对齐能力检测；报告见 `docs/features/reports/FD-041-implementation-r4.md`。
- Reviewer R4 outcome：verification-passed；source event `FD-041-000017-implementation-ready`；独立 session `fd041-reviewer-r4-20261009-8d27af`；审查基线 `4a445b27de935b39d45cfb1bd4276258abf33d36`，HEAD `c0261ba463761dcc4c5a8897313f0e6b9f704e96`。R3 文档 finding 已修正，表格来源、编码安全、ANSI 条件及列宽经实际代码和差异静态审查支持验收；报告见 `docs/features/reviews/FD-041-review-r4.md`。未运行 Reviewer 测试、构建、compile-only 或运行时终端验证；Worker 报告的 compile-only 结果未重复执行。交付与归档仍由 PM 后续执行。

## TODO

- [x] 按用户决定收紧 CLI 和 FD 的着色条件说明，与实际能力检测一致。
- [x] 完成本次人工授权的独立复审。

- [x] 实现基于 FD 文档的分组终端表格和安全标题渲染。
- [x] 添加支持终端检测、状态/优先级颜色及纯文本回退。
- [x] 更新 CLI 文档，完成静态审查与 compile-only 检查。

## Sources

- `plugins/aiw-fd.py`：`list` 当前直接 `print()` UTF-8 Markdown 索引；`all_files()` 提供活动和归档 FD 文档，`status()` / `title()` 提供列表字段。
- `docs/features/FEATURE_INDEX.md`：当前派生索引含有归档标题 BOM；直接打印在 GBK stdout 触发 `UnicodeEncodeError`。
- `docs/usage/aiw-fd.md`：当前 CLI 用法文档。
- Planner handoff: `FD-041-000002-design-requested`; session `fd041-planner-20261008-a17c4d`.

**Completed:** 2026-10-09
