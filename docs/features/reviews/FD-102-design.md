# FD-102 独立设计审查

<!-- aiw-data: FD-102-design.json -->

结论：ready。审查 FD Revision 2，来源事件为 `FD-102-000002-design-requested`。这是设计证据，不是实现验收或发布许可。

用户已明确要求全部插件和全部技能分别合包，并豁免兼容。两个真实载荷包比保留单包再增加集合更直接符合目标；删除旧目标、要求干净安装根、不自动迁移旧安装记录，范围明确，没有要求恢复已豁免的兼容性。

静态追踪 `build.py` 的目录发现、catalog/aliases、expand、compile_target/latest_compile、build manifest 和 install 根选择，以及 `scripts/package_definitions.py`、`scripts/local_packages.py`、`skill_management.source_root`。未发现需要先修改设计才能实施的阻断项。

- 原生编译不能直接沿用包级 expand：新 plugins 标记本身不在 GO 表中。设计已经单列源码编译组件展开，并要求编译、复用、inventory 和失效使用同一展开，覆盖 req/say/packer/file-convert 双平台输出。实现审查应据此追踪所有调用点。
- 原单插件规则含 file/patch/notify 私有 helper、say 配置/资源与 usage 文档。设计明确复用这些规则后合并文件列表，避免简单复制整树引入开发 shim 或遗漏原生文件。
- skills 整树与根共享 Markdown 同归 aiw-skills，保留 `skills/<name>` 路径，符合 source_root 默认位置及后续 agent 安装。移除 main/skill 文本推断出的精确依赖，可避免重建原有版本联动；core/all 承担整套选择。
- plugins/skills 必须退出 groups，alias 对应唯一实体，manifest/install 仅选一个显式根。现有 LocalPackages 可以处理这两种类别及一个 ZIP，不需要扩大通用仓库接口；输入 ZIP 防覆盖与安装所有权保护继续适用。
- 旧包已发布产物、旧仓库与旧根不修改；generic pkg 仍可读取历史包，这与删除旧构建目标并不冲突。

保留风险：合包可能接近 512MiB 限制；设计保留限制，未声称实际尺寸或运行通过。原有 tree_entries 的排除规则应维持，配置及平台标记不能在合并时丢失。两者属于实现审查点，没有未决产品选择。

实际仅执行 PowerShell 的 Get-Content、Select-Object 和 rg 读取源码及设计；一次读取不存在的设计报告模板失败，未重试该命令。未执行测试、编译、构建、功能模拟、网络、Git 写入或工作流状态操作。
