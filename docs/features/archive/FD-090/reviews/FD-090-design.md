# FD-090 独立设计审阅

<!-- aiw-data: FD-090-design.json -->

结论：**ready**。审阅对象为 FD-090 Revision 1 及父会话提供的 `LocalPackages.from_release(root, version, name=None)` 方案；这不是实现验收，也不是插件故障修复结论。

未发现阻止设计进入实现的未决事项。让 publish 读取已有 ZIP、保留 build 的组包职责，符合用户澄清的阶段边界。直接扫描五类版本目录，能避免 latest manifest 和当前源码目录改变旧版本发布集合；无需另建索引文件或复制暂存包。

## 实现必须落实的边界

- `publish-all` 应枚举指定版本目录内全部 ZIP，并对每个实际路径验证包内版本、文件名和唯一身份。现有 `_add` 允许内容相同的重复身份，因此新 release factory 仍须拒绝跨类别重复路径，不能仅依赖 `_add`。
- `_find` 适合作为共享的精确身份查找入口，保留 install 的既有契约。依赖和集合成员必须遵循包内声明的精确版本，不得从当前 `expand` 或 latest 清单补齐。递归时按身份去重，避免循环引用导致无终止遍历。
- `safe_path` 应在遍历类别目录、版本目录和读取 ZIP 之前生效，不能先 resolve 掉原始链接再验证。`read_package` 已验证文件名、包内数据及逐文件摘要，但目录版本仍需 factory 验证。
- 所有根包、完整依赖闭包及 fetch 再校验应在第一次 `Repo.publish` 前完成。例如先物化完整原址路径列表，再交给现有 export 循环，不能把 fetch 作为写入循环中的惰性生成器。发布期间源文件被并发修改仍由既有逐包摘要检查防护，不声称跨包原子性。
- named publish 的短名映射仅用于规范化根身份；解析发布闭包不能再次调用源码目录展开。保留当前已存在别名即可满足本 FD 的兼容目标；不需要新增分组发布语义。
- `openspec/specs/aiw-packages/spec.md` 当前仍要求 publish 重新 build，且将 publish-all 绑定当前 catalog。Work Item 1.2 应同步改写这些要求及对应场景，保留配置优先级、仓库不可覆盖和部分成功语义。

## 静态依据与范围

已追踪 `build.py` 中 publish → build_packages → snapshot 检查及组包路径；`LocalPackages` 的闭包、查找、原址 fetch 与安装预检；`entry_for` / `read_package` 的身份及摘要验证；`Repo.publish` 的本地不可覆盖和逐包写入。相关范围未发现本地更深层 AGENTS.md。

实际执行：四批仅包含 Get-Content、rg、Select-Object/Select-String 的定向静态读取。第四批用于补足前三批定位到的共享包引擎路径安全、摘要和发布首写边界证据。未执行测试、编译、build、publish、安装、联网、Git 写入或工作流事件。

`source_event` 为 null：本次是父 Planner 委派的有界设计反馈，没有分配独立设计事件，不创建或冒充实现 Reviewer 事件。

残留风险：实际发布行为尚未运行验证；多包发布保留部分成功风险；插件目录 EvalSymlinks 故障单独诊断，不受本结论覆盖。下一步由 Planner 吸收上述实现边界并进入既有 FD 工作流。
