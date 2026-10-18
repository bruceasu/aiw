# Issue Plan：aiw-file-convert 文件转文本插件

- Issue：`ISSUE-00008`
- 日期：2026-10-18
- 状态：扩展需求记录，INCOMPLETE。用户已确认 Go 移植与轻依赖方向，本次 capture 保存讨论和决定。Pandoc 首批格式及离线资源策略仍待确认；历史 APPROVED 不代表这些未决扩展已批准。未创建 FD、未开始实现。
- 需求提出者：当前用户；记录者：Codex。
- 方法：通用需求发现、Engineering Options、grill-with-docs 当前范围与依赖检查点。

## 问题与目标

用户希望通过 AIW 插件将不同格式文件转换为可供阅读、检索或后续处理的 text 或 Markdown，减少逐个使用专用工具的操作。插件名称明确为 `aiw-file-convert`，主要使用 Microsoft MarkItDown；旧版 `.doc` 如不受支持，可以增加其他库或转换后端。

## 已确认事实

1. 用户指定插件名、MarkItDown 项目和文件转文本的目标；允许为 `.doc` 增加依赖。用户进一步明确：不能假设安装了 MS Office 或 LibreOffice，要求考察 Python 库，并提出 antiword 及其提取质量问题。用户随后以 confirm 确认 MarkItDown 主路径与 antiword 的 .doc 正文提取方案，接受轻量本地二进制；具体 Python 封装及分发方式留给 FD 设计。
2. MarkItDown 官方说明定位为面向文本分析的 Markdown 转换工具，支持 PDF、Office、HTML、部分文本格式等；不是高保真版式转换器。不同格式需要相应可选依赖。
3. 官方 Word 转换器接受 `.docx`，不能据 README 中的 Word 字样推断它支持旧版二进制 `.doc`。
4. 不以 MS Office 或 LibreOffice 作为运行前提。此前 LibreOffice 候选不再推荐为首版路径。Antiword 可独立提取旧版 Word 文本，其 FAQ 明确 UTF-8 输出可表示中日韩文字，但受源文档编码影响，不能保证所有历史文件正确。
5. 仓库稳定规格约定 `aiw-<name>` 插件发现方式；现有 Python 插件和 `plugin.toml` 可作为后续设计参考。入口为 `aiw file-convert`，CLI 规则见已确认补充。

## 已确认范围

- 首版处理单个本地文件；优先 `.pdf`（已有文本层）、`.docx`、`.pptx`、`.xlsx`、`.xls`、`.html`、`.htm`、`.txt`、`.csv`、`.json`、`.xml`，并提供 `.doc` 的补充路径。
- 支持 UTF-8 Markdown 与纯文本输出，默认 Markdown；纯文本应消除 Markdown 语法，保留正文和可读的段落、列表及表格内容，不能仅修改扩展名。
- 提供标准输出及显式输出文件两种使用方式；诊断与正文分离。
- 保留源文件，默认拒绝覆盖已有目标；缺少依赖、文件损坏、加密、格式不支持或未提取到可用文本时给出可行动提示，避免伪报成功。
- 文档处理默认在本地完成，不自动上传文件或下载依赖；具体离线边界需要后续设计检查所选后端及配置。

用户于 2026-10-18 回复 confirm，确认此前讨论的首版范围及 MarkItDown + antiword 方案。批准仅针对需求与技术方向，不表示质量已验证或授权安装、测试、发布。

## 已确认 CLI 补充

用户在基础方案批准后补充 CLI 要求，并以“confirm, capture it”确认本节建议和 JSON/文件输出组合规则，授权写回同一 Issue Plan。捕获前保留基础方案快照 `issue-plan.before-cli-confirmation.md`，原审批日志不改写。

### 用户明确要求

- 帮助输出符合 AIW plugin 体系，包含 Usage、Options 和 Examples；插件声明中的 help 与插件自身帮助保持一致。
- `-h` / `--help` 打印 Usage 和 Examples。
- `-i` / `--input <file>` 指定输入文件。
- `-o` / `--output <file>` 指定输出。
- 输出参数值为 `-` 时表示 stdout；输出值为空时使用完整输入文件名再追加 `.md` 或 `.txt`，例如 `report.docx.md`，不是替换原扩展名。
- `--json` 输出 JSON，包含 `arguments`、`result-data`、`content-type`、`result-code`、`error-message` 等字段。`result-code` 成功时为 `ok`，失败时为错误代码。

