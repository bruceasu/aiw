# FD-056 Worker 修复报告 r2

<!-- aiw-data: FD-056-implementation-r2.json -->

**FD：** FD-056  
**来源事件：** `FD-056-000006-changes-requested`  
**修复范围：** 首轮独立审查发现的 P2 错误传播问题

## 变更

自然语言帮助搜索现在会检查 `listPlugins` 的返回错误。`searchDocs` 返回 `(matches, error)`，`searchAndAnswer` 将插件清单错误包装为 `search docs` 错误并交给帮助命令的既有错误呈现路径。无效的 `plugin.toml` 不会再被当成空插件列表或“没有匹配结果”。

FD-056 的 TODO 和 Verification 已更新，记录该修复和验证结果。

## 验证

- 静态检查：阅读帮助搜索调用链，确认 `listPlugins` 错误会沿 `searchDocs`、`searchAndAnswer` 返回调用者。
- 编译：`$env:GOPROXY='off'; $env:GOTOOLCHAIN='local'; python scripts/compile.py`，退出码 `0`。
- 未运行测试、格式化、lint 或最终构建。

## 剩余风险

没有运行时测试覆盖无效清单的自然语言帮助路径；等待第二轮独立审查。
