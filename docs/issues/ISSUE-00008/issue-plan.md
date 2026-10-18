# Issue Plan：aiw-file-convert 文件格式转换插件

- Issue：`ISSUE-00008`
- 日期：2026-10-18
- 需求提出者与确认者：当前用户；整理者：Codex。
- 当前结论：用户明确要求“把讨论的内容都确认成正式的 issue 内容”。本文为合并后的正式需求来源，取代此前正文与扩展草案中的重复、建议及相互冲突的描述。
- 阶段：Issue 内容确认与 capture；未创建 FD、未开始实现。
- 审批说明：保留历史批准日志和审批摘要；当前用户的整体确认记录在本文。capture 不等于 CLI 重新批准或移交 FD，不能声称历史审批摘要已自动绑定本文。

## 问题、使用者与目标

AIW 使用者需要通过一个符合插件体系的命令，将常见文件提取为文本或 Markdown，并将 Markdown/HTML 导出为可阅读、可打印的 HTML/PDF。目标是统一参数、结果、错误和依赖提示，减少重复使用专用工具的操作。

插件名为 `aiw-file-convert`，入口为 `aiw file-convert`。优先减少运行依赖，支持便携分发，不假设机器安装 MS Office、LibreOffice、Pandoc、Playwright 或任何指定浏览器。源文件不得被修改，已有输出默认不得覆盖。

## 正式范围与后端职责

| 能力 | 已确认方向 | 边界 |
| --- | --- | --- |
| 常见文档 → MD/TXT | Python 插件调用 Microsoft MarkItDown | 按格式检测所需依赖，不将项目支持范围全部变成本插件承诺 |
| 旧版二进制 DOC → 正文文本 | Python 封装或调用 antiword | 不依赖 Office 套件；不承诺恢复复杂排版或结构化 Markdown |
| MD → HTML | 移植参考 Go 转换器 | 完整 HTML、CSS、Mermaid 及本地图片处理 |
| MD → PDF | Go 转换器生成 HTML，再使用浏览器打印 | 等待图表、字体、图片完成，不以仅生成 HTML 代替 PDF 成功 |
| HTML → PDF | Go 转换器扩展本地 HTML 输入 | 保持资源基路径，明确资源失败和超时处理 |
| 额外文件格式转换 | Pandoc 可选后端 | 默认不下载、不捆绑、不要求安装，仅在请求需要它的格式时使用 |

常见文档提取的目标输入：有文本层的 PDF、DOCX、PPTX、XLSX、XLS、HTML/HTM、TXT、CSV、JSON、XML；另以 antiword 支持旧版 DOC。输出正文采用 UTF-8，Markdown 为默认目标。

Pandoc 保留为扩展能力，不要求自研替代其全部格式。DOCX、ODT、EPUB、RST 与 Markdown 的转换是已讨论候选；每个对外声明支持的输入/输出组合都必须在后续 FD 列出后端、依赖和验收边界，不能宣传任意格式互转。尚未选定的专用 Go 库不算已承诺后端。

### Go 转换器与轻依赖要求

- 源码参考：`D:/03_projects/mermaid-tools/go`；移植目标：`src/programs/md-converter`。Python 插件统一调度该程序与其他后端。
- MD → HTML 运行时不需要安装 Go 工具链、Node/npm、Python Markdown 渲染库或 Playwright。必要 CSS 和固定版本 Mermaid JS 随程序提供。
- PDF 优先复用受支持的已安装 Edge、Chrome 或 Chromium；缺失时提供安装指导，获得具体授权后才下载安装。
- 默认不引入 Playwright。Go 中如需 CDP 或其他控制方式，后续 FD 评估具体库及体积；不能用固定延时替代可靠的渲染完成判定。
- PDF 必须等 Mermaid 渲染、字体和图片加载完成；失败、超时、资源缺失要明确报告。
- 继承本插件输出命名和错误契约，不照搬参考程序的“替换扩展名”及“跳过 PDF 仍成功”行为。
- 参考程序现有 Windows AMD64 EXE 为 7,561,216 字节（约 7.21 MiB），内嵌 Mermaid JS 为 2,749,172 字节。仅静态读取现有文件；不保证它对应当前源码，也不保证移植后大小相同。

### 本地与离线边界

