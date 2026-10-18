# aiw-file-convert 文件转换

`aiw file-convert` 是 Python 3.11+ 插件。将 `src/plugins/aiw-file-convert`
作为完整插件目录分发，保持入口、Python 模块、help.txt 和 plugin.toml 同目录。
Go 转换器源码在 `src/programs/md-converter`，需单独编译/分发对应平台程序；
源码交付不会自动更新已安装的 AIW 或生成可执行文件。

```text
aiw file-convert --help
aiw file-convert -i report.docx
aiw file-convert -i legacy.doc --format txt
aiw file-convert -i "my report.pdf" -o -
aiw file-convert -i report.docx -o - --json
aiw file-convert -i design.md --format html -o design.html
aiw file-convert -i design.md --format pdf -o design.pdf --json
aiw file-convert -i page.html --format pdf -o page.pdf
aiw file-convert -i design.md --format docx -o design.docx
```

`-i/--input` 必需；`-o/--output` 支持兼容别名 `--ouput`。
省略输出或传入空字符串时，完整输入名追加目标后缀，例如 `report.docx.md`、
`design.md.pdf`。`--format` 默认 md，不根据输出后缀推断，后缀冲突报错。
已有目标一律不覆盖，源文件不修改；临时转换成功后通过 hard link 不覆盖发布，
不支持该能力的文件系统明确失败。目标目录须已存在。

文本可用 `-o -`；PDF/DOCX/ODT/EPUB 必须输出文件，JSON、管道、重定向也不例外。
`--help` 显示 Usage、Options、Examples，不加载转换依赖。

| 输入 | 输出 | 后端与边界 |
| --- | --- | --- |
| PDF（有文本层）、DOCX、PPTX、XLSX、XLS、HTML/HTM | md/txt | MarkItDown；TXT 额外用 Go 将生成的 Markdown 语法节点转为正文 |
| TXT、CSV、JSON、XML、MD/Markdown | md/txt | UTF-8 本地文本；结构文本保留为代码块，TXT 去掉生成的语法；Markdown 正文提取使用 Go |
| 旧 DOC | md/txt | antiword 正文，非排版重建，不依赖 Office/LibreOffice |
| MD/Markdown | html/pdf | Go Goldmark + 内嵌 Mermaid/CSS；PDF 需要浏览器 |
| HTML/HTM | pdf | 本地被动 HTML 预处理 + Go 浏览器打印 |
| MD/Markdown | docx/odt/epub/rst | Go 安全 HTML + 可选 Pandoc；正文/表格/图片，Mermaid 保留代码，不承诺图形 |
| ODT、EPUB、RST | md/txt | Pandoc `--sandbox` 与资源预检；TXT 额外用 Go |

不声明其它任意格式互转。无 OCR、音视频、批量、远程 URL、云服务或 Office 高保真排版。
中文需要系统字体；UTF-8 并不代表旧 DOC 编码、字体和视觉质量已验证。

## 依赖

只检查当前路径需要的后端。程序不自动下载安装或编译依赖，不要求 Playwright、
Node/npm、Office、LibreOffice 或 LaTeX。

