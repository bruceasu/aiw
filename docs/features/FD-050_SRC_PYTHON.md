# FD-050: 将仓库源码统一迁入 `src/` 并用 Python 重写构建入口

**Status:** Open
**Revision:** 4
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

- 主 Go 模块整体迁入 `src/`：`go.mod`、`go.sum`、`.go-build.jsonc`、`cmd/` 和 `internal/`。模块路径仍为 `aiw`，避免 Go import 路径变化；从 `src/` 作为工作目录执行主模块构建。
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

- [ ] 1.1 将主 Go 模块迁入 `src/`，并更新主 CLI、`aiw-req`、`aiw-say` 的 Go 构建入口和编译脚本。验收：模块名和 `aiw/internal/...` 导入保持不变，所有仓库内主模块构建命令都从新模块根执行。**规模：小；难度：中；依赖：无。**
- [ ] 1.2 将 `program/` 迁为 `src/programs/`，更新独立项目构建、配置样例和文档引用。验收：Gateway、Agent、Say、AI Code Tools 和 Windows runner 的项目边界及各自模块文件保持完整。**规模：小；难度：中；依赖：无。**
- [ ] 1.3 将插件源码迁入 `src/plugins/`，更新仓库内插件发现与插件相关脚本路径。验收：仓库开发模式能发现源码插件，已安装目录优先级、插件命名和执行环境变量保持不变。**规模：中；难度：中；依赖：1.1。**
- [ ] 1.4 将技能与 Agent 模板迁入 `src/skills/`、`src/agent-templates/`，更新安装/复制来源和忽略规则。验收：安装包仍包含原有技能与模板内容，隐藏仓库配置不受影响。**规模：小；难度：低；依赖：无。**
- [ ] 1.5 新增 `build.py` 并替代 `build.bat`。验收：既有构建动作、顺序、默认行为、错误退出、离线 Go 环境、输出路径、安装及文件保留/排除行为均有对应实现；构建产物不写入源码树。**规模：中；难度：中；依赖：1.1–1.4。**
- [ ] 1.6 更新活动文档、仓库规则、测试夹具、规格源码链接和根级路径引用。验收：源码路径检索不再留下需维护的旧根级目录引用；历史归档报告保持不变。**规模：中；难度：中；依赖：1.1–1.5。**

## Acceptance

- 主 Go 模块位于 `src/`，`module aiw` 不变；根目录不再保留主模块源码目录或模块清单。
- `src/programs/`、`src/plugins/`、`src/skills/`、`src/agent-templates/` 成为对应源内容的唯一维护位置；旧的根级目录不存在。
- `build.py` 是仓库构建入口，`build.bat` 不再作为入口；现有命令语义与安装文件布局保持兼容。
- AIW 在仓库开发环境和安装环境都能按原有插件命名规则发现并执行插件，安装插件优先于源码插件。
- 所有活动构建说明、仓库指引、配置和代码链接指向新位置；历史归档材料不改写。
- 编译型检查通过；静态检查确认迁移后的路径、构建配置、插件候选顺序和安装来源一致。

## Verification

- 按仓库预算执行一次主 Go 模块 compile-only 检查，使用更新后的 `scripts/compile.py`；不生成可分发产物。
- 静态检查最终差异，追踪 `build.py` 的每个动作、子进程工作目录/环境、安装来源及失败码；检查所有迁移路径引用和 `.gitignore`。
- 不运行测试、完整构建、安装动作、格式化、lint 或验证脚本。测试夹具随迁移更新；若要运行黑盒/安装行为测试，需单独授权并使用 `$fd-test`。

## Sources

- Issue: none; scope and layout confirmed directly by the user.
- `AGENTS.md`：FD-first、worktree、验证预算和路径迁移规则。
- `.agents/skills/work-management.md`、`.agents/skills/fd-workflow/SKILL.md`：FD 生命周期与隔离工作区要求。
- `build.bat`、`.go-build.jsonc`、`scripts/compile.py`：现有构建入口及动作。
- `internal/plugin/discover.go`、`internal/plugin/environment.go`：插件发现顺序和执行环境契约。
- `internal/repo/root.go`：仓库根目录由 Git common dir 解析，不依赖根级 `go.mod`。
- `program/agent-gateway/go.mod`、各 `program/*/package.json`：独立项目边界。
- `plugins/aiw-skills/tests/test_skills_cli.py`、`tests/` 中的构建夹具：旧入口和源码路径耦合。
- `openspec/specs/cli-and-plugins/spec.md`、`openspec/specs/requirement/spec.md`、`openspec/specs/ai-routing/spec.md`：存在源码路径引用；仅在路径失效时修正链接。
- Planner session: `fd050-planner-20261010-b72f31`; source event: `FD-050-000002-design-requested`.
