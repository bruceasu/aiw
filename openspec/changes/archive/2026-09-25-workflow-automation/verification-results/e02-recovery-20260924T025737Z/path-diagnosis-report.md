# E02 根路径解析诊断

用户授权继续处理；Task workflow-automation，wi-0012 / 2.2。

## 代码与编译

worktree 的 execution_inputs.go 为根目录解析、目标路径解析和正文读取增加独立错误上下文。EvalSymlinks 失败通过 os.PathError 保存实际输入路径和原始错误链；原来的路径归属、符号链接、内容摘要校验全部保留。

离线 python scripts/compile.py 退出 0，临时产物由脚本清理。沿用直接补丁方式，没有 Git 写入。

## 两次同范围运行

本轮先执行 path-diagnostic-run，再针对已观察到的默认临时目录错误，仅改变测试子进程 TMP/TEMP 到 worktree 内并执行 workspace-temp-run。每个目录各自保存计划、授权上下文、源码摘要、命令/环境、时间、原始 stdout/stderr 与退出码。

两次均使用原三个 E02 用例、相同源码摘要 41f31a0891077f8d361636832d4f9253a665618c6d7dbba31e55d5a4f12f5f10、GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local；均退出 1，三个测试均失败，执行期间输入未变化。没有更改系统环境、ACL、运行用户或沙箱权限。

默认临时目录：C:/Users/svictor/AppData/Local/Temp/TestDurable.../001/runtime/task-1。
worktree 临时目录：C:/Users/svictor/workspace/tools/aiw/.wt/workflow-automation/.ai/test-temp/e02-recovery/TestDurable.../001/runtime/task-1。

两者都在 `resolve input root links ...: Access is denied` 失败：确认是 confinedFile 对 Task 根路径执行 filepath.EvalSymlinks 时返回拒绝，尚未解析目标路径或读取工件正文。此前创建、写入该目录及工件成功，因此不能将其笼统解释为目录不可写。

本机 Go Windows EvalSymlinks 的名称规范化逐级执行 FindFirstFile；这与当前错误形态一致，但尚未得到被拒绝的具体祖先目录或权限来源。换到 worktree 没有解决问题，不再重复调整路径或运行测试，不绕过安全检查。

## 后续取证

已准备 change 目录中的 run-e02-terminal.py，需操作者在正常终端显式调用。它从自身位置解析 worktree，保持测试范围和离线环境，运行一次并保存独立证据，不提权、不安装、不改变 Core/Gate、不自动重试。脚本仅静态审阅，未在当前环境执行；工具环境未获新的外部运行权限。

```powershell
python C:\Users\svictor\workspace\tools\aiw\.wt\workflow-automation\openspec\changes\workflow-automation\run-e02-terminal.py
```

脚本打印 Evidence 路径及退出码。如果正常终端也失败，需基于该环境的权限信息继续诊断，不能直接认定沙箱是唯一原因。

%% BLOCKED: 当前环境的根路径解析权限问题未解决；完整 E02/AC 验收未通过。2.2/3.1/3.2 与 Gate 保留。诊断增强不是权限问题修复，不安装 worktree 版本或放宽生产路径检查。
