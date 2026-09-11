# Requirement Artifact

## Metadata
- Artifact Type: Business Case
- Requirement ID: aiw-ask
- Stage: value-assessment
- Status: VALIDATE_FIRST
- Based On: Problem Brief
- Human Approval Required: yes

## Facts
- AIW 当前处于推广期，许多使用者不熟悉可用功能。
- 查阅 `help` 或 `README.md` 以寻找操作方式较费时间。
- 开放性问题的持续记录有助于发现产品演化方向。
- 计划每周人工汇总分析问答记录。
- Chat UX 是首期必需，但保持简洁：支持多行输入和常用快捷键提示。
- 首期不支持复杂编辑能力，也不支持工具调用。

## Assumptions
- 主要价值是提升决策效率和降低学习成本，而非替代 AIW 命令执行。
- 初期问答分析由维护者人工每周执行。

## Value Evidence
| Dimension | Evidence | Confidence | Notes |
|---|---|---|---|
| 决策效率 | 用户查阅 help/README 较费时 | 中 | 尚无具体分钟数 |
| 用户体验 | 推广期用户需要即时 AI 辅助 | 中 | 适合先做小范围验证 |
| 产品演化 | 每周分析开放问题，发现高频缺口 | 中 | 需要定义分析输出 |

## Scope and Alternatives
### Smallest Useful Scope
- 单轮 `aiw ask`。
- 简洁 `--chat` 与 `--resume` 会话。
- 多行输入和基础快捷键提示。
- system prompt、本地 Markdown 记录和目录访问边界。
- 每周人工汇总分析，不做自动分析。

### Explicitly Deferred
- 复杂文本编辑能力。
- Chat 内工具调用。
- 自动问题聚类、趋势分析或插件自动生成。

## Validation Path
- 在推广期试用若干周，每周统计问题数量、重复问题、无法回答问题及可转化为文档或插件的候选。
- 观察用户完成 AIW 任务前后的查找时间和成功率变化。
- 首期试用周期和参与用户范围在工程/推广阶段确定。

## Decision / Recommendation
- `VALIDATE_FIRST`：价值方向明确，但需要通过推广期问答数据验证使用频率和产品化收益。

## Open Questions
- %% 首期试用周期和参与用户范围待确定。
- %% 需要确定每周人工汇总的最小输出格式。

## Suggested Next Stage
- 进入工程可行性与边界澄清，重点确认现有 LLM 接口、promptui 集成、配置读取、HOME 路径兼容性及只读访问限制。