### 已确认的参数补全

- 省略 `-o` 与显式空字符串均走自动文件名；单独 `-o` 缺少值属于参数错误，不能吞掉下一个选项。
- 增加 `--format md|txt`，默认 `md`，作为自动后缀、stdout 和 JSON 模式下内容类型的选择依据。
- `--help` 不要求提供输入，不加载转换依赖，不开始转换。
- JSON 模式下 stdout 仅包含一个 JSON 对象，不能混入正文、进度或日志；实际转换失败返回非零进程退出码。具体退出码表在 FD 定义。

### Usage 与 Examples

下列帮助使用 Easy English，插件声明和自身帮助须体现同一参数与示例契约。

```text
Usage:
  aiw file-convert -i <file> [-o <file|->] [--format md|txt] [--json]
  aiw file-convert -h | --help

Options:
  -h, --help                 Print usage and examples.
  -i, --input <file>         Input file.
  -o, --output <file|->      Output file. Use - for stdout.
      --ouput <file|->      Alias for --output.
                            If empty or omitted, append .md or .txt
                            to the full input file name.
      --format <md|txt>     Content format. Default: md.
      --json                Return a JSON response.

Examples:
  aiw file-convert --help
  aiw file-convert -i report.docx
  aiw file-convert -i report.docx -o report.md
  aiw file-convert -i legacy.doc --format txt
  aiw file-convert -i "my report.pdf" -o -
  aiw file-convert -i report.docx -o - --json
```

### JSON 返回规则与示例

以下为成功输出示例，`content-type` 描述转换正文；失败时 `result-data` 可为 null，`result-code` 为具体错误代码，`error-message` 为错误说明。字段固定类型与 `arguments` 中原始值/解析值的区分留待 FD 明确。

```json
{
  "arguments": {
    "input": "report.docx",
    "output": "-",
    "format": "md"
  },
  "result-data": "# Report\n\nDocument text.\n",
  "content-type": "text/markdown",
  "result-code": "ok",
  "error-message": ""
}
```

已确认 `--json` 与输出目标的组合规则：

- 指定输出文件或使用自动文件名时，文件保存转换后的 Markdown/纯文本，stdout 返回 JSON 结果，并记录最终输出路径；文件本身不改为 JSON。
- `-o - --json` 时，正文置于 JSON 的 `result-data` 中，stdout 只输出 JSON 对象，不另外打印一份正文。
- 不带 `--json` 且 `-o -` 时，stdout 直接输出转换正文。
- `--help` 打印 Usage 和 Examples，不执行转换；与其他开关同时出现时的优先级由 FD 明确。

## 新增转换与依赖管理（扩展草案）

### 用户明确追加的目标

1. 参考 `D:/03_projects/mermaid-tools/go`，支持 Markdown → HTML、Markdown → PDF、HTML → PDF。
2. 利用 Pandoc 增加文件格式转换能力，不预设系统已安装 Pandoc。
3. MarkItDown、antiword、Pandoc 以及渲染所需第三方依赖缺失时，提示安装，提供官网和对应平台的参考命令；用户明确授权后才自动下载、安装，优先绿色版。

本节扩展原“文件转文本”的目标。下文保留的原非目标、`--format md|txt`、默认后缀和文本 JSON 规则是历史基线；本节确认前不静默重写它们。新增 Markdown/HTML/PDF 与其他格式输出在此单独提出，不受原“反向生成非目标”排除。

### 参考项目的静态事实

