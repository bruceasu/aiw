# FD-003: 精简 aiw-cz 发布包

**Status:** Complete
**Revision:** 3
**Priority:** Medium

## Problem

`aiw-cz` 使用 TypeScript 开发，但 `aiw cz` 运行编译后的 JavaScript，依赖 Node.js 22.12.0+。当前 `build.bat cz` 在插件目录执行 `npm install` 和 `npm run build`，`build.bat plugins` 随后复制插件目录；该目录可能含 `src`、`dist` 和完整的 `node_modules`。开发依赖和源码不应因为安装目录复制而进入发布包。需要验证能否在不改写功能的前提下交付更小、更明确的运行产物。

## Options and decision

1. **保留 TypeScript，打包运行代码及必要资源（选定）。** 在构建阶段把编译后的入口和 JavaScript 依赖打包，单独列出无法内联的 SDK 原生二进制或其他运行资源；发布清单只包含入口、打包产物与实际必需的资源。目标环境仍提供 Node.js 22.12.0+。这样保留现有实现和配置语义，同时验证能否不复制 `src` 与 `node_modules` 目录。
2. 改写为 Python：仍需 Python 解释器及对应 SDK/依赖，且要迁移交互、配置和三个 LLM provider；当前没有证据表明总交付成本更低。
3. 原样复制编译结果和完整依赖目录：改动最少，但不能解决体积和交付边界不清的问题。

先用选定方案制作隔离的发布样本并测量。若 Copilot SDK、Codex SDK 或 OpenAI SDK 需要的动态加载、原生组件或平台资源无法可靠打包，记录具体阻塞与测量数据；不要悄悄删除 provider、改变回退顺序或宣布方案已通过。

## Solution

保持 `aiw-cz.js` 作为插件入口、`plugins/aiw-cz/src` 作为 TypeScript 源码，以及现有 `aiw cz` 命令和配置格式。选择一个构建时 JavaScript 打包工具，产出可由入口加载的运行文件；使用显式发布清单构建隔离的 `aiw-cz` 目录，而不是镜像复制开发目录。把无法内联的原生二进制、辅助可执行文件和许可证等资源逐项列入清单，并按目标平台生成。发布包不含 `src`、`node_modules`、TypeScript 编译器、锁文件或构建缓存；Node.js 作为外部运行前置条件保留。

构建时可以使用现有锁文件和本地依赖。此 FD 不批准下载依赖、运行构建、执行交互验证或发布；实施这些动作须遵守仓库的运行时和网络授权规则。发布流程改动还须先核对 Windows 与 Linux 的目标路径和原有插件发现逻辑。不要创建 OpenSpec change；只有验证结果确实改变稳定行为时才更新相关稳定规格。

## Scope

涉及 `plugins/aiw-cz` 的构建配置与入口、`build.bat` 的 CZ 安装路径、必要的发布清单，以及 `docs/usage/cz-configuration.md` 和相关说明。现有手动向导、Copilot → Codex → OpenAI 回退、配置优先级、交互审阅和 Git 提交行为保持不变。

不迁移到 Python，不把 Node.js 打进产物，不改变 AIW Go 命令派发和其他插件，不发布或推送软件。

## Work items

- [x] 1.1 记录当前 `build.bat cz`、`build.bat plugins` 的文件流和开发目录基线大小；列出三种 SDK 的动态加载与平台资源。证据：路径/依赖清单和可复核的体积记录。
- [x] 1.2 在隔离目录验证 TypeScript 打包方案，写出明确的运行资源清单和可重现构建命令。证据：产物目录列表、依赖解析检查和与基线的体积对比；运行验证仅在获得授权后执行。
- [x] 1.3 若 1.2 满足验收条件，调整 CZ 发布/安装路径以只交付清单文件，并保留原插件入口与 Node.js 版本要求。证据：静态追踪构建、安装、发现和执行路径；必要的编译检查。
- [x] 1.4 更新 CZ 安装文档和 Verification，记录各平台、手动向导及三个 provider 的实际验证范围、未运行项目与剩余风险。由独立 Reviewer 检查实际差异和证据后再考虑关闭 FD。

保持 Work Item 编号稳定。若打包验证不通过，记录阻塞，不把未完成项标记为完成。

## Acceptance

1. 发布样本不包含 `src` 或 `node_modules` 目录；清单说明每个额外运行文件的用途。开发目录的 TypeScript 源码与锁文件仍可用于构建。
2. 在已安装 Node.js 22.12.0+ 的目标环境中，插件入口可找到打包产物；缺少 Node.js 或必需资源时给出明确错误。Node.js 前置条件在文档中可见。
3. 手动向导和 `--llm` 的三个 provider、现有回退顺序与配置优先级不因打包改变。若某个平台或 provider 无法验证，明确标为未验证，不宣称兼容。
4. Windows 与 Linux 发布清单均可说明原生/辅助资源的来源和去向；发布目录大小与当前完整插件目录基线并列记录，以证实是否真正缩小。
5. 不复制 TypeScript 源码和开发依赖到发布目录，不从目标环境联网安装依赖；没有改变 Go 插件派发契约。

