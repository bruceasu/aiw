# Requirement Artifact

## Metadata
- Artifact Type: Engineering Options
- Requirement ID: aiw-ask
- Stage: engineering-options
- Status: CONSTRAINED
- Based On: Problem Brief / Business Case
- Human Approval Required: yes

## Facts
- `aiw ask`、Chat、Session、Markdown memory、system prompt、权限边界和只读约束属于 AIW 核心。
- LLM 已在核心实现，本需求直接复用现有 LLM，不另行提取或重构 provider 接口。
- Plugin 仅作为未来扩展边界，例如特定外部工具集成或问答分析；首期不实现这些扩展。
- Chat UX 首期保持简洁：多行输入、常用快捷键提示和 `/exit`；不支持复杂编辑或工具调用。
- 默认只允许读取当前目录及其子目录；仅在问题与文件内容相关时读取。
- 目录外读取可由命令行路径授权和对话内确认共同控制。
- `--resume` 无匹配 session 时直接创建新 session。
- `Ctrl+C` 用于取消当前输入或操作；Windows 下不依赖 `Ctrl+D`。
- 每周直接分析 Markdown 文件，不增加自动整理工具。

## Assumptions
- 现有核心 LLM 调用入口可以被 `ask` 命令安全复用。
- 终端层可将结构化回答渲染为友好文本。

## Options
| Option | Benefits | Constraints | Dependencies | Risk | Open Decision |
|---|---|---|---|---|---|
| 核心实现 ask 与 Session | 开箱即用，统一权限和个人 memory | 增加核心 CLI 与本地持久化代码 | 现有 LLM、配置、HOME 路径 | 核心边界扩大 | 在正式设计中确定模块位置 |
| 未来插件扩展 provider/分析 | 保留演化空间 | 首期不实现、不依赖插件才能使用 | 插件机制 | 过早抽象 | 后续根据问答数据决定 |
| JSON 规范数据 + Markdown 持久化 + rendered 展示 | 可分析且用户友好 | 需维护 schema 版本和渲染器 | JSON 序列化、Markdown writer | LLM 输出不符合 schema | 设计阶段确定校验/修复策略 |

## Structured Answer Contract
规范回答使用 JSON，至少包含：

```json
{
  "schema_version": "1.0",
  "status": "ok|needs_clarification|unsupported|error|interrupted",
  "summary": "...",
  "interpretation": "...",
  "steps": [{
    "order": 1,
    "title": "...",
    "description": "...",
    "command": "...",
    "working_directory": ".",
    "read_only": true,
    "expected_result": "..."
  }],
  "alternatives": [],
  "warnings": [],
  "references": [],
  "follow_up_questions": [],
  "capability": {
    "supported": true,
    "unsupported_reason": "",
    "plugin_candidate": false
  },
  "safety": {
    "modifies_files": false,
    "reads_paths": ["."],
    "requires_authorization": false
  }
}
```

- `status=unsupported` 时应提供 `alternatives`，并说明是否可能成为插件候选。
- `error` 或 `interrupted` 时只保存状态、问题和错误提示，不生成 answer 内容。
- JSON 是规范数据；Markdown 中可同时保存 JSON 和由其渲染的 `rendered` 文本。
- `schema_version` 初始为 `1.0`，后续通过版本升级保持兼容。

## Boundary and Data Considerations
- `ask` 只能回答和读取，不得调用文件写入、任务变更或其他副作用工具。
- 记录写入 `$HOME/.aiw/ask/<date>/`，文件名为 `<datetime>-<sha256(prompt)>.md`，不进入 Git。
- 每轮记录问题、时间和结构化回答；失败/中断不写 answer。
- Session 可以跨日期目录继续。

## Permission and Audit Considerations
- 默认上下文范围为当前目录及子目录，且只读取与问题相关的内容。
- 目录外路径必须有显式命令行授权和/或对话确认。
- %% 外部 Codex/Copilot 的授权传递协议尚未确认。

## Failure and Operational Considerations
- LLM 失败或中断：非 Chat 输出友好提示；Chat 输出提示并保持会话可退出。
- `/exit` 是稳定的 Chat 退出方式；`Ctrl+C` 取消当前输入或操作。
- `--resume` 找不到 session 时自动开启新 session。
- %% 需确定 JSON schema 校验失败时的重试或降级策略。

## Open Questions
- %% 现有核心 LLM 调用入口、system prompt 配置格式和路径授权参数名称待正式设计确认。
- %% 需确定每周人工汇总的最小输出格式。

## Suggested Next Stage
- Requirement Management 完成 Requirement Plan 后，由用户决定 approve/defer/reject；批准后再 promote 为一个 AIW Task 并进入 OpenSpec 设计。
