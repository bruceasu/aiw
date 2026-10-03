# FD-025 实现记录 R1

<!-- aiw-data: FD-025-implementation-r1.json -->

Worker：`fd025-host-20261004-729bf1`；来源：`FD-025-000003-design-ready`。新增本地认证停止入口、stop CLI 和停止脚本，修正启动脚本命令名。

`POST /internal/shutdown` 复用现有取消、进程树清理和 HTTP Shutdown；等待定期内容清理结束后才释放锁。loopback + 现有 enabled Key、POST、空 body、无 query 是控制边界。审计失败不阻止安全本地关停，其他 AI 路径仍保持存储门禁。

停止 CLI 不打开存储或要求后端可用；只从配置读取凭据，禁止重定向，不自动重试。接收 202 后有限等待监听和状态锁释放；失败不强杀、不删除锁。脚本转发参数和退出码。`build.bat bin` 定向安装 Gateway 文件，保留配置并关闭 Go 下载。

实际命令：`python program/agent-gateway/scripts/compile.py` 退出 0，Windows/Linux 无产物编译；`cmd.exe /d /c build.bat bin` 退出 0，按用户授权安装 Windows/Linux AIW 与 Gateway 到 C:\green\aiw。构建是用户明确授权的例外。

静态审阅：控制入口认证/loopback/输入检查，取消与 finally/defer 路径，HTTP 和定期清理完成后释放锁，固定安装目标及配置保护。未运行 Go 测试、lint、formatter、下载或真实模型请求，无 Git 写操作。活动模型取消的运行证据仍缺失；Tester 与 Reviewer 待独立验证，不以实现进度标记 Complete。