- `go.mod` 使用 `github.com/yuin/goldmark`；`internal/markdown/renderer.go` 启用 GFM，处理 Mermaid fenced code block 为 `pre.mermaid`，默认转义原始 HTML。
- `internal/document/document.go` 包装完整 HTML，并调用 Mermaid 浏览器脚本。README 说明固定版本 CDN 与 `--offline` 内嵌脚本两种方式。
- `internal/browser/browser.go` 在 PATH 与平台常用路径查找 Chrome/Edge/Chromium，并通过 headless `--print-to-pdf` 打印本地 HTML；没有浏览器下载逻辑。
- 参考项目当前未提供独立 HTML → PDF CLI 输入模式。该能力需为新插件明确设计，不能声称原程序直接具有。
- 参考代码直接启动打印，未看到等待 Mermaid Promise、图片与字体完成的机制。可借鉴转换路径，不能将它作为渲染完整性的验证证据。
- 参考项目自动 HTML 文件名替换扩展名；本插件此前确认追加扩展名。保留本插件命名契约，不照搬参考项目差异。
- 只读参考路径；没有修改外部仓库、运行程序、编译或测试。参考现有方案不意味着已选择跨仓库导入、复制 Go 源码或新增 Go 运行时依赖。

### Python 与 Go 移植的可行性评估

结论：Python 可以实现参考程序的同类能力；目前没有必须移植 Go 程序的技术阻碍。此结论来自源码和官方 API 的静态映射，不是已完成的兼容性验证，也不表示用户已经选定 Python 或授权实现。

| 参考能力 | Python 对应方案 | 可行性与注意点 |
| --- | --- | --- |
| Goldmark GFM → HTML | markdown-it-py 的解析、扩展及自定义 renderer | 可实现；GFM 边缘语法、任务列表和标题锚点需明确配置，不能声称与 Goldmark 字节级一致 |
| Mermaid code fence → 图表 | 自定义 fence renderer 输出 mermaid 容器，HTML 内嵌固定版 Mermaid JS | 可实现；和参考程序一样，图表布局由浏览器内 JS 完成，不需要 Node/npm 运行时 |
| 元数据、标题、完整 HTML 与 CSS | Python 字符串/模板和标准库 HTML 转义 | 可实现；元数据优先级与转义规则须与选定契约一致 |
| 内嵌本地图片 | pathlib 路径解析、base64 与 MIME 处理 | 可实现；参考项目的根目录边界、符号链接处理及文件类型限制仍需保留 |
| MD/HTML → PDF | Python 调用 Chrome/Edge/Chromium headless，或使用 Playwright Python 控制浏览器 | 两者可行；直接进程调用更接近参考程序，Playwright 能显式等待 JS/资源完成后调用 page.pdf |
| 单文件绿色分发 | Python 环境 + 包/资源；或额外打包方案 | Python 源码分发不等同于 Go 单可执行文件；不过插件原本已需要 Python/MarkItDown，新增 Go 程序也不能消除这些依赖 |

初步架构选项：Python 统一插件入口，markdown-it-py 负责 HTML，固定本地 Mermaid JS 负责图表，浏览器负责 PDF，Pandoc 负责选定额外格式。该选项功能可行；后续用户明确优先减少依赖，因此不再默认推荐新增 Playwright。它仍可作为需要浏览器自动化时的备选，而不是必装项。

用户后续关注依赖体积：Go 参考程序直接调用现有浏览器，不需要 Playwright，也不需要目标机器安装 Go 工具链。静态检查 `go.mod` 仅列 Goldmark；Mermaid JS 资源为 2,749,172 字节，已嵌入可执行文件。现有 Windows AMD64 EXE 为 7,561,216 字节，约 7.21 MiB；仅读取文件大小，不执行、不重编译。现有产物大小不能证明与当前源码一致，也不能作为后续移植产物的大小保证。

用户以最新“confirm”确认：将现有 Go 转换器移植到 `src/programs/md-converter`，由 Python 插件统一调度；MD → HTML 无需额外 Python Markdown/Playwright 包，PDF 优先复用已安装 Edge/Chrome/Chromium，默认不引入 Playwright。若需要可靠等待 Mermaid/图片/字体，可在 Go 中评估轻量 CDP 控制方案；具体库、其编译体积与依赖变更属于后续 FD 设计，尚未确定。参考程序现有打印路径仍需补齐就绪判定，不能以体积优先为由降低 PDF 完整性要求。

