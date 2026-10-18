# FD-079 获授权的聚焦静态复查

<!-- aiw-data: FD-079-implementation-review-20261018-r2.json -->

- 当前结论：**verification-passed**。原 R1/R2 已修复，本轮限定范围内无新增阻塞；结合首轮其他实现范围的静态审阅证据，当前可通过独立 Verification。
- Reviewer：`codex-fd079-reviewer-20261018-r2`，独立于 Worker。
- 来源事件：`FD-079-000010-implementation-ready`；读取真实 pending receipt 后由本会话领取，FD Revision 10。
- 本轮依据：用户明确授权的一次 R1/R2 聚焦静态复查；不是重跑整个实现审阅或新增运行验证。没有第三轮授权。
- 原实现：`bd63c3d`；修复源码：`c3d2c27`；文档 HEAD：`48a7b47d8e1d47c025147dd9c4643414a1874128`。当前未提交差异为 CLI 的 FD/索引状态更新。

## R1：链接逐组件解析，已解决

核对 `safety.go` 的 resolve、目录关系图及 `publish.go` 链接恢复路径。现在使用路径栈，遇到中间链接先递归展开其原始目标，返回后才继续处理后续 `.`/`..`。栈为空时拒绝 `..`，经过普通文件的路径分量拒绝继续遍历；目标的存在性、非绝对路径、链深和循环检查保留。依赖链接在当前链接前加入恢复顺序，普通文件和目录仍先于全部链接发布。

首轮反例 `p/q/a -> ../../x`、`p/q/b -> a/../../z`：解析 b 时先把 a 展开到根内 x，之后第一次 `..` 回到根，第二次 `..` 报越界；不会清理成另一个存在的 p/z。发布仍使用原始 target，但校验现已按同一组件语义处理。还核对本机 `os/root_windows.go`：目录链接判定把 newname 所在目录前缀加到 oldname，再在 root 内 Stat；依赖顺序与该机制相符。目录图仍拒绝根回链及目录环。

以上为静态控制流分析，未创建链接或执行反例。

## R2：ZIP 实际消费范围受预扫描约束，已解决

核对 `zip_limits.go`、`unpack.go` 的先预检后 OpenReader，以及本机 `archive/zip/reader.go` 的 EOCD 搜索、ZIP64 选择、baseOffset、预分配和中央目录循环。

- 普通 ZIP 要求 offset + length 精确等于 EOCD 起点，预扫描遍历完整区域且实际数量与声明数量一致。标准库由相同 offset 开始，紧接区域的 EOCD 签名不是目录头，不能继续消费额外目录记录；修复前的正 baseOffset 容错绕行不再成立。
- ZIP64 验证 end 记录签名、磁盘、数量、recordLength 和 locator 的连续范围，并要求中央目录精确结束于 ZIP64 end 起点。标准库读取同一有效 locator/end，紧接的 ZIP64 end 同样终止目录循环。元数据 64 MiB 和实际记录 100000 的前置限制仍保留。
- 对标准库的 directorySize==0xffff 兼容分支做了相应检查；缺少必需 ZIP64、异常偏移及不一致布局在 OpenReader 前拒绝。单独 0xffffffff size 导致标准库不采用 ZIP64 的异常布局，即使前检接受有效 ZIP64 记录，也会因标准库偏移计算不成立而失败，不能绕过前置数量分配保护。
- 尾部候选选择限制与标准库的最后可容纳 EOCD 对齐，并拒绝尾随数据。若尾部还有截断的伪 EOCD 导致标准库拒绝，则发生在目录分配前，不能恢复原资源绕过。

首轮反例声明 count=1、length=首个目录头长度但实际连续 131073 条：精确结束条件立即拒绝，尚未进入标准库分配。没有生成或执行恶意包。此次安全结论以本机 Go 1.25.1 源码为依据；更新工具链时须保留此消费范围一致性。

## 范围、命令及证据限制

实际只读命令：`Get-Content -Encoding utf8`（本轮 FD、Worker 双报告、receipt、受影响源码和本机 Go 源码），`git diff bd63c3d c3d2c27 -- src/programs/aiw-packer`，`git diff --name-only bd63c3d c3d2c27 -- src/programs/aiw-packer src/plugins/aiw-packer openspec/specs/aiw-packer`，`git rev-parse HEAD`，`git status --short`，以及针对本机 rootSymlink 的 `rg -n`。

工作流操作：`aiw fd claim FD-079 FD-079-000010-implementation-ready --session codex-fd079-reviewer-20261018-r2` 成功。本报告用于同一 source event 的真实 verification-passed。仅写本次 Markdown/JSON 和 FD Verification，不改实现、不提交 Git、不合并归档。

本次没有重新审阅未改变的全部格式/编码/plugin/删除代码，其静态证据沿用首轮报告；首轮 changes-requested 保留为真实历史，未改成通过。Worker 修复报告记录 c3d2c27 的 compile-only 退出 0，本 Reviewer 未重跑或冒认独立编译结果。

Reviewer 未运行任何测试、compile、packer、恶意包、最终构建、formatter、lint、vet、运行验证或网络，也未创建测试。跨平台/历史格式互操作、Windows 链接权限及故障行为仍无运行证据；源树并发变化和多文件发布/删除非事务风险仍保留。当前通过结论是现有授权和静态证据下的本地交付 Gate，不是运行测试或发布认证。
