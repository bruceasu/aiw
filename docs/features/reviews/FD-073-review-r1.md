# FD-073 独立实现评审 r1

<!-- aiw-data: FD-073-review-r1.json -->

结论：**changes-requested**。Reviewer：`fd073-reviewer-20261017-r1`；来源事件：`FD-073-000004-implementation-ready`；基线 `91aaec3`，审查 HEAD `6b34ce3`，实现提交 `c569141`。

## R1：旧块迁移未检查与其他来源块的重叠，会删除其他 FD 内容

严重性：高；属于正确性和验收违例，非风格建议。

位置：`src/skills/publish-github-issue/scripts/projection.py:115`、`:158`–`:167`，以及项目安装副本。

新来源路径先由 `_validate_source_blocks` 检查 `aiw:source` 标记，再单独取得旧 OpenSpec span。两种标记的边界没有共同验证。假设 existing 按以下顺序包含完整标记：旧 OpenSpec start、FD-B start、FD-B 内容、FD-B end、旧 OpenSpec end。两个检查各自通过；选择不存在的 FD-A 并显式设置 `migrate_legacy=True` 后，`span = legacy`，最终切片替换删除整个 FD-B 块。旧块与来源块交叉时同样可能破坏其他来源。该推导来自实际控制流，未执行 helper。

这违反 FD Acceptance 的“保留其他来源”及稳定规格 `github-issue-projection/spec.md` 的非嵌套与来源隔离要求；迁移授权仅允许转换指定旧块，不授权删除其他 FD 的块。

最小修正：新来源替换路径同时检查旧 OpenSpec 与新来源标记区间，拒绝任何交叉、包含或被包含的受管块，以及残缺/重复/逆序旧标记。保持互不重叠块的正常更新/显式迁移，以及旧两参数接口的历史空白行为。同步源和项目副本，静态追踪上述嵌套和交叉场景后重新交接；不要求新增或运行测试。

## 已核对的其余范围

- Issue/REQ、FD 和可选 change 的入口、Issue 所有者与投影来源分离、多 FD 聚合、明确独立 FD 策略均已落入 Skill/发布引用和规格。
- 稳定映射路径按 canonical ID 构造并检查链接；基本字段拒绝布尔/非正 Issue 编号及不匹配 URL；保存先验证旧目标，再临时文件加原子替换。旧 version-1 字段与函数名称仍保留。
- 新来源常规替换通过原始切片保留块外内容；生成块验证、来源标记重复/残缺/逆序和 64 KiB 限额已有实现，但不弥补 R1。
- pending 的独占/原子登记、响应先落凭据再写映射、未知 create 不重放是执行者合同。helper 明确不实现后台发布或锁服务；本轮不实际联网发布。此边界与设计反馈 r2 一致。
- 阅读了 Worker 双格式报告、设计 r1/r2、真实源码差异、README、UI metadata、目录和稳定规格变更。Worker 的内存 compile 结果作为已有证据引用，没有将静态检查解释为运行结果。

## 实际操作与限制

运行了定向 `Get-Content`、`Select-String`、`rg`、`git diff --stat`、`git diff`、`git status --short` 进行本地静态读取；首批 PowerShell 未指定 UTF-8 导致中文显示乱码，随后关键正文使用 UTF-8 读取，未修改原文编码。通过 `aiw fd show` 检查待事件，并以独立 Reviewer session claim。没有导入或执行 helper，没有运行测试、编译、模型、网络、安装、格式化、最终构建或验证脚本。

剩余风险：真实 GitHub、文件系统失败与并发行为尚未运行验证；完整远程 body 更新本身仍无 CAS 保证。以上不是测试通过声明。R1 修复前不支持本地交付或 Complete 归档。
