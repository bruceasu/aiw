# E07 文本通知实施记录

2026-09-18；Task/change：workflow-automation；Work Item：wi-0009 / 1.7。

## 实现与证据

- `internal/workflow/notification.go`：复用 outbox；v2 绑定事实、正文摘要、冻结目标、配置引用、尝试 ID 与结果。真实发送最多两次，仅第一次明确 `network_failure` 保存失败记录后至少 60 秒重发；不使用 Task 全局通知恢复余额。缺配置等待且不计发送，未知结果保留预约，不因重启释放额度。旧记录禁止重放，旧 delivered 在读取时显示 `legacy-local-ack`，原 receipt 保留。
- `notification_facts.go` 与 Store 提交接缝：完成事实、可行动 Gate 和重要晚到辅助/交付结果在同一提交登记；普通进度不通知，晚到消息关联原通知。正文分开展示开发、交付、知识和审查，不包含任意诊断正文或虚构链接。过期提醒取消尚未发送项，在途结果只对账。辅助失败不撤销已接受开发成果。
- 登记时保存项目 `aiw.toml` 的有界内容摘要，避免异步解析前配置被替换导致旧事实改投。首次合法预检后保存规范化配置和目标引用，后续按这两个引用检查。配置文件不存在时，完成通知取消，当前有效人工提醒可等待配置。此登记保护在首次绑定前保守地覆盖整个配置文件，包括无关编辑；不自动猜测原目标。
- `internal/notification/plugin.go`：只调用项目根 `plugins/aiw-notify.py`；两份脚本按 LF 规范化源码 SHA-256 匹配明确支持的版本，未知改动不覆盖、不回退全局插件。正文只经 stdin；UTF-8、单对象、完整字段、重复键、未知字段、身份、错误码和 64 KiB 输出均受检查。每次进程最长 45 秒，非法/缺失/超限结果为 unknown。
- `plugins/aiw-notify.py`：v2 console/Teams 统一协议；v1 仅兼容旧本地确认。显式 `[notifications]`，默认关闭；严格字段和 1–30 秒请求限制；正文最多 16 KiB。Teams 子进程只有必要环境和指定凭据引用，不读 .env，不拼 shell，不上传文件。凭据值不进入 argv/stdin/结果。Python 3.11+ 的标准库 `tomllib` 负责读取配置，无新增项目依赖。
- `plugins/send_teams_msg.py`：新增隔离的 `--managed-json` text 入口；保留旧独立 CLI。受管入口验证 TLS/主机名，可用显式 CA，禁用隐式代理、重定向及内部重试；不读取响应体。DNS/拒绝连接/重置/不可达按异常类型分类，未知阶段超时不重发，HTTP 拒绝与权限拒绝不当作网络故障。
- `internal/notification/host.go` 和 E05 helper/config 接线：共享既有独立进程与项目宿主锁；通知和模型工作分别执行，通知无需模型配置；每轮最多处理 32 项并受 helper 十分钟上限约束。等待原 due time，不刷新额度；每次真实发送前复核配置、Stop、提醒有效性。其余等待项留给下次受管启动。console 使用独立本地显示流；后台宿主继承空显示设备时，可从 outbox 查看冻结正文，不宣称远端送达。

## TODO

- [x] 实现 E07 Core、Go adapter、插件 v2、Teams 受管入口及生产宿主接线。
- [x] 更新 1.7 authored checklist；不操作 Task/Core 显示状态、租约或 Gate。
- [x] 补写 Go 聚焦用例：跨重启 due time、两次上限、非网络/未知终止、缺配置不计次、冻结目标、禁用不补完成通知及严格结果协议。
- [ ] supervisor 执行冻结 Compile Plan，并在需要时提供编译诊断。
- [ ] 获授权后执行 AC28–AC30 / AX04 的替身验证；真实 Teams 配置及发送另行授权。

## Verification

本轮仅进行源码/工件读取、编辑和一次静态差异检查；未运行测试、编译、构建、格式化、lint、vet、验证脚本或联网。编译按本轮用户指令交由 supervisor；新写测试尚未执行，不作为通过证据。

实际命令类别：`Get-Content -Encoding UTF8`、`rg`、`Get-ChildItem`、`Get-Command aiw`；`aiw patch --help`；按用户给定精确前缀执行 Git status/diff；PowerShell 标准库计算两份脚本摘要并写入 adapter 常量。初次 UTF-8 JSON 被 PowerShell 默认编码误读，随后显式采用 UTF-8；不存在的 `cmd/`、`internal/config` 搜索没有修改文件。`aiw patch` 内部使用未获授权的裸 Git apply，因此以直接文件补丁作为编辑回退；没有修改 Git 配置、索引、分支或提交。

静态关注：同一 Store 提交内的事实登记、条件版本写入、派发前预约、身份绑定结果、未知不可重放、首次失败 60 秒原 deadline、逐消息额度、旧 ack 不外发、插件→脚本 stdin、TLS/redirect/HTTP 分类及宿主接线。

静态检查后修正第二次预约的状态不变量：当前 due time 在预约消费时清除，原失败观察时间与 due time 保留在首次尝试历史。相同事实换 ID 或重渲染也不能新增发送额度。遵守一次静态命令预算，未追加验证命令；修正后的编译仍交 supervisor。

%% ACTIVATION_PENDING：默认通知关闭；需要受支持脚本、Python 3.11+、显式目标/凭据引用、可信 CA（如需要）和宿主权限。未读取 .env 或运行通知程序。现有脚本旧独立 CLI 不属于受管安全入口。
%% VALIDATION_PENDING：编译由 supervisor 执行；所有 AC/AX 运行结果仍未知。尤其需用替身覆盖进程中断、TLS 错误、HTTP 拒绝、网络重置重复投递风险及宿主并发提交冲突。未把实现完成等同于测试通过或通知送达。
