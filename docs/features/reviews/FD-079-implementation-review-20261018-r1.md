# FD-079 独立实现审阅

<!-- aiw-data: FD-079-implementation-review-20261018-r1.json -->

- 结论：**changes-requested**；两项实质缺陷阻止 Verification 通过。
- Reviewer：`codex-fd079-reviewer-20261018-r1`；与 Worker 会话独立。
- 真实来源事件：`FD-079-000006-implementation-ready`，确认 pending 后已领取；FD Revision 6。
- 基线：`28dd3bf`；实现提交：`bd63c3d`；证据 HEAD：`125dd19d91f61fe78c3a30423da50a5826def35f`。当前未提交差异是 CLI 的 FD/索引状态更新。

## 必须修复

### R1 / P1：链接目标清理顺序可能生成解包范围外的链接

位置：`src/programs/aiw-packer/safety.go:143`，以及 `publish.go:165`。

FD 的链接安全策略与稳定 spec 要求 -l 拒绝越界、悬空目标。当前 `resolve` 先对整个目标执行 `path.Clean(path.Join(...))`，再查找链接。这会在解析中间符号链接前消去 `..`，与操作系统按路径组件处理链接的语义不同；发布却保留原始 target 文本。

静态反例：归档包含目录 `x/`、普通文件 `p/z`、目录 `p/q/`、链接 `p/q/a -> ../../x` 和 `p/q/b -> a/../../z`。当前校验将 b 的目标清理为 `p/z` 并接受；实际访问先把 a 展开到解包根中的 x，再执行两次 `..`，结果指向解包根的父目录下 z。该链接可能越界，也可能悬空。已有目录环检查不会拒绝这组条目。Go `os.Root.Symlink` 的本地源码明确说明不验证 oldname，不能修补这个缺口。

最小修复：解析目标时逐组件展开链接，再解释后续 `..`，在真实解析语义下检查边界、存在性与依赖顺序；或明确拒绝此类含 `..` 的复杂链接目标。不能只对清理后的路径验证，再原样恢复另一种语义的链接。此为静态推导，未执行反例。

### R2 / P1：ZIP 预扫描范围与标准库实际读取范围不一致

位置：`src/programs/aiw-packer/zip_limits.go:51`（范围判断及后续扫描），`unpack.go:84`（调用 `zip.OpenReader`）。

稳定 spec 要求在标准库分配条目表前限制实际条目数和中央目录元数据。当前预扫描只保证 `offset + length <= endOffset`，并且只扫描声明的 length。Go 1.25.1 本地 `archive/zip/reader.go` 则会在声明 offset 存在有效中央目录头时将正 baseOffset 回退到零，并从该位置持续读中央目录头，直到遇到非目录头；它不以声明 length 为循环边界，最终数量也仅按 uint16 比较。

因此可构造连续 131073 个中央目录记录，EOCD 声明 count=1、length=第一个记录长度、offset=第一个目录头。当前预扫描只读取一个记录并通过，标准库随后为全部 131073 个记录分配对象，数量模 65536 仍为 1；后置 `len(z.File)` 检查已经太晚。后续记录的 extra/comment 也能使实际元数据超过 64 MiB。即使稍后因文件名或内容错误失败，也已失去预分配资源保护；整个输入仍可远低于 4 GiB。

最小修复：预扫描必须约束并验证标准库实际会消费的完整目录区域，拒绝声明范围与实际布局不一致的输入，或让标准库只能读取已经证明受限的布局；不能仅依赖 EOCD 的较小 length/count。ZIP64 亦应采用同样原则。此为两份实际源码的静态控制流对照，未生成或执行恶意包。

## 已审阅证据与范围

读取批准 Issue 的主要规则、issue.toml/decision-log、FD/Work Items、独立设计报告和 Worker 双报告；Issue TOML 确认 APPROVED，历史 Plan 的未批准描述未作为当前决策。核对实际 diff 范围以及 options、encoding、source、pack、unpack、safety、publish、zip_limits 的完整调用路径，涵盖六类格式、编码优先级、stdout 预暂存、覆盖、删除快照、失败传播及资源计数。读取 plugin manifest/launcher、准备脚本、README 和相关稳定 specs。第三方审阅限于固定版本记录、本地 replace、导入闭包与许可证，未展开生成编码表，也未重新下载。

其他主要路径有可追踪的静态支持：stdout 在完整解码后发布；-d 的输入组合受限，输出成功后才重新核对源并删除；目标通过目录句柄检查且不覆盖已有链接；历史编码使用往返检查。上述静态支持不消除 R1/R2，也不代表运行兼容性已验证。

## 实际命令与跳过项

实际运行只读 `Get-Content -Encoding utf8`、定向 `rg/rg --files`、`Get-ChildItem`、`Get-Command`、`git status --short`、`git diff --stat 28dd3bf HEAD`、`git rev-parse HEAD`，及 `aiw fd --help`、`aiw fd claim --help`、`aiw fd emit --help`。曾因 worktree 无 receipt.json 和猜测的 Go 标准库路径不存在而读取失败；后分别更正为父工作区真实 events 文件和 `C:/green/go/src`，没有权限探测或运行重试。

实际工作流写操作：`aiw fd claim FD-079 FD-079-000006-implementation-ready --session codex-fd079-reviewer-20261018-r1` 成功；本报告作为 reviewer 的 `changes-requested` artifact，引用同一 source event。仅修改本报告双文件与 FD Verification；未修改实现或提交 Git。

Reviewer 未运行测试、compile、packer、最终构建、formatter、lint、vet、验证脚本或网络；未创建测试。Worker 报告的两次 compile-only 成功按其证据记录，未冒充 Reviewer 独立重跑结果。没有以未授权的运行检查作为拒绝通过的理由；拒绝依据是以上可定位的代码缺陷。

## 剩余风险

跨平台格式互操作、历史编码包、Windows 链接权限/恢复及故障注入未运行验证。源树并发修改和多文件发布/删除非事务边界已有披露。应先修复 R1/R2；后续验证须沿用现有预算，不自动扩大测试或网络权限。
