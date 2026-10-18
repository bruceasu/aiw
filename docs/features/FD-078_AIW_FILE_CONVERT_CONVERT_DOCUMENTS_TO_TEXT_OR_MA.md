# FD-078: aiw-file-convert: convert documents to text or Markdown

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

ISSUE-00008 用户需要统一的本地文件转换入口：文档提取为 MD/TXT，以及 MD/HTML 发布为 HTML/PDF。不得假设 Office、LibreOffice、Pandoc、Playwright 或浏览器已安装。

## Options and decision

采用 Python 插件协调 MarkItDown、antiword、可选 Pandoc 和独立 Go 转换器。复用参考程序的 Goldmark/Mermaid 资源，修正输出命名、资源边界和 PDF 完成判断。全部 Python 渲染会增加运行依赖；完整重写 Pandoc 无收益，因此不采用。保持一个 FD，因为统一 CLI、离线及输出保护规则贯穿所有后端；按小型成果拆 Work Items。

使用 codebase-design 的接口职责方法：Python 拥有参数、输出事务、JSON、依赖提示/授权安装；Go 只接收本地 MD/预处理 HTML 并写临时 HTML/PDF。不改变 AIW 核心插件协议。新增独立 Go module，Goldmark v1.7.13 从本地缓存复制源码与许可证，不联网；Mermaid 11.12.1 复制参考资源并保留 MIT 许可。不引入 CDP 库：仅实现本需求的顺序 CDP JSON/WebSocket 通信，不提供公共通用客户端。

## Solution

入口 src/plugins/aiw-file-convert/plugin.toml，startup=python。-h/--help 优先于转换，无需依赖；-i/--input，-o/--output/--ouput，--format（默认 md），--json。输出空值追加完整输入名的目标后缀；显式 -o 缺值报错。格式由 --format 决定，与输出后缀冲突报错。文本可 -o -；pdf/docx/odt/epub 提前拒绝 stdout，JSON 也不例外。

JSON 固定 arguments 对象、result-data（stdout 文本为字符串，否则 null）、content-type 字符串、result-code 字符串、error-message 字符串，文件成功额外 output-file 对象 {path,size}；错误可有 dependency 对象。退出码 0=ok，2=arguments，3=missing-dependency，4=conversion，5=output，6=install。所有诊断 stderr，JSON stdout 单对象。写入同目标目录临时文件，成功以不覆盖的 hard-link 发布；不支持安全发布的文件系统明确失败，保留已有目标；源与目标相同提前拒绝。

转换矩阵：pdf/docx/pptx/xlsx/xls/html/htm -> md/txt 使用 MarkItDown；txt/csv/json/xml/md -> md/txt 使用本地 UTF-8 文本或 Markdown 处理；doc -> md/txt 使用 antiword -m UTF-8.txt，旧 DOC 仅正文；md -> html/pdf、html/htm -> pdf 使用 Go；md -> docx/odt/epub/rst、odt/epub/rst -> md/txt 使用 Pandoc。DOCX -> md/txt 仍优先 MarkItDown。Pandoc 写入路径先由 Go 将 MD 转为离线自包含 HTML，再以 HTML reader 转换，额外依赖 Go；仅承诺正文/表格/本地图片，Mermaid 仍作为代码保留而非矢量图。所有 Pandoc 使用 --sandbox；ODT/EPUB ZIP XML/CSS 预检外部资源，RST 拒绝 include/raw 与远程图片，避免后台联网。不声明其余任意组合。TXT 使用本地 HTMLParser 将 Markdown 渲染后提取段落/列表/表格正文，避免用标点删除法。

Go PDF 用用户指定 AIW_CONVERT_BROWSER 或本机 Edge/Chrome/Chromium，隔离临时 profile，进程有超时并清理。仅 loopback CDP 调试端口，不使用 Playwright。CDP 阻断所有外部 HTTP/HTTPS/file 请求；文档内 CSP 默认拒绝外部资源。MD 禁止原始 HTML，图片必须位于输入目录内（解析符号链接后检查），嵌入 data URL。HTML 由 Python 预处理：拒绝脚本、事件属性、frame/object/embed、base/refresh，图片和 CSS/字体引用仅允许目录内资源并内嵌，远程资源明确报错。可信 Mermaid 内嵌脚本；等 mermaid.run、document.fonts.ready、图片 decode 完成才 Page.printToPDF，失败/超时非零。返回 PDF 魔术头校验后发布。

