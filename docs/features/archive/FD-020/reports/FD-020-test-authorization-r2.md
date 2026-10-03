# FD-020 Planner 测试授权，第 2 轮（一次修正重跑）

<!-- aiw-data: FD-020-test-authorization-r2.json -->

批准在仓库根目录再次运行唯一命令 `python tests/fd020_observability_blackbox.py`，这是针对测试源码修正的一次重跑。事件、FD revision/digest、Tester session 及用户授权依据与 r1 相同；本次绑定新源码 SHA256 2b4e2945c77ae484b129d5534979acb9dbe3ff9980891ca0d62b18c001919944。

首轮 28 项通过，并发拒绝用例因测试擅自固定 HTTP 429 而失败；公开契约未固定并发拒绝状态码。修正接受 429/503，并额外要求 rejected、execution_started=false 和非空错误码；失败诊断保留实际状态和 traceback。已检查该增量及源码摘要，测试范围和临时副作用不变。原始 r1 证据保持，输出写入 FD-020-test-raw-r2.json。

本次仍禁止真实凭据、外部 AI、远端服务、下载、发布和 Git 写操作，临时编译与 loopback HTTP 仅用于本次聚焦测试。若再失败，不再自动重跑或扩大范围。首轮失败必须保留为测试假设修正的历史，不得篡改为首轮通过。
