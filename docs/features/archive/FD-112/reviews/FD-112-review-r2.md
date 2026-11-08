# FD-112 文档修复聚焦复审（r2）

<!-- aiw-data: FD-112-review-r2.json -->

## 结论

**r1 的两项文档 finding 已解决；另有一项残留 stable spec finding，发出 `changes-requested`。** README 持久化范围和 development/shared listener 规则现已准确记录，但 Local graceful shutdown 仍要求 App Server 才需要的进程树清理，而当前实现已移除该执行路径。

Reviewer session：`fd112-review-doc-fixes-20261108`。来源事件：`FD-112-000008-implementation-ready`。当前 FD revision：8。

## r1 findings 核对

- **README 持久化范围：已解决。** README 明确 `openai_codex` 将 HTTP 元数据写入 `state_dir/requests` 并提供本地 usage 统计；另明确 `openai_proxy` 不持久化 HTTP 请求元数据、不提供本地 usage 统计。与 `main.go` 的 store 创建逻辑和 `server.go` 的条件持久化路径一致。
- **development/shared 监听约束：已解决。** Stable spec 新增独立的 Deployment listener and peer boundaries 要求及场景：development 仅 loopback listener/peer，shared 必须显式设置 listen。配置和 server 仍实现这些限制。新增段落没有恢复 App Server 进程池、MCP、sandbox 或 workspace 的要求。

## Finding

1. **稳定规格仍要求已删除的进程树清理**

   - 位置：`openspec/specs/agent-gateway/spec.md`，Requirement: Local graceful shutdown control，关停语义段落（当前约第 135 行）。
   - 当前要求：shutdown “MUST 复用全局取消和现有进程树清理”。
   - 观察到的实现：FD-112 删除了 App Server 子进程启动、进程池和 Linux/Windows 进程树管理文件。Gateway 当前关停路径执行 HTTP server graceful shutdown、关闭代理连接并等待内容清理收敛；没有 App Server 子进程或进程树清理实现。该规范句子超出了当前实现能力，并与本 FD 删除 App Server 生命周期依赖的范围不一致。
   - 影响：stable spec 继续要求不存在的 App Server 进程树语义，可能造成对当前关停保证的错误预期。
   - 最小修正：删除“现有进程树清理”要求，保留全局取消、活动 HTTP/代理连接关闭、定期内容清理等待及状态锁释放等仍适用的关停语义。

## 范围与证据

- 当前 FD revision 8；Worker 记录称实现代码自 r1 未改，仅修订 README 和 stable spec。当前完整 diff 与 r1 review 基线相同的 Gateway 实现变更仍包含旧 App Server 文件删除；本轮重点核对其不再被规格要求。未修改代码或文档。
- Worker 在 FD Verification 中记录 `python scripts/compile.py` 的 Windows/Linux compile-only 成功；本 Reviewer 未重跑。未运行测试、构建、编译、网络或运行时命令，符合 Go 子树规则。
- 本次仅作静态读取：`aiw fd show`、`aiw fd claim`、`git status`、`git diff --stat`、`git diff`（README/spec/FD）、`rg`、`Select-String`、`Get-Content`。未运行 Go 测试。

## 剩余风险

除上述关停规范残留外，r1 两项文档 findings 已处理。编译成功不能验证配置迁移或旧进程停止行为；本轮按授权未运行这些行为验证。
