## Context

现有 `PreflightSupervisedGitWorkspace` 校验规范工作树、Git 注册信息和实际根目录，返回仅信任该目录的 GIT_CONFIG 环境。Session 启动会向 provider 传递该环境，但它不能保证 Agent 的嵌套沙箱工具继承配置。

`internal/commands/task/agent.go` 的 `supervisedWorkItemInstruction` 仅在标题包含 `no unrelated changes` 时生成 `git -c safe.directory=... -C ...`。同一段提示还限制只能修改检查框；不能简单把整个提示无条件复用到普通实现项。

证据来自已保存的 `.ai/sessions/requirement-artifact-generation/outputs/0001-events.jsonl` 和 0001-final.txt：Git 明确报告工作树所有者和沙箱执行用户不同。当前判断依据为静态调用链和已有日志，未重新运行故障命令。

## Design Readiness

FD_NOT_REQUIRED。修复边界已明确：沿用现有预检和命令级信任机制，将通用查询指令与既有审查限制拆开。不引入新的权限来源、存储模型或 CLI。用户已确认沿用现有请求组装与 Git 指令测试边界。

## Goals / Non-Goals

**Goals:** 每次受管理监督派发都提供当前工作树的 Git 查询方式；不依赖环境继承；保持实现与审查范围不同；保留可审计的失败信息。

**Non-Goals:** 提升系统权限、扩大受信任目录、修改 Git 配置、自动重试历史阻塞项、重构基于标题识别审查项的现有规则。

## Decisions

### 1. 通用指令只从当前预检产生

在 supervised 请求组装边界统一附加只读 Git 查询说明，覆盖普通实现和 scope-review 工作项，以及共享该组装路径的 turn/chat。不能依赖标题中的英文字符串。仅消费当前 Task 预检产生的目录，不从旧 handoff、用户标题或任意继承环境猜测信任路径。

复用 `supervisedGitCommandPrefix` 的正斜杠与 shell 引号处理。保持 `git -c <safe.directory=canonical-path> -C <canonical-path>` 两处目录一致，授权 status、branch 查询、diff 和其他原本允许的只读检查。明确这是单次命令选项，不授权 Git 配置、提交、分支或索引写操作，也不绕过文件系统限制。

保留父进程现有 scoped 环境传递；命令级指令用于补足嵌套工具不继承环境的场景，不能宣称预检成功等同于所有子环境必然可访问。

### 2. 单独保留工作项编辑约束

普通实现项接收通用 Git 指令和原有实现范围，不接收“只能修改所选 checkbox”的审查限制。既有 scope-review 项接收同样的通用指令，并单独附加原有审查限制。不得因为读取权限修复扩大任何一类工作项的编辑或 Git 写权限。

### 3. 缺少依据时不静默派发

当前预检失败时保留现有 Core 阻塞处理；请求组装发现当前预检信任目录缺失或无法对应当前请求工作区时，返回明确错误并停止 provider 派发，沿用既有准备失败的清理路径。不得仅返回空提示继续执行，也不能引入 `safe.directory=*` 或全局配置兜底。

当前已验证指令可以明确替代旧 handoff 中针对此次 Git 查询失败的过时阻塞说明，但不覆盖其他权限限制。若限定目录查询仍失败，Agent 保留原始错误并停止，不自动扩大目录或循环尝试。既有 Agent workspace-access 降为 unknown 的分类保持不变。

### 4. 既有任务恢复与兼容

修复交付后新派发自动获得新指令，不批量改写历史 Session、handoff 或既有 Gate。恢复旧阻塞任务前，需要确认执行环境实际使用修复版本及限定目录查询有效；再执行 gate resolved、reopen 对应 Work Item，最后显式 supervise start。不能将门禁解析为自动授权或清除原始失败证据。

本 change 仅记录恢复流程，不执行 requirement-artifact-generation 的恢复、代码实施或 Git 交付。

### 5. 测试边界

优先使用 `internal/commands/task` 的现有 Task Agent 请求组装测试和本地 provider 替身，断言最终发给 provider 的请求内容，而不只断言格式化 helper。

覆盖普通中文标题实现项获得指令且没有 checkbox-only 限制；审查项获得两类说明；缺失/失败预检不会派发；目录切换不复用旧 handoff 路径；Windows 与 POSIX 路径的空格、单引号及美元符号仍保持字面值。保留 `internal/taskx/supervised_git_test.go` 的限定环境检查。所有测试离线，不使用真实模型、不更改系统 Git 配置；运行需另行授权。

## Risks / Trade-offs

- 文本指令不能保证 Agent 一定遵循：测试覆盖真实请求组装；沙箱下的实际 Git 查询仍需独立运行证据。
- 本修复解决已发现的信任指令遗漏，不保证解决 ACL 或其他访问问题；仍失败时保留诊断。
- 标题识别审查项的现状保留，避免本修复扩大为工作项类型模型变更。

## Migration Plan

无数据迁移。交付修复后对新请求生效；旧阻塞项通过显式 Gate/reopen/start 流程恢复，保留 Task、Session、Attempt 与失败证据。

## Open Questions

%% VERIFICATION: 本轮未运行沙箱复现或回归测试，实际恢复效果待修复后的聚焦验证。
%% VERIFICATION: 本会话已知 OpenSpec PowerShell 入口受执行策略限制；沿用仓库共享 spec-driven 模板进行静态结构核对，不绕过执行策略，不运行 validator。
