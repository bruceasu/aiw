## Why

监督任务能够创建工作树并通过 Git 预检，但沙箱 Agent 的普通 Git 查询仍可能触发 dubious ownership。已有命令级目录信任指令仅在工作项标题包含 `no unrelated changes` 时生成，普通实现项没有收到，因此在编辑前核验阶段阻塞。

已观察到 requirement-artifact-generation 的工作项 1.1 因此进入 BLOCKED：工作树属于 svictor，而执行查询的是 CodexSandboxOffline。该任务未开始代码修改，重启监督循环也不能自行消除指令缺口。

## What Changes

- 为每个通过当前 Supervisor 工作区预检的受管理工作项提供限定目录的只读 Git 查询指令，不依赖工作项标题或语言。
- 分离通用 Git 查询授权与审查项专用编辑限制，避免普通实现项被错误限制为只能修改检查框。
- 复用现有命令级 safe.directory、规范路径和 shell 引号处理，保留现有环境传递；不修改全局、系统或仓库 Git 配置。
- 缺少当前预检信任依据时明确阻止派发，不能回退为任意路径信任。
- 增加请求组装回归覆盖，并记录已阻塞任务的人工恢复顺序。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `workflow-supervision`：监督请求必须携带与当前预检绑定的通用 Git 查询指令，同时保留工作项自身编辑范围。

## Impact

影响监督 Task Agent 请求组装、提示内容和相邻测试及恢复文档。不改变任务身份、完成语义、Gate/outcome 分类、重试预算或 CLI 参数。不自动恢复、重跑或修改作为故障证据的 requirement-artifact-generation。

## User Stories

1. 作为使用者，我希望普通实现项在沙箱身份不同的情况下仍能完成已授权工作树的 Git 查询。
2. 作为使用者，我希望目录信任仅限本次已验证工作树，不扩大到其他仓库。
3. 作为使用者，我希望通用查询授权不改变普通实现项或审查项的编辑权限。
4. 作为使用者，我希望缺少预检依据时看到明确错误，而不是 Agent 运行后再次遇到相同权限阻塞。
5. 作为使用者，我希望修复后能按明确步骤恢复旧的阻塞项，而不是删除工作树或清空历史。

## Out of Scope

全局 safe.directory 配置、管理员提权、更改目录所有者、关闭沙箱、重做 Git/Workflow 权限模型、修改归档功能、自动解除既有 Gate，以及扩大 Agent 的 Git 写权限。
