# FD-100 独立设计审查

<!-- aiw-data: FD-100-design.json -->

结论：**ready**。FD Revision 2 对问题、输出契约、兼容性和本轮阶段限制的描述足以进入后续实施准备；本结论不启动 Worker，也不是实现验收通过。

本次来源事件：`FD-100-000002-design-requested`。审查对象为 `FD-100_IMPROVE_PACKAGE_SEARCH_AND_SHOW_OUTPUT.md` Revision 2；仅作一轮有界静态设计审查。

## 发现

无阻塞发现。

1. 名称诊断正确。`main` 的 search 使用 casefold 后的片段包含匹配，`repository.select` 使用完整名称精确相等。因此示例中的短名无法安装，添加版本也不会补全名称。保持安装解析不变、展示完整包名并提供正确命令，能解释现象而不扩大有副作用命令的范围。
2. search 的表格复用合理。`format_package_list` 使用 sorted 生成展示序列，没有原地修改输入列表；每个匹配版本一行可以保留多版本信息。JSON 分支须直接序列化原结果，保留原顺序和全部元数据，FD 已明确这一约束。
3. show 的本地语义完整。现有 `Manager.record` 接受 installed/removed，校验文件路径、配置标记、依赖/成员引用及安装原因后返回原记录。设计保留该调用及异常路径，缺少 reason 时显示 manual、缺少可选字段时展示占位符，不将错误记录伪装成有效记录，也不写回默认值。
4. `Members (recipe)` 明确表达历史配方版本，与 aiw-packages 中集合成员可独立升级的规则一致；不查询仓库或推断当前成员版本可以避免误报。完整文件和额外字段继续通过 show --json 输出。
5. 公共输出兼容性变化已明确。search/show 默认文本和新增 --json 需要同步使用文档及稳定规范；尤其要修改现有规范中“Other subcommands MUST retain their existing output”的表述，使 search/show 成为明确的新契约。FD 的范围与 Work Item 1.3 已覆盖这一工作，无需本轮修改稳定规范。
6. 参数路由可局部修改。show 与 install/remove/purge 共用参数创建循环，search 与 info 共用另一循环；新增 --json 应仅注册到 show/search，并只在对应分支读取。不能把最终输出判定改成所有 action 都直接访问 args.json。FD 的 info 原输出及其他子命令不变验收已覆盖此风险。

## 范围与证据

三个 Work Items 均有单一结果、small/low 估计、依赖和完成证据；Acceptance 可映射到相应条目。没有材料表明需要新的别名机制、记录迁移、依赖或持久化设计。

实际执行了四批只读 PowerShell 检索/读取，使用 `Get-Content`、`Select-String`、`Select-Object`、`rg` 和一次限定路径的 `Get-ChildItem`；读取了 FD、fd-workflow 规划规则、共享契约/资源预算，以及 `aiw-pkg.py`、`repository.select`、`Manager.record`、相关文档和稳定规范。第四批用于补足本地记录校验及文件元数据的关键证据。第一批文件路径筛选没有命中而返回非零，未重跑该命令；其余读取完成。随后仅写入本审查 Markdown 与 JSON。

未执行测试、编译、可执行功能验证、网络、Git 写操作、安装或构建；未改源码、稳定规范、FD 状态或原有未跟踪脚本。

## 后续边界

用户清理父工作区后，Planner 可按本设计继续准备正常 FD 隔离实施。真正实现仍需检查 JSON 分支字段完整性、空记录、removed 状态、历史成员版本、配置文件计数及参数隔离，并取得独立实现审查；这些尚无运行结果。
