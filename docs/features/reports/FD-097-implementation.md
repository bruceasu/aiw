# FD-097 安装目录 dotenv 启动加载报告

<!-- aiw-data: FD-097-implementation.json -->

已在 aiw main 分发任何命令前调用 LoadExecutableDefaults，仅加载 executable 目录的 .env。
不存在不报错；读取、展开或导出失败在 dispatch 前以 error 退出，不输出变量值。
不使用 cwd、AIW_ROOT、向上搜索或 profile 来选择启动文件，不改变 build.py 行为。

复用 env.Loader，新增私有 defaults-only 模式。已有进程变量（包括空值）在单引号、
多行和普通赋值中均保留，普通/多行展开前跳过覆盖。展开优先读取父进程值；Windows
新默认键及引用/循环检测键归一化，避免大小写变体随机覆盖。新增 UTF-8 BOM 支持；
种子环境忽略无法由普通 Setenv 表示的空 pseudo-key，但不修改实际父环境条目。
循环展开从 panic 改为返回错误；默认非 defaults 模式继续原有文件覆盖语义。

插件仍通过 os.Environ 继承，所以 aiw git/pkg 及其子命令可见安装目录默认变量。
原有引擎专属 metadata 注入不变；直接运行独立插件没有此 main hook。
文档明确：导入的安装默认值成为进程环境，因此优先于 Go LLM 的项目 dotenv 回退，
显式功能配置优先级不变；各消费者对空值的原有回退不重写。

实际执行离线仅编译命令，两个 Go entrypoint 输出 NUL，exit 0：

```powershell
$env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTOOLCHAIN='local'; $env:GOWORK='off'; $env:CGO_ENABLED='0'; $env:GOFLAGS=''; python scripts/compile.py
```

一批最终 source diff/new helper 读取和工作区状态静态检查已完成。未运行测试、gofmt、
任何实际启动命令、构建、安装、发布或网络。已安装 aiw.exe 和所有 .env 文件均未修改。

用户指定主工作区，无 Worker handoff，source_event=null，使用受管状态/review 路由。
原先已有的 src/plugins/aiw-git/git-guide.py 修改保持原样，未纳入本 FD 暂存/提交。
独立实现审阅尚待完成；不声称已安装程序已加载 dotenv。