依赖提示按请求惰性检测，Python 环境不用自动 pip。MarkItDown 建议 0.1.3、Pandoc 建议 3.6.4、antiword 0.37；浏览器复用本机版本，仅要求支持 CDP 所用接口。提示官网和按平台参考命令；版本为兼容目标而非运行质量保证。Go 程序查 AIW_MD_CONVERTER、同包及 src/programs/md-converter/bin，不在转换时自动编译。

授权安装入口 --install-plan <json>：先只显示具体安装清单及其 SHA256 标识，--approve-install <plan-sha256> 才执行。清单必须含 dependency、version、https url、sha256、平台 system/machine、用户指定全新 target；只允许 zip 类型便携包（不支持 wheel 安装；无全局 PATH/提权）。首版实现便携 ZIP 安装：限制为 Pandoc/antiword/browser/Go-helper，需用户自备可信发布清单，不内置未经核验的下载链接；MarkItDown 使用提示的专用 venv/pip 命令，不声称存在安全绿色包。清单采用排序键、紧凑分隔符、ensure_ascii=True 的 JSON UTF-8 SHA256；严格字段/当前OS与CPU匹配。URL 主机白名单（github.com、release-assets.githubusercontent.com、objects.githubusercontent.com、downloads.sourceforge.net、storage.googleapis.com），拒绝所有重定向（用户提供可信直链）、下载哈希、路径穿越/符号链接防护、解压限额、临时目录验证后独占创建目标目录并逐文件独占复制；已有目录一律失败，不覆盖。成功返回目标与配置 AIW_CONVERT_PANDOC/ANTIWORD/BROWSER/AIW_MD_CONVERTER 的明确路径。执行安装是单独模式，不隐式重试转换；用户之后显式重试原命令。无批准、非交互、拒绝和安装失败均不静默安装。当前 FD 不执行任何下载安装。下载官方包的分发许可由用户清单来源保持；不捆绑 antiword/Pandoc/browser 二进制。

## Scope

完整覆盖 Issue 的统一输出契约、基本提取、Go HTML/PDF、有限 Pandoc 和明确授权便携安装。首版支持 Windows/Linux/macOS 的路径/浏览器发现；PDF 运行适配未实测必须如实报告。无 GUI、批量、OCR、云、远程输入、任意互转或完整 Office 排版。新增 docs/usage/aiw-file-convert.md、独立稳定 spec；无 OpenSpec change。

## Work items

- [ ] 1.1 插件参数/JSON/不覆盖输出事务与统一帮助（小/中，无依赖；验收 Issue 6-9）。
- [ ] 1.2 MarkItDown/antiword/Pandoc 惰性适配与 TXT 正文（小/中，依赖1.1；验收 Issue 1-3、11）。
- [ ] 1.3 Go MD/HTML、固定离线资源与许可证（小/中，依赖1.1；验收 Issue 4）。
- [ ] 1.4 HTML 资源预处理与 Go CDP PDF 完成/超时（小/中，依赖1.3；验收 Issue 5，离线边界）。
- [ ] 1.5 依赖指导与清单摘要授权的便携安装（小/中，依赖1.1；验收 Issue 10）。
- [ ] 1.6 文档/spec、静态证据、编译与独立审查（小/低，依赖全部；不执行转换测试）。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

以 ISSUE-00008 全部 11 项验收及上述矩阵为准。输入/输出含中文空格通过参数数组传递；已有输出、二进制 stdout 在依赖调用前拒绝。转换失败清理临时文件、保留输入和已有输出。缺依赖同时提供人类和 JSON 指导；未批准安装不访问网络。HTML/PDF 不加载远程资源、不读取输入目录以外资源。PDF 必须输出真实文件，依赖缺失/资源失败/超时准确失败。

## Verification

- Compile-only check and static review by default; record unrun tests explicitly.
- 仅一次 scoped compile 脚本：Python 源码 compile()（不导入/运行插件），Go module 离线 go build -o NUL。无测试、格式化、lint、最终构建、网络/依赖下载；不重建已安装 AIW 或 converter。独立设计审查覆盖 CLI、输出、资源、安装授权；独立实现审查负责验证结论。视觉质量、真实第三方集成和跨平台 PDF 均为未验证风险。
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

## Sources

- Issue: ISSUE-00008
- openspec/specs/cli-and-plugins/spec.md；D:/03_projects/mermaid-tools/go 的 renderer/images/browser/document 和 Mermaid 资源；本地 Goldmark 缓存。

Link relevant specs and evidence used for decisions.
