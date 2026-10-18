# FD-086 独立设计审阅

结论：changes-needed。审阅 FD Revision 1；仅使用本地静态证据。本报告属于设计证据，不代表实现验证通过。

## 必须补齐的设计约束

- D1（中）：TypeScript 目录清单必须比较完整文件路径集合，而不只是核对已记录文件的 SHA-256。现有 `legacy_package_component` 会通过 `tree_entries(bin/http-openai-proxy)` 收集当前目录内容；如果 compile 后新增一个未记录的 JS 文件，仅检查原清单的文件存在及哈希仍可通过，而打包会带入未经此次 compile 记录授权的内容。请明确：所选记录覆盖的每个原生组件，当前实际完整文件集合必须等于记录清单，新增、缺失、改变或链接均拒绝；Go 每个组件固定为 Windows/Linux 两个文件。该约束归入 Work Item 1.1 及验收。

## 已确认的决策

- 选择最新且覆盖所需展开组件的单条记录，允许 `compile all -> build aiw`；不得将不同版本或不同完成批次的记录拼接为一次授权。
- 在首次写原生输出前，持久化失效所有原生组件有交集及同 target 的记录；整个 compile 成功后再原子记录成功。该顺序可防止 Windows 成功而 Linux 失败后仍复用前次混合结果；失效写入失败应先停止，不能继续改动输出。
- `--version` 及 publish 强制重新编译；`AIW_VERSION` 仅复用相同版本记录；无匹配记录才生成/编译新版本。选中记录失效时明确报错，不退回旧记录或自动打包。
- 资源包允许空原生清单，保持 compile 版本关联；重用编译结果与当前资源组合的边界已经明示，不承诺源码新鲜度。
- `latest` 保持已成功 build 语义；`latest-compile` 是编译记录查询，应先验证所选记录再输出版本。

## clean 边界

设计中先检查 bin/dist 两棵树再删除的顺序合理。实施时应在 resolve 之前检查原始路径及祖先；现有 `archive.safe_path` 和 `reject_link` 使用 lstat 和 Windows reparse 属性，能够覆盖符号链接、junction 及损坏链接。只对 resolve 后路径做检查会丢失链接证据，应避免。遍历必须不跟随链接，并在任何删除前发现所有链接、非目录根和越界路径；预检失败不应删除另一棵树。

单写者约束明确，不新增锁或并发保证；删除中途操作系统错误可能留下部分清理结果，已有风险说明足够。配置到 checkout 外的发布仓库及安装目录均不属于删除目标。

## 规格与证据

`openspec/specs/aiw-packages/spec.md` 当前仍要求每次 build 先 compile，且无版本时生成 snapshot；Work Item 1.2 必须同步为“可验证 compile 重用优先”的新要求，保留显式发布新编译及 latest build 规则。

静态阅读：FD-086、build.py、src/plugins/aiw-pkg/archive.py、scripts/package_definitions.py 相关路径、aiw-packages 稳定规格及 fd-workflow/references/planning.md。未运行 compile、测试、构建、clean、网络、Git 写入或工作流事件。一次性独立审阅完成；Planner 可将 D1 的明确约束写入 FD 后推进，不要求自动重复审阅。