## Verification

- 设计阶段：仅静态阅读 `plugins/aiw-cz/package.json`、`src/index.ts`、`src/llm.ts`、`src/config.ts`、`aiw-cz.js`、`build.bat`、`internal/plugin/exec.go` 和相关稳定规格。未运行构建、测试、发布或体积测量。
- 1.1 基线（本机 Windows x64）：`Get-ChildItem -LiteralPath plugins/aiw-cz -Recurse -File -Force | Measure-Object Length -Sum` 得 623,895,219 bytes；其中 `node_modules` 为 623,816,483 bytes，`dist` 为 23,423 bytes。原路径：`build.bat cz` 执行 `npm install` 与 `npm run build`；`build.bat plugins` 再通过 `cp-mul.bat` 镜像复制整个 `plugins` 目录。`src/llm.ts` 动态导入 Copilot、Codex、OpenAI SDK；Copilot 平台包含 `copilot-runtime`、`runtime.node` 等资源，Codex 平台包含 `codex` 二进制和资源，OpenAI SDK 为 JavaScript 客户端。现有构建器只有 TypeScript，未安装 esbuild、Rollup 或 ncc。
- Worker 已写入打包器入口、显式资源清单、安装目录过滤和运行时二进制路径适配。`npm install --save-dev esbuild` 成功，锁定 esbuild 0.28.2。首轮 `npm run build:release` 因 Koffi 不导出 `package.json` 子路径而失败；修正定位后重跑成功，得到 Windows x64 初始样本 570,362,000 bytes，对应安装依赖后的开发目录基线 635,764,195 bytes。初始样本仍含 Koffi 的 `src`，因此已把 Koffi 交付改为最小运行文件；**修订后样本尚未重建，不能将初始体积视为最终结果**。
- 编译检查 `npm exec tsc -- --noEmit -p tsconfig.json` 首次发现 Node 类型声明中的 `process.report` 类型过宽；修正类型收窄后同一命令重跑通过。Windows 与 Linux 的资源路径均在脚本中声明；Linux、手动向导及三个 provider 均未运行验证。`build.bat plugins` 未执行，未安装到目标目录。
- 最终 Windows x64 样本：额外一次获授权的 `npm run build:release` 成功；安装 esbuild 后的开发目录基线为 635,766,603 bytes，发布样本为 568,674,974 bytes。`release-manifest.json` 列出 132 个文件；只读核对确认清单文件均存在，且样本内没有 `src` 或 `node_modules` 目录。`node release/aiw-cz.js --help` 在隔离样本中成功，显示三个 provider 顺序及 Node.js 版本要求。`git diff --check` 通过。
- 构建/安装/发现路径已静态追踪；`build.bat plugins`、目标目录安装、Linux 打包、手动向导和三个 provider 实际调用未运行。无法从 `--help` 推断它们的运行结果。详细命令、体积与风险见 `docs/features/archive/FD-003/reports/FD-003-implementation.md`。
- Worker：先完成基线与打包可行性验证，再决定是否进入发布路径改动；记录实际执行命令、平台和结果。遵守仓库对构建、运行、网络和验证范围的授权限制。
- Reviewer：独立核对发布清单、打包资源和验收证据；缺少真实运行证据时明确保留风险。
- 独立 Reviewer 已审阅当前 FD-003 差异和 Windows x64 样本证据，结论为静态审阅通过；报告见 `docs/features/archive/FD-003/reviews/FD-003-review.md`。Linux 打包、目标安装、手动向导和三个 provider 的实际调用仍未验证，`--help` 不代表这些路径已通过。

## Sources

- 用户请求：比较 TypeScript、Python 与无需复制源码/`node_modules` 的发布方式，并选择一个方案处理。
- `plugins/aiw-cz/package.json`、`tsconfig.json`、`aiw-cz.js` 和 `src/`：当前 TypeScript、Node.js 与三种 SDK 依赖。
- `build.bat`：当前 CZ 构建和插件目录安装路径。
- `internal/plugin/exec.go`、`internal/plugin/runtime_config.go`：JavaScript 插件使用 Node.js 启动。
- `openspec/specs/cz-configuration-priority/spec.md`：CZ 插件、provider 顺序与 Node.js 运行约束。
- `openspec/specs/cli-and-plugins/spec.md`：插件发现与执行边界。

**Completed:** 2026-09-29
