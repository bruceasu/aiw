# FD-025 Planner 测试授权 R2

<!-- aiw-data: FD-025-test-authorization-r2.json -->

批准同一 Tester、FD Revision 4 与实现事件绑定的精确命令 `python tests/fd025_shutdown_blackbox.py` 在仓库根目录进行唯一一次测试入口纠正重试。首次执行只在启动就绪场景失败，10 个后续场景未运行，历史 raw 保留。

已审阅纠正片段：向 Windows Popen 传递原始 cmd `/d /s /c` 命令 string，避免 argv list 的二次引号转义；临时 stderr 文件不输出原文，只提供退出码与允许的错误类别。授权绑定新的测试源码 SHA-256，安装二进制及 FD 摘要未变；授权文件和原始输出使用 R2 独占路径，不覆盖 R1。

范围、临时目录清理边界、隐藏后台运行、loopback 临时端口、合成 Key、零真实模型、无下载/构建/强杀/遗留锁删除均保持 R1 限制。Planner 直接批准该可检查、聚焦、临时文件内的低风险入口纠正；不允许第三次执行或扩大验证范围。

