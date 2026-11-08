# FD-112 最终修订独立复审（r4）

<!-- aiw-data: FD-112-review-r4.json -->

## 结论

**最终修订复审通过，无 findings。** 本次审查的 FD revision 16 与其实现、最终 spec/README、历史 review 及编译/测试记录一致。随后按流程将本报告路径和通过结论写回 FD，形成 revision 17 的最终证据修订；该文档性更新不改变实现。

来源事件：`FD-112-000016-review-requested`。Reviewer session：`fd112-review-final-20261108`。

## 核对结果

- **实现与迁移：** `loadConfig` 要求明确的 `backend_mode`，缺失/null 和旧 `codex` 给出迁移到 `openai_codex` 的提示；其它值被拒绝。新启动配置会拒绝 `workspace_dir`、`codex_path`、`codex_script`。专用 `stopGateway` 解码结构保留这些旧字段并允许旧缺省/`codex` 配置用于停止旧实例，不会启动旧 App Server。
- **执行链与删除：** Gateway 初始化不创建 App Server pool；proxy 模式在代理分支转发后返回，非 proxy inference 走 `executeOpenAICodex`。旧 App Server runner、pool、JSON-RPC 与平台进程管理文件均已从工作区 diff 删除；剩余 `codex` / workspace 字符串仅出现在显式迁移错误或停止旧进程的兼容解析中。AIW Codex 登录和其他 Codex 调用范围未见被本 FD 删除。
- **r1 README finding：已解决。** README 将 `state_dir/requests` 持久化和本地 `/v1/usage` 统计限定于 `openai_codex`，并说明 `openai_proxy` 不持久化 HTTP 请求元数据且不提供本地 usage 统计；与对应代码一致。
- **r1 stable-spec finding：已解决。** Stable spec 有独立的 development loopback listener/peer 及 shared 显式 listen 要求。
- **r2 shutdown-spec finding：已解决。** 关停规范现在要求取消活动后端请求、优雅停止 HTTP、关闭代理连接、等待定期内容清理退出并释放状态锁；停止命令继续有界等待，只有 listener 停止且锁释放后才报告成功。进程树清理和其他 App Server 生命周期要求均未恢复。对应代码路径执行服务级 context cancellation、HTTP shutdown、proxy connection close、cleanup goroutine 等待，并在正常退出时 deferred close Store。
- **示例和 README：** 默认/Codex 示例显式配置 `openai_codex`；proxy 与 DeepSeek 示例显式配置 `openai_proxy`。README 要求新配置显式设置 backend_mode，并说明删除旧字段，与配置加载行为一致。
- **历史与验证记录：** r1 sidecar 为 changes-requested（2 findings），r2 为 changes-requested（1 finding），r3 为 verification-passed（0 findings）；本次核实前两轮指出的问题均已修复。FD Verification 记录 Windows/Linux `python scripts/compile.py` 成功，且记录未运行 Go tests、runtime、network 或依赖安装。仓库操作日志记录 PM 将此前 event-derived Complete 重置为 In Progress，并明确 review 尚未完成；本次独立复审完成该 gate。

## Findings

无。

## 命令与未运行检查

只运行了本地只读检查：`aiw fd show FD-112`、`aiw fd claim FD-112 FD-112-000016-review-requested --session fd112-review-final-20261108`、`git status`、`git diff --stat`、`git diff --name-status`、`git diff`（最终实现/spec/README/FD）、`Get-Content`、`Select-String`、`rg`，并读取历史 review sidecar 与状态操作 audit JSON。

未运行测试、编译、构建、网络或运行时命令。compile-only Windows/Linux 成功仅引用 FD 记录，本 Reviewer 未重跑；未创建或运行 Go tests。

## 剩余风险

配置迁移、旧实例停止和关停行为没有进行运行验证。编译成功不替代运行时验证；这些检查因当前授权边界保持未运行。
