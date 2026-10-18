# FD-090 独立实现审阅

<!-- aiw-data: FD-090-review.json -->

结论：通过。未发现阻塞验收的实现缺陷。

审阅者：`codex-fd090-reviewer-20261018-1427`；已独立认领
`FD-090-000004-review-requested`。审阅 HEAD `7551f5d`，比较基线 `1851532`，
FD Revision 4；审阅者未编写实现。

同一审阅中发现两个 argparse 帮助仍写 `Build and publish`。Worker 在
`7551f5d` 中仅修正为发布已有包的文案并更新实现证据；已定向确认该补充 diff，
不涉及功能逻辑变动。最终没有未解决 findings。

静态证据：

- publish 分支只获取目标、构造 release adapter、物化所有 fetch 路径，再调用
  export_repository；没有 compile、build、manifest 或 snapshot 写入路径。
- publish-all 直接扫描五个 dist 类别的指定版本目录；不使用 latest 清单或
  expand(all)，因此可选择旧版本和已移除来源的包。
- named publish 用规范别名查找唯一根 ZIP；closure 从包元数据的精确版本引用
  递归解析，循环由已选身份集合终止。共享 install 的原 exact lookup 语义保留。
- safe_path 在目录遍历和原始文件读取前拒绝链接/reparse point；entry_for/read_package
  检查文件名、元数据、内部文件和哈希。根重复身份即使内容相同也被拒绝；
  closure 的跨类别歧义由唯一 exact lookup 拒绝。
- 所有 entry 的 fetch 校验和原址列表物化均在首次 Repo.publish 前完成。
  原 Repo 不可覆盖、逐包发布及可能部分成功的语义保留，未改共享引擎接口。
- usage 和稳定 spec 已更新为独立阶段；实现报告如实区分 Python 语法编译与
  未执行的包发布运行验证。

实际执行：定向 Get-Content、rg、git diff/--stat 读取源代码、FD、报告、规范和
主工作区事件；初次读取推定 receipt 路径不存在，之后读取实际 events 文件。
另执行 `git diff b8c90ba 7551f5d -- build.py docs/features/reports/FD-090-implementation.md docs/features/reports/FD-090-implementation.json`
确认帮助修订及证据；执行 `aiw fd --help`、`aiw fd show FD-090`、
`aiw fd claim FD-090 FD-090-000004-review-requested --session codex-fd090-reviewer-20261018-1427`、
`aiw fd emit --help`；最终使用受管 emit 记录本报告。只写本审阅证据及 FD TODO/Verification，
未改实现、未执行 Git 写入。

未运行测试、编译、build、publish、install、clean、产品帮助验证或网络请求；
Worker 的两文件 compile-only 通过属于已提供的实现证据，本审阅没有重跑。

%% 实际多包发布尚无本轮运行证据，仍可能按已有语义部分成功。
%% 安装目录的 aiw pkg EvalSymlinks 故障独立于本 FD，用户批准的一次帮助命令已失败；
本审阅未重跑、未声称已解决，交回主代理在单独范围继续处理。
