# FD-094 独立实现审阅

<!-- aiw-data: FD-094-review.json -->

结论：请求修改。存在一项物理路径解析错误，当前不能通过 Verification。

- FD：FD-094。
- 来源事件：FD-094-000006-review-requested。
- 独立 Reviewer session：codex-fd094-junction-review-20261019，已真实 claim。
- 基线：7a3c2d8；被审实现：8f4b7c9。

## 阻塞发现 R1：解析相对链接之前必须解析其父目录

位置：`src/internal/plugin/manifest.go`，`resolveManifestPathLinks` 的叶到根循环及
`target = filepath.Join(filepath.Dir(current), target)`。

FD Acceptance 和稳定规格要求返回实际物理目标，并拒绝指向插件物理根目录之外的入口。
当前先检查叶子链接，尚未处理祖先 junction；相对目标中的 `..` 被 Join 提前按词法消除。
这会丢失祖先链接的物理路径语义。

可静态推导的例子（未执行、未创建文件）：插件根为 `D:/plugins/p`，其子目录
`sub` 是指向 `E:/outside/sub` 的 junction，实际 `E:/outside/sub/run.py` 是指向
`../payload.py` 的相对 symlink；两侧对应 payload 文件都存在。清单入口 `sub/run.py`
实际指向 `E:/outside/payload.py`，必须被物理 containment 拒绝。然而当前代码先 Readlink
叶子，再 Join `D:/plugins/p/sub` 与 `../payload.py`，清成 `D:/plugins/p/payload.py`。
后者普通祖先检查成功且位于 root 内，于是返回了不同的文件并绕过应有的越界拒绝。
相反方向的组合也可能误拒绝原本合法的入口。

最小修正方向：按卷根到叶子的顺序解析每个组件；或保证相对链接的父目录已先完整物理
解析，再组合目标。所有目标替换和父目录解析必须共享 255 次遍历预算，保留未知
irregular fail-closed、卷相对目标拒绝和最终 root/entrypoint 的物理 Rel 检查。
不要仅在 Join 后重新检查已经丢失 junction 的词法路径。

## 已审证据和限制

已阅读 FD、Worker 双报告、完整 manifest resolver、调用方和精确三文件实现 diff，
以及本机 `C:/green/go/src/os/types_windows.go`、`os/file_windows.go` 和
`path/filepath/symlink.go`：junction 的 ModeIrregular、Readlink 的 MOUNT_POINT 支持，
以及标准 walker 从根逐段处理链接和 `..` 的语义均有源码支持。

普通路径快路径、混合 ModeDir/ModeSymlink 识别、未知 irregular Readlink 错误、
递归次数限制和最终 Stat/Rel 守卫本身均已静态检查；这些不能抵消 R1。
Worker 报告记录离线仅编译 exit 0；Reviewer 未重新运行编译。已消耗的用户单次
路径诊断授权未重用，未声称修正后的已安装程序可运行。

实际命令仅静态读取（Get-Content、rg、git diff）及受管工作流 help/claim/emit。
首次猜测 FD 文件名不存在，后续以 rg 查得准确文件；首次 `aiw fd show FD-094`
因 GBK 无法输出 BOM 失败，未将其当作成功证据。随后 claim 使用 UTF-8 输出。
未运行 tests、compile、runtime、build、install、publish、formatter 或 network。

剩余风险：R1 尚未修复；修正后 launcher 的实际运行和现有文件替换竞争条件均未验证。
