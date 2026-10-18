# FD-091 实现与诊断报告

<!-- aiw-data: FD-091-implementation.json -->

用户报告 aiw pkg 在 C:\green\aiw\plugins\aiw-pkg 的目录解析失败。已获明确批准的一次
`& 'C:\green\aiw\aiw.exe' pkg --help` 退出码 1，复现同一错误；未重跑。目录、manifest
和 Python 入口经只读检查均存在，没有 reparse 属性。未修改任何已安装文件。

诊断结论属于有依据的机制推断：当前 Go 1.25.1 标准库 EvalSymlinks 在 Lstat 遍历后
调用 Windows FindFirstFile 做大小写规范化，返回裸系统错误；用户观察到的是此类
错误形态。具体失败路径段未测量，不能声称已证明其值或更新后的运行成功。

新增内部 resolveManifestPath：绝对化输入，逐一 Lstat 非卷根的所有路径组件。
只有类型精确为普通文件或普通目录才直接返回绝对路径，避免多余 case enumeration。
任何 symlink、junction 类 irregular 或其它类型仍走原 EvalSymlinks 并传播失败，
没有“忽略错误后使用词法路径”的回退。使用完整 ModeType 比较，兼顾旧 Windows
可能同时带 Dir/Symlink 位的类型。目录和入口均使用同一 helper；相对路径约束、
filepath.Rel 边界、最终 Stat/目录拒绝及 manifest 错误上下文保持不变。

静态证据：一次限定路径 git diff b4aba07，追踪所有祖先、普通类型证明、链接物理
解析及错误分支；检查文档和 stable plugin contract。共享 CLI/启动方式没有改动。

实际 compile-only 命令（PowerShell）：

```powershell
$env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTOOLCHAIN='local'; $env:GOWORK='off'; $env:CGO_ENABLED='0'; $env:GOFLAGS=''; python scripts/compile.py
```

退出码 0。既有脚本将 cmd/aiw、cmd/aiw-req 编译到 NUL，缓存限定当前 worktree，
没有保留最终二进制，没有依赖下载。另运行定向文件/Go 标准库读取、Git 状态与
提交、aiw fd 状态及 aiw git wt add FD-091。用户直接请求无 Worker 派发，source_event
为 null，后续通过受管 request-review 获取独立结论。

未运行测试、gofmt、vet、lint、最终构建、安装、发布或网络；未重新执行旧安装程序。
%% 当前 C:\green\aiw\aiw.exe 仍为旧文件；新解析逻辑需重新 compile/build/install 主程序。
%% 更新后的目标运行尚未验证。链接/异常对象不承诺修复全部配置，失败保持 fail-closed。
%% 文件系统并发替换竞态沿用既有单写入者边界。
