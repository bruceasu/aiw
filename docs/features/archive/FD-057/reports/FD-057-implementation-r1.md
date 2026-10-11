# FD-057 Worker 实施报告

<!-- aiw-data: FD-057-implementation-r1.json -->

**FD：** FD-057  
**源事件：** `FD-057-000004-design-ready`  
**Worker session：** `fd057-worker-20261011-c4e91b`

## 完成内容

- 保留现有 `SystemPrompt(Request)` 接口、动态翻译配置和源文作为不可信数据的安全指令。
- 通用提示词明确要求翻译完整内容，不省略或总结；启用 Simple 时简化表达并保留原意。
- 仅当 Style 为 `document` 时追加 Markdown 文档规则，保护现有结构、代码与技术标识符，翻译链接显示文字和解释性自然语言，并保护现有 `XPROTECT` 六位数字标记。
- 更新 AIW Say 指南，说明 `--simple` 和 `--style document` 的行为与提示词保真限制。

## 验证

- 离线 compile-only：在 `src/` 执行 `$env:GOPROXY='off'; $env:GOSUMDB='off'; go build -o NUL ./cmd/aiw-say`，退出码 0，无输出。未下载依赖，未保留可执行文件。
- 静态检查：完成最终 diff 检查，确认文档规则只由 document 风格拼接，原有配置与源文安全约束仍在，文档与 FD 一致。
- 未运行测试、真实 API 请求、最终构建、formatter、linter、vet 或验证脚本。

## 剩余风险

提示词不能保证模型完全遵从结构和标记保护规则；本次没有真实模型翻译质量证据，也没有程序级 Markdown 或占位符保护机制。
