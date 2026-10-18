# FD-086 实现报告

<!-- aiw-data: FD-086-implementation.json -->

两个 Work Items 已实现。compile 和内部 build/publish 编译在 bin/.compile.json
记录版本、覆盖组件、时间及完整产物哈希；任何重叠原生编译开始前先使旧记录失效。
只有全部编译及产物扫描成功才登记。build 无 --version 时优先选覆盖目标的最新
成功记录，AIW_VERSION 限制复用版本；没有记录才生成/选择版本重新编译。
显式 --version 和 publish 仍重新编译。latest-compile [target] 查询有效编译版本。

复用比较完整原生路径集合与 SHA-256，包括两套 Go 产物及所有 TypeScript 输出，
拒绝新增、删除、改写或链接。设计审阅 D1 已在规划和实现中解决。无原生产物的
资源编译也能保存版本。既有无 provenance 的输出不会直接复用。

clean 仅处理 checkout bin/dist；删除前检查两个绝对目标归属和所有后代，拒绝
链接、junction、非常规文件及异常目录。两个树都预检完成才开始删除，缺失目录
不报错；不会操作外部仓库、安装目录、源码或 .ai 工作流记录。

静态证据为一次限定路径的 git diff 23caf71，追踪缓存选择/失效、先失效后覆盖、
完整文件集合比较、别名/分组覆盖、版本分支、clean 双树预检及 docs/spec 一致性。

实际 compile-only 命令：

```text
python -c "from pathlib import Path; compile(Path('build.py').read_bytes(), 'build.py', 'exec'); print('Python compile-only passed; no imports or output artifacts')"
```

结果退出码 0，无模块执行或最终产物。scripts/compile.py 仅针对无关 Go 入口，
故使用窄范围 Python 编译器。另运行定向 Get-Content/rg、Git 状态/提交和 aiw fd
状态、aiw git wt add FD-086 等本地流程命令。直接用户请求无 Worker 派发，因此
source_event 为 null，后续通过受管 request-review 获取独立结论。

未运行测试、实际编译目标、包构建、clean、安装、发布、网络或验证脚本。
%% 复用不检测源码变化，源码/资源改动后如需新原生二进制须重新 compile。
%% 同版本内容变更仍受 Repo 不可覆盖规则约束；需要新版本时重新 compile。
%% 哈希增加本地磁盘读取，流程假设单写入者；清理遇到系统错误可能部分完成。
%% 实际运行行为尚未验证，独立静态审阅待完成。
