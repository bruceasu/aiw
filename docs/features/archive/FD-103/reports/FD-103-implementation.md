# FD-103 实现报告

<!-- aiw-data: FD-103-implementation.json -->

实现版本：3e0d89c。基线：08f8fe5。Worker：fd103-host-20261021-b873。来源事件：FD-103-000003-design-ready。

## 修改及原因

活动 FD 使用独立目录，保留设计文件名，新增局部 index.md，并将本轮设计审阅移入本 FD 的 reviews。102 个既有归档目录新增局部导航，历史设计与证据正文均未移动或改写；重复历史编号保留多个设计链接。

全局 FEATURE_INDEX.md 改为固定目录导航。update_index 只更新所属 FD 的活动/归档导航，且内容不包含状态；不同 FD 不再共同改写状态表。未安装 merge driver，Git 的真实源码冲突与原恢复门禁保持有效。

创建、发现、重新打开、重新审阅、归档、Git worktree 发现及四种 shell 补全支持目录结构。旧平铺设计和证据仍可读取，缺失旧 artifact 路径按精确文件名唯一映射；歧义明确失败。归档在任何移动前检查新旧证据同名碰撞；归档、重新打开与归档审阅失败时还原两处局部索引。

当前技能源、模板、使用文档与稳定规格已更新。已安装的技能或插件未发布或安装；需要匹配版本才能使用新结构。

## 静态证据

检查实际 diff、活动/归档发现、artifact 和旧 receipt 路径、证据清单、归档碰撞与回滚、索引调用点和补全路径。git diff --check 无空白错误。修正归档遗留空目录造成活动导航残留的问题。独立设计审阅为 ready。

## 实际命令与结果

- Python compile() 编译 src/plugins/aiw-fd.py、src/plugins/aiw-git/git-wt.py：退出 0，不写 pyc。
- GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local 环境下 go -C src build ./internal/commands/completion：退出 0，仅编译包，不留可分发产物。
- 一次静态批次：git diff 08f8fe5、定向 rg 调用点/文档检查、git diff --check 08f8fe5：退出 0。
- 工作流命令已完成 new、claim、design-ready、wt add，以及本 FD 范围提交。无测试、网络、下载、最终构建、格式化或 lint。

## 兼容边界与剩余风险

全局索引不再提供状态表格，解析表格的消费者需改用 FD 文件或 aiw fd list。物理路径变化不提供浏览器重定向；CLI artifact 映射仅在旧路径不存在且目标唯一时生效。历史归档原路径保留。

并发运行、归档失败注入和 shell 交互未执行。并发独立状态变更消除共享写入的结论来自静态分析；相同源码修改仍可能产生真实冲突。交付继续要求干净父工作区并逐个进行。本报告不是运行测试通过记录。
