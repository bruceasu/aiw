# FD-089 独立实现审阅

<!-- aiw-data: FD-089-review.json -->

结论：通过。未发现阻断验收的正确性、规范或证据问题。此结论来自静态审阅，
不代表实际构建或安装已验证。

- FD：FD-089，Revision 4。
- 来源事件：FD-089-000004-review-requested。
- 独立 Reviewer session：reviewer-FD089-direct-20261018。
- 实现提交：4ed79d6；比较基线：7c0b070。

检查 build.py、scripts/local_packages.py 的完整变更及 usage/spec 变更，
并追踪 archive、repository、Operation、Manager 和 package_definitions 的相关调用。
build 已移除隐式发布；组件清单使用 expand(component) 保存闭包，包括 build all
生成的组件清单。显式 publish/publish-all/export 仍调用原发布路径。

LocalPackages 在内存建立精确身份目录，原址 fetch 不写 ZIP 或索引；源路径使用
safe_path 拒绝链接及逃逸，entry_for 校验元数据、完整归档哈希和大小。
重复身份内容冲突被拒绝；旧清单只解析声明的 name/version，并在五个类别内要求
唯一候选，拒绝缺失、歧义和身份不符。所有清单源和闭包先于 Manager 构造读取。

单组件及集合仅选唯一请求包为 root，目录闭包不变成手工安装 roots；分组保留
原清单 roots。Manager 构造前，全部当前平台载荷目标与全部源 ZIP 及其祖先比较；
安装根先按 Manager 同样策略解析，再检查根内路径。共享 Operation 的精确版本、
循环、平台、目标归属和最终依赖图检查未变；prepared 仍在写入前保存载荷与脚本字节，
Manager 继续使用既有事务与备份机制。

实际命令：定向 Get-Content/rg；一次 git diff 7c0b070 4ed79d6（含 stat 和
指定源码、usage、spec）；aiw fd --help/show/claim/emit 的帮助及受管事件操作。
最初 worktree 内 .ai/fd 读取路径不存在，改由 aiw fd show 获取真实 pending receipt；
未写入或伪造 receipt。一次 rg 使用 PowerShell 未展开的文件通配符而报路径错误，
之后以明确源文件定向读取，没有执行代码。

Worker 报告记录两个 Python 文件 compile-only 成功；Reviewer 未重复编译。
未运行测试、实际编译/组包、安装、发布、clean、网络、lint 或验证脚本。

%% 实际安装行为未运行验证；仍保留既有单写入者假设和必要事务暂存。
%% 原始 ZIP 有额外元数据与哈希读取；旧清单缺失精确依赖时需要重新构建。

下一步由主代理提交审阅证据，按当前受管通过事件检查交付、清理和归档门禁。
