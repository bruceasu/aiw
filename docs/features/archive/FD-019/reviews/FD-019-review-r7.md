# FD-019 独立静态复审（修订 7）
<!-- aiw-data: FD-019-review-r7.json -->

结论：静态审查通过，上一轮 R1 已消除，无新增阻塞发现。该结论不代表运行测试通过或已经部署。

Reviewer：`codex-reviewer-FD019-20261004-r7-82de`。来源事件：`FD-019-000007-implementation-ready`，领取前收据为 pending，已由本 Reviewer 独立领取。Worker 为 `codex-worker-FD019-20261004-r6-a71c`，身份不同。差异基线为 HEAD `c9b6d51cd2308e1ffbaf51755d697f6acda3ad83`，本轮审查该基线到工作区的 FD-019 定向差异，未创建提交；其他 FD 和已有无关改动未纳入结论且保留。

## R1 修复证据

- `plugins/aiw-ai/aiw-ai.mjs` 的 HELP 使用服务端 `principals[].keys`，直接比较且不计算哈希，明确 `key_hashes` 已废除、旧摘要仅作为两端相同的不透明字符串使用。定向差异仅改变帮助模板文字，认证、请求和输入实现未变。
- `plugins/aiw-ai/README.md` 的最少配置与 Key 配置说明采用同一字符串；删除当前配置步骤形式的 BAT 调用和“服务端存哈希、客户端发原始 Key”要求。历史 BAT 仅作为退役工具提及。
- `openspec/specs/agent-proxy-client/spec.md` 记录同一指引，与 `openspec/specs/agent-proxy/spec.md` 已有直接匹配、不接受 `key_hashes` 的契约一致。网关 `program/agent-gateway/config.go` 的 Keys 字段、非空检查与 `subtle.ConstantTimeCompare` 为实际静态支持。

## 工作项与证据边界

| 工作项 | 复审结论与真实证据 |
| --- | --- |
| 1.1 | 当前 DEFAULT_MODEL 为 gpt-6-luna，config 的非空模型环境变量覆盖；Key 缺失/格式错误提示帮助且不显示凭据。 |
| 1.2 | -h/--help 返回同一 HELP，main 在读取 stdin、config 和 fetch 前返回；最少认证配置 R1 已修复。help 路由的 -h 转发沿用 r5 已审查证据。 |
| 1.3 | README、客户端规格与默认模型一致，保留 models/allowed_models 配置要求；认证说明已对齐网关契约。 |
| 1.4 | 历史 BAT 与先前已审查的隐藏输入、UTF-8 无换行哈希实现保留，此轮未改或运行；当前文档明确它不用于当前认证配置。 |
| 1.5 | 当前 parseArgs 分离选项及值，以空格拼接位置参数，并拒绝 - 与其他提示词混用。 |
| 1.6 | 当前 editor.mjs 解析引号与参数，spawn shell:false，读取有界且 fatal UTF-8，finally 清理；失败在请求前返回错误。 |
| 1.7 | 当前 Ctrl+E 携带 rl.line，关闭 readline 并移除 keypress 后转交终端；正常编辑结果返回 main，readline SIGINT 取消。终端与编辑器实际行为未运行验证。 |
| 1.8 | 帮助、README 与客户端规格当前一致。原实现阶段本机同步只存在 FD 历史记录，本轮未重做或核验；本轮明确限于仓库指引修复，安装副本尚需运营者更新。 |

请求路径仍是一次 Bearer POST /v1/responses，input/instructions/text.format 分离，stdout 输出结果、stderr 输出诊断；当前文档改动未改变 URL、超时、重试或认证实现。无 Issue，无关联 OpenSpec change。FD 使用 External 策略，未运行测试不构成本轮阻塞，不能当作通过结果。

## 命令和剩余风险

实际执行：定向 `Get-Content`（中文随后指定 `-Encoding UTF8`）、`Get-ChildItem`、`rg` 和 `rg --files` 读取规则、FD、事件、报告及相关源码/规格；`git status --short`、`git rev-parse HEAD`、`git diff -- plugins/aiw-ai/aiw-ai.mjs plugins/aiw-ai/README.md openspec/specs/agent-proxy-client/spec.md`；`aiw fd --help`、`aiw fd claim --help`、`aiw fd emit --help`；`aiw fd claim FD-019 FD-019-000007-implementation-ready --session codex-reviewer-FD019-20261004-r7-82de` 成功。首次默认编码读取中文显示乱码，随后 UTF-8 读取确认真实文本。报告/FD 编辑后仅进行一次静态读取核对，并使用正式 CLI 发出通过交接，以收据为准。

Worker 报告记录本轮 `node --check plugins/aiw-ai/aiw-ai.mjs` 退出码 0，之后仅修改帮助措辞且未重复检查；Reviewer 未运行或重新证明该命令结果，也不把旧编辑器编译、本机同步和哈希核对当作本轮验证。

未执行测试、编译、客户端、编辑器、BAT、网关、认证/模型请求、网络、最终构建、依赖下载或 Git 写操作。实际认证、模型可用性、跨平台 Ctrl+E/编辑器等待/Ctrl+C 仍由外部验证；安装副本未更新。强制终止或编辑器备份可能残留临时 Prompt，现有 FD 已记录。历史报告不覆盖。通过审查仅允许父会话决定 close 归档，不代表推送、发布或部署。

