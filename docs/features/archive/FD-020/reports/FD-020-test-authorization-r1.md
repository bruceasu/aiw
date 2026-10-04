# FD-020 Planner 测试授权，第 1 轮

<!-- aiw-data: FD-020-test-authorization-r1.json -->

批准唯一命令 `python tests/fd020_observability_blackbox.py`，工作目录为仓库根目录。依据是用户本轮明确要求 `test then review FD-020_AGENT_GATEWAY_REQUEST_OBSERVABILITY`，并已逐段检查测试源码及修正后的统计计数断言。授权绑定 FD revision 8、事件 FD-020-000008-test-requested、Tester session fd020-tester-20261004-c6a92b 和下列源码摘要。

预计 20–90 秒，临时 Go 编译最多 120 秒。测试设置 GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local、CGO_ENABLED=0；临时编译网关、启动随机 loopback 端口，通过 HTTP 请求验证公开行为，模拟 node 后端仅有限延时退出。配置、凭据、日志、状态及可执行文件全部为测试专属临时文件，结束时终止测试进程并清理；只在仓库保存 FD-020-test-raw-r1.json。编译可能写本机 Go 编译缓存。不得访问真实 gateway.json、真实凭据、外部 AI、远端服务、依赖下载、发布或 Git 写操作。此记录不授权扩大测试范围或重复运行。

范围包括认证、禁用/重复 Key、key_hashes 拒绝、HTTP 日志与脱敏、请求统计及额度隔离、后端失败/超时、RPM/并发拒绝、轮换、中断恢复、存储损坏及目录缺失。成功后端、SSE、取消、跨午夜、写入故障等未执行场景必须如实记录。分支覆盖未测量不得称为通过。
