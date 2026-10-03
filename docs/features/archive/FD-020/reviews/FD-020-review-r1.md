# FD-020 独立审查，第 1 轮

<!-- aiw-data: FD-020-review-r1.json -->

结论：通过独立审查，未发现阻断当前 FD 验收的实现缺陷。该结论由已执行黑盒证据及本次静态调用路径审查支持；14 个未执行场景仍未得到运行验证，分支覆盖未测量。

## 身份、交接及固定基线

- Reviewer：`fd020-reviewer-20261004-93d81a`，不同于 Worker 和 Tester。
- 精确认领：`FD-020-000010-test-accepted`，FD revision 10，摘要 `7fc3d07f5601a9388866844345f1a0f8ed90a681742749b7b83d1d5a31f5f928`。
- 审查提交：`c9b6d51cd2308e1ffbaf51755d697f6acda3ad83`；差异基线：父提交 `9ae91484d4835a50dad6d2aebbcad93c509583f9`。
- 此提交一次加入当前 Go 网关，不能将整份提交都归为 FD-020 独占改动。本次审查按 Worker r1/r2 声明范围及当前 FD 验收聚焦认证、HTTP 记录、存储和聚合；当前网关及 `openspec/specs/agent-proxy/spec.md` 对 HEAD 没有工作区差异。其它 FD 的脏文件未改动。
- 读取 implementation-r1/r2/r3；r1 的摘要兼容已由 r2 keys-only 决策替代，r3 为历史汇总和正式恢复证据，不视为新 Worker 完成。

## 实现与契约依据

| 检查 | 静态证据及结论 |
|---|---|
| 全入口日志、主体和脱敏 | `server.go:ServeHTTP` 在拒绝分支前输出开始并注册统一 defer；完成及可捕获 panic 均进入 `observation.go:finishHTTPRequest`。主体只来自认证；路径/方法白名单归一，逻辑模型只取已配置值。日志序列化固定 HTTP 字段，不包含请求正文、凭据、任意查询或响应正文。 |
| SSE、交付与取消 | `responses` 先保存 `execute` 的 State/Code/Started，已启动 SSE 的最终事件不会覆盖为 HTTP 成功；包装 writer 捕获 Write/Flush 错误，send 的 deadline 错误由 deliveryError 捕获。成功执行交付失败可改变 HTTP 状态结果，但执行 metadata/额度保持；已失败执行仍保留执行结果及独立 delivery_failed。 |
| 统计口径及跨日 | `AggregateHTTP` 使用 received_at 时区日期且只筛当前主体；`Aggregate` 仅累计 StartedAt 非空并按其日期归属，未知 usage 独立计数。`Reserve/countLocked/Put` 使用同一锁，保留跨日预留保守占额。 |
| 恢复与损坏拒绝 | 执行 metadata 先加载，再处理 HTTP 文件。HTTP in_progress 恢复 interrupted/request_interrupted；finished_at/duration_ms 保持 null；execution_started 根据匹配执行记录 StartedAt 确认。manifest/记录严格解码、版本、ID、状态与时间必要字段被检查；已标记后目录缺失拒绝启动，旧存储只增量创建目录和标记。 |
| 写失败阻止新执行 | `PutHTTPRequest` 在 atomicJSON 失败时锁内设置 fatal，返回固定 storage_error；入口和模型更新失败提前返回，终态失败输出脱敏 request_audit_failed，不宣称落盘成功；后续入口/Reserve 拒绝。原 in_progress 可在重启按中断恢复。 |
| keys-only 认证 | `config.go` 没有 KeyHashes/摘要计算；strictDecode 拒绝 key_hashes。所有配置 Key 经长度/空白和全局重复检查，同主体重复也拒绝；authenticate 用 subtle.ConstantTimeCompare 直接比较，禁用主体不认证，原摘要字符串按普通不透明 Key 处理。 |
| 文档及兼容 | README、插件 README、稳定 agent-proxy 规格与当前 keys-only、独立 HTTP 元数据/分组约定一致。旧执行 groups 和额度字段保留。没有变更依赖或扩展公共能力。 |

## 测试与授权证据审查

已检查 `FD-020-test-report-r1.md/json`、raw-r1/r2、authorization-r1/r2、当前测试源码及 PM decision-r1。两次精确命令都是 `python tests/fd020_observability_blackbox.py`，绑定事件 8、revision 8、相同 Tester session；r2 当前源码 SHA256 为 `2b4e2945c77ae484b129d5534979acb9dbe3ff9980891ca0d62b18c001919944`，与第二次授权一致。脚本使用临时目录、离线 Go 编译、模拟 Node 后端和 loopback HTTP，不调用真实 AI；副作用已写在授权中。

raw-r1 保留 28 通过、1 失败；该失败源于测试擅自限定并发拒绝为 429，而公开契约未限定。r2 改为 429/503，同时检查 rejected、未启动及非空错误码，29/29 通过；没有将首轮失败抹除或将未执行场景计为通过。报告逐场景列出 54 个可观察需求，40 个有执行证据（74.07%），14 个未覆盖，测试数和场景数使用不同分母。

PM 明确接受分支覆盖未测量的例外。Reviewer 尊重该决策，并静态核对未测路径；不把静态检查计为新增测试覆盖。AI 成功/token、SSE、取消/交付、跨日、旧存储升级、写失败、请求记录损坏、同主体重复 Key、Key 边界、执行确认恢复、全局并发、响应正文脱敏、全部提前返回日志仍有运行风险。

## 实际操作与限制

实际执行：定向 `Get-Content`、`rg`、`Get-FileHash`；`git rev-parse HEAD`/`HEAD^`、`git status --short`、限定范围 `git log` 和 `git diff`；`aiw fd --help`、`aiw fd claim --help`、`aiw fd emit --help`；精确 `aiw fd claim FD-020 FD-020-000010-test-accepted --session fd020-reviewer-20261004-93d81a`。首次读取误用不存在的 `.ai/fd/FD-020/receipt.json`，随后定位真实 events 文件；没有推测或伪造回执。

本次只写此独立报告及 JSON、更新 FD TODO/Verification，并以该报告发布 verification-passed。未运行测试、服务、编译、最终构建、格式化、lint、vet、网络、Git 写操作或归档；已有编译和运行结果仅引用先前角色的证据。当前实现尚未证明真实 Codex、生产部署、高流量或长期文件增长的运行表现。
