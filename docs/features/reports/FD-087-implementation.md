# FD-087 实现报告

<!-- aiw-data: FD-087-implementation.json -->

已实现 --compile-version 显式来源与 --version 包版本分离。来源按精确版本选择
覆盖当前 target 的成功记录，并经过既有完整路径集合/哈希校验；缺失或失效在
组包前报错，不会重编译或回退。--version、AIW_VERSION、来源版本依次决定包版本，
显式来源不受 AIW_VERSION 的编译筛选影响。未指定来源时保留 FD-086 原有行为。

组包路径、info、依赖和集合成员继续统一使用包版本；原生二进制及其内嵌版本保持
编译版本。控制台同时显示两者，组件、target、all 的分组清单写入 compile_version
来源字段。重新组包不改 compile records，latest 仍查询包版本。Publish 保持新编译。
同步更新 usage 与 stable spec 的例子、环境优先级及不可用历史产物边界。

静态证据：一次 git diff fb67c60，限定 build.py、usage 和 stable spec。追踪参数
分支、来源验证先于写入、两种版本传递、独立组包脚本的 --version、精确依赖/集合
版本、manifest provenance 与既有查询兼容性；直接检查文档措辞及场景位置修订。

实际 compile-only 命令：

```text
python -c "from pathlib import Path; compile(Path('build.py').read_bytes(), 'build.py', 'exec'); print('Python compile-only passed; no imports or output artifacts')"
```

退出码 0，没有导入执行或保留产物。既有 scripts/compile.py 仅编译无关 Go 入口，
因此选 Python 编译器。另执行定向 Get-Content/rg、Git 状态及提交、aiw fd 状态和
aiw git wt add FD-087 等本地工作流命令。直接用户请求无 Worker 派发，source_event
为 null；独立实现审阅通过受管 request-review 获取。

未运行测试、实际目标编译、包构建、CLI 行为验证、clean、安装、发布或网络。
%% 可执行程序报告的版本将按设计保留 snapshot，不会变成包版本 1.0.1。
%% 已覆盖的历史编译不保存；当前源码/资源可能与所选二进制的源码时点不同。
%% 保留单写入者与不可覆盖发布规则；运行行为尚未验证。
