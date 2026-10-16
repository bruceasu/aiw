# FD-061 实施报告

<!-- aiw-data: FD-061-implementation-r1.json -->

- Worker：fd061-worker-20261017-a9d2
- 来源事件：FD-061-000003-design-ready
- 实现提交：9a2e0bd、36aa594；计划/基线 2d62ef9，父分支 develop。

保留 issue-management 名称和 Issue 至 FD 的阶段边界。将五个需求讨论 Skill 正文迁为其 references/intake.md、value-assessment.md、metric-brief.md、engineering-options.md、synthesis.md，删除对应源目录和项目副本的 SKILL.md 与 UI 元数据。references 无独立 Skill frontmatter，主入口使用按需相对链接。分析约束、输入输出模板、金融条件分支保持；交叉引用改为同目录文件，措辞改为内部方法。

CLI 按固定 gap 映射加载新 reference 正文，Methods 的 Path 指向项目内新路径。历史 finance-* 选择标识仅留在选择元数据和日志中，兼容旧 Session 和 JSON，不再表示独立安装依赖。domain-modeling、grill-with-docs 和发布评估保持独立；必需方法错误阻塞、可选追问失败诊断、摘要与引用、预算、固定路径、审批与模型调用数不变。

安装器的 materialize_skill_source 和安装流程使用整目录 copytree，references 可随 issue-management 安装；未修改或运行安装器。同步了项目副本，未触碰用户级安装和已归档 FD。

## 静态证据与实际命令

- Get-Content、rg、git status/branch 定向读取相关技能、方法路由、安装器复制符号、规格和工作流规则。两处初始路径定位未命中，按实际 src/plugins/aiw-skills/aiw-skills.py 路径继续，未运行未知脚本。
- 一次静态命令：`git diff --find-renames develop -- src/skills src/internal/issue openspec/specs/requirement/spec.md`，结合六个源/项目副本 Get-FileHash 比较及 git status。五个 references 与主入口比较均为 True；差异确认正文迁移、旧 UI 移除、相对链接和新固定路径。
- 静态检查后仅统一 reference 中残留的 this Skill / sibling Skills 措辞，源/项目副本同步修改，无 Go 变化；由独立 Reviewer 检查最终版本。
- 在 src 执行 `$env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTOOLCHAIN='local'; go build ./internal/issue`，退出码 0，仅编译库包，无最终分发产物。
- 工作流执行 aiw fd new/claim/emit、aiw git wt add 与定向 git add/commit。专用分支 feature/FD-061、worktree .wt/FD-061、登记父分支 develop 均已核对。

## 未执行和风险

未创建或运行测试、最终构建、格式化、lint、vet、网络或下载。没有运行安装/同步命令，安装兼容判断基于现有复制逻辑的静态证据。

已安装旧 CLI 仍读旧路径，需要与新 references 一同更新；用户级安装和当前会话旧目录不自动清理。历史方法标识刻意保留，诊断里仍可能显示 finance-*。真实模型讨论行为与实际安装未运行验证。改动尚未合并或归档，等待独立 Reviewer。