- 延续已确认的本地处理方向：不自动上传文档，不隐式访问远程资源或下载依赖。
- 采用此前建议的本地资源方案：生成的 HTML/PDF 默认使用本地固定 Mermaid/CSS，不依赖 CDN 才能渲染图表。
- 本地图片保留正确的相对路径或按设计嵌入；不得任意越过允许的文档资源目录读取文件。
- 输入 HTML 的远程图片、脚本、字体等不能静默联网获取；缺失或被限制的资源需要可见提示。是否另增显式联网模式留待单独需求，本次不承诺。
- 中文显示依赖实际可用字体；不将 UTF-8 编码等同于字体或排版质量已经验证。

## CLI、Usage 与 Examples

- `-h` / `--help`：打印 Usage、Options、Examples；无需输入文件，不加载转换依赖，不执行转换。
- `-i` / `--input <file>`：指定单个本地输入文件。
- `-o` / `--output <file>`：指定输出；保留此前确认的 `--ouput` 兼容别名。
- `--format <format>`：目标格式，默认 `md`；基本目标为 `md`、`txt`、`html`、`pdf`，额外目标以实际声明的 Pandoc 格式组合为准。
- `--json`：输出结构化执行结果；不得与转换目标格式混淆。
- 省略 `-o` 或显式空字符串：完整输入文件名追加目标扩展名，如 `report.docx.md`、`report.docx.txt`、`design.md.html`、`design.md.pdf`。
- 单独 `-o` 缺少值属于参数错误，不能吞掉下一个选项。
- 文本目标允许 `-o -` 输出 stdout；二进制目标必须写文件，详见下节。
- 诊断、进度与正文分离；JSON 模式的 stdout 只能有一个 JSON 对象。
- 插件声明中的帮助和程序自身帮助保持一致，使用 Easy English 描述参数与示例。

```text
Usage:
  aiw file-convert -i <file> [-o <file|->] [--format <format>] [--json]
  aiw file-convert -h | --help

Options:
  -h, --help                Print usage and examples.
  -i, --input <file>        Input file.
  -o, --output <file|->     Output file. Use - for text output to stdout.
      --ouput <file|->     Alias for --output.
                           If empty or omitted, append the target extension
                           to the full input file name.
      --format <format>    Target format. Default: md.
                           Basic formats: md, txt, html, pdf.
      --json               Return a JSON response on stdout.

Examples:
  aiw file-convert --help
  aiw file-convert -i report.docx
  aiw file-convert -i legacy.doc --format txt
  aiw file-convert -i "my report.pdf" -o -
  aiw file-convert -i report.docx -o - --json
  aiw file-convert -i design.md --format html -o design.html
  aiw file-convert -i design.md --format pdf -o design.pdf --json
  aiw file-convert -i page.html --format pdf -o page.pdf
```

## 输出与 JSON 契约

### 文本目标

- 指定输出文件或自动文件名时，文件保存转换正文；`--json` 使 stdout 返回 JSON 结果，并记录最终输出路径，文件本身不变成 JSON。
- `-o -` 不带 `--json`：stdout 直接输出正文。
- `-o - --json`：正文置于 `result-data`，stdout 只输出 JSON，不另打印正文。
- 纯文本转换必须消除生成的 Markdown 语法并保留正文可读性，不得仅改扩展名或删除所有标点。

### 二进制目标

- PDF、DOCX、ODT、EPUB 等二进制结果必须写文件，不能输出 stdout 或 stderr。
- 显式 `-o -` 一律返回参数错误，包括终端、管道、重定向及 `--json` 场景；此错误应在转换、下载或安装之前报告。
- 省略输出仍自动生成追加目标扩展名的文件。
- `--json` 可以返回路径、MIME、大小等文件元数据，不包含二进制正文，也不以 Base64 内嵌正文。

### JSON 字段

保留 `arguments`、`result-data`、`content-type`、`result-code`、`error-message`。`content-type` 描述转换结果；成功代码为 `ok`，失败为具体错误代码。错误时提供错误说明，失败进程退出码非零。字段固定类型、输出路径字段名、文件元数据对象与退出码表在 FD 明确。

```json
{
  "arguments": {"input": "report.docx", "output": "-", "format": "md"},
  "result-data": "# Report\n\nDocument text.\n",
  "content-type": "text/markdown",
  "result-code": "ok",
  "error-message": ""
}
```

## 依赖检测、安装提示与授权

