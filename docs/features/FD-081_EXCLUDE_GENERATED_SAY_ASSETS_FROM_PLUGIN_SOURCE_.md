# FD-081: Exclude generated Say assets from plugin source packaging

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户执行 python build.py build all --version 1.0.0，在 Say 打包阶段遇到
Duplicate source: common/plugins/aiw-say/aiw.toml.example；随后 install all 因缺少
all 发布清单失败。用户日志提供运行失败证据，本会话不重跑完整构建。
主工作区存在旧式安装/暂存留下的 ignored 示例配置及 profiles；干净 worktree 中无这些资源，
FD-080 静态审查未覆盖这一源码目录已有生成资源的情形。

## Options and decision

不放宽 write_package 的重复检查，也不删除用户目录中的旧资源。
选择为 tree_entries 增加明确的相对路径排除参数：仅 Say 插件扫描排除
aiw.toml.example、profiles 和两个热键脚本；这些资源统一从 canonical programs/aiw-say 收集。
不采用按文件内容去重或后写覆盖，避免隐藏不同内容的路径冲突。

## Solution

tree_entries 的排除同时作用于目录遍历和文件收集；未传排除参数的调用保持现有行为。
Say manifest/README/入口继续从插件目录读取；双平台 Go 程序继续从 bin 获取；
示例、profiles、热键继续从 programs/aiw-say 获取。构建成功前不生成 all 清单的原有规则不变。
无依赖、公开 CLI、状态格式或权限变更，属于内部收集规则修复，不需要额外独立设计审查。

## Scope

仅修改 build.py 的遍历排除及 Say 调用，补充已生成资源存在时的使用说明。
不改变包格式和安装语义，不删除或修改用户的 bin/dist/src 下现有生成资源。
不创建测试，不执行发布构建或安装。原仓库功能候选仍未生成新 FD。

## Work items

- [ ] 1.1 为 tree_entries 增加精确相对路径排除并接入 Say；小/低；无依赖；目录和文件均排除，canonical 资源只收集一次。
- [ ] 1.2 更新说明与 Worker 证据；小/低；依赖 1.1；Python 内存 compile 和一次静态 diff 检查，独立审查后本地交付归档。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

源码插件目录无生成资源或存在旧示例/profiles/热键时，Say 包的资源来源一致且不重复。
profiles 子目录在扫描前被排除，不能仅过滤扫描后的文件列表。
默认遍历及其他插件不受影响；archive.write_package 仍拒绝真正重复来源。
all 清单仍只在完整组包成功后写出；不把当前缺清单误诊为独立安装缺陷。

## Verification

静态证据：用户两条失败命令日志；源码 Say 目录确有 ignored aiw.toml.example 和 profiles；
package_component 先全量 tree_entries 后再次显式收集 canonical 资源，确认因果链。
实现后仅编译 build.py（compile()，不导入或执行）及一次 git diff --check。
未授权测试、完整构建、安装、网络或依赖下载。运行复现和构建成功仍待用户验证。

## TODO

提交就绪计划后在 feature/FD-081、.wt/FD-081 实现；独立审查、交付与归档按默认生命周期推进。

- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

## Sources

- Issue: none

- build.py；src/plugins/aiw-pkg/archive.py。
- openspec/specs/aiw-packages/spec.md；归档 FD-080。
- 用户本轮 build all/install all 失败日志。