轻量替代是仅用 Python subprocess 调用已安装浏览器，依赖较少，但必须另行解决可靠的渲染就绪判定，固定延时不足以保证 PDF 完整。Go 当前参考实现同样存在该问题，直接移植不能自动解决。

默认优先复用受支持的已安装 Chrome/Edge；兼容的浏览器发行物、Playwright 版本和平台支持应在 FD 固定。缺少合适浏览器时，遵守用户授权后下载安装、尽量绿色的既定要求，不自动执行 Playwright 的浏览器下载。

用户已选定将 Go 程序移植到 `src/programs/md-converter` 的方案方向。Python 方案仍然可行，当前选择依据是减少依赖与便携分发。后续 FD 必须适配插件 JSON/错误协议、强制二进制文件输出、HTML 输入路径、PDF 就绪判定及依赖指导。本次确认发生在 Issue 阶段，不启动实现、构建或安装。

评估依据：本轮进一步只读检查参考项目 `internal/markdown/images.go`、`internal/document/document.go`、`internal/metadata/metadata.go`、`internal/assets/assets.go` 和 CLI 参数相关符号；查看官方 markdown-it-py、Playwright Python、Mermaid 文档。未进行原型、安装、性能基准、PDF 导出或视觉验证。

- https://markdown-it-py.readthedocs.io/en/latest/using.html ：解析配置、renderer 与扩展 API。
- https://playwright.dev/python/docs/library ：Python 浏览器控制能力与安装要求。
- https://playwright.dev/python/docs/api/class-page#page-pdf ：PDF 生成接口。
- https://playwright.dev/python/docs/browsers#google-chrome--microsoft-edge ：Chrome/Edge 通道及浏览器安装约束。
- https://mermaid.js.org/config/usage.html ：HTML 集成与异步 Mermaid 渲染。

### 转换能力边界与选项

| 用户结果 | 候选路径 | 约束 |
| --- | --- | --- |
| MD → HTML | 参考项目的 Markdown、文档模板及 Mermaid 渲染思路 | 待确定实现复用方式、样式与离线资源；本地图片相对路径需正确处理 |
| MD → PDF | MD → HTML → 已安装或授权安装的浏览器打印 | 必须等待图表、图片及字体渲染完成；中文字体和分页质量未验证 |
| HTML → PDF | 本地 HTML → 浏览器打印 | 需保留资源基路径，明确脚本、远程资源与本地读取边界 |
| 更多格式 | Pandoc reader/writer 支持的转换组合 | 不是任意扩展名互转；版本、具体 reader/writer、附加资源影响能力 |
| 原文档 → MD/TXT | 保留 MarkItDown 与 antiword 的已确认职责 | Pandoc 不自动取代旧版 .doc 的 antiword 路径 |

Pandoc 官方文档说明可列出输入与输出格式，但本轮没有执行 `pandoc --list-input-formats` 或 `--list-output-formats`。候选扩展可包括 DOCX、ODT、EPUB、RST 等与 Markdown 的转换；具体支持方向、验收样本和首批清单待定，不承诺完整矩阵。Pandoc PDF 需要相应 PDF 引擎，不能把仅安装 Pandoc 当作 PDF 环境就绪。

### 第三方依赖检测、提示与安装要求

- 按当前转换路径检测实际需要的依赖：Python 包/运行环境与系统可执行文件分别说明。MarkItDown 是 Python 包，其格式 extras 也可能缺失；antiword、Pandoc、浏览器有各自的平台与资源要求。
- 先查找已有可用依赖；正常转换和 `--help` 不隐式联网下载安装。不因非当前路径的可选依赖缺失而阻塞其他格式。
- 缺失时给出依赖名、受影响能力、官网/发布页、匹配 OS/CPU 的手动安装或解压说明及参考命令。JSON 模式仍返回单个结构化错误，安装说明也必须能被程序读取；具体字段由 FD 确定。
- 用户明确授权具体下载来源、版本、安装目录、命令和预期副作用后，才执行自动下载、解压或安装。拒绝授权或非交互场景未提供授权时，保持缺失状态并退出，不重复强制提示或静默下载。
- 优先官方绿色压缩包，安装到用户可写目录；避免默认提权、系统级安装和修改全局 PATH。没有合适绿色发行版时说明替代方式，不将 Python 虚拟环境等同于单文件绿色程序。
- 下载方案需固定版本、匹配平台，并明确发布物校验、资源文件和许可证要求；失败不破坏已安装工具，不报告安装成功。安装目录、升级/清理边界与授权机制留待独立设计。
- 用户当前要求是设计该能力，不是授权本会话实际安装、下载或执行依赖探测。