1. 按请求的转换路径检测所需 Python 包、格式 extras、可执行文件和资源；可选依赖缺失不影响无关转换。
2. 未安装时明确依赖名、受影响能力、官网/可信发布页、匹配 OS/CPU 的安装或解压步骤及参考命令。JSON 模式也须提供程序可读的缺失依赖与指导信息。
3. MarkItDown 按需安装格式 extras；antiword 需要匹配平台的程序与资源。Python 封装不等于消除了本地二进制依赖。
4. Pandoc 默认不下载、不捆绑；基本 MD/HTML/PDF 路径不依赖 Pandoc 或 LaTeX。仅在需要额外格式时提示其可选安装。
5. 优先官方绿色压缩包和用户可写安装目录。没有可靠绿色版时如实说明替代方式，不默认提权、全局安装或修改全局 PATH。
6. 自动下载安装是产品能力：必须先展示具体来源、版本、安装目录、命令和副作用，获得用户明确授权后才执行。当前 Issue 确认不是本会话下载或安装授权。
7. 用户拒绝授权、非交互且无授权、下载失败或安装失败时，保持真实失败状态，不能静默安装或报告成功。安装失败不得破坏原有可用依赖。
8. 下载需要匹配平台、固定版本、校验发布物，并处理资源与许可证；安装成功后仅按约定重试本次转换。具体授权入口、安装目录、升级清理和重试机制由 FD 设计。

安装指导的已有来源示例（仅记录，不执行）：

```text
MarkItDown: https://github.com/microsoft/markitdown
  python -m pip install "markitdown[pdf,docx,pptx,xlsx,xls]"
  Select extras for the requested formats and use a dedicated environment.

Pandoc: https://pandoc.org/installing.html
Releases: https://github.com/jgm/pandoc/releases
  Preferred Windows option: download and unzip the matching official ZIP.
  Alternative: winget install --source winget --exact --id JohnMacFarlane.Pandoc
```

%% NEEDS_EVIDENCE: 后续 FD 固定各后端版本、支持平台和可执行安装指导；antiword/浏览器绿色发布物来源与校验方案尚未核实，不得编造官网或下载地址。

## 选型结论与不采用的默认方案

- Python 能实现参考程序同类能力；选 Go 移植是为了减少新增依赖和便携分发，不是 Python 无法实现。
- 默认不增加 Playwright；浏览器依然是 PDF 的运行依赖，程序语言不能消除浏览器渲染需求。
- 不要求用 Go 全量替代 Pandoc。本次检索未找到覆盖其完整能力且已验证更轻的等价替代。
- `html-to-markdown`、`go-word`、`go-epub` 是已讨论的专用 Go 候选，分别面向 HTML → MD、Markdown/DOCX、EPUB 生成；尚未选定，不承诺大小、质量或完整格式覆盖。
- antiword 的正文提取方向已确认。`doc2txt` 只是候选封装，需核查平台、超时、解码行为及内附二进制分发条款；`legacy-doc` 纯 Python 方案仅为比较备选，不纳入首版承诺。
- Pandoc 本体与 LaTeX 套件分开看待；采用浏览器 PDF 路径，不要求安装 LaTeX。

## 验收示例

1. DOCX 的中文、标题、列表、表格正文能输出可读 Markdown；选择 TXT 后保留段落和内容，无生成的 Markdown 语法。
2. 有文本层 PDF、PPTX、Excel、HTML 有各自代表性内容验收，不无提示丢失表格数据。
3. 旧 DOC 在无 MS Office/LibreOffice 环境下通过 antiword 提取正文；后端缺失、乱码或无法处理时准确提示，不伪报成功。
4. MD → HTML 支持表格、代码、Mermaid、中文和本地图片，必要脚本/CSS 本地可用；默认不依赖 CDN。
5. MD → PDF、HTML → PDF 生成真实文件，等待图表、图片和字体完成；缺浏览器、渲染失败或超时返回错误及依赖指导。
6. 省略输出按完整输入名追加目标后缀；含中文、空格路径能定位；已有目标默认拒绝覆盖。
7. 文本 stdout、文本 JSON、文件输出加 JSON 遵守相同结构，正文和诊断不混合。
8. PDF 等二进制 `-o -` 及 `-o - --json` 均提前失败；文件输出加 JSON 只返回元数据，不泄露二进制正文或 Base64。
9. `--help` 展示一致的 Usage/Examples，无需输入或第三方转换依赖。
10. 缺依赖时给官网及参考命令；拒绝授权不下载；授权安装失败返回真实错误。没有 Pandoc 时基本路径仍可使用。
11. 每条后续声明的 Pandoc 额外转换均有明确格式组合、所需依赖及独立验收输入，不将全部 reader/writer 的组合视为支持承诺。

