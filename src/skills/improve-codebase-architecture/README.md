# improve-codebase-architecture：寻找架构改善机会

## 用途与适用场景

现有系统职责分散、接口负担重或难以理解时，先探索架构摩擦，列出可以深入讨论的
模块改善候选，再由用户选择方向。

## 输入与输出

- 输入：代码范围、领域词汇、已有架构决定和关注点。
- 输出：候选说明与自包含 HTML 报告，随后对选中方向进行设计讨论。
- 当前项目报告路径遵守工作区覆盖规则：`.ai/tmp/architecture-review-<timestamp>.html`。

## 调用示例

```text
$improve-codebase-architecture Find improvement opportunities in this package.
```

## 边界与资料

探索不等于实施。领域文档修改、进一步子 Agent 设计和运行操作仍遵守各自范围与
授权，不因为生成报告而获得重构权限。报告路径以入口中的工作区覆盖规则优先，
不要照搬其中保留的历史 OS 临时目录表述。见 [SKILL.md](SKILL.md)。