已核实的安装资料示例（仅展示，不执行）：

```text
MarkItDown official project:
  https://github.com/microsoft/markitdown
  python -m pip install "markitdown[pdf,docx,pptx,xlsx,xls]"
  Use a dedicated Python environment. Select extras for the requested formats.

Pandoc official install guide:
  https://pandoc.org/installing.html
Pandoc official releases:
  https://github.com/jgm/pandoc/releases
  Portable Windows option: download the matching official ZIP and unzip it.
  Windows package manager alternative:
  winget install --source winget --exact --id JohnMacFarlane.Pandoc
```

MarkItDown extras 的最终版本和安装命令需在 FD 固定。Antiword 及浏览器的官方可信发布物、绿色包、平台支持和安装命令尚未核实，不能编造下载地址；应在实现前补齐对应安装指导。

### CLI 与验收扩展草案

- 建议保留已确认 `--format`，扩展 `html`、`pdf` 及选定 Pandoc 输出格式；完整输入文件名追加目标后缀，默认 `md` 不变。后缀与 `--format` 冲突规则待设计，不自动凭输出扩展名修改已指定格式。
- 新帮助和 Examples 须列出三条转换路径及依赖提示，例如：

```text
aiw file-convert -i design.md --format html -o design.html
aiw file-convert -i design.md --format pdf -o design.pdf
aiw file-convert -i page.html --format pdf -o page.pdf
```

- 验收覆盖中文、标题、表格、代码、Mermaid 与相对路径图片；PDF 不得在图表尚未渲染时伪报完整成功。具体外部资源与离线策略确认后补充可判定示例。
- PDF 明确要求下缺少浏览器/引擎应报告失败及安装指导，不能把只产生 HTML 当作 PDF 转换成功。
- 缺失依赖时先提示；未授权不下载；授权安装成功后仅在约定范围重新执行该转换；安装失败仍返回结构化错误。具体重试和执行范围须在 FD 明确。

### 已确认：二进制结果必须输出到文件

用户明确要求：“二进制格式文件不能输出终端，必须输出成文件。”本条取代此前关于二进制 stdout/JSON 的开放问题，适用于 PDF、DOCX、ODT、EPUB 等二进制目标格式。

- 二进制格式必须生成文件。指定 `-o <file>` 时使用该路径；省略输出或显式空值时，延续自动命名规则，在完整输入文件名后追加目标扩展名，例如 `design.md.pdf`。
- 显式 `-o -` 一律返回参数错误，无论 stdout 是否连接终端、管道或重定向文件，也无论是否启用 `--json`；不能通过重定向绕过必须指定文件输出的契约。
- 参数检查在转换开始前完成；无效的二进制 stdout 请求不得触发转换或依赖下载、安装。
- `--json` 可以在 stdout 返回文件路径、MIME、大小等元数据；`result-data` 不包含二进制正文，也不将正文转成 Base64 内嵌。具体元数据对象字段由 FD 定义。
- 禁止将二进制正文写入 stdout 或 stderr；原先文本目标的 stdout/JSON 规则继续适用于文本格式。
- 验收：`--format pdf -o result.pdf` 生成文件；省略 `-o` 生成追加 `.pdf` 的文件；`--format pdf -o -` 与 `--format pdf -o - --json` 均失败且不产生二进制输出。其他二进制目标使用相同规则。

%% NEEDS_INPUT: “更多格式”的首批转换清单尚未指定。建议以 DOCX/ODT/EPUB/RST 与 Markdown 的选定方向作为候选，由用户确认后列出逐条验收范围，不将 Pandoc 全部 reader/writer 的笛卡尔积作为承诺。

