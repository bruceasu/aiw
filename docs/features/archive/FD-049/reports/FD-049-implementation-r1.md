# FD-049 实施报告 r1

<!-- aiw-data: FD-049-implementation-r1.json -->

## 结果与范围

已实现 Codex 后端的 `POST /v1/chat/completions` 非流式文本子集，匹配现有 Say 的 model/system/user/stream:false 请求及单个 choice/stop 响应。FD-048 的真实上游代理及统计/计费预留未在本次实现；完整 Chat Completions 功能也不在范围内。

Worker 会话：`fd049-worker-20261010-9a721c`；source event：`FD-049-000004-work-requested`；FD revision：4。工作树 `.wt/FD-049`，分支 `feature/FD-049`，父分支 develop，计划基点 `f653509`。

## Work Items 与证据

- 1.1 完成，提交 `731cc38`：新增 `chat.go`，严格拒绝未知/重复/null 字段和不支持角色/内容/执行选项；开头 system/developer 映射为 instructions，user/assistant 对话复用内部 `decodeRequest` 与 64 KiB 限制。成功对象生成 chatcmpl_ ID、单个 assistant content/stop，usage 重命名且未知保持 null。
- 1.2 完成，提交 `e9e0590`：新增路由，抽取原 Responses handler 为共享 inference 入口，只按 API 切换 decoder 和非流式最终对象。认证、模型授权、额度/并发、timeout/取消、App Server 调用及 Responses SSE 路径复用；route 观测新增 Chat Completions。
- 1.3 完成：更新稳定 spec、Gateway/Say 文档、FD TODO/Verification 和本 Dual 报告。说明严格子集、配置方式及直接降级旧二进制读取新 route 的限制。

## 检查与实际命令

- `python -B scripts/compile.py`，cwd=`program/agent-gateway`：退出码 0。脚本以 GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local、CGO_ENABLED=0 编译 Windows/Linux amd64，输出到系统空设备，不保留发布产物。
- 已运行 `git diff --check f653509`（无差异格式错误）、`git diff --stat f653509` 和源文件定向 `git diff f653509 -- ...`。静态核对 Say 的请求/响应解码、Chat 字段/角色/大小映射、共享执行链、usage、错误与未改动的 Responses SSE 分支，以及新增 route 的载入校验；最终实现差异交独立 Reviewer。
- 已按 Work Item 分别提交代码；未运行 Go tests、真实 HTTP/Say/App Server 请求、SDK、完整构建、formatter/linter/vet 或网络下载。

## 限制与风险

- 编译结果只证明两个目标平台的编译成功，不证明 Codex 安装/登录、App Server RPC 或 Say 运行兼容性。真实请求验证需单独授权。
- 首批只接受字符串消息，stream 省略/false，n 省略/1；tools、图片/音频、content 数组及生成参数均拒绝。
- 观测记录结构与 schema 版本不变，但新 route 枚举值可能被旧二进制拒绝，不能保证直接降级读取新记录。
- Worker 报告不是 Reviewer 通过证据；当前等待独立审查。
