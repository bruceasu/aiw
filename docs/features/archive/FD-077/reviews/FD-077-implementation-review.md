# FD-077 独立实现审查

<!-- aiw-data: FD-077-implementation-review.json -->

- 结论：changes-requested；存在 1 项阻塞发现。
- Reviewer 会话：fd077-reviewer-20261018-c9d283，独立于 Worker。
- 来源事件：FD-077-000004-implementation-ready；核对 pending 后已独立 claim。
- 审查基线：193f0de；实现：0dfc816、aa3be06；证据提交/审查 HEAD：a6e3db96c134cc9bdf13121c7b2b2e680be87614。

## 发现

### P1：预检与打开之间替换草稿根可扩大删除范围

位置：`src/internal/issue/capture_draft.go:75`、`:82`、`:87`、`:130`。

当前代码检查 family/root 为非链接目录，却丢弃这次 Lstat 的身份；OpenRoot 后才建立 rootInfo。若在预检后、OpenRoot 前将 family 或 drafts 换成指向另一目录的链接，OpenRoot 会绑定替换后的目录，sourceInfo 和读取也以其为准。cleanup 的 os.Stat 跟随这个链接，仍能匹配替换后建立的 rootInfo；随后隔离、核对和删除可全部成功，最终删除指定草稿根之外的文件。

这违反 FD Solution 的“根本身被替换……保留并警告”、Acceptance 3，以及稳定规范中不安全链接不得扩大清理范围的要求。该窗口发生在建立受信根时，与 FD 明确排除的“恶意修改私有隔离目录”不同；后续 SameFile 和字节核对不能补救错误的初始根身份。

最小修正：保留预检的 family/root 身份，打开后将固定根句柄的身份与预检身份比较；清理前重新以不跟随末级链接的检查核对两层目录类型和身份。发现替换、链接或无法安全核对时保留源并警告，不把替换后的根作为新的可信快照。

## 其余静态证据

- Capture 的所有保存失败出口均在 cleanup 之前；正式工件与 Write(meta) 成功后才进入清理。
- 两个确切草稿根、原始 `..`、ADS、根以下链接、普通文件身份和实际捕获字节均有针对性检查；隔离后不一致时采用不覆盖的 Link 恢复，失败保留恢复路径；不递归删除目录。
- 外部来源不进入清理；正式工件、批准日志与其他草稿没有扫描清理入口。
- 直接与聊天调用都在成功 Capture 后发出 stderr 警告；聊天后续事实记录错误沿用现有边界。
- CleanupWarning 标记 `json:"-"`；Write 仅显式序列化 Path/Digest；返回警告在 meta map 赋值和保存之后添加，批准源 JSON 摘要不变。
- 帮助、使用文档、稳定规范与 FD 的目标一致；根保护发现修复前不能认为这些承诺已全部满足。
- 已读取 Worker 报告和 compile 脚本；Worker 记录的离线 `python scripts/compile.py` 退出码为 0。本 Reviewer 未重新执行编译，不将其当成本会话运行结果。

## 命令与边界

实际执行：定向 `Get-Content`/`Select-Object`/`Select-String`/`rg` 静态读取；`git status --short`、`git diff 193f0de..a6e3db9 --stat`、该范围的定向 diff 与 `--name-only`、`git rev-parse HEAD`；`aiw fd --help`、`aiw fd show FD-077`、`aiw fd claim FD-077 FD-077-000004-implementation-ready --session fd077-reviewer-20261018-c9d283`、`aiw fd emit --help`。报告完成后发出本报告对应的 changes-requested 事件。

读取时发生并保留的非产品错误：`aiw fd status FD-077` 不是支持的子命令，已根据帮助改用 show；worktree 内 `.ai/fd/FD-077` 不存在，未继续猜测目录；定向 rg 包含不存在的 `src/internal/issue/digest.go`，已直接从 store.go 阅读摘要实现。这些均不是运行验证。

未运行测试、实际 capture/删除/恢复、编译、格式化、lint/vet、最终构建、网络或下载；未新增测试或实现修复，未提交、合并、归档。Windows 文件占用、硬链接兼容性及跨平台行为仍为未验证风险；测试缺席本身不是本次阻塞原因。修复后应聚焦复核根身份边界及相关 diff。
