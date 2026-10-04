# FD-025 修复交接 R2

<!-- aiw-data: FD-025-implementation-r2.json -->

来源为已 claim 的 `FD-025-000006-test-rejected`，Worker 为 `fd025-host-20261004-729bf1`。修正 Windows 停止完成及重复停止判断：网络拒绝按 Winsock 10061 识别，经 errors.Is 保留包装错误匹配。Linux 使用原 ECONNREFUSED。只新增平台内部辅助函数，不改变控制接口、配置、超时或认证行为。

静态追踪两个错误匹配调用点、锁释放和监听停止组合条件；默认启动与进程取消流程保留。此前失败测试不变成通过。修复后 `python program/agent-gateway/scripts/compile.py` 退出 0，Windows/Linux amd64 无最终产物编译通过；按用户授权 `cmd.exe /d /c build.bat bin` 退出 0，定向安装到 C:\green\aiw，未复制 gateway.json。

交独立 Tester 作同范围修复验证，须新事件、FD 和二进制摘要绑定授权。活动模型取消、Linux 运行、故障期限及审计失败未实测，由 Reviewer 静态追踪并保留风险。不运行 Go 测试、扩大构建、下载、真实模型、Git 写操作。
