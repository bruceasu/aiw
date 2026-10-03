# FD-020 独立测试报告，第 1 轮

<!-- aiw-data: FD-020-test-report-r1.json -->

## 结论

修正后的本地黑盒测试 29 项全部通过，命令退出 0；需求场景覆盖为 40/54（74.07%），业务代码分支覆盖未测量。建议 PM 在明确接受未覆盖风险后进入独立审查，不能把未测场景称为通过。

首轮实际执行 29 项：28 通过、1 失败，退出 1。唯一失败是测试将并发拒绝状态码擅自固定为 429；公开契约只要求即时拒绝。经 Planner 检查后改为接受 429/503，同时要求 rejected、未执行和非空错误码，并按新的源码摘要授权重跑一次，29 项全部通过。首轮原始证据保留，实施源码没有修改。当前摘要计数指修正后这一轮，不覆盖首轮历史。

## 绑定与依据

- FD：FD-020，revision 8。
- FD SHA256：1c878661ac18e3aa6be4ea7c741e673229bcbd9496e4567320e1a16f614d4115。
- Implementation event / source event：FD-020-000008-test-requested（正式 PM 刷新，保留历史 Worker provenance）。
- Tester session：fd020-tester-20261004-c6a92b；已认领精确事件。
- 依据：当前 FD、openspec/specs/agent-proxy/spec.md、program/agent-gateway/README.md。没有阅读网关实施源码。
- 最终测试源码 SHA256：2b4e2945c77ae484b129d5534979acb9dbe3ff9980891ca0d62b18c001919944。

## 场景与证据

每项场景独立列出；一个测试可验证多个独立行为，测试数与需求场景数采用不同分母。已执行需求场景使用原始 r2 cases 结果及测试中的断言作为证据。

| 场景 | 可观察行为 | 结果 / 用例 |
|---|---|---|
| models_success | Models 成功查询记录 HTTP 200 与已知处理耗时 | 通过：models_success |
| stable_principal | 已认证请求关联稳定主体和服务端 request-id | 通过：models_success |
| missing_key | 缺少 Key 返回401且主体为空 | 通过：unauthenticated |
| disabled_key | 禁用主体的Key返回401且主体为空 | 通过：disabled_key |
| unknown_key | 未匹配Key返回401且主体为空 | 通过：unknown_key |
| opaque_digest_key | hex样式原摘要可作为不透明Key直接认证 | 通过：former_digest_key |
| no_digest_conversion | 请求发送配置Key的SHA256不能获得认证 | 通过：digest_conversion_not_accepted |
| invalid_parameter | 未知执行参数在启动前400拒绝并保留主体 | 通过：parameter_rejection |
| model_rejection | 未配置模型拒绝且不记录任意模型原文 | 通过：model_rejection |
| invalid_usage | 非法日期查询拒绝且保留主体和Usage请求记录 | 通过：usage_query_rejection |
| unknown_route | 未知路径归other且不记录路径查询原文 | 通过：unknown_route |
| wrong_method | Models错误方法拒绝并持久化请求 | 通过：unknown_method |
| method_other | 未知BREW方法归other | 通过：unknown_method_normalization |
| caller_b | 第二主体使用自己的Key成功调用Models | 通过：caller_b_models |
| usage_principal | Usage响应只标识当前主体 | 通过：usage_isolation_and_no_execution |
| usage_isolation | Usage GET Models分组不包含B或未认证请求 | 通过：usage_isolation_and_no_execution |
| no_execution_groups | 执行前拒绝及查询不新增执行groups或executed数 | 通过：usage_isolation_and_no_execution |
| no_rejection_quota | 执行前拒绝及查询不消耗每日额度 | 通过：usage_isolation_and_no_execution |
| query_in_progress | Usage查询自身按in_progress计入HTTP分组 | 通过：usage_isolation_and_no_execution |
| method_groups | POST拒绝与未知方法拒绝形成独立方法分组 | 通过：usage_isolation_and_no_execution |
| error_status_counts | HTTP分组包含拒绝数、400状态计数和错误码计数 | 通过：http_status_error_duration_aggregation |
| duration_samples | 已知终态数等于duration_samples，总时长非负 | 通过：http_status_error_duration_aggregation |
| backend_failed | 模拟后端失败记录failed、502和确认启动 | 通过：backend_failure_and_linkage |
| execution_linkage | HTTP request-id与既有执行元数据文件名一致 | 通过：backend_failure_and_linkage |
| timeout | 模拟后端超时记录timed_out、504和确认启动 | 通过：backend_timeout |
| concurrency | 主体并发即时拒绝、错误码非空且不启动执行 | 通过：concurrency_rejection |
| failed_quota | 已启动失败或超时均扣一次额度且不退回 | 通过：failed_execution_quota_unknown_tokens |
| unknown_tokens | 缺失token单独计unknown_usage_requests | 通过：failed_execution_quota_unknown_tokens |
| log_linkage | Models开始与结束日志通过request-id关联 | 通过：log_linkage_and_privacy |
| redaction | 日志和HTTP记录不含测试Key、Authorization、提示词及查询路径原文 | 通过：log_linkage_and_privacy |
| old_key | 轮换重启后旧Key不能认证 | 通过：rotation_old_key_rejected |
| new_key | 新Key关联原稳定主体 | 通过：rotation_new_key |
| rotation_quota | 轮换保持已有实际执行额度 | 通过：rotation_quota_preserved |
| rpm | RPM即时拒绝且不启动执行 | 通过：rpm_rejection |
| recovery_state | 持久in_progress重启变为interrupted/request_interrupted | 通过：interrupted_recovery |
| recovery_time | 恢复未完成请求不编造finished_at和duration_ms | 通过：interrupted_recovery |
| duplicate_keys | 不同主体重复Key拒绝配置加载 | 通过：duplicate_keys_rejected |
| removed_field | keys存在时也拒绝已移除key_hashes字段 | 通过：key_hashes_rejected |
| corrupt_manifest | HTTP存储标记损坏拒绝启动 | 通过：corrupt_manifest_rejected |
| missing_directory | 已升级requests目录缺失拒绝启动 | 通过：missing_upgraded_directory_rejected |
| ai_success | AI Responses成功结果与已知token累计 | 未覆盖：没有实现模拟成功JSONL协议，不调用外部AI |
| sse_failure | SSE已输出200后失败/超时仍可辨认执行结果 | 未覆盖：本轮只非流式HTTP |
| cancellation | 客户端取消单独计cancelled并保留已启动额度 | 未覆盖：未引入断连/取消测试 |
| delivery | 交付失败单独计delivery_failed且不返还额度 | 未覆盖：未模拟传输故障 |
| cross_midnight | HTTP接收日与执行启动日跨午夜分属两日 | 未覆盖：未修改系统时间或注入服务时钟 |
| legacy_upgrade | 升级旧存储保持历史执行、主体和额度 | 未覆盖：未建立旧版本执行存储 |
| write_failure | HTTP入口/终态写入故障脱敏并阻止新执行 | 未覆盖：未模拟存储权限或IO故障 |
| record_corruption | 请求记录损坏及未知schema拒绝启动 | 未覆盖：仅覆盖manifest损坏和目录缺失 |
| same_principal_duplicate | 同主体重复Key拒绝加载 | 未覆盖：仅验证跨主体重复Key |
| key_bounds | 非法长度/空白Bearer Key拒绝且脱敏 | 未覆盖：仅验证缺失、未匹配、禁用及摘要转换 |
| confirmed_recovery | HTTP恢复actual executed只依据已确认执行元数据 | 未覆盖：本轮恢复的是未执行Models记录 |
| global_concurrency | 全局并发即时拒绝及其统计 | 未覆盖：仅覆盖主体并发 |
| response_redaction | AI响应正文不出现在日志或HTTP元数据 | 未覆盖：模拟后端失败没有取得响应正文 |
| every_route_logs | 所有路由提前返回均有开始/终止日志 | 未覆盖：只对Models成功逐条断言日志对；其他请求断言持久化终态 |

