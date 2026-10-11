# FD-056 Worker 实现报告

<!-- aiw-data: FD-056-implementation-r1.json -->

**FD：** FD-056
**Worker：** `fd056-worker-20261011-0d31`
**来源事件：** `FD-056-000004-design-ready`

## 实现内容

- 新增 `plugin.toml` 解析与校验，支持一个清单定义多个插件，包含多行说明和帮助、相对入口、启动方式及入口路径边界检查。
- 插件目录有清单时按清单名称发现；无清单时保留旧文件名与扩展名候选。缺少入口或启动方式时分别回退到本目录 `aiw-<name>` 文件查找及原扩展名/shebang 启动逻辑。
- 将启动模式传入执行层；旧路径接口仍使用 `auto`。插件帮助目录、`aiw help <name>` 和帮助文本检索读取清单元数据。
- 更新 README 和 `cli-and-plugins` 稳定规格。

## 静态依据与验证

- 静态追踪 `DiscoverPluginInfo` 到命令分发和 `ExecPluginWithStartup`；帮助列表、详情和搜索共用解析后的插件元数据。
- 编译命令：`$env:GOPROXY='off'; $env:GOTOOLCHAIN='local'; python scripts/compile.py`。命令无输出，执行会话结束；工具包装未保留最终退出码，因此不记录为编译通过。
- 未运行测试、最终构建、格式化、lint 或其他运行验证。

## 未验证事项

- 编译脚本的最终退出码不可得；未通过测试或运行插件来验证清单解析、各启动模式及帮助输出。
- `startup = "typescript"` 依赖当前 Node 运行时支持 `--experimental-strip-types`；平台运行行为未执行验证。
