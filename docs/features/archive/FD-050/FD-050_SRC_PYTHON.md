# FD-050: 将仓库源码统一迁入 `src/` 并用 Python 重写构建入口

**Status:** Complete
**Revision:** 10
**Priority:** High  
**Evidence policy:** Dual

## Problem

仓库根目录当前同时承载主 Go 模块和多个可维护的项目内容。主 CLI 在 `cmd/`、`internal/`，独立程序在 `program/`，插件、技能和 Agent 模板分别在根目录的 `plugins/`、`skills/`、`agent-templates/`。构建和安装逻辑集中在 Windows 批处理脚本 `build.bat`，其中混合了 Go 编译、插件打包、文档复制和安装。

维护者希望将所有仓库维护的源码与内容源统一放进 `src/`，并将构建入口改为 Python，同时保留现有模块导入路径、插件发现行为、构建动作和安装布局。

## Options and decision

1. 只迁移主 Go 模块，保留 `program/`、`plugins/`、`skills/` 和 `agent-templates/`：影响较小，但仍把多类源内容留在根目录，不满足已确认的组织目标。
2. 把整个仓库内容混放到 `src/`，包含文档、测试和仓库管理规则：根目录更少，但混淆源码、验证、规范和发布产物的职责。
3. 将主 Go 模块、独立程序、插件源码、技能和模板作为源内容迁入 `src/`；保留根级文档、测试、仓库工具和管理配置，并将构建入口改为 Python。**选择此方案。**它满足用户确认的目标，同时保留各独立项目的目录边界和根级仓库职责。

## Solution

- 主 Go 模块整体迁入 `src/`：`go.mod`、`go.sum`、`cmd/` 和 `internal/`。模块路径仍为 `aiw`，避免 Go import 路径变化；从 `src/` 作为工作目录执行主模块构建。旧 `.go-build.jsonc` 已由父分支提交 `2287e4b` 删除，不重新创建；新 Python 构建入口直接配置 Go 调用。
- 将 `program/` 改为 `src/programs/`，保留各程序内部的源码、依赖清单、配置和项目文档。独立 Go/Node 模块继续使用自己的模块清单。
- 将根级 `plugins/`、`skills/`、`agent-templates/` 分别迁入 `src/plugins/`、`src/skills/`、`src/agent-templates/`。隐藏的 `.agents/`、`.codex/` 配置不迁移。
- 增加根级 `build.py`，直接使用 Python 标准库和 Go 工具链协调构建，不依赖 `gbuild` 或批处理。完成调用方迁移后移除 `build.bat`。保留现有动作名称、参数顺序、组合动作、无参数默认行为、版本和离线环境变量、输出平台、安装目标及配置保留/排除规则。
- 主二进制继续输出到根级 `bin/`；插件构建和打包中间产物放到 `dist/`。安装后的插件仍在安装目录的 `plugins/` 下。
- 插件发现继续优先使用可执行文件旁的已安装 `plugins/`，并支持从 Git 仓库根下的 `src/plugins/` 加载开发源码。`AIW_ROOT`、`AIW_WORKSPACE` 等进程环境语义不变。
- 更新主模块构建脚本、技能/插件安装路径、相关测试夹具、活动文档、仓库指引和指向旧源码路径的稳定规格链接。规格只更新失效路径，不改变能力要求。

## Scope

包含：主 Go 模块、`program/`、根级 `plugins/`、根级 `skills/`、根级 `agent-templates/` 的迁移；构建入口替换；构建、发现、安装、忽略规则、测试夹具和活动引用更新。

不包含：迁移 `docs/`、`openspec/`、`tests/`、`scripts/`、`.agents/`、`.codex/`、`.github/`；改变插件命令契约或安装布局；改变各独立项目的依赖或模块路径；提交、发布或部署安装产物。

## Work items

