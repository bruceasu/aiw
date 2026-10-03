# FD-025 修复交接 R3

<!-- aiw-data: FD-025-implementation-r3.json -->

来源为已 claim 的 FD-025-000009-test-rejected，Worker 为 fd025-host-20261004-729bf1。Python 插件默认配置追加位置现在同时识别 start/stop，确保动作仍位于 argv 首位，避免无参数 stop 变成 Go 解析剩余参数。显式配置不变。

R2 Windows 拒绝连接识别修复和 Windows/Linux 无产物编译结果保留，此次一行 Python 入口修改不改变 Go 源。用户授权 build.bat bin 安装后交独立 Tester 作唯一产品修复运行验证；需要新版本摘要及入口摘要绑定授权。R3 测试报告明确未运行，不能当作运行失败或通过。

不扩大网络、依赖、Git、模型执行范围；活动模型取消、Linux运行及故障注入保留静态审查风险。

实际执行 `cmd.exe /d /c build.bat bin` 退出 0，安装修复入口并保留现有配置；Python 内存 `compile()` 对该文件编译退出 0，不执行入口、不生成字节码产物。精确命令见 JSON。
