# FD-078 独立设计审查

<!-- aiw-data: FD-078-design-review.json -->

结论：**ready**。审查对象为 FD-078 Revision 2 及 Planner 在本次审查中明确接受的设计补充，来源事件为 `FD-078-000002-design-requested`。下述 R1/R2 已在设计讨论中解决，补充必须写入 FD 后再实现。这是设计证据，不是实现验收或流程状态变更。

整体职责划分可行：Python 管理插件参数、输出事务、适配器和安装；独立 Go 程序负责 Goldmark/Mermaid 与浏览器 PDF。CLI、二进制禁止 stdout、JSON 类型、无覆盖发布和依赖惰性提示均已形成具体契约。参考程序只执行浏览器打印命令，没有可靠就绪等待；FD 增加 CDP 和异步完成判断是必要修正。安装、浏览器以及跨平台能力未实测，不能作为已验证质量陈述。

## 已解决的设计问题

### R1：离线和资源边界尚未覆盖 Pandoc 等非浏览器后端

FD Solution 把远程请求控制集中在 Go PDF/CDP 和 HTML 预处理，但转换矩阵还公开承诺 `md -> docx/odt/epub`。这些后端可能处理 Markdown 图片与其他资源引用；浏览器的 CSP、Network.setBlockedURLs 不约束外部 Pandoc 进程。Issue 的“不隐式访问远程资源”是跨后端约束，不能只覆盖 HTML/PDF。

在设计中明确每个资源消费后端的责任。建议在调用 Pandoc 前解析 Markdown 资源引用，拒绝网络 URL、目录外文件、原始 HTML/扩展资源指令；把允许的图片复制或嵌入独立临时资源空间，并固定输入格式和工作目录。不要把仅检查一类 Markdown 图片语法或设置工作目录当成完整限制。无法可靠界定的资源语法必须明确拒绝，不能交给后端自由读取。HTML 提取路径同样应明确关闭 MarkItDown 的可选联网/LLM能力，不传入远程 URL。

验收例：包含 HTTPS 图片、`../outside.png` 或绕过检查的原始 HTML 资源时，Markdown 导出 DOCX/EPUB 必须在启动 Pandoc 前明确失败；正常目录内图片可经受控临时资源继续。此处要求设计与静态调用路径证据，不要求当前运行测试。

Planner 已接受的补齐：Markdown 先经 Go 安全 HTML 渲染，关闭原始 HTML、目录内图片内嵌，再以固定 HTML reader 交给 Pandoc；Pandoc 全路径加 `--sandbox`。输入 ZIP 的 XML 预检外部 src/href/Target 与事件，RST 拒绝 include/raw 和远程图片。Pandoc 输出只承诺正文，不承诺 Mermaid 图表。这个组合使边界有明确责任归属；实现应确保传给 Pandoc 的实际文件就是检查/生成后的快照，而不是检查后重新读取不受控路径。

### R2：安装授权与下载信任规则尚有歧义

同一段同时允许“ZIP 或 wheel 的 pip 安装”和“首版仅便携 ZIP”，会给实现与帮助带来不同的公开契约。“URL 主机白名单与最终重定向校验”也不足以说明是否在跟随每一跳之前检查；如果请求后才检查最终 URL，就可能已访问未授权主机。

明确首版只实现 ZIP，MarkItDown 只提供手动专用 venv/pip 指南。安装计划规定准确字段类型、平台别名规范、允许依赖标识、必须的包内入口/资源位置和规范化摘要算法；审批必须绑定同一份已解析清单快照及完整 target，而不是执行时重读清单。固定各依赖允许的可信主机列表，不接受清单自行扩大白名单；自动重定向应关闭，在每次发出下一跳请求前校验 HTTPS、主机和重定向次数。下载安装全程限时限量，目标必须全新；解压保留或明确恢复 Linux/macOS 可执行位并在发布前检查入口。安装后通过显式环境变量/路径指导定位，不暗改全局 PATH。

验收例：展示后清单被修改、批准摘要不匹配、平台不匹配、目标已存在、重定向到非允许主机和 ZIP 链接/穿越成员时都必须拒绝；没有批准摘要不产生网络请求。按 FD 既定授权，本轮只实现这些产品行为，不执行安装或下载。

Planner 已接受的补齐：ZIP-only；对已解析快照以 `json.dumps(sort_keys=True,separators=(',',':'),ensure_ascii=True)` 的 UTF-8 计算 SHA256，严格字段及平台检查；每跳 HTTPS 白名单，首版可直接拒绝全部重定向；全新 target 以 mkdir 预占后 exclusive 复制，拒绝已有目标；安装后指导用户设置明确环境变量。目录发布并非整目录原子操作：成功之前不得宣称安装可用；失败需清理本次创建内容或保留明确失败恢复路径，不能破坏已有依赖。ZIP 可执行位和入口检查仍属于实施约束。

## 实现约束与剩余风险

- 补充设计已说明使用 `Page.setDocumentContent` 输入自包含 HTML，并在载入前启用 Network 阻断。必须在发出 CDP 命令时继续分派异步事件，关联响应 ID；如果采用 Fetch 拦截，则必须在等待导航/运行脚本响应期间处理暂停请求，避免顺序等待死锁。只使用 Network 阻断时无需额外引入 Fetch。
- PDF 就绪应由单一可等待 Promise 覆盖 `mermaid.run`、`document.fonts.ready` 和全部图片 decode，检查 `exceptionDetails` 与资源失败；打印成功后检查真实 PDF。CSP 对可信内嵌 Mermaid 使用受控许可，对用户内容保持限制。
- Python 临时文件必须在启动后端前关闭，确保 Windows 子进程能写入。发布只使用不覆盖操作；发布成功而临时清理失败时，不应误报为未生成输出或删除最终文件。明确记录硬链接文件系统限制。
- Work Item 1.4 同时承担 HTML/CSS 资源解析、浏览器发现/生命周期、WebSocket/CDP 和就绪判断，工程体量明显高于简单“小/中”标签。实施前可拆为明确子项及接口，保持一个 FD 和既定总范围；不要求新建第二套计划。
- 未运行测试、浏览器或转换程序；未联网。手写 WebSocket/CDP、CSS 引用解析、ZIP 安全、Windows 文件锁和跨平台权限仍需独立实现审查，不能由本次设计审查背书。

## 本地证据

读取 FD-078、ISSUE-00008、`openspec/specs/cli-and-plugins/spec.md`、现有插件清单/执行入口，以及参考 Go 的 browser、renderer、images、document 和 assets 源码。参考 images 已有目录限制与本地嵌入思路，可复用；参考 document 的 Mermaid Promise 没有作为打印屏障，不能原样移植。没有实现或修改流程状态。

本轮仅三批本地读取/搜索。第二批误读已归档 FD-077 的旧路径，随后按实际归档路径读取；未重复原错误命令。
