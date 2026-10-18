# FD-096 有界独立设计反馈

<!-- aiw-data: FD-096-design.json -->

本次仅审阅 FD 第 2 版和现有静态调用链，不代表实现验证。未领取工作流事件；由 Planner 授权提供一次有界设计反馈，source_event 为 null。

结论：**ready**，以 Planner 已确认纳入 FD 的以下有限澄清为准。索引优先和接受本地修改的方向符合用户要求，无需增加新缓存、记录格式或检查体系。

## 需明确的契约

FD 第 2 版写明所有 dry-run 都不获取归档，但仓库索引没有 files，未获取新包时无法完成同样的载荷预检。Planner 已采纳反馈：仅全部节点匹配的 no-op dry-run 零获取；changed dry-run 仍完整 prepare/preflight，不写安装状态、不运行 hook。这保留现有预览契约。

Planner 同时确认零目标文件检查承诺仅用于仓库 named install/update；build.py 的 LocalPackages 初始 catalog 和先于 Manager 的输入 ZIP 防覆盖检查保留，因此不能把 build-local install 宣称为零解包或零目标检查。这里只通过精确 root_requests 消除后续重复的 root decode，不扩大本 FD 改动范围。

## 可实施的主体设计

- 使用 index 的 name/version/fingerprint 对比安装记录，匹配节点不调用 fetch、read_package、Manager.install 或目标文件检查。锁下复核记录快照，并仅在原因或成员元数据变化时保存记录，足以保留并发和手动安装语义。
- retained 节点仍从 index 获取完整 dependencies/members；update-all 对集合套用旧成员交集，显式安装集合仍展开完整成员。不可用已裁剪的 current.members 替代完整 index members。最终状态保留未选择的已安装节点并校验精确版本和循环。
- changed 节点统一 prepare 返回原路径及已校验载荷，Repo.fetch 保留原返回类型。获取时原始 SHA 只算一次，解码后的身份、依赖、成员、平台必须与已验证索引绑定；在任何 self-update 写入前完成相关输入的准备。
- metadata-only record 仍验证相对路径语法和保留状态路径排除。删除 install/update 的旧内容哈希拒绝，保留写入时路径、类型、外包归属、快照和配置保护；不必将新的本地完整性检查移到其他入口来补偿。
- LocalPackages 初始索引仍需读取 ZIP，这是当前本地 manifest 仅存路径的限制。其 preflight_destination 现会检查所有文件目标；Planner 已明确保留此输入保护并将其排除于仓库 no-op 零目标检查承诺之外。仍保护全部输入 ZIP 及其祖先免受变化载荷覆盖。
- 512 MiB 总量只计算实际获取的归档，跳过节点不应调用 ZIP stat 或累加大小；无需改变现有单包上限。

## 静态依据与限制

检查了 operations.py 的根选择、visit、preflight、final 校验及 reason；manager.py 的 record、preflight、unchanged、snapshot 和 install 快路径；repository.py 的索引校验及 fetch；LocalPackages 的索引、fetch、输入保护；build.py 安装入口。现有 Manager.install 即使同指纹仍先遍历目标，不能作为 retained 节点的执行入口。

只执行读取和检索命令并写入此反馈双报告；未改实现、FD 状态、事件或 Git，未运行测试、运行验证、编译、构建、安装或网络。性能收益尚无测量，不能声称具体提速比例。
