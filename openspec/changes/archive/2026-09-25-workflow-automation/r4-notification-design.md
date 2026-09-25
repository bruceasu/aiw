# R4 本地通知适配协议

2026-09-18，按用户“继续推进”授权闭合 R4 设计，FD_APPLIED。供 E07 使用，追踪 SW28–SW30/AC28–AC30/AX04，依赖 E05 登记/恢复与 R3 资源政策。这里只写协议，不发送消息、不读取凭据、不创建实际目标配置。

## 本地证据与兼容策略

`plugins/aiw-notify.py` 当前从 stdin 接收通知并返回 accepted/receipt，仅有本地确认；`internal/notification/plugin.go` 将该回执作为成功协议。`plugins/send_teams_msg.py` 支持 text/card 模式、隐式加载 .env/send_message.env、CLI > env > default 解析参数，关闭证书校验，并将 URL、远端错误体或响应打印出来。新受管协议不得继承这些隐式行为。

为 aiw-notify 引入显式 schema_version=2 请求与结果；保留旧 v1 解析仅作旧客户端兼容，不将旧 accepted/receipt 解释为外部发送证据。新 Core 必须与插件 v2 同时接通；收到旧/未知/不匹配结果时记录 unknown，不自动降级或重发。插件元数据须准确声明潜在网络副作用，不能继续以只读确认器描述 Teams 通道。

Teams 仍调用项目解析后的 `send_teams_msg.py`；增加隔离的 `--managed-json` 入口，其他 argv 不携带消息/目标/密码。脚本现有独立 CLI 可保留，但受管入口不得调用旧 main 的默认解析或不验证 TLS 的 helper。若插件/脚本有用户改动，基于实际内容摘要选择显式受支持版本，不覆盖文件或偷偷换用其他安装位置；不兼容则 configuration_failure。

## 1. 项目配置与冻结目标

项目根现有 aiw.toml 的 `[notifications]` 为受管配置唯一入口，schema_version=1、enabled=false 默认关闭；channel 仅 console/teams。无显式配置不得选择隐式 Teams 目标。字段为 enabled、channel、target_id、teams_url、credential_env、ca_file、sender、timeout_seconds；未知字段拒绝，console 不要求 Teams 字段，Teams 必须具备明确目标与 credential_env。timeout_seconds 默认 30，首期允许 1–30，不自动提高。

teams_url 必须是完整 HTTPS 文本发送端点，不从旧默认 host/profile/channel 推导；禁止 userinfo、fragment、凭据查询串，URL 不承担凭据传递。credential_env 只保存环境变量名；ca_file 可选，按项目根解析到明确可信 CA 文件，缺失/不可读不回退为关闭验证。sender 可为空，其他必需配置缺失进入 waiting，不消费真实发送次数。

冻结配置引用包括 schema、规范化配置摘要、通道、target_id、明确端点及 CA 身份；消息持久保存事实 ID、内容版本、纯文本和该目标引用。目标不得包含秘密；部署环境若将目标地址本身当作 bearer secret，则当前协议不适用，等待独立秘密引用适配，不能将其落入工件。发送前配置必须仍与冻结引用一致；变更目标不改投旧消息。凭据值可轮换，值不入摘要，只在发送前读取同一环境变量引用。

## 2. 进程请求和结果

Core → aiw-notify `dispatch --json`：stdin 单个 UTF-8 JSON 对象，字段固定 schema_version=2、notification_id、attempt_id、content_digest、text、channel、target_reference、config_reference。插件从已核验项目配置解析实际参数，不信任消息提供的任意程序路径。正文只用 stdin，禁止 shell 拼接、自动上传文件或展开正文中的路径。

aiw-notify → Teams 脚本 `--managed-json`：stdin 固定 schema_version=1、notification_id、attempt_id、text、endpoint、credential_env、ca_file、sender、timeout_seconds。使用参数数组、冻结项目根及脚本绝对路径，传递最低必要环境（约定 credential_env 的值与必要运行环境），不读 .env、不用默认凭据/默认目标。凭据不进入 argv、stdin、stdout、stderr、Task 工件或消息正文；脚本在进程环境内取值，仅按现有 text 请求格式置入 HTTPS 请求体的 ePassword 字段。

结果使用单个 JSON 对象：schema_version=2、notification_id、attempt_id、content_digest、attempted（布尔）、outcome（attempted/network_failure/configuration_failure/permission_failure/service_failure/unknown）、error_code（固定枚举或空）、http_status（整数或 null）。脚本结果由插件归一化并补齐 Core 的引用；禁止任意远端 response body/error 字符串进入结果。stdout 仅协议，stderr 只允许固定脱敏诊断码；console 把文本写入独立本地显示流，协议结果仍只说明本地尝试。

