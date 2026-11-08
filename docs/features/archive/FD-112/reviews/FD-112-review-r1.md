# FD-112 独立实现审查（r1）

<!-- aiw-data: FD-112-review-r1.json -->

## 结论

**要求修正文档后复审，发出 `changes-requested`。** 核心后端迁移实现及旧 stop 配置兼容路径静态检查未发现阻断缺陷；但 README 对代理模式持久化行为自相矛盾，稳定规格删除时也丢失仍在实现中的部署监听约束。

Reviewer session：`fd112-review-20261108`。来源事件：`FD-112-000005-review-requested`。审查基线：HEAD `2e9d0c1` 加完整未提交 diff。

## Findings

1. **文档矛盾：代理模式是否持久化 HTTP 请求记录不一致**

   - 位置：[README.md](../../../src/programs/agent-gateway/README.md)，第 51 行与第 60 行。
   - 当前要求/行为：`openai_proxy` 不打开 Codex usage store；`main.go` 在该模式下不调用 `openStore`，HTTP handler 只在 `g.store != nil` 时调用 `PutHTTPRequest`。README 第 60 行也明确写着代理模式不持久化请求记录。
   - 问题：README 第 51 行却笼统声明所有 HTTP 请求的元数据都保存至 `state_dir/requests`，与代理模式实际行为及 README 后文直接冲突。
   - 影响：代理模式运营者可能误以为请求元数据有持久化审计记录。
   - 最小修正：把第 51 行的持久化说明限定为 `openai_codex`，并说明 `openai_proxy` 只记录日志、不持久化请求元数据；或在代码确实要持久化时另行实现并更新规范。当前 FD 目标是保留既有代理行为，优先采用文档范围限定。

2. **稳定规格误删仍有效的 development/shared 监听规则**

   - 位置：`openspec/specs/agent-gateway/spec.md`，删除的 `Requirement: Controlled process lifecycle` 段落。
   - 当前要求/行为：删除段落同时含有 App Server 专属的进程树、临时工作区规则，也含有不依赖 App Server 的规则：development 限 loopback 监听及 peer，shared 模式要求显式 listen。当前 `config.go` 仍拒绝 development 非 loopback 地址及 shared 未显式 listen；`server.go` 仍实施 development peer 限制。全规格已无其它 active requirement 描述前两项模式规则。
   - 问题：为清理 App Server 规范而删掉了仍由代码强制的部署/监听契约，造成稳定规格对现存行为不完整。
   - 影响：后续维护者可能将这些限制视为无规范依据的实现细节并误删或改变。
   - 最小修正：恢复 development/shared 监听与 peer 的要求/场景，同时不恢复 App Server 进程、沙箱或 workspace 规则。

## 已通过的静态检查点

- `loadConfig` 在严格解码前显式检查 `backend_mode`：缺省/null、旧 `codex` 和非法类型不会默认切换后端；缺省/旧值错误明确指向 `openai_codex`。仅接受 `openai_codex` 或 `openai_proxy`。
- 新启动配置对 `workspace_dir`、`codex_path`、`codex_script` 明确报旧配置字段错误；这三项未再进入启动 `Config`。
- `stopGateway` 使用专用兼容结构保留三个旧字段的 JSON 解码，缺失旧 `backend_mode` 时仍按旧 `codex` 解释；stop 不走新启动 `loadConfig`，不启动 App Server。它仍按旧 state_dir 查锁并向运行中的旧实例发起本地 shutdown。
- 运行时 `main.go` 仅为非 proxy 模式打开 store；Gateway 创建时不再创建 App Server pool；`server.go` 的 proxy 分支透传后返回，其余请求统一进入 `executeOpenAICodex`。App Server runner/pool/protocol 与平台进程启动文件均在 diff 中删除；源码搜索未见残留运行时引用。
- 默认 `gateway-example.json` 与 Codex 样例明确设为 `openai_codex`，代理与 DeepSeek 样例明确设为 `openai_proxy`；README 的新启动配置说明和这四个样例一致。FD 所列 compile-only 命令成功仅作为 FD 记录，本 Reviewer 未运行。

## 命令与未运行检查

仅运行本地只读审查命令：`aiw fd show`、`aiw fd claim`、`git status`、`git diff --stat`、`git diff --name-status`、`git diff`（配置/入口/README/spec 全部变更）、`git log`、`Get-Content`、`Get-ChildItem`、`rg`、`Select-String`。没有修改实现。

未运行测试、编译、构建、格式化、静态检查器、网络或运行时命令。FD 中记录的 `python scripts/compile.py` Windows/Linux 成功及 `git diff --check` 通过属于 Worker 证据，本 Reviewer 未复跑。依 Go 子树规则，未创建或运行 Go 测试。

## 剩余风险

- 未运行行为验证；compile-only 不能证明配置错误、stop 旧进程兼容或后端运行时行为。
- 上述两项文档 finding 修正后，应基于更新后的 FD/revision 再请求复审。实现代码本次静态未见其它问题。