- [x] 1.1 将主 Go 模块迁入 `src/`，并更新主 CLI、`aiw-req`、`aiw-say` 的 Go 构建入口和编译脚本。验收：模块名和 `aiw/internal/...` 导入保持不变，所有仓库内主模块构建命令都从新模块根执行；不恢复已删除的 `.go-build.jsonc`。**规模：小；难度：中；依赖：无。**
- [x] 1.2 将 `program/` 迁为 `src/programs/`，更新独立项目构建、配置样例和文档引用。验收：Gateway、Agent、Say、AI Code Tools 和 Windows runner 的项目边界及各自模块文件保持完整。**规模：小；难度：中；依赖：无。**
- [x] 1.3 将插件源码迁入 `src/plugins/`，更新仓库内插件发现与插件相关脚本路径。验收：仓库开发模式能发现源码插件，已安装目录优先级、插件命名和执行环境变量保持不变。**规模：中；难度：中；依赖：1.1。**
- [x] 1.4 将技能与 Agent 模板迁入 `src/skills/`、`src/agent-templates/`，更新安装/复制来源和忽略规则。验收：安装包仍包含原有技能与模板内容，隐藏仓库配置不受影响。**规模：小；难度：低；依赖：无。**
- [x] 1.5 新增 `build.py` 并替代 `build.bat`。验收：既有构建动作、顺序、默认行为、错误退出、离线 Go 环境、输出路径、安装及文件保留/排除行为均有对应实现；构建产物不写入源码树。**规模：中；难度：中；依赖：1.1–1.4。**
- [x] 1.6 更新活动文档、仓库规则、测试夹具、规格源码链接和根级路径引用。验收：源码路径检索不再留下需维护的旧根级目录引用；历史归档报告保持不变。**规模：中；难度：中；依赖：1.1–1.5。**

## Acceptance

- 主 Go 模块位于 `src/`，`module aiw` 不变；根目录不再保留主模块源码目录或模块清单。
- `src/programs/`、`src/plugins/`、`src/skills/`、`src/agent-templates/` 成为对应源内容的唯一维护位置；旧的根级目录不存在。
- `build.py` 是仓库构建入口，`build.bat` 不再作为入口；现有命令语义与安装文件布局保持兼容。
- AIW 在仓库开发环境和安装环境都能按原有插件命名规则发现并执行插件，安装插件优先于源码插件。
- 所有活动构建说明、仓库指引、配置和代码链接指向新位置；历史归档材料不改写。
- 编译型检查通过；静态检查确认迁移后的路径、构建配置、插件候选顺序和安装来源一致。

## Verification

- 编译型检查：`python scripts/compile.py` 成功；输出写入 NUL，没有保留最终产物。
- 静态检查：审阅最终差异和 `build.py` 的动作、子进程环境与安装来源；`git diff --check develop...HEAD` 无空白错误，根级旧源码目录均不存在。
- 未运行测试、`build.py` 构建/安装动作、完整构建、格式化、lint 或验证脚本。测试夹具已更新；黑盒安装测试需通过 `$fd-test` 单独授权。
- 独立 Reviewer：请求修改，报告见 `docs/features/reviews/FD-050-review-20261010T030000Z.md`；handoff `FD-050-000007-implementation-ready`。阻塞项为插件发现未严格保证已安装目录优先，以及 `.github/copilot-instructions.md` 残留旧 `cmd/` 路径。
- Worker 修正：插件候选现在按 `paths` 顺序逐目录选择，并且只在首个命中目录内比较扩展名；PATH 仅作目录候选的后备。Go CLI 指引已改为 `src/cmd/`。`python scripts/compile.py` 修正后通过；未运行测试、构建或安装动作。新实现报告见 `docs/features/reports/FD-050-implementation-20261010T021625Z.md`。
- 独立 Reviewer 复审：通过，报告见 `docs/features/reviews/FD-050-review-20261010T021900Z.md`；handoff `FD-050-000009-implementation-ready`。静态确认插件候选按目录优先级查找，Copilot Go CLI 指引已指向 `src/cmd/`。测试、构建和安装未运行。

## Sources

- Issue: none; scope and layout confirmed directly by the user.
- `AGENTS.md`：FD-first、worktree、验证预算和路径迁移规则。
- `.agents/skills/work-management.md`、`.agents/skills/fd-workflow/SKILL.md`：FD 生命周期与隔离工作区要求。
- `build.py`、`scripts/compile.py`：现有构建入口及动作；父提交 `2287e4b` 已移除旧 `.go-build.jsonc`，新构建配置由 Python 入口负责。
- `src/internal/plugin/discover.go`、`src/internal/plugin/environment.go`：插件发现顺序和执行环境契约。
- `src/internal/repo/root.go`：仓库根目录由 Git common dir 解析，不依赖根级 `go.mod`。
- `src/programs/agent-gateway/go.mod`、各 `src/programs/*/package.json`：独立项目边界。
- `src/plugins/aiw-skills/tests/test_skills_cli.py`、`tests/` 中的构建夹具：旧入口和源码路径耦合。
- `openspec/specs/cli-and-plugins/spec.md`、`openspec/specs/requirement/spec.md`、`openspec/specs/ai-routing/spec.md`：存在源码路径引用；仅在路径失效时修正链接。
- Planner session: `fd050-planner-20261010-b72f31`; source event: `FD-050-000002-design-requested`.

**Completed:** 2026-10-10
