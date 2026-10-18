# FD-082: Support linked install roots and explicit legacy adoption

**Status:** Open
**Revision:** 3
**Priority:** Medium
**Evidence policy:** Dual

## Problem

用户 install all 到 C:\green\aiw 时因上层 junction 被拒绝；改用 D:\green\aiw 后
又因旧 aiw.exe 无包归属被拒绝。静态检查确认 C:\green 是 D:\green 的 junction，
Manager 在解析用户安装根之前拒绝所有祖先链接，且没有旧式安装接管路径。

## Options and decision

用户明确选择普通 install 自动备份并接管，不增加 --adopt-existing。
拒绝让用户先删除旧程序，也不全面放宽链接检查。安装根解析一次为 canonical 绝对路径，
安装根内部的链接/状态区保护保持不变；接管只涉及当前包声明、当前平台的普通文件。

## Solution

Manager 保存用户指定根的解析后真实路径，使 C:/D: 两种写法共享同一锁和状态。
safe_path 的严格链接检查不变，只改变 Manager 初始根解析。
preflight 允许当前包将覆盖的无归属普通文件；其他包归属冲突、目录/文件冲突、
链接、已有受管文件被修改的拒绝规则保留。配置继续保留内容并登记。

写入前 before 保存所有受影响的原文件及实际 current 记录；失败恢复原文件及原记录，
不能因为接管而把失败安装记录为成功。没有现有 installed 记录但有待覆盖的旧普通文件时，
成功后将它们作为唯一 previous，生成 version=legacy、legacy_snapshot=true 的恢复记录，
不假称知道旧软件版本，也不伪造原包 ZIP。内容指纹由快照的文件路径/哈希生成。
remove 恢复旧文件、撤销本次新增文件并消费备份，保留配置；后续 remove/purge 按记录处理。
若已有 installed 记录，正常上一版仍为唯一备份；新增接管文件也由 before 快照保护。
正式 legacy 备份只在新安装成功后发布，失败保留原 previous；整个旧安装目录不搬动、不接管。

## Scope

共享 Manager 和调用方文档/稳定规范；不改变包 ZIP 格式，不增加 CLI 参数或依赖。
不修改 C:\green 或 D:\green 的链接、ACL、既有文件，不实际执行安装。
不新增并发机制、服务、自更新或脚本沙箱。锁、事务和平台机制复用 FD-080。

## Work items

- [x] 1.1 独立设计审查 ready，无阻塞，见 reviews/FD-082-design-r1.md/JSON；legacy previous 与 before 真记录分开，恢复不依赖旧 ZIP，仅承诺 mode 不承诺 ACL。
- [ ] 2.1 解析显式安装根并保留内部链接保护；小/低；依赖 1.1；别名路径共享 canonical 状态。
- [ ] 2.2 实现自动接管与 legacy 单备份；小/中；依赖 2.1；普通文件备份后覆盖，失败恢复，remove 可恢复旧安装。
- [ ] 3.1 更新帮助/文档/规范和 Worker 证据；小/低；依赖 2.2；一次 Python 内存 compile 与静态 diff，独立实现审查后交付归档。

Keep item numbers stable after implementation starts. Use `- [-]` only for an
explicitly cancelled item, with its reason on the same line.

## Acceptance

C:\green\aiw 与 D:\green\aiw 解析到相同根；安装根内部 junction 仍拒绝。
新目录安装不产生虚构 legacy 备份；旧目录安装自动接管当前包目标文件，保留无关文件/配置。
成功后仅一个上一版备份；接管失败恢复旧文件，不生成成功安装记录；原备份不被提前删除。
remove 可恢复 legacy 原字节和权限，不依赖缺失的旧 ZIP；其他包归属仍拒绝覆盖。

## Verification

用户提供两条真实失败命令，Get-Item 静态检查 junction，源码调用链确认根因。
本会话只做静态检查及改后 compile，不执行用户目标安装、链接变更或测试/构建。
设计/实现分别由独立角色审查，不把静态结论表示为运行结果。

## TODO

等待独立设计审查后 design-ready、计划提交与 worktree；之后实现、独立审查和本地归档。

- Compile-only check and static review by default; record unrun tests explicitly.
- For explicitly requested test support (any wording/language), record suitable
  TDD scope, independent fd-test scope, command/revision authorization and finite
  budget here. Otherwise no testing stage is added. On exhausted unexpected-
  failure budget, stop affected test repair/execution, diagnose statically and
  continue independent work; unresolved acceptance defects still block delivery.

## Sources

- Issue: none

- 用户本轮两条失败日志，以及“普通 install 自动备份并接管”的明确选择。
- src/plugins/aiw-pkg/manager.py；archive.py；openspec/specs/aiw-packages/spec.md。
- 归档 FD-080：原来拒绝无归属文件的规则由本次确认明确替代，其他安全规则不变。
