# FD-078 独立实现审查

<!-- aiw-data: FD-078-implementation-review.json -->

结论：**changes-requested**。Reviewer：`fd078-reviewer-20261018-d2a7c4`；来源事件 `FD-078-000004-implementation-ready`。审查基线 `bc918f8`，代码提交 `fd13b2c`，实现报告提交 `1c0e504`。本次为一次完整静态审查，不代表运行验收。

## 必须修复

### R1 / P2：TXT 保留 Markdown 的转义语法和实体

`src/programs/md-converter/internal/markdown/plain.go` 的 `ast.Text`、`ast.String` 分支直接写入原始字节。普通正文如 `\*literal\* &amp; &#20013;` 因此保留反斜杠和实体源码，而非可读正文 `*literal* & 中`。Goldmark 自带 HTML renderer 的 `renderText` / `renderString` 明确区分 raw/code 与需解析转义的普通文本，说明解析器没有替此处完成该工作。它影响 Markdown、MarkItDown 及 Pandoc 所有经过 `plain()` 的 TXT 路径，违反 Issue 验收 1 及 TXT 正文契约。

最小修复：普通文本按 Markdown 规则解除标点转义并解析数值/名称实体；raw/code 节点保持字面值，避免破坏代码中的反斜杠和实体文本。无需增加无授权运行测试。

### R2 / P2：安装批准未绑定实际完整目标路径

`src/plugins/aiw-file-convert/install.py` 将 `plan["target"]` 经 `expanduser().absolute()` 转为实际目标，但摘要及预览仍使用原始字符串。相同的相对 `target` 在两个工作目录中可产生相同批准摘要，却写入不同目录；`~` 同样依赖执行环境。这不满足 Issue 的具体安装目录授权及设计审查 R2 要求的完整 target 绑定。

最小修复：要求目标为明确绝对路径，拒绝依赖当前目录或 home 的表达；在展示、摘要和执行前使用同一份规范目标快照。若路径解析结果变化，原批准必须失效。文档同步明确该规则。

### R3 / P2：字体加载完成未等于字体成功加载

`src/programs/md-converter/browser.go` 的 readiness 只等待 `document.fonts.ready`，随后检查图片而不检查字体失败状态。字体 ready 在加载流程结束时也会完成；例如目录内损坏的 WOFF2 经预处理嵌为 data URL，可能字体解码失败并回退字体后仍打印成功。`Network.loadingFailed` 不能替代字体解码结果检查。此路径违反 FD Acceptance 的资源失败准确失败要求。

最小修复：在字体 ready 后检查文档 FontFaceSet 的 error 状态，必要时记录 loadingerror，并在失败时拒绝打印。保留当前 Mermaid Promise 和图片 decode 的等待顺序。

## 已检查的边界

审查了 CLI 参数优先级、别名与格式矩阵、二进制 stdout 拒绝、临时输出及 hard-link 无覆盖发布、JSON/诊断、MarkItDown 禁用插件与本地输入、antiword 严格 UTF-8、Pandoc 快照预检与 sandbox、HTML 被动资源内嵌、Go 图片目录边界、CSP 脚本摘要、CDP loopback/请求阻断/异步事件、Mermaid 单一 readiness、便携 ZIP 的批准前无网络、摘要校验、重定向拒绝、解压限额/路径/权限/入口和失败恢复说明。

未发现必须扩展设计范围的其它阻塞项。`third_party/goldmark/_tools`、`_benchmark`、`testutil` 的未使用辅助源码可按程序体积目标删除，属于非阻塞建议。

## 证据与限制

实际命令为定向 `Get-Content`、`Select-Object`、`rg`、`git diff --stat bc918f8..fd13b2c`，以及生命周期 `aiw fd claim` 和 `aiw fd emit --help`。首次误在父目录认领被摘要保护拒绝；切换到正确 FD worktree 后一次修正重试成功，没有接管他人事件或修改 receipt。报告落盘后按来源事件发出 changes-requested。

Worker 报告的 `python src/programs/md-converter/scripts/compile.py` exit 0 与脚本静态行为一致；Reviewer 未重复编译，也不把 Worker 结果称作自身执行结果。未运行测试、转换程序、浏览器、下载安装、最终构建、formatter/lint/vet 或网络请求；不要求以未经授权的测试补足本轮修复。

真实文档/旧 DOC 质量、Mermaid/PDF 视觉和分页、字体及跨平台浏览器、手写 CDP/WebSocket、ZIP 安装和输出事务的运行行为仍未实测。固定版本兼容性与来源许可仍依赖实际发行物；硬链接发布依赖文件系统。上述风险应保留在最终交付说明中。
