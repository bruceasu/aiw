# FD-090 实现报告

<!-- aiw-data: FD-090-implementation.json -->

已移除 publish/publish-all 对 build_packages 的调用。新增 LocalPackages.from_release：
按指定版本读取五个 dist 类别下已有 ZIP；单包使用规范身份查找，其依赖/集合按包内
精确声明解析。publish-all 不依赖当前 source catalog 或 latest 清单，可发布旧版本
和已从源码删除的包。无产物时提示先 build，不会自动编译或组包。

原址路径检查先于遍历与读取，拒绝根文件名/版本不符、重复身份（即使相同内容）、
歧义及缺失依赖。完整 closure 的 fetch 校验和路径列表先全部物化，再调用 Repo；
只写配置仓库，未改 ZIP、构建清单或 snapshot 状态。保留不可覆盖及部分发布语义。
共享安装 adapter 的 exact lookup 提取为类方法，原安装流程保持不变。

静态证据：一次 git diff 1851532（两个 Python 文件）和 docs/spec 发布条款 rg，
追踪发布分支、源文件验证、闭包完整预检、原址传递和显式目标。同步移除文档中
“publish 会编译 snapshot/重新组包”的旧条款。

实际 compile-only 命令：

```text
python -c "from pathlib import Path; paths = ('build.py', 'scripts/local_packages.py'); [compile(Path(p).read_bytes(), p, 'exec') for p in paths]; print('Python compile-only: 2 files passed; no imports or output artifacts')"
```

退出码 0，无导入或输出产物。根 compile 脚本只编译 Go，因此使用 Python 编译器。
独立审阅进行中指出两个帮助字符串仍称 Build and publish，现仅更正为 Publish
existing；本次源变更后重复同一窄范围 compile-only 命令并通过，没有额外运行验证。
另执行定向文件读取、Git 状态/提交、aiw fd 状态和 aiw git wt add FD-090。
直接用户请求无 Worker 派发，source_event 为 null，使用受管 request-review。

另一个已安装插件故障已获用户明确批准运行一次：

```powershell
& 'C:\green\aiw\aiw.exe' pkg --help
```

退出码 1，复现 resolve plugin directory: The system cannot find the path specified。
静态检查 C:\green\aiw\plugins\aiw-pkg 及文件均存在，未修改安装目录；此故障独立于
本 FD 发布验收，将在后续独立范围处理。没有重跑获准的帮助命令。

未执行测试、实际包构建、发布、安装、clean 或网络；未声称运行发布验证通过。
%% 多包发布仍可能部分成功，同身份不同内容仍要求新版本。
%% 插件路径问题尚未由本 FD 修复，发布通过不代表该问题解决。
