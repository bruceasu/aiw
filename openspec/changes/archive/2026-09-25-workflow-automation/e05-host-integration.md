# E05 受控宿主接入

Work Item：wi-0007 / checklist 1.5。Attempt：attempt-1789723715632420700。

## 本轮实现

- `main.go` 在命令分派前安装生产 Store/Session 接线，并解释内部 `--auxiliary-root <absolute .ai path> --auxiliary-task <id>`。隐藏独立进程重建相同配置，使用项目系统锁防止多个宿主同时派发；前台退出不取消子进程。宿主最长 10 分钟，调用最长 120 秒，没有永久服务或自动启动安装。
- `execution/auxiliary_config.go` 从项目 `.ai/auxiliary-host.json` 读取显式 Task/操作授权和能力文件摘要；不沿用一般生成器的无界工具或默认模型能力。禁用配置仍允许原在途请求只读对账。
- `execution/auxiliary_http.go` 实现实际的只读文本生成 Worker。完整 JSON 请求计入字节和有证据的 token 上界；协议只含一条 user message、固定 model、stream=false、n=1、max_tokens，没有 tools、shell、子进程或追加上下文。HTTPS 验证 TLS，不继承代理，不跟随重定向，不自动重试。只接入 memory；E06/E08 的业务处理仍由后续工作项完成。
- 发送前在来源 Task 保存请求日志，结果日志仍使用项目预约和 Task 内容寻址工件。Reconcile 只读取该预约的证据，不产生调用。完整终态响应才允许 valid/invalid；请求超时、响应丢失、超大/截断响应、未识别终态或超出能力的 usage 保持 unknown，保留项目槽和完整预算。缺日志或 PID 消失不证明可以重调。有效文本最多 64 KiB；提供可靠 usage 才结算，缺失则保留满额。
- `auxiliary_maintenance.go` 提供受管政策安装、初始盘点和存储结算；后者保留所有模型调用、恢复和政策历史，只释放已终态模型队列的空间预约。新的 Task 可通过重新盘点加入存储基线。来源 seal 仍保守保留峰值，不删除原件。
- 所有生产 Store 通过构造接线获得辅助服务；没有配置时仅记录 `HostGap`，不撤销接受。helper 不再启动 helper。当前配置内其他 Task 的已登记工作也在同一个十分钟窗口内处理。
- 非冻结普通/交互 Session 从 Task 加载投影；冻结 Coder/Tester 保留原版本，派发前复核该版本的原始工件。人工 Session memory 文件保持原文；仅准备另一轮 prompt 不触发新摘要。首次 schema-10 来源在使用前持久固定。

## 本地配置契约

项目 `.ai/auxiliary-host.json` 是版本 1 的 JSON 对象，字段为：

| 字段 | 含义 |
| --- | --- |
| version / enabled | `1` 和显式启用值；不触发 schema 迁移 |
| endpoint | 完整 HTTPS chat completions URL，无凭据、query 或 fragment |
| api_key_env | 保存凭据的环境变量名；缺值时局部 waiting |
| tasks | Task ID 到允许操作名数组的映射；E05 只执行 `memory`，来源和赞助 Task 均须允许 |
| capability | `{kind,path,sha256}`，path 相对项目 `.ai`，精确匹配本地能力记录 |

能力记录字段：`version=1`、`endpoint`、`selection`（profile/provider/model/digest）、`context_tokens`、`output_tokens`、`counter_version=utf8-bytes-upper-bound-v1`、`envelope_tokens`、`protocol=chat-completions-max-tokens-v1`、`closed_input=true`、`enforced_output=true`、`evidence`、`reviewed_by`。

`evidence` 必须说明该精确 provider/model 的 UTF-8 字节数为何是 token 上界、服务端封装 token 上界，以及 max_tokens 与终态响应的实际契约。实现将完整序列化请求字节数加 envelope_tokens 作为保守输入预约；不根据字符比例猜 token，不联网取得 tokenizer。服务端返回 model 必须与审核的精确模型一致；动态别名须先固定为有证据的实际身份。能力记录不是本轮生成或验证的模型证明。

