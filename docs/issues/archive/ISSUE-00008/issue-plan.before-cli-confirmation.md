# Issue Plan：aiw-file-convert 文件转文本插件

- Issue：`ISSUE-00008`
- 日期：2026-10-18
- 状态：需求方案已由用户 confirm 确认；本次通过 CLI 记录 APPROVED，未创建 FD、未开始实现。
- 需求提出者：当前用户；记录者：Codex。
- 方法：通用需求发现、Engineering Options、grill-with-docs 当前范围与依赖检查点。

## 问题与目标

用户希望通过 AIW 插件将不同格式文件转换为可供阅读、检索或后续处理的 text 或 Markdown，减少逐个使用专用工具的操作。插件名称明确为 `aiw-file-convert`，主要使用 Microsoft MarkItDown；旧版 `.doc` 如不受支持，可以增加其他库或转换后端。

## 已确认事实

1. 用户指定插件名、MarkItDown 项目和文件转文本的目标；允许为 `.doc` 增加依赖。用户进一步明确：不能假设安装了 MS Office 或 LibreOffice，要求考察 Python 库，并提出 antiword 及其提取质量问题。用户随后以 confirm 确认 MarkItDown 主路径与 antiword 的 .doc 正文提取方案，接受轻量本地二进制；具体 Python 封装及分发方式留给 FD 设计。
2. MarkItDown 官方说明定位为面向文本分析的 Markdown 转换工具，支持 PDF、Office、HTML、部分文本格式等；不是高保真版式转换器。不同格式需要相应可选依赖。
3. 官方 Word 转换器接受 `.docx`，不能据 README 中的 Word 字样推断它支持旧版二进制 `.doc`。
4. 不以 MS Office 或 LibreOffice 作为运行前提。此前 LibreOffice 候选不再推荐为首版路径。Antiword 可独立提取旧版 Word 文本，其 FAQ 明确 UTF-8 输出可表示中日韩文字，但受源文档编码影响，不能保证所有历史文件正确。
5. 仓库稳定规格约定 `aiw-<name>` 插件发现方式；现有 Python 插件和 `plugin.toml` 可作为后续设计参考。候选入口为 `aiw file-convert`，尚未冻结 CLI 参数。

## 已确认范围

- 首版处理单个本地文件；优先 `.pdf`（已有文本层）、`.docx`、`.pptx`、`.xlsx`、`.xls`、`.html`、`.htm`、`.txt`、`.csv`、`.json`、`.xml`，并提供 `.doc` 的补充路径。
- 支持 UTF-8 Markdown 与纯文本输出，默认 Markdown；纯文本应消除 Markdown 语法，保留正文和可读的段落、列表及表格内容，不能仅修改扩展名。
- 提供标准输出及显式输出文件两种使用方式；诊断与正文分离。
- 保留源文件，默认拒绝覆盖已有目标；缺少依赖、文件损坏、加密、格式不支持或未提取到可用文本时给出可行动提示，避免伪报成功。
- 文档处理默认在本地完成，不自动上传文件或下载依赖；具体离线边界需要后续设计检查所选后端及配置。

用户于 2026-10-18 回复 confirm，确认此前讨论的首版范围及 MarkItDown + antiword 方案。批准仅针对需求与技术方向，不表示质量已验证或授权安装、测试、发布。

## 非目标

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

这些是后续设计与验证的验收输入，本轮未执行转换或测试。

## 规模与拆分

中等规模，单一用户可验收结果：通过一个 AIW 插件完成文件转文本。主要不确定性为 `.doc` 外部依赖与实际文档提取质量。暂不拆分为子 Issue；各格式后端是同一结果的组成部分。若增加 OCR、批量或远程内容，应另行评估独立范围。

## 待决定事项与 Gates

- 已确认：首版本地单文件、常见文档格式、Markdown/纯文本输出；OCR、音视频、批量和远程 URL 不在首版范围。

- 已确认：`.doc` 由 Python 封装或调用 antiword，以正文提取为验收目标；不要求纯 Python，不能依赖 MS Office 或 LibreOffice。具体封装包尚未选定。

%% NEEDS_EVIDENCE: Antiword 与纯 Python 候选尚无代表性文件实测结果。后续如获授权，应以中英混排、表格、页眉页脚、文本框、修订及旧编码样本检查正文遗漏、顺序与乱码；不将文档宣传视为质量验证。

业务范围与后端方向已确认，可记录 Issue 批准。样本质量、平台与分发风险保留为后续 FD 设计和交付约束；需求批准不表示这些风险已经解除。FD 移交是独立操作，本轮不执行。

后续 FD 需要确定：具体格式矩阵和依赖版本、CLI 参数与退出状态、纯文本渲染规则、输出覆盖策略、临时文件生命周期、转换失败与空内容处理。用户确认方向后，这些工程细节由后续 FD 设计明确；未解决前不能宣称设计或实现就绪。

## 来源与静态证据

- 用户当前请求：明确插件名、MarkItDown 主路径、text/Markdown 目标及 `.doc` 的补充依赖许可。
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

## 本轮结果与下一步

用户回复 confirm 后更新同一 Issue Plan，并执行 capture 与 approve 记录需求批准；通过 show --json 核对实际审批状态。不 promote、不修改源代码、不创建分支、不执行 Git 写操作；未安装依赖、运行测试、编译、构建或运行转换。此前已静态检查本地契约、上游文档、维护者说明及一个封装实现。本轮实际管理命令为 aiw issue --help、capture、approve、show --json。下一步是单独授权移交编号 FD，完成具体集成、平台支持与分发方案设计。
