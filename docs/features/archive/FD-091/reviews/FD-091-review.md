# FD-091 独立静态审阅

<!-- aiw-data: FD-091-review.json -->

结论：通过。未发现阻塞性缺陷；没有需要 Worker 修改的条目。

审阅者会话：`fd091-independent-reviewer-plugin-path-review`。来源事件：
`FD-091-000004-review-requested`，已通过受管 claim 独占领取。
实现提交 `286faae`，比较基线 `b4aba07`；审阅当前 FD revision 4。

已读取 FD、Work Items、Worker 双报告、相关稳定插件规范和完整变更，追踪
`src/internal/plugin/manifest.go` 的发现、入口解析及错误包装调用路径：

- 普通路径绝对化后逐级 Lstat，检查所有非卷根祖先；只有完整 ModeType 精确为
  0 或 ModeDir 才省略 EvalSymlinks。卷根终止避免向卷外遍历，普通安装目录不因此漏检。
- 符号链接、junction 的 ModeIrregular 及其它非普通类型仍调用 EvalSymlinks，
  失败直接传播，没有错误后的词法路径回退。本机 Go 源码确认相应模式及 Windows
  walkSymlinks 后 FindFirstFile 大小写规范化机制；组合类型不会误当普通目录。
- 根目录和入口采用同一绝对路径解析；原相对路径约束、filepath.Rel 物理边界、
  最终 Stat、目录入口拒绝及 manifest 上下文均保留。共享 CLI 和启动策略没有变更。
- 文档与稳定规范对应本次变更。Worker 记录离线 compile-only 退出码 0，审阅未重跑。
  现有帮助命令的失败只能支持机制推断，不能证明具体失败路径段或修复后的运行结果。

实际命令：Get-Content/Select-String 读取上述文件、共享审阅方法和核心规则；
`git diff b4aba07 286faae --`；`aiw fd show FD-091`；`aiw fd --help`、
`aiw fd claim --help`、`aiw fd emit --help`；上述精确 claim；
Get-ChildItem 检查受管目录、Get-Command go 定位工具链并只读其 Windows 标准库源码。
首次读取不存在的 receipt.json 和默认 Go 路径失败，随后使用受管 show 和实际工具链路径；
没有重复执行同一失败命令。完成报告后提交受管 verification-passed 事件。

未执行测试、编译、gofmt、vet、lint、最终构建、安装、发布、网络或任何 pkg 运行。
本报告是独立静态判断，未将 Worker 的编译证据或旧程序失败伪称为修复运行成功。

%% 已安装 C:\green\aiw\aiw.exe 没有更新；源码交付不能直接改变其行为。
%% 具体失败路径段和修复后的目标环境运行结果尚未测得。
%% 文件系统并发替换假设沿用原实现；异常链接解析仍可能失败，保持 fail-closed。