本轮没有创建以上实际配置或能力文件，没有注入凭据，没有启用后台调用。

## 受管维护入口

以下为实现后的入口说明，**本轮均未执行**：

1. `aiw workflow auxiliary policy <reviewed-policy.json>`：使用 R3 已批准硬上限校验全部字段；更新必须有更高 version 和 reason；保留原使用量。
2. `aiw workflow auxiliary inventory`：输出只读盘点草稿。全局最多 16,384 个条目、120 秒、4 GiB 既有运行内容；拒绝不可读/链接文件、重复 legacy/canonical Task 和活动写入状态。对同一物理文件去重，保守计入全部既有运行目录内容。该维护盘点不读取项目源码或提取历史知识。
3. 将审核后的草稿放到 `.ai/auxiliary-inventory.json`，填写 reason、reviewed_by；初始建账必须明确 no_prior_dispatches=true，且实际状态不得已有辅助调用。缺账本不自动证明历史为零。
4. `aiw workflow auxiliary initialize`：只允许不存在的资源账本，重新核对现场快照，封存不可变证据，再建立资源账；已有或不可读账本均保留。
5. `aiw workflow auxiliary settle`：在宿主空闲、无未知在途及 Task 静止时消费新审核盘点，更新物理存储基线。未知模型预算不释放，来源恢复额度不重置，旧 Task 不得从盘点消失。

维护前必须暂停相关写入；文件元数据变化会要求重新盘点。盘点不是迁移或执行授权。schema 10 仍须通过 E01–E04/R1 原有联合启用条件。

## 验证、限制与交接

静态检查范围：helper 参数到 Store/Worker、发送前日志到 Reconcile/Observe、完整输入/输出约束、原始来源到 Session、政策/盘点到持久预算、清单依赖。使用直接补丁工具，因为仓库 `aiw-patch.py` 走 Git 写路径，本轮仅授权只读 Git。

本轮实际验证：一次编辑后静态检查批次执行用户指定前缀的 `git diff --check` 和指定路径的 `git diff --stat`，并通过 `rg`/UTF-8 `Get-Content` 核对生产绑定、Session 类型、资源维护边界及 1.5/下游清单。`diff --check` 未报告空白错误，仅有已有 LF/CRLF 转换提示；它不覆盖未跟踪文件，新增 Worker/helper/配置/维护文件通过补丁文本和局部读取检查，未作编译声明。`diff --stat` 含前次 Attempt 已有改动，不能当作本轮独立改动量。本段记录补充后未再运行检查。

发现阶段执行了本地 `Get-Content`、`Get-ChildItem`、`Get-Command aiw`、`rg` 和指定前缀的 `git status --short --branch`。一次默认编码读取显示乱码，后续正文采用 UTF-8；个别旧路径/PowerShell glob 定位失败，使用已定位的实际路径继续，没有重试权限或访问网络。缺失的 `prompts/task-modes/feature.md` 未作为已读取规则；已应用根规则、Core 资源/验证规则、Go CLI 规则及 implement 契约。为补齐交接中未定位的真实接口，发现批次超过三批；没有运行任何执行性验证。

编译交给当前 supervisor 的冻结 Compile Plan；本执行者未编译、未测试，未运行 formatter/linter/vet/验证脚本、最终制品构建、网络、模型、后台宿主、通知或维护入口。没有修改 Core/Gate/租约/Task 生命周期或进行 Git 写操作。E05 实现完成不代表 E06–E08 或任何 AC/AX 已通过。

%% ACTIVATION_EVIDENCE：本地真实能力、凭据、授权配置、资源政策/盘点和 R1 平台证据仍为启用输入；缺失时只限制辅助调用。默认 schema 9 保持不变。

%% VALIDATION_PENDING：必须由 supervisor 编译后再接受本 Work Item；AC09/AC18/AC19/AC24/AC25/AC27 及 AX02/AX04 未执行。网络中断后无法从完整终态日志复原的请求保留 unknown，需要原宿主/服务端可靠证据，不能强制重试。来源封存和非终态任务仍采用保守存储预约，可能先于物理硬上限停止新分配。
