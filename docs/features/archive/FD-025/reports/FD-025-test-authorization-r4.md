# FD-025 Planner 产品修复验证授权 R4

<!-- aiw-data: FD-025-test-authorization-r4.json -->

审阅测试源码及变更后，批准根目录精确命令 `python tests/fd025_shutdown_blackbox.py` 一次。绑定 FD Revision 10、事件 FD-025-000010-implementation-ready、独立 Tester 和测试/已安装 exe/入口三份摘要。此前 R2 是入口纠正重试，R3 未执行；本次是在真实 Windows 错误码和默认配置入口修复后允许的一次产品验证，不允许失败后继续重试。

原 11 个临时生命周期行为保留，新增同一临时目录内复制已安装 exe、Python 入口及合成配置后的无 config stop。均不使用真实 Key、状态或 43127，不调用模型；只访问独立 loopback 临时端口，不下载、不构建、不改权限、不强杀、不删除活跃锁。最终清理仅限 OS 临时目录内确认已停止且无锁的自有目录。原始结果使用独占 R4 路径；不覆盖历史。

Planner 依低风险例外批准可检查的离线本地控制测试；真实服务、真实模型、故障注入与更大范围测试不在此次授权内。