%% NEEDS_INPUT: 参考项目含 Mermaid；新插件是否要求生成的 HTML/PDF 默认可离线渲染（含 Mermaid 和必要资源），以及如何处理输入 HTML 的远程资源？建议延续原默认本地、不自动访问外部资源的边界。

### 规模与拆分复评

范围已从正文提取扩展到可独立验收的文档发布输出、Pandoc 格式扩展与授权依赖安装，预计中到大规模，存在多个有独立价值的结果。建议在本 Issue 保留整体目标与跨项约束，准备按以下用户结果拆分；尚未创建子 Issue 或分配 ID：

1. 保留原文档 → MD/TXT 基线。
2. 新增 MD → HTML/PDF 与 HTML → PDF；共享浏览器、字体、图表与离线验收边界。
3. 按确认清单扩展 Pandoc 转换；依赖通用的缺失提示。
4. 用户授权后的绿色依赖获取；转换先具备手动安装指引，不以实现自动安装作为手动可用路径的必要前提。

各项仅规划独立结果，不重复同一格式交付范围。跨项约束包括统一 CLI/JSON、依赖提示、离线默认和安装授权。子 Issue 需各自确认范围并通过 CLI 创建、关联；本草案不代表原批准覆盖新增结果，也不触发 FD 设计或实现。

### 新增来源与本轮工作

- 用户本轮三项补充要求；直接输入来源。
- 用户后续明确二进制格式必须输出成文件，禁止终端输出；据此关闭二进制输出路由问题，更新参数与 JSON 验收规则。
- `D:/03_projects/mermaid-tools/go/README.md`、`go.mod`、`internal/markdown/renderer.go`、`internal/browser/browser.go`，以及针对 `cmd/md2html/main.go`、`internal/document/document.go`、`internal/assets/assets.go` 的定向静态搜索。
- https://pandoc.org/installing.html ：Windows ZIP 绿色版、官方发布页、winget 示例和 PDF 引擎依赖，2026-10-18 查阅。
- https://pandoc.org/MANUAL.html ：输入输出格式能力、格式列表命令与 PDF 引擎说明，同日查阅。
- 前序执行了 Get-Content、rg、官方网页读取及现有 EXE 大小读取。本次依据用户确认保存正式 Plan 的扩展前快照，更新并通过 aiw issue capture 保存扩展内容；原审批日志与摘要不改写，不宣称修订版已经重新获批。未运行转换、检测依赖、下载安装、测试、编译或构建。

## 非目标（原批准基线，新增目标见上节）

首版不包含 GUI、批量目录、远程 URL、压缩包递归、图片或扫描 PDF OCR、音视频转录、云端转换、高保真排版及文档反向生成。MarkItDown 支持的其他能力不自动成为本插件承诺。

## 工程可行性与选项

总体判断：CONSTRAINED，主路径可行；`.doc` 在不依赖 Office 套件的条件下存在候选方案，质量与分发策略尚未验证。

| 路径 | 优点 | 约束与风险 | 当前结论 |
| --- | --- | --- | --- |
| MarkItDown 主路径 | 直接复用现有格式转换 | 分格式依赖；不同文档的文本与结构质量有差异 | 用户指定方向 |
| `.doc` 由 Python 调用 antiword | 不依赖 Office 套件；默认输出正文文本；可指定 UTF-8 及取消固定宽度换行 | 需要匹配平台的可执行文件及资源；复杂结构还原与中文质量未实测；分发需核对 GPL 条款 | 已确认采用方向；集成方式由 FD 确定 |
| Python 包 `doc2txt` 封装 antiword | 项目声明内置 Windows AMD64、Linux AMD64、macOS ARM64 二进制及资源，便于 Python 集成 | 本质仍调用本地程序，不是纯 Python；不覆盖所有架构；需核对二进制来源和许可证，不能只看封装声明 MIT | 候选封装，不直接承诺使用 |
| 纯 Python 包 `legacy-doc` | 项目声明无第三方依赖及外部二进制，直接解析 Word 97–2003 OLE 文件的 piece table | 当前标记 Alpha、要求 Python >=3.11；仅提取文本，不承诺格式与修订还原；真实中文及复杂文档覆盖未知 | 纯 Python 备选，须先静态审查及另行授权样本验证 |