## 非目标

GUI、批量目录、远程输入 URL、压缩包递归、扫描 PDF/图片 OCR、音视频转录、云端转换、高保真 Office 排版复刻、自研 Pandoc 全量替代、默认捆绑浏览器或 LaTeX。本地 HTML/PDF 生成和授权依赖获取属于本次正式范围，不再沿用早期将全部文档生成排除的表述。

## 规模、拆分与后续设计边界

范围中到大，已包含多个可独立验收结果：正文提取、MD/HTML 发布输出、可选 Pandoc 扩展、授权依赖获取。保留一个整体需求来源以统一 CLI/JSON、命名和依赖授权规则；后续拆分建议按上述用户结果，各自明确范围与验收，不按文件或语言分层。

当前未创建子 Issue、不分配子 ID、不触发 FD。原提取范围和后续扩展均保留在本记录中；拆分若执行，须通过 CLI 建立父子关系并避免重复交付。方案确认不能代替 FD 设计就绪。

后续 FD 必须明确：各格式组合及版本/平台、Go 移植边界和许可证、可靠 PDF 就绪检测、资源路径边界、纯文本渲染、字体及分页、具体 CLI 帮助优先级、格式/后缀冲突规则、错误码与 JSON 类型、安装授权和生命周期、临时文件及覆盖保护。

%% NEEDS_EVIDENCE: 尚无代表性旧 DOC、中文、表格、文本框、页眉页脚、修订、Mermaid 或 PDF 视觉实测。本 Issue 记录目标和证据缺口，不宣称兼容性或质量已验证；运行验证仍需仓库规则要求的独立授权。

## 来源与证据

- 本会话用户的原始请求、追加 CLI/JSON/二进制规则、MD/HTML/PDF/Pandoc/依赖安装需求及历次确认；最新整体确认明确授权正式记录全部讨论结论。
- `openspec/specs/cli-and-plugins/spec.md`，`src/plugins/examples/plugin-example.toml`，`src/plugins/aiw-github/plugin.toml`：插件契约与帮助示例。
- `D:/03_projects/mermaid-tools/go/README.md`、`go.mod`、`cmd/md2html/main.go`、`internal/markdown/renderer.go`、`images.go`、`internal/document/document.go`、`internal/browser/browser.go`、`internal/metadata/metadata.go`、`internal/assets/assets.go`：已静态检查的参考方案。
- https://github.com/microsoft/markitdown 和其 `packages/markitdown/src/markitdown/converters/_docx_converter.py`：Markdown 主路径与 DOCX 边界。
- https://github.com/grobian/antiword/blob/main/Docs/FAQ 、https://github.com/grobian/antiword/blob/main/Docs/antiword.1 、https://github.com/grobian/antiword/blob/main/Docs/ReadMe ：UTF-8/历史编码、输出选项与许可证说明。
- https://pypi.org/project/doc2txt/ 和 https://github.com/Quantatirsk/doc2txt-pypi/blob/21bb4d1001187ee40839a35ae68c6803a2be2021/doc2txt/antiword_wrapper.py ：候选封装及已静态读取实现。
- https://pypi.org/project/legacy-doc/ ：纯 Python 备选说明，未审查解析实现。
- https://markdown-it-py.readthedocs.io/en/latest/using.html 、https://playwright.dev/python/docs/api/class-page#page-pdf 、https://playwright.dev/python/docs/browsers 、https://mermaid.js.org/config/usage.html ：Python 可行性及 Mermaid 异步渲染依据。
- https://pandoc.org/installing.html 、https://pandoc.org/MANUAL.html ：绿色分发、输入输出能力与独立 PDF 引擎。
- https://github.com/JohannesKaufmann/html-to-markdown 、https://github.com/drumkitai/go-word 、https://github.com/bmaupin/go-epub ：专用 Go 替代候选；未下载、编译或质量验证。
- 外部来源已在本会话前序检索，主要为可变版本说明；FD 须固定所用版本，不将来源宣传视为运行证据。

## 本轮结果与执行边界

根据用户整体确认，将分散讨论整理为本统一 Plan，通过 `aiw issue capture` 保存。保留 capture 前正式 Plan 快照及原审批日志，不创建分支、不进行 Git 写操作、不 promote、不实现代码。仅进行文档编辑、Issue 管理操作与一次静态核对；没有安装或下载软件、运行转换、测试、编译、构建。
