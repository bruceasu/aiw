# FD-088 实现报告

<!-- aiw-data: FD-088-implementation.json -->

已按用户的新模型完成两个 Work Items。compile 不接受版本参数且忽略 AIW_VERSION，
编译函数不再接收版本，Go 内嵌标识固定 snapshot。build/publish 共用固定 snapshot
前置流程：有有效覆盖记录就校验后组包，无记录/旧版本记录/覆盖不足/普通产物缺失时
自动编译 snapshot。build 的显式版本不再强制重编译，--compile-version 已移除。

包版本仍由 --version、AIW_VERSION、UTC snapshot-日期时间依次决定；原生产物始终
snapshot。成功清单 compile_version 固定 snapshot。latest 查询包版本；保留的
latest-compile 仅验证并打印 snapshot，失败时不触发编译。clean 范围和保护不变。

缺失检查先完整扫描其余产物，再区分缺失与新增/修改。链接、非常规文件、权限和
损坏记录异常不会被补编译吞掉；即使无可用 provenance，也先预检现存原生产物。
原有覆盖前失效、全部成功后记录、双平台和完整哈希保护保留。旧非 snapshot 记录
可读取但不复用。同步替换 stable spec 和 usage 的旧选择规则，更新 .env.example。

静态证据：一次限定路径 git diff 7893cd6 加 rg 调用/遗留选项搜索，追踪所有编译
入口、固定 linker、包版本与快照解耦、全部剩余文件扫描、纯查询和历史记录边界。

实际 compile-only 命令：

```text
python -c "from pathlib import Path; compile(Path('build.py').read_bytes(), 'build.py', 'exec'); print('Python compile-only passed; no imports or output artifacts')"
```

结果退出码 0，无模块导入执行或产物。scripts/compile.py 仅针对无关 Go 入口，故用
窄范围 Python 编译器。另运行定向 Get-Content/rg、Git 状态/提交、aiw fd 状态、
aiw git wt add FD-088；以一次 Python 文本编辑更新已有 stable spec 的相关章节。
直接用户请求无 Worker 派发，source_event 为 null，采用受管 request-review。

未执行测试、真正编译目标、包构建、运行 CLI 验证、clean、安装、发布或网络。
%% compile --version 和 build --compile-version 按用户新模型有意移除。
%% snapshot 不检测源码变化；需要更新原生代码时须显式 compile。
%% 包版本相同但内容不同仍被 Repo 拒绝；单写入者和清理部分失败限制保留。
%% 运行行为尚未验证，等待独立静态审阅。
