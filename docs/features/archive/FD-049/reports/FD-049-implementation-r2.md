# FD-049 实施报告 r2

<!-- aiw-data: FD-049-implementation-r2.json -->

Worker 会话 `fd049-worker-20261010-9a721c`，source event `FD-049-000006-changes-requested`，FD revision 6。工作树 `.wt/FD-049`、分支 `feature/FD-049`、父分支 develop、计划基点 `f653509`。

## 当前结果

1.1–1.3 已完成：提供 Say 所需的非流式文本 Chat Completions，支持 model/messages、stream 省略或 false、n 省略或 1；共享认证、模型授权、额度、取消与 App Server 执行，返回单个 assistant content/stop，未知 usage 保持 null。首轮实现详情及未运行验证见 r1 历史报告。

R1 审查请求修改已修复，提交 `d076f03`：除既有严格 JSON 校验外，Chat 顶层和各消息对象还按原始 JSON 键名检查精确白名单，拒绝大小写变体和标准键/别名同时出现的覆盖组合。只允许 model/messages/stream/n 与 role/content；消息字段 null 也显式拒绝。未修改 Responses 解码契约；稳定 spec 补充精确键名要求。

原实现提交 `731cc38`、`e9e0590`，文档/证据提交 `514174f`；原 Reviewer 报告 `FD-049-review-r1.md` 保留历史，不把其请求修改结论改写为通过。

## 检查与命令

- R1 源码修复后在 `program/agent-gateway` 重跑 `python -B scripts/compile.py` 一次，退出码 0。连同首轮，共两次 compile-only；Windows/Linux amd64 均通过。脚本关闭依赖网络，输出到系统空设备，不保留发布产物。
- 静态跟踪原始字段精确检查、消息数组与 typed decode 的对应关系、重复字段拦截、null/UTF-8/大小边界和共享执行入口。全部检查为静态证据，未执行真实请求。
- 已运行 `git diff --check a3e5b23`（无差异格式错误）、`git diff a3e5b23 -- program/agent-gateway/chat.go` 和 `git diff --stat a3e5b23`，检查 R1 审查提交后的修复范围。
- 未运行 tests、HTTP/Say/App Server、SDK、完整构建、格式化/lint/vet 或依赖下载。

## 限制与下一步

首批仍是非流式字符串文本子集；FD-048 的真实上游代理未实现。真实 Codex 登录/RPC/翻译行为未经运行验证。新 route 的旧二进制降级限制见 Gateway 文档；原数据不删除。修复后的结果交第二次独立 Reviewer，尚不宣称审查通过。
