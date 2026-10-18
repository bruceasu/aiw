# FD-084 独立实现审阅

<!-- aiw-data: FD-084-implementation-review.json -->

结论：**通过静态审阅**。未发现阻塞性问题；不宣称构建、发布或安装已经实际运行。

审阅者 session：`reviewer-FD084-20261018-implementation-review`。
已真实 claim 来源事件 `FD-084-000004-review-requested`。审阅实现为
`14c73de`，比较基线为 `16a106d`，FD 为 Revision 4。工作树已有的 Revision
3→4 属于受管请求审阅更新。设计最终 ready 证据读取自主工作区
`docs/features/reviews/FD-084-design.md`；父分支该证据提交为 `2390d44`，
不包含本次实现变化。

静态验收证据：

- `build.py` 的 `target_catalog` 覆盖程序、动态插件、Skill、共享资源、docs 和集合；
  `expand(all)` 对每个组件递归依赖与成员后统一去重。aiw-main 特殊映射到 aiw，
  集合名称保持原样，其他规范包名通过目录映射回组件；原短名和兼容分组仍可接受。
- 新 publish 命令先检查必需的 `AIW_PACKAGE_REPO`，复用 Repo 的本地/远程和
  Windows 盘符判定；相对目标以 ROOT 解析。缺失或远程配置在 compile/package 前失败。
  `build_packages` 返回本次指定版本的 ZIP，直接交给目标 Repo；发布路径不读旧 manifest。
  原 dist/repository 暂存、不可变发布及失败即停止行为保留，多包不是原子事务。
- `.env` 固定从 checkout ROOT 读取，无新增依赖、执行或插值；`setdefault` 保留进程优先级。
  Go/Node 工具从环境查找，安装默认值在加载环境后取得；双平台 amd64 常量保持。
  独立包脚本经 `package_definitions.run` 加载同一环境，显式版本优先。
- `scripts/packages/docs.py` 采用已有独立包入口。aiw-docs 仅取 docs/usage 与可选根 README，
  安装到 docs/reference/usage 与 docs/reference/README.md；不与现有程序插件 docs/usage
  所有权重叠，不读取 features 报告、根 .env 或 dist。
- 检查原 compile/build/install/export 分支、manifest 与安装调用，及用法、稳定规范、
  `.env.example`、ignore 规则的一致性。Worker 报告明确区分 compile-only 与未运行项；
  三个 Python 文件的编译成功为 Worker 已记录证据，本 Reviewer 未重跑。

实际命令：定向 `Get-Content -Encoding utf8`、`Select-Object`、`Select-String`、
`Get-ChildItem`、`rg`、限定路径 `git diff 16a106d`、`git diff --stat 16a106d`、
`git status --short`，以及 `aiw fd --help`、`aiw fd show FD-084`、claim/emit 子命令帮助、
精确 claim 和随后真实 emit。首次部分共享资源、模板与 receipt 路径不存在；根据已发现的
真实路径读取，未运行测试或通过更换 shell 探测权限。写入仅为本报告、JSON 与 FD 审阅记录。

%% 未运行测试、编译、包构建、publish/install、网络、验证脚本或 codex-auto-review。
%% dotenv 边界值、实际工具调用及双平台包完整性仍缺运行证据；本次授权以静态审阅为准。
%% 本地仓库多包发布可能部分成功；同版本内容改变仍由既有 Repo 拒绝。