正文 UTF-8 上限 16 KiB，协议请求/结果各 64 KiB；输出超限/非法 UTF-8/多 JSON 值/字段未知或 ID 不匹配为协议故障。不得为满足正文上限截断关键状态并冒充完整通知：渲染器产生本地简要文本，保留事实状态和待办；无法在限额内生成则 configuration_failure，不调用模型扩写。父进程单次等待上限 45 秒覆盖 30 秒请求和启动收尾；超时/退出结果不明标 unknown，保存派发事实，不自动重发。

## 3. TLS、HTTP 与错误分类

受管发送采用验证主机名和证书的 TLS 默认上下文，可显式加载可信 CA；禁止 CERT_NONE/check_hostname=false。禁止自动 HTTP redirect，3xx 记 service_failure，避免目标被重定向。只发送 text 格式，禁用 card、附件和自动链接扩展。真实服务兼容性未验证时不宣称已送达。

| 本地观察 | outcome / 是否自动重发 |
| --- | --- |
| console 显示成功或 HTTP 2xx | attempted；不声称已送达，不重发 |
| 有类型依据的 DNS/连接拒绝/网络不可达/连接重置 | network_failure；首次才允许 60 秒后一次重发，重置可能重复投递的风险已接受 |
| 明确连接建立前超时 | network_failure；只有适配器能证明发生阶段时适用 |
| 读响应超时、进程中断、无法确定阶段的超时或无合法结果 | unknown；不自动重发 |
| 缺配置/凭据/CA、协议版本不支持、证书验证失败 | configuration_failure；不关闭 TLS、不自动重发；确认未尝试的缺配置可以等待补齐 |
| 宿主禁止网络/进程或 HTTP 401/403 | permission_failure；不扩权、不重发 |
| HTTP 3xx、其他 4xx/5xx（含 429）或明确服务拒绝 | service_failure；不自动重发，不按 Retry-After 扩大既定政策 |

错误分类只使用异常类型、可验证阶段或 HTTP 状态，不通过自由文本“包含 network”猜测。未知异常归 unknown，禁止返回远端响应体、原始 URL、请求体或环境快照。插件/脚本内部不做 HTTP 自动重试，所有重发由 Core 管理。

## 4. 计次、迁移与恢复

Core 在派发前完成配置/宿主预检并持久预约原消息的 attempt_id；缺配置且确认未执行保持等待不计真实发送。启动插件后结果未知保留预约，不靠重启释放。合法结果证明 attempted=false 且未进入外发/console 显示时可对账释放；进入发送调用的 DNS/连接失败仍计一次尝试。

每内容版本总发送最多两次，首次明确 network_failure 才保存 retry_due_at=该失败持久记录时间+60秒；再次启动只读原 due time 与次数。due time 是最早允许时刻，不是准点送达保证，时钟倒退只能延迟。第二次任何结果均终止自动发送。修改展示时间、配置、重新渲染、重建索引不创建新事实或新额度；重要晚到更新须有新的实质事实来源及显式关联。

发送前复核 Stop、enabled、提醒仍有效及冻结目标。主动禁用取消尚未派发的历史完成通知，不全量补发；人工提醒仅在仍有效且原额度允许时恢复。对在途消息只对账，不能声称撤回。网络能力在宿主不可用时等待/失败，不以通知规格作为本轮联网权限。

旧 delivered/receipt 保留为 legacy-local-ack 证据性质，不推定远端已收到，不为“补正状态”自动外发旧消息。旧实际发送次数或分类无法证明则 unknown，禁止补零并自动发送。新 outbox 使用按消息内容版本的次数与错误类别，不沿用 Task 全局通知重试计数；迁移不得扩大旧已用额度。

## TODO 与 Verification

- [x] 固定配置、stdin 协议、环境凭据、TLS、分类、迁移及唯一重试所有者。
- [x] E07 同时实现 Core outbox、Go 插件适配器、aiw-notify v2 与 Teams 受管入口；保留旧独立 CLI，未知脚本版本拒绝执行。实现及验证缺口见 [E07 实施记录](e07-implementation.md)。
- [ ] 在已有 Store/通知派发接缝以进程/传输替身检查：缺配置不计次、冻结目标、两次上限、60 秒跨重启、TLS/redirect/HTTP 拒绝、未知不重发、协议错配、输出超限与敏感内容不外泄。

本次只读分析了现有源码，没有读取 .env 或真实凭据，没有运行通知脚本、服务探测、网络请求或测试。真实 Teams 发送验证须用户明确指定目标并授权发送；这不阻止接口实现或替身场景设计。

%% ACTIVATION_PENDING：E07 接线已实现，编译由 supervisor 执行，运行验证未执行；真实目标、凭据引用、可信 CA 和宿主网络能力留作显式部署配置，默认关闭。R4 本地契约已确定，不把实际发送成功作为设计闭合的前置条件。
