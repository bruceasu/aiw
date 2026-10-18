# FD-094 junction 路径解析修正报告

<!-- aiw-data: FD-094-implementation.json -->

已确认已安装二进制与 snapshot 的 SHA-256 相同，且包含 FD-091 resolver 符号。
这次不是仅凭猜测认为旧二进制。PowerShell 确认 C:\green 是指向 D:\green 的 junction。
用户明确批准一次离线 Go 路径诊断，实际退出 0：

- 插件目录及其下层祖先 Lstat 为普通目录。
- C:\green 的 Go Lstat 类型为 ModeIrregular。
- 对完整插件路径及 C:\green\aiw 的 EvalSymlinks 失败。
- 对 C:\green 本身 EvalSymlinks 返回 C:\green，未转换物理目标。

本机 Go 标准库源码确认：mount-point tag 被标为 irregular，filepath walker 仅跟随
ModeSymlink；Windows os.Readlink 明确支持 SYMLINK 和 MOUNT_POINT 两种 reparse tag。

实现显式 Readlink 跟随 symlink/irregular 祖先，拼接已检查的 suffix，重新完整检查。
最多跟随 255 次。相对目标基于链接父目录；卷相对目标拒绝。读取失败带具体路径，
未知 irregular 对象不允许退回未检查的词法路径。原相对入口约束、Rel containment、
最终 Stat 和目录入口拒绝均保留；普通路径不依赖 case enumeration。

实际命令：

```powershell
# 用户明确批准的一次诊断，已经消耗，不重跑
$env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTOOLCHAIN='local'; go run ./.ai/diagnostics/plugin-path.go
# 修改后仅编译，exit 0，两个 entrypoint 输出到 NUL
$env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTOOLCHAIN='local'; $env:GOWORK='off'; $env:CGO_ENABLED='0'; $env:GOFLAGS=''; python scripts/compile.py
```

已作一次最终实现 diff 静态检查。未执行测试、gofmt、最终构建、安装、发布或网络。
未执行修正后的运行验证，未替换 C:\green\aiw\aiw.exe 或现有 ZIP。
当前主工作区实现无 Worker handoff，source_event=null，走受管状态和独立 review 路由。
独立实现审阅待完成；不声称已安装程序恢复运行。
