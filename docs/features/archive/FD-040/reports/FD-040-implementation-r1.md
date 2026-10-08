# FD-040 实施报告 R1

<!-- aiw-data: FD-040-implementation-r1.json -->

## 完成内容

- 将 `docs/features/` 顶层的 14 个模板文档迁入 `docs/templates/`。
- `aiw fd new` 从 `docs/templates/TEMPLATE.md` 读取模板；模板不存在时在新目录生成内置默认模板。
- 更新 `fd-workflow`、`fd-test`、工作管理约定、FD 用法文档和可移植布局说明中的模板路径。
- 更新 `fd_smoke.py` 与 3 个黑盒测试 fixture 的模板来源和缺失模板场景路径。
- 保持 FD 文件、FEATURE_INDEX、活动/归档报告与审查路径不变；未重写历史报告。

## 验证

- Python 内存 compile-only 命令对 5 个变更 Python 文件调用内置 `compile()`；全部通过，未生成 `.pyc`。
- 静态路径扫描确认活跃代码、技能、文档和 fixture 使用 `docs/templates/` 作为模板目录。
- 未运行测试、smoke、最终构建、lint 或格式化。运行时行为尚未验证。

## 剩余风险

CLI 创建 FD 的运行时行为与测试 fixture 未执行；本报告只记录 compile-only 和静态证据。旧 `docs/features/TEMPLATE.md` 布局不再作为读取回退路径。