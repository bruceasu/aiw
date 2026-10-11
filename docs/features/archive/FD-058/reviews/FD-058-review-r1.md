# FD-058 独立审查 r1

<!-- aiw-data: FD-058-review-r1.json -->

**FD：** FD-058  
**来源事件：** `FD-058-000008-implementation-ready`  
**审查会话：** `reviewer-fd058-r2-20261011`  
**审查版本：** `feature/FD-058` HEAD `a8c1200`；重点核对 `756cc24` 引入的发现代码与规格，以及 `71514a9` 的范围收敛提交。

## 发现

### 必须修正：搜索根清单支持仍留在实现与稳定规格中

`src/internal/plugin/discover.go` 的 `DiscoverPluginInfoIn` 仍会在每个搜索根读取 `plugin.toml`（约第 103–119 行），并在根清单存在时跳过该根的平铺旧式插件扫描（约第 150 行）。`src/internal/plugin/manifest.go` 的 `ListPluginsIn` 也仍读取搜索根清单并列出其插件（约第 161–181 行），还用 `hasRootManifest` 改变根目录旧规则扫描（约第 219 行）。这些代码来自本 FD 的 `756cc24`，并未由最终范围收敛提交移除。

此外，`openspec/specs/cli-and-plugins/spec.md` 仍声明搜索根可以放置 `plugin.toml`，并包含对应场景（约第 234、238–250 行）。这与 Worker r2 报告中“回退了搜索根清单发现/列表逻辑与 OpenSpec 规格改动”的陈述不一致，也超出当前 FD 仅为七个插件子目录补充清单的范围。

请移除本 FD 引入的搜索根清单支持和对应规格改动，同时保留七个正式插件子目录清单及无根清单时的平铺旧式发现行为；随后更新实现报告并重新交 Reviewer。

## 已核对项与限制

- 七个正式子目录各有一个 `plugin.toml`；名称与目录对应，均填写非空说明和多行 `help`，均省略 `entrypoint` 与 `startup`。
- 根目录 `src/plugins/plugin.toml` 已删除；平铺插件文件没有被新增清单覆盖。
- Worker 报告准确说明了编译命令因内存/分页文件不足失败，且没有将其表述为通过。
- 未运行测试、编译、runtime、TOML 解析器、格式化、lint 或验证脚本。静态审查命令仅读取 Git 差异、文件和目录内容。
- 清单解析与插件调用仍未运行时验证；本审查未尝试弥补该证据缺口。

## 结论

**请求修改（changes-requested）。** 搜索根清单支持与当前明确范围冲突，且 Worker 报告称该逻辑已回退但代码和规格仍保留。
