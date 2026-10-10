# FD-047 活动与归档 FD ID 冲突修复

<!-- aiw-data: FD-047-implementation.json -->

## 定位

- FD：`FD-047`
- 阶段与角色：Implementation / Worker
- 设计就绪事件：`FD-047-000003-design-ready`
- 状态：实现完成，等待独立 Reviewer

## 实现内容

- `plugins/aiw-fd.py` 的 `resolve_fd()` 现在优先解析唯一活动 FD；没有活动记录时，回退到唯一归档记录。
- 活动记录或归档候选在各自优先级范围内存在歧义时，错误信息列出候选路径，避免任意选择。
- 对所选路径继续执行仓库边界与符号链接检查。
- `docs/usage/aiw-fd.md` 补充重开 FD 的解析优先级、归档回退、歧义处理，以及归档目标已存在时关闭仍会失败的说明。
- 稳定 spec 未定义 ID 解析策略，因此没有修改稳定 spec。

## 检查与验证

- 静态检查：检查了解析器候选分类与选择分支、共享解析器调用路径、选中路径的边界与符号链接检查，以及 CLI 文档和 FD 验收项的一致性。
- 编译型检查通过：`python -B -c "from pathlib import Path; compile(Path('plugins/aiw-fd.py').read_text(encoding='utf-8'), 'plugins/aiw-fd.py', 'exec')"`。
- 未新增或运行测试，也未运行其他运行时验证。

## 剩余事项

- 独立 Reviewer 尚未审查实现。
- FD-022 仍需在本 FD 交付后使用已修复解析器执行此前授权的活动状态修正；其归档历史必须保持不变。
- 重开 FD 的归档目标碰撞处理不属于本次实现，关闭操作仍可能因目标已存在而拒绝覆盖。