不得以把 `.doc` 改名为 `.docx` 代替转换。外部后端缺失时应准确说明 `.doc` 条件支持，不影响已安装依赖的其他格式。

### Antiword 的质量边界与集成证据

- 已确认为正文文本提取后端方向，不承诺恢复 Markdown 标题、表格及复杂排版。手册提供默认 text 输出、`-w 0` 按段落输出以及隐藏文字、删除修订文字选项；相关内容是否包含需在 FD 明确。
- FAQ 说明 UTF-8 可表示中日韩文字，但旧文件编码可能导致乱码；“支持 UTF-8”不能代替中文样本验证。
- 已静态读取 `doc2txt` 发布提交的 `antiword_wrapper.py`：通过 `subprocess.run` 调用内置程序，设置 `ANTIWORDHOME` 并指定 UTF-8。实现没有超时，输出采用替换方式解码；若采用，FD 需解决卡住与乱码被掩盖的风险。
- 其 Windows/Linux 分支未按架构严格筛选，不能仅凭 Python wheel 标签推断所有 CPU 或 Linux 环境兼容。
- Antiword 源码仓库 README 标注 GPL；封装包的 MIT 标记不能替代所附二进制的分发条款核对。本轮仅记录选型约束，未作法律结论。
- 用户确认采用 antiword 路径，纯 Python legacy-doc 不纳入首版承诺。两者均未实测，不给出质量排名或成功率。

## 验收示例

1. 含中文、标题、列表和表格的 DOCX 能输出可读 UTF-8 Markdown，正文和顺序合理；不要求视觉版式一致。
2. 同一文档选择纯文本后，输出保留正文和段落，无人为生成的 Markdown 标题或链接语法；不能通过删除所有标点破坏原始内容。
3. 有文本层 PDF、PPTX、Excel 和 HTML 各提供代表文件，输出对应正文；表格数据不能无提示丢失。精确的工作表、幻灯片边界及读取顺序在 FD 中明确。
4. `.doc` 后端可用时能转换含中文的旧版文档；不可用时明确说明缺少的可选依赖及安装或替代操作，不返回成功空文件。
5. 输入不存在、损坏、加密、不支持或没有可提取文本时，报告原因；不损坏源文件和既有目标文件。
6. 输出到标准输出时，正文不混入诊断；输出到指定文件时使用 UTF-8。路径含空格、中文时仍可正确定位输入输出。
7. 在未安装 MS Office 和 LibreOffice 的环境中，声明支持的 `.doc` 路径仍可运行；安装与帮助信息应明确是否携带或需要 antiword 等本地二进制。
8. `-h` 与 `--help` 展示插件体系一致的 Usage、Options 和 Examples；无需输入文件或转换依赖。
9. 省略输出或显式空值时，`report.docx` 默认生成 `report.docx.md`，指定 `--format txt` 则生成 `report.docx.txt`；`-o -` 不创建自动命名的输出文件。
10. `--output` 和 `--ouput` 均可作为 `-o` 的长参数；缺少参数值明确报错。
11. 文件输出结合 `--json` 时，目标文件仍为正文，stdout 为包含结果与输出路径的 JSON；`-o - --json` 时，正文仅在 JSON 的 `result-data` 中出现。失败使用错误代码与错误说明，不混入非 JSON 内容。

这些是后续设计与验证的验收输入，本轮未执行转换或测试。

## 规模与拆分（原批准基线；扩展复评见上节）

中等规模，单一用户可验收结果：通过一个 AIW 插件完成文件转文本。主要不确定性为 `.doc` 外部依赖与实际文档提取质量。暂不拆分为子 Issue；各格式后端是同一结果的组成部分。若增加 OCR、批量或远程内容，应另行评估独立范围。

## 待决定事项与 Gates

- 已确认：首版本地单文件、常见文档格式、Markdown/纯文本输出；OCR、音视频、批量和远程 URL 不在首版范围。

