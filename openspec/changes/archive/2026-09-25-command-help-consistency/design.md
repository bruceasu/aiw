## Context

当前 AIW 的公开命令由多个分派器和帮助文本分别维护。workflow 的实际 operation 已包含 routing、failure reporting、delivery、focused test 和 metadata repair 等路径，但帮助与补全没有始终同步；README 也保留了较早的命令摘要。该问题属于 CLI 表面一致性问题，不需要新的运行时抽象。

## Goals / Non-Goals

**Goals:**

- 让帮助输出完整列出当前公开命令、子命令形式和重要选项。
- 让 README 的命令摘要与示例反映实际源码支持的入口。
- 让 Shell 补全至少覆盖所有公开 workflow operation。
- 通过检查避免帮助、补全和 README 再次静默落后。

**Non-Goals:**

- 不重构命令分派器。
- 不改变命令名称、参数解析、执行流程或错误语义。
- 不为内部函数或仅供测试使用的路径创建新的公开命令。
- 不引入文档生成依赖或网络步骤。

## Decisions

1. **源码行为是唯一真相。** 实现时以顶层/Task/workflow 分派、参数校验和补全定义为事实来源，逐项核对帮助与 README；不能根据旧 README 推断命令是否存在。

2. **保持现有帮助结构。** 在现有分类和命令别名下补齐内容，保留 `aiw workflow` 与 `aiw task workflow` 两种入口，不引入新的帮助框架。

3. **至少覆盖公开 operation 和选项。** workflow 帮助必须包含所有用户可直接调用的 operation，包括 `delivery-failed`、`report`、`local-merge`、focused-test、metadata repair 等；`supervise` 必须说明启动时支持的 provider/model 覆盖选项。

4. **补全与文档同步核对。** 补全列表必须覆盖公开 workflow operation；README 的摘要和示例必须使用真实入口与真实参数。对仅用于内部恢复或自动流程的命令，若仍对 CLI 公开，也必须明确决定是记录、隐藏还是收回；本 change 默认按现有可调用入口记录。

5. **设计准备度：FD_NOT_REQUIRED。** 这是窄范围、可逆的帮助和文档一致性修复，源码边界、兼容性策略和验证 seam 已明确，不存在需要独立 Feature Design 的未决架构选择。

## Risks / Trade-offs

- [新增命令再次遗漏文档] → 增加命令集合与帮助/补全/README 的一致性检查，并在 checklist 中要求覆盖所有入口。
- [README 示例与参数校验不一致] → 对示例中的 operation、参数顺序和选项做静态逐项核对；不新增未经源码支持的示例。
- [内部命令是否应公开存在边界] → 本 change 不改变公开性；对现有可通过 CLI 分派的 operation 按公开命令处理，并在实现中保留备注供后续产品决策。

## Migration Plan

无需迁移。修改帮助、补全、README 和检查后即可随 CLI 一起发布；回滚只需恢复这些文档/帮助变更，不涉及运行时数据。

## Open Questions

无。
