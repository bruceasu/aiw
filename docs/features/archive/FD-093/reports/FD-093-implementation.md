# FD-093 安装优化实现报告

<!-- aiw-data: FD-093-implementation.json -->

在当前主工作区实现一次多根依赖规划。build.py 将全部选定根交给一个 Operation，
共享依赖与 collection 成员按名称去重，所有根先登记并保持 manual 语义。
完整图先预检，再逐包事务安装；不承诺跨包整体回滚。

本地索引复用已校验 prepared 元数据，避免再次解包；fetch 仍检查安全路径、
原 ZIP 大小和 SHA-256。Operation 仍完整解包校验并固定所有载荷，支持自更新。
相同版本及指纹的 Manager 分支已移到锁内临时文件创建之前，保留事务守卫与
reason/dependencies/members 更新，变更包保留原暂存、snapshot 和失败恢复路径。

设计独立反馈 ready。保留既有单次 Operation 累计 512 MiB 输入 ZIP 限制，
现在整个多根选择受此限制；超大 all 应拆分显式 target，文档已说明兼容影响。

实际执行五个修改文件的 Python 内存 compile，exit 0；一次 git diff 静态检查。

```powershell
python -c "from pathlib import Path; paths = ['build.py', 'scripts/local_packages.py', 'src/plugins/aiw-pkg/repository.py', 'src/plugins/aiw-pkg/operations.py', 'src/plugins/aiw-pkg/manager.py']; [compile(Path(p).read_text(encoding='utf-8-sig'), p, 'exec') for p in paths]; print('Python compile-only: OK')"
```

未运行测试、benchmark、构建、安装、发布或网络；未声称测得提速比例。
没有持久化缓存或安装记录 schema 变化。用户选择主工作区，无 Worker handoff，
source_event 为 null，经受管状态/review 路由取得独立实现验证。
独立实现审阅尚待进行。