- 已确认：`.doc` 由 Python 封装或调用 antiword，以正文提取为验收目标；不要求纯 Python，不能依赖 MS Office 或 LibreOffice。具体封装包尚未选定。

%% NEEDS_EVIDENCE: Antiword 与纯 Python 候选尚无代表性文件实测结果。后续如获授权，应以中英混排、表格、页眉页脚、文本框、修订及旧编码样本检查正文遗漏、顺序与乱码；不将文档宣传视为质量验证。

业务范围与后端方向已确认，可记录 Issue 批准。样本质量、平台与分发风险保留为后续 FD 设计和交付约束；需求批准不表示这些风险已经解除。FD 移交是独立操作，本轮不执行。

后续 FD 需要确定：具体格式矩阵和依赖版本、退出状态表、JSON 字段固定类型与参数值表示、帮助优先级、纯文本渲染规则、输出覆盖策略、临时文件生命周期、转换失败与空内容处理。已确认的 CLI 参数与输出路由不得在设计中静默变更；其余工程细节未解决前不能宣称设计或实现就绪。

## 来源与静态证据

- 用户当前请求：明确插件名、MarkItDown 主路径、text/Markdown 目标及 `.doc` 的补充依赖许可。
- 用户后续请求明确 Usage、Examples、输入输出参数、JSON 结果结构；最新“confirm, capture it”确认补充建议及 JSON/文件输出组合，并授权 capture。
- https://github.com/microsoft/markitdown ：官方定位、格式与可选依赖说明，2026-10-18 查阅。
- https://github.com/microsoft/markitdown/blob/main/packages/markitdown/src/markitdown/converters/_docx_converter.py ：`ACCEPTED_FILE_EXTENSIONS = [".docx"]`，同日查阅；main 为可变来源，FD 选型时需固定版本。
- https://help.libreoffice.org/latest/en-US/text/shared/guide/convertfilters.html ：`.doc` 输入过滤器、DOCX 与文本输出依据，同日查阅。
- 用户追加约束：不能假设安装 MS Office / LibreOffice，询问 Python 库及 antiword 质量；替代此前 LibreOffice 推荐。
- https://github.com/grobian/antiword/blob/main/Docs/FAQ ：中日韩 UTF-8 与源编码限制；这是源码镜像中的上游文档，2026-10-18 查阅。
- https://github.com/grobian/antiword/blob/main/Docs/antiword.1 ：文本输出、段落宽度及修订选项，同日查阅。
- https://github.com/grobian/antiword/blob/main/Docs/ReadMe ：版本说明及 GPL 标记，同日查阅。
- https://pypi.org/project/doc2txt/ ：维护者对内置二进制、平台和 API 的说明，同日查阅，尚未运行验证。
- https://github.com/Quantatirsk/doc2txt-pypi/blob/21bb4d1001187ee40839a35ae68c6803a2be2021/doc2txt/antiword_wrapper.py ：发布提交的实际调用路径与边界，同日静态读取。
- https://pypi.org/project/legacy-doc/ ：纯 Python 候选的声明范围、Alpha 状态及 Python 版本要求，同日查阅；未审查其解析实现。
- `openspec/specs/cli-and-plugins/spec.md`：插件发现契约。
- `src/plugins/examples/plugin-example.toml`、`src/plugins/aiw-say/plugin.toml`：现有插件声明示例；未改动。
- `src/cmd/aiw-req/issue-create.go`、`issue-capture.go`、`aiw issue --help`：Issue 创建与捕获入口。

## 基线捕获历史与下一步

基础方案此前通过 capture 与 approve 记录批准。此次依据“confirm, capture it”保存原 Plan 快照，更新草稿并执行 capture；原审批日志及摘要保留，不伪造修订版重新批准结果。本轮执行 aiw issue --help、capture，以及一次静态读取/状态查询。未 promote、修改源代码、创建分支、执行 Git 写操作、安装依赖、运行测试、编译、构建或转换。下一步为单独决定 FD 移交；移交时须核对当前修订与历史审批摘要的绑定状态。
