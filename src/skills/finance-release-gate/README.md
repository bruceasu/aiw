# finance-release-gate：评估真实发布准备

## 用途与适用场景

实现已完成、确实计划发布，并具备运维证据时使用。适用于各领域；`finance-` 前缀
为兼容保留，不是金融场景专属触发条件。

## 输入与输出

- 输入：已实现变更、发布范围，以及适用的迁移、数据、权限、审计、回滚、监控和责任人证据。
- 输出：Release Review、开放风险、下一步，以及 `GO`／`GO_WITH_RISK`／`NO_GO`。

## 调用示例

```text
$finance-release-gate Assess release readiness from this evidence.
```

## 边界与资料

只读评估，不编造缺失程序或责任人。不是 Issue 讨论方法，也不是每个 FD 本地
合并／归档的门槛；GO 不授权部署。不修改迁移、回滚或监控文件来补证据。
见 [SKILL.md](SKILL.md)。
