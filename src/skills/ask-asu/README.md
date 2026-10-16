# ask-asu：只读流程导航

## 用途与适用场景

不清楚当前 Issue／FD 处于哪个阶段、由谁处理、下一步应做什么时使用。它读取实际
状态、交接事件和授权范围，给出路由建议；不是自动执行入口。

## 输入与输出

- 输入：请求、Issue／FD ID、已有证据和阶段限制。
- 输出：当前阶段、下一角色／Skill、建议命令及阻塞原因。

## 调用示例

```text
$ask-asu Explain the current stage and next step for FD-XXX.
```

这是宿主 Skill 调用。需要实际实施时，明确转入 fd-workflow。

## 边界与资料

不修改文件，不执行变更命令，不派发角色，不宣称建议已经完成。
行为合同见 [SKILL.md](SKILL.md)，流程参考见
[development-workflow.md](references/development-workflow.md)。
