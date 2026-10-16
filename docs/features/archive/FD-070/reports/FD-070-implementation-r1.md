# FD-070 实现报告

<!-- aiw-data: FD-070-implementation-r1.json -->

- Worker：fd070-worker-20261017-4c82；来源事件：FD-070-000003-design-ready。
- 实现修订：bd1aa3b；基线：8f4d6bb。后续提交仅包含本报告和进度登记。

四个辅助命令共享 Session、提供商路由、资料与 Skill 快照、结构化输出和本地历史。research 根据明确提供的本地资料研究；prototype 保存一次性原型源码但不运行；edit-article 保存新稿但不覆盖原文；teach 持续教与练，以独立 Session 保存多个课程并支持暂停、选择和继续。辅助运行不创建 Issue/FD、不操作 Git。

主命令、帮助和补全均已接入，list/help 不请求模型。普通输入先澄清，只有 `/run` 使用已就绪摘要生成材料。新输入或新资料使旧摘要失效，无自动模型重试。Skill 入口与模式适配引用被冻结，系统阶段协议位于其后；共享改动仅增加可选 FrozenTurn.SystemPrompt、OpenAI Instructions 映射及 Codex aux 阶段 model 参数。

模型调用前登记 pending；保存原始结果后按轮次提交成果、历史和元数据。恢复发现不一致时阻止新调用；recover 仅从保存的结果补交本地材料，相同内容可复用，不同内容保留现场。失败历史写入后元数据保存失败，也复用原失败记录。来源路径、链接、文件类型、大小、成果名称和数量均有约束。课程保留全部本地历史，模型上下文限最近六轮和显式 recall，可通过 window 缩小，无隐式摘要调用。

## 验证证据

- 执行两次 `python scripts/compile.py`，均退出 0：一次是初始实现，另一次是 bd1aa3b 的恢复加固。环境为 GOPROXY=off、GOSUMDB=off、GOTOOLCHAIN=local、GOFLAGS=-mod=readonly。脚本编译 aiw 和 aiw-req 到系统空设备，不保留发行二进制。
- 一批只读静态检查确认四命令分发/帮助/补全可达，十二组 Skill/README/适配引用源文件与项目副本一致，相关链接存在，没有依赖或测试文件变化。
- 静态追踪已覆盖冻结提示、来源选择、取消传递、pending/LastTurn 校验、部分成果恢复、失败历史复用、只读模式、就绪失效与课程上下文预算；实现通过仍须独立 Reviewer 决定。
- Git 操作仅限本 FD 计划与实现提交；读取了 FD、设计反馈、相关实现和 stable specs。未运行测试、真实模型对话、联网、安装、原型执行、格式化、lint、vet 或最终发行构建。

## 限制与风险

CLI 提供商的只读选项不构成全面文件/网络隔离；敏感文件名过滤不等于秘密检测。research 首版不自动联网；teach 不凭空证明已掌握知识。未知遗留锁和 running 状态拒绝自动接管。Ctrl+C、Windows 链接和多文件提交恢复目前只有静态证据，没有运行验证。当前 PATH 中的 aiw 尚未重建安装。

独立设计反馈两轮已解决 D1/D2；下一步为独立实现评审，随后按正常流程本地合并与归档。
