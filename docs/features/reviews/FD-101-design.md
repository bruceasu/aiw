# FD-101 独立设计审查

<!-- aiw-data: FD-101-design.json -->

结论：**ready**。审查 FD 修订 2；未发现阻止实施的设计问题。本报告仅为设计证据，不代表实现验收。

来源事件：`FD-101-000002-design-requested`（由 Planner 提供）；审查会话：`fd101-design-independent`。

## 设计判断

- 用户问题来自两处重复规则：`component_dependencies` 为所有插件扩展 runtime，`build_component` 又为所有插件生成同版本 runtime 引用。用一个实际消费者声明同时驱动规划和元数据，能解决根因而无需修改安装器。
- 全插件静态名称搜索中，`file`、`patch` 直接导入 `aiw_codec`，`notify` 通过子进程执行 `plugins/send_teams_msg.py`。设计保留三者的精确 runtime 依赖，覆盖了非 import 的真实消费路径。`aiw-pkg` 的源码导入属于标准库或自身载荷模块，无这两个共享模块的使用。
- `build.py expand` 的普通目标递归使用 `component_dependencies`；`build_packages` 以 expand 结果构建并写入每个组件的版本清单。因此改变声明并让插件元数据复用它，可同步消除非消费者的构建闭包和安装依赖。`plugins` 兼容组仍显式含 runtime，集合通过真实消费者展开仍含 runtime，范围一致。
- 新 `aiw-pkg` 包的依赖为空，升级它不要求替换已安装 runtime；其他插件的旧精确依赖因此保持满足。旧 ZIP、索引、安装记录和全局精确反向依赖检查均保留，不应把新规则描述为已经修复现有已发布包。
- 保留真实消费者的现有精确同发行版本契约，避免未经证明的 ABI 兼容承诺。显式消费者声明是人工维护点；新增共享模块消费者必须更新该声明，FD 已在验收中记录。

## 覆盖与边界

工作项 1.1 覆盖依赖唯一来源、非消费者解耦、三个消费者和集合闭包；工作项 1.2 覆盖稳定规范、使用文档、旧包边界与真实验证记录。两个工作项规模和依赖关系合理，无额外设计 Gate。

实施时保持包名、载荷、集合配方和安装器校验不变；不要扩展为 ABI 范围、依赖解析器或 runtime 版本治理重构。实际安装结果仍须后续另行授权验证。

## 实际证据与限制

仅执行 PowerShell `Get-Content`、`rg`、`Select-Object` 静态读取，检查 FD、共享打包定义、expand/build_packages 路径、源码共享模块引用和相关规范。两次读取不存在的历史审查/模板路径返回缺失；未重试，报告按工作管理契约编写。未执行测试、编译、构建、规划模拟、网络、安装或 Git 写入。

下一步：Planner 可按本 FD 进入 design-ready；实现后仍须独立实现审查。
