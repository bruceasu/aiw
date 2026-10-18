# FD-078 实现交接

<!-- aiw-data: FD-078-implementation.json -->

- Worker：fd078-host-20261018-a83f92；来源 FD-078-000003-design-ready；代码提交 fd13b2c；基线 bc918f8。
- 已实现 Python file-convert 插件、参数/帮助/JSON/不覆盖输出、MarkItDown/antiword/Pandoc 适配、TXT 语法节点正文、Go 离线 HTML 和基于本机浏览器 CDP 的 PDF、依赖指导与摘要授权 ZIP 安装。
- 资源来自本地参考源码及 Goldmark 1.7.13 缓存、Mermaid 11.12.1；保留许可。未增加 AIW 根 module 依赖，未下载。
- 静态检查：参数先验证、二进制 stdout 先拒绝；后端临时结果校验后 hard link 发布；Pandoc 检查快照和自包含 HTML；HTML 拒绝主动内容并嵌入本地资源；CDP 命令循环处理错误事件；安装清单规范摘要、授权后下载、哈希/ZIP检查、独占新目录及逐文件复制。
- 实际命令：`python src/programs/md-converter/scripts/compile.py`，退出0。仅编译 Python 源文本和当前平台 Go module 到 NUL，网络关闭，无保留执行文件。已执行一次暂存 diff 静态检查和本地 Git 提交。
- 未运行：测试、转换/删除/安装运行验证、PDF视觉、跨平台编译/运行、formatter/lint/vet、最终构建、网络、依赖下载、全局安装和 push。
- 剩余风险：手写 CDP/WebSocket 与浏览器兼容、字体和真实排版；MarkItDown/antiword/Pandoc 未集成实测；安装主机白名单不替代用户确认发布来源；官方入口可能重定向，首版需可信直链；hard link 文件系统支持；并发恶意同用户文件替换不是 OS 沙箱。
- 当前完整范围交给独立 Reviewer，未声称其已通过。Go helper 尚需授权构建/分发才可运行相应路径；已安装 AIW 不会被源码交付自动更新。
