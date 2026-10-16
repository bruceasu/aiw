# publish-github-issue：发布 OpenSpec 的远程投影

## 用途与适用场景

用户明确要求把一个 OpenSpec change 发布或更新到 GitHub Issue 时使用。
它处理本地变更文档到远程 Issue 的单向投影，不是普通需求创建入口。

## 输入与输出

- 输入：唯一且明确指定的 change、相关 FD、目标仓库、现有映射与发布范围。
- 输出：远程 Issue URL 和本地映射记录，保留原始变更的权威归属。

## 调用示例

```text
$publish-github-issue Publish the selected OpenSpec change to owner/repo.
```

## 边界与资料

必须明确请求外部发布；目标不唯一时不能猜。发布失败不改变 FD 状态、归档或
清理资源；不自动关闭 Issue 或把远程投影变成工程计划。凭据、联网和远程写入
仍遵守现有授权。具体映射和投影规则见 [SKILL.md](SKILL.md)。
