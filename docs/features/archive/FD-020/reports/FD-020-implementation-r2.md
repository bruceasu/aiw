# FD-020 实施修订：仅支持直接 Key 认证

<!-- aiw-data: FD-020-implementation-r2.json -->

## 结果与授权

用户明确要求不再支持 key_hashes，并确认已把原摘要字符串作为 keys 使用。本轮落实 FD-020 revision 6 的工作项 1.6，替代 r1 的摘要兼容方案；历史报告保持不变。

## 修改

- config.go 删除 KeyHashes、crypto/sha256、encoding/hex 和摘要认证回退。配置及请求只使用 keys，认证仍为常量时间直接比较。
- 配置加载以原 Key 字符串检查同一主体及跨主体重复值，保留原长度/格式、禁用、主体/模型权限和限额检查。
- 严格 JSON 配置解析会拒绝已删除的 key_hashes 字段；原摘要值作为 keys 的值时按同一字符串匹配，客户端应发送该值，不再发送摘要计算前的旧 Key。
- 更新稳定 agent-proxy 规格、程序/插件说明及 FD 的决策、验收、TODO 和 Verification。用户的 gateway.json 已采用 keys，本轮只检查字段名，保留用户凭据值。

## 静态证据及实际命令

使用 Get-Content 与 rg 定向读取 FD、认证源码、稳定规格及说明，检查 key_hashes/KeyHashes/SHA-256/hex 引用；实现后执行一次只读源码和文档检查。认证路径没有摘要计算，配置重复检查也没有摘要计算；移除字段后 strictDecode 的未知字段规则继续生效。

运行 `python program/agent-gateway/scripts/compile.py`，退出码 0，Windows/Linux amd64 编译通过。已读取脚本，确认 GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local、CGO_ENABLED=0，输出 os.devnull，不保留发布产物。

读取 `aiw fd --help`、`aiw fd emit --help` 和 `aiw fd show FD-020`，并静态检查现有 CLI 的状态及角色约束；未运行 mutating FD 命令、测试、服务、最终产物构建、格式化、lint、vet、网络请求或 Git 写操作。

## 未解决的交接与限制

旧待认领 Tester 事件 FD-020-000005-implementation-ready 绑定 revision 5，本轮调整后的 revision 6 与其不匹配。当前 CLI 不支持 Pending Test 的范围修订或重新发起 Worker 交接；不得伪造测试拒绝、PM 决定或手写派发回执。本轮未认领该 Tester 事件，也未新建或声称完成独立验证，FD 保持待测试并在 TODO 记录该风险。后续需流程维护者提供正式恢复入口后才能交接当前修订。

源码编译通过不表示运行测试通过。现有二进制未替换，服务未启动；客户端须使用 keys 配置中的原字符串。日志/统计实现本轮未改，长期请求文件增长的既有风险仍在。
