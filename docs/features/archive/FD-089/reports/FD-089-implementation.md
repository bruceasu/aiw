# FD-089 实现报告

<!-- aiw-data: FD-089-implementation.json -->

已完成三个 Work Items：移除 build 对 dist/repository 的隐式发布，组件清单保存
完整依赖/集合闭包；install 使用 scripts/local_packages.py 的内存目录，fetch
返回原始 ZIP 路径，不复制 ZIP 或写索引。只有显式 publish/export 调用 Repo 写入。
旧 dist/repository 不删除且不再读取；保留 clean 的显式清理方式。

LocalPackages 先读取所有清单 ZIP，再递归解析精确依赖/成员。旧组件清单缺少闭包时，
只在五个已知 dist 类别下查找声明的 name/version 文件；缺失、歧义、身份不符或
重复身份内容冲突均拒绝。路径检查和 fetch 时的完整 entry/hash 重校验保留。

设计 D1-D3 已处理：单组件/集合仅选请求包作为 root，目录闭包不提升为 manual；
分组沿用原显式 roots。Manager 构造前预检当前平台的全部载荷目标与全部原始 ZIP，
拒绝覆盖其他输入或占用其祖先；支持安装根既有链接别名策略，根内仍拒绝链接。
共享 Operation 的依赖图/精确版本/循环/平台/归属检查和事务暂存代码未修改。

静态证据：一次 git diff 7c0b070（build.py、local_packages.py）、diff stat 和 rg
发布/原址 fetch/roots/旧暂存仓库引用，追踪安装路径及 docs/spec 一致性。

实际 compile-only 命令：

```text
python -c "from pathlib import Path; paths = ('build.py', 'scripts/local_packages.py'); [compile(Path(p).read_bytes(), p, 'exec') for p in paths]; print('Python compile-only: 2 files passed; no imports or output artifacts')"
```

退出码 0，无模块执行或保留产物。scripts/compile.py 仅针对无关 Go 入口，使用窄范围
Python 编译器。另执行定向 Get-Content/rg、Git 状态及提交、aiw fd 状态和
aiw git wt add FD-089。本次用户直接请求没有 Worker 派发，source_event 为 null，
独立审阅经受管 request-review 获取。

未执行测试、目标编译、实际组包/安装/发布/清理、网络、lint 或验证脚本。
%% 安装仍需必要的载荷解包与事务快照；仅移除重复的仓库/下载 ZIP。
%% 原始 ZIP 读取次数增加，但不会额外复制归档；运行安装尚未验证。
%% 旧不完整清单只能使用现存精确类别文件，缺失时须重建；保留单写入者限制。