| 后端 | 定位与安装参考 |
| --- | --- |
| MarkItDown | [官网](https://github.com/microsoft/markitdown)；在专用 venv 中 `python -m pip install "markitdown[pdf,docx,pptx,xlsx,xls]==0.1.3"`，再用该 Python 运行插件。目标版本非实测保证 |
| antiword 0.37 | [源码与文档](https://github.com/grobian/antiword)；Linux `apt install antiword`，macOS `brew install antiword`；Windows 需可信匹配平台程序及 UTF-8.txt 映射。`AIW_CONVERT_ANTIWORD` 指向程序，`ANTIWORDHOME` 指向映射目录；未捆绑未经核验的 Windows 下载 |
| Pandoc 3.6.4 或兼容版 | [安装指导](https://pandoc.org/installing.html)、[官方发布](https://github.com/jgm/pandoc/releases)；优先 ZIP 解压到用户目录，设置 `AIW_CONVERT_PANDOC`；Windows 可选 `winget install --source winget --exact --id JohnMacFarlane.Pandoc` |
| Go helper | 设置 `AIW_MD_CONVERTER`，或放插件目录/md-converter/bin；源码用户显式构建见 [README](../../src/programs/md-converter/README.md) |
| 浏览器 | 复用 Edge/Chrome/Chromium 或设置 `AIW_CONVERT_BROWSER`；[Edge](https://www.microsoft.com/edge/download)、[Chrome](https://www.google.com/chrome/)、[Chromium](https://www.chromium.org/getting-involved/download-chromium/)；不捆绑浏览器 |

Markdown 禁止原始 HTML；图片只允许输入目录内的 PNG/JPEG/GIF/WebP，解析符号链接后检查。
HTML 图片、CSS 和字体须本地内嵌；拒绝脚本、事件、iframe/object、刷新、base、远程资源、
CSS import/转义/注释和 SVG。嵌入 CSS 的资源只能位于其目录之内。
不支持的资源明确失败。PDF 等待 Mermaid、字体和图片，渲染最长 45 秒，总体最长 90 秒。
Pandoc 只处理检查后的 ZIP/RST 快照或 Go 生成的自包含 HTML；拒绝外部引用、DTD/实体、
CSS url/import 和 RST include/raw/图片指令。这是首版保守边界，部分含超链接的 EPUB/ODT
也可能被拒绝，不能声称完整覆盖所有文档。文档资源约束不等于操作系统沙箱。

## JSON 与错误

stdout 只返回一个 JSON。`result-data` 对文本 stdout 为字符串，文件输出时为 null；
安装模式为清单结果对象。`content-type` 描述转换结果。

```json
{"arguments":{"input":"report.docx","output":"-","format":"md","json":true},"result-data":"# Report\n","content-type":"text/markdown","result-code":"ok","error-message":""}
```

文件成功附加 `output-file: {path, size}`，二进制不进入 JSON 或 Base64。
缺依赖附加 `dependency: {name, website, instructions, platform, detail}`。
非 JSON 的文件成功输出最终路径，文本 stdout 输出正文，错误诊断 stderr。

| 退出码 | result-code | 含义 |
| --- | --- | --- |
| 0 | ok | 成功；安装预览也成功但不执行安装 |
| 2 | arguments | 参数/格式组合/批准摘要错误 |
| 3 | missing-dependency | 当前路径需要的依赖缺失或不可用 |
| 4 | conversion | 转换、输入读取、资源、超时或格式失败 |
| 5 | output | 输出冲突或不能安全发布 |
| 6 | install | 下载/校验/安装失败 |

## 明确授权的便携安装

转换绝不触发安装。首版只支持用户提供的固定版本 ZIP 清单，未内置未经核验的发行 URL、
哈希或浏览器绿色包。Python/MarkItDown 使用上述专用环境命令，不通过 ZIP 安装。
清单恰好包含以下字符串字段（例子为结构示意，URL、哈希必须用可信发布信息替换）：

```json
{
  "dependency": "pandoc",
  "version": "3.6.4",
  "url": "https://release-assets.githubusercontent.com/REPLACE_WITH_TRUSTED_DIRECT_ZIP_URL",
  "sha256": "REPLACE_WITH_64_LOWERCASE_HEX_DIGITS",
  "system": "Windows",
  "machine": "AMD64",
  "target": "D:/tools/pandoc-3.6.4",
  "entrypoint": "pandoc-3.6.4/pandoc.exe"
}
```

```text
aiw file-convert --install-plan portable.json --json
aiw file-convert --install-plan portable.json --approve-install <returned-plan-sha256> --json
```

预览返回完整来源、版本、安装路径、入口和摘要，不访问网络。摘要是解析后 JSON 的
sort_keys=True、separators=(",", ":")、ensure_ascii=True、UTF-8 SHA256。
目标必须是明确绝对路径，不接受相对路径或 `~`；先解析既有祖先链接和路径，再将规范目标
写回清单用于展示、摘要和执行。路径解析变化会使原批准失效。
批准标识必须完全匹配当前清单。OS/CPU 按 `platform.system()/machine()` 精确匹配（忽略大小写）。
dependency 为 pandoc/antiword/browser/md-converter；来源限 HTTPS 的 github.com、
release-assets.githubusercontent.com、objects.githubusercontent.com、downloads.sourceforge.net、
storage.googleapis.com；拒绝重定向，需提供可信直链（官方发布入口通常会重定向，不能直接代替直链）。
主机白名单不替代用户核实来源及 SHA256。下载上限512MiB，解压上限1GiB/20000条目；
拒绝穿越、符号链接、特殊文件和重复路径，保留可执行位，检查入口。
只创建全新目标目录，不提权、不改全局 PATH、不覆盖已有工具；复制失败保留明确恢复路径，
不会报告安装成功。成功返回环境变量设置指导，用户显式重试转换，不自动重试。

本轮交付只做静态审查与编译；真实转换、PDF 视觉、第三方后端、跨平台行为和下载安装均未实测。