纯内部要求“常量时间比較且无摘要计算”由 Reviewer 静态检查，不计入上述可观察场景分母；公开认证行为已验证直接Key与计算后的SHA256区分。Go 黑盒进程未加分支插桩，无法提供分支覆盖率；本轮未用语句覆盖替代分支覆盖。

## 命令、原始证据与授权

两次工作目录均为 D:/03_projects/AI-tools/aiw：

1. `python tests/fd020_observability_blackbox.py`，首轮退出 1，stdout 为 `{"passed": 28, "failed": 1}`，约 6.1 秒。授权 FD-020-test-authorization-r1.md/json；原始证据 FD-020-test-raw-r1.json。
2. 同一命令，修正重跑退出 0，stdout 为 `{"passed": 29, "failed": 0}`，约 5.7 秒。授权 FD-020-test-authorization-r2.md/json；原始证据 FD-020-test-raw-r2.json。

脚本实际调用离线 `go build -o <测试临时目录>/gateway.exe .`（cwd program/agent-gateway，设置 GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local、CGO_ENABLED=0），以及临时网关 `start --config <测试临时目录>/gateway.json`。构建均退出 0。临时 node 后端只有限延时退出，没有调用真实 Codex 或外部 AI。网关只监听随机 loopback 端口，配置/状态/工作区/日志/可执行文件均在测试专属临时目录；结束已清理。测试原始证据仅保留用例结果，不包含测试 Key、配置或日志原文。

另实际执行只读文档/回执读取、CLI帮助、源码摘要读取、精确 claim；报告交接使用正式 emit。未执行仓库全量测试、分支覆盖、发布构建、下载、外部网络、真实凭据访问或 Git 写操作。

## 剩余风险

成功AI执行、流式交付、取消、跨日、写入故障及部分存储恢复分支未覆盖。测试没有证明所有提前返回路径都输出成对日志，也没有测量生产吞吐、长期存储增长或恶意后端进程行为。HTTP 分组耗时只验证已知样本计数与总值非负，不证明时钟精度。原始证据为测试断言结果，没有保存HTTP正文或完整日志快照；独立审查需结合测试源码和实施差异评估其充分性。

