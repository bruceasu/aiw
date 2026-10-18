# FD-082 独立设计审查 r1

- 结论：**ready**，未发现阻塞设计就绪的问题。
- 审查对象：FD-082，Design revision 2。
- 来源事件：`FD-082-000002-design-requested`。
- 独立会话：`/root/fd080_design`。
- 范围：本地静态设计审查，不是实现审查或运行验证。

## 设计判断

1. **链接边界**：用户显式选择的安装根先解析为 canonical 绝对路径，随后仍由原 safe_path 检查根内路径；这能够接受 C:/D: 的安装根别名，并使锁和包状态落在同一真实目录。没有理由放宽载荷目标、状态目录或恢复路径的链接检查。工作项 2.1 与验收覆盖这一边界。
2. **自动接管**：用户已明确选择普通 install 自动接管。只放宽当前包、当前平台声明目标中的无归属普通文件，其他包归属冲突、目录冲突、链接、受管文件被修改仍拒绝；配置继续保留。此改动与稳定规范中原来的拒绝规则存在有意变更，由工作项 3.1 同步规范及帮助即可，不需新增采用开关。
3. **单备份**：没有 installed 记录而实际覆盖旧文件时，以真实文件快照构造 synthetic legacy previous，避免伪造旧 ZIP 或未知软件版本；新目录不产生虚构历史。有 installed 记录时继续采用正常上一版。工作项 2.2 已覆盖成功才发布 previous、消费唯一备份、权限和字节恢复。
4. **失败恢复**：事务 before 保留真正原 current，synthetic legacy 只用于成功后的 previous，二者不能混用。由此接管失败不会生成 legacy 成功记录，原 previous 也能沿既有轮换恢复流程保留。没有增加脚本副作用或崩溃原子性的承诺。

## 实现时需保持的约束

- **D1，低，非阻塞**：构造 synthetic previous 应修改独立副本，不能改写事务 before 中的原记录。legacy files 应只登记实际被覆盖的旧非配置普通文件及实际摘要；新安装带来的新增文件须能在 remove 时撤销。保留配置继续由现有 merged_configs 处理，并保持后续 purge 可定位配置。该约束已由 FD 的恢复规则表达，不要求新范围。
- **D2，低，非阻塞**：synthetic legacy 没有旧 ZIP，应确保 snapshot/restore、后续升级和第二次 remove 都接受这种有记录、无 current.zip 的状态；不能在恢复时重新依赖包读取器。现有 restore 以快照恢复文件，saved package 缺失时删除 current.zip，具备复用基础。不能把 mode 位恢复宣传为 Windows ACL 的完整恢复；FD 已明确不修改 ACL。

## 静态依据及未运行项

阅读 FD-082、`src/plugins/aiw-pkg/manager.py` 的根解析、预检、snapshot/restore、install/remove 和失败恢复路径，`archive.py` 的 safe_path/reject_link，以及 `openspec/specs/aiw-packages/spec.md`。

仅运行 PowerShell 文件阅读和 rg 搜索，写入本报告及同名 JSON。未运行 aiw CLI、测试、编译、构建、安装、网络或工作流事件操作，未修改实现。
