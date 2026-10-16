# FD-063 独立审查 r1

<!-- aiw-data: FD-063-review-r1.json -->

结论：请求修改。显式只实现或部分工作项的请求，尚缺少交接动作之前的明确停止条件。

- Reviewer：`fd063-reviewer-20261017-r1`，独立于 Worker。
- 来源：`FD-063-000004-implementation-ready`；已检查 pending receipt 后独立 claim。
- 范围：`785ead2..2e0a42b`；实现提交 `1e886f5`、`afd698f`、`a7f6a3a`。
- FD：revision 4；接手时 SHA256 为 `f2665d6328f949034bf6c846fc5a040560678b52434ddf338850530100dd4794`，与来源事件一致。

## 发现

1. **P1：阶段限制没有落实在派发之前。** `src/skills/implement/SKILL.md:50` 在所有所选工作项完成后无条件要求 emit `implementation-ready`，并明确该事件直接路由 Reviewer；`src/skills/fd-workflow/SKILL.md:248` 的第 3 步同样要求 emit。新增的阶段限制只作为一般规则或审查后收尾规则，未限定这两处具体动作。只实现请求会因此继续审查；部分工作项也可能满足 “all scoped Work Items”，产生代表整份 FD 就绪的事件。配置 role runner 时，emit 可立即启动 Reviewer，等事件发出后才停止已经太迟。应在两处 emit/dispatch 之前明确判断用户阶段与完整 FD 范围：只实现、所选部分工作项或尚未完成整个 FD 时，仅记录所做工作与证据并停止，不发出整份 FD 的 implementation-ready；整份 FD 交接必须同时满足用户允许进入审查和全部 FD 范围完成。同步项目副本，保留只审查、暂不合并等各自边界。该问题不需要运行 runner 才能判定。

另有一处低风险措辞建议：`src/skills/fd-workflow/SKILL.md:156` 的 blocker 段仍说主工作区只有另行授权才能提交，宜明确引用共享契约中的默认 FD 聚焦提交授权，避免被理解成仍需额外询问。此项不单独构成阻塞。

## 静态证据

- 已读当前 FD、Worker 双格式报告、根规则、共享工作管理与 reviewed-skill 契约，以及主源实现/审查/流程技能、角色、模板、README 和稳定规格差异。
- canonical 共享契约 SHA256：`9d296056badacf060888a479ba9458acd577f5d83a6ac03faf9d7e8f7b4bfb0e`；31 份已跟踪生成副本均相同；六对变更技能源/项目文件全部一致。
- 默认完整 FD 无需第二次 auto、独立 session、最新 revision/digest、实现无漂移、完整范围、无 blocker、坐标及干净工作区等交付门槛已有明确条款；未引入 CLI、schema、依赖或调度实现变化。
- 本地交付只面向登记父分支；先由 local-merge 校验 squash/FD-Source 并清理，再 close 与归档提交；失败、冲突、部分清理与归档拒绝保留状态，禁止自动解决冲突或绕过 Gate。
- 主工作区路径支持真实的 managed 独立审查而不伪造 Worker receipt；在途角色与 foreign claim 保留；未扩展测试、构建、网络、push、release、部署或提权权限。
- 接手时两处未提交差异仅为 implementation-ready 导致 FD/index 状态和 revision 更新，不属于实现漂移；它们及后续审查证据仍需宿主在交付前聚焦提交并重新检查当前 Gate。本次未认定工作区已经满足交付的干净条件。

## 实际命令与未执行检查

实际执行：定向 `Get-Content`、`Get-ChildItem`、`rg`、`git diff/status/rev-parse/ls-files`、`Get-FileHash`，以及 `aiw fd --help`、`aiw fd emit --help` 和独立 `aiw fd claim`。首次较大组合读取发生输出截断，随后分段读取必要差异和条款；未重复失败命令。

未执行编译、测试、验证脚本、构建、格式化、lint、vet、安装、网络或 runner；修改仅为 Markdown/YAML，无适用编译目标。结论只基于静态指令与派发契约，未宣称真实多会话或错误恢复运行结果。

宿主下一步：Worker 修复交接前阶段判断并提交新证据，再独立审查。通过后由宿主 PM 在当前 Gate 满足时继续本地交付、清理和归档；Reviewer 不代替 PM。
