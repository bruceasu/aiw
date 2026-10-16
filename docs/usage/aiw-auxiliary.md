# 辅助对话：研究、原型、文章修改与学习

`aiw research`、`aiw prototype`、`aiw edit-article`、`aiw teach` 是独立辅助命令。
它们不会创建、批准或推广 Issue／FD，不执行 Git，不把课程或文章变成工程任务。
需要把结论转成产品变更时，再另行交给 Issue／FD。

## 开始一个任务或课程

在希望保存任务的目录中运行。每个目录有独立的 `.ai/sessions/`；从另一个目录执行
不会自动找到这里的课程。命令沿用共享 AI 配置，支持本次 `--provider`／`--model`。
模型调用会按所选后端使用服务或 CLI；`list`、`--help` 和本地恢复不调用模型。

```powershell
aiw research "Explain the module boundaries" --source src/internal
aiw edit-article "Improve clarity for new readers" --source docs/draft.md
aiw prototype "Explore this state model" --kind logic
aiw prototype "Compare UI layouts" --kind ui
aiw teach new "Learn event-driven architecture"
aiw teach new "Practice technical writing"
```

没有参数时先显示同类型的已保存任务／课程，可选编号继续或输入 0 新建。逻辑原型是
prototype 默认分支，`--kind ui` 选择 UI；分支在创建时固定。以上是新源码命令，旧发行
版未必已更新；发布时需同步主程序及对应 Skill 资源，不是只安装一个 README。

每行输入发送一次；长文章或代码使用 `--source` 或对话中的 `/source`。

## 对话与产物

| 输入 | 行为 |
| --- | --- |
| 普通文字 | 研究／原型／文章任务继续澄清；teach 直接讲解、练习与反馈。 |
| `/brief` | 查看当前摘要或学习进展，不调用模型。 |
| `/run` | 从已展示且就绪的摘要生成成果；未就绪不执行。 |
| `/source PATH` | 明确增加当前目录内的文本资料，原文件不修改；旧就绪状态失效。 |
| `/window N` | 选择近期上下文轮数，0–6；完整历史不删除。 |
| `/recall N` | 显式加入一个较早轮次；0 取消回顾。 |
| `/pause`、`/exit` | 结束当前对话，保留进度。 |
| Ctrl+C | 取消当前输入或模型请求并保留可恢复记录，不启动后台守护服务。 |
| EOF | 结束后续输入；若已有模型轮次正在处理，完成并保存该轮后退出。 |

新的讨论输入会使旧的生成就绪状态失效，需看到新摘要后再 `/run`，不另加重复确认。
teach 的 `/run` 是生成下一份课件，不表示课程结束；可以在同一课程继续学习和生成
多份材料。历史、课件、目标、摘要和当前回顾范围都按课程隔离。

成果保存在对应 Session 的 `artifacts/result-<turn>/`，命令展示实际路径。文章为修改
副本，原稿保留；原型为未运行的一次性源码／HTML；课程可保存后续课件。模型不会直接
获得写产品代码的步骤，host 只接受限定类型、数量、大小的安全单文件名，不运行成果。

## 中断后继续

```powershell
aiw teach list
aiw teach resume teach-20261017T120000-example
aiw research list
aiw research resume research-20261017T120000-example
```

用实际输出的 ID 替换示例 ID；`resume`、`recover` 不接受新的来源／分支参数，新增
资料请恢复对话后用 `/source`。provider／model 可以按本次调用覆盖，不改旧任务配置。

每次模型调用前保存 pending，模型轮次完成后再提交辅助历史和成果登记。若提交中断，
下一次不会自动重新调用模型；显式使用：

```powershell
aiw teach recover teach-20261017T120000-example
```

recover 只读取已有模型输出并补本地提交：相同成果可校对，缺失文件可补写，不覆盖
冲突内容，不调用模型。若返回失败但已明确记录失败轮次，可随后 resume 继续；原始
失败证据保留。完整历史按 `aux-turn-<turn>.json` 保存，不依赖 provider 长期线程。

硬杀进程、系统断电或通用 Session 工具并发修改可能留下未知 running 状态或
`auxiliary.lock`。程序不会抢锁或猜测进程已死：先确认原进程已停止、检查记录，再处理
那个具体锁文件；不能删除仍运行任务的锁。无法核对的轮次保持阻塞，不能以 recover
为名重放模型。恢复记录本身损坏时也会报错，不伪装成新课程。

## 知识来源、Skill 与预算

research 首版从显式提供的本地文本研究，可以使用已保存的官方资料；不会自动联网
搜索或声称启动了独立后台研究 Agent。没有来源时应说明证据缺口。teach 使用课程资料
和可见历史，未提供来源的讲解不是已外部核实的事实；不默认下载教材。

Skill 定位顺序：显式 `AIW_SKILLS_SOURCE_ROOT`，否则当前目录的 `src/skills`、发行
程序旁 `skills`、项目 `.agents/skills`。加载入口和辅助 CLI 适配引用；prototype 只选
LOGIC 或 UI 分支。开始时冻结 Skill／资料快照及摘要，恢复不悄悄换成新 Skill 内容。
Skill 不可用时清楚报错，不退化成没有方法的普通聊天。

- 单次用户输入最多 8 KiB；来源每次选择最多 64 个文本文件，单文件 32 KiB，累计资料 128 KiB。
- 仅允许当前目录内的支持文本类型；拒绝符号链接／越界，排除 `.ai`、`.git`、依赖／构建目录和已知凭据文件名。不是通用秘密检测器，选择文件前仍应核对内容。
- Skill 快照最多 48 KiB；目标／摘要／近期或显式回顾历史合计最多 32 KiB；预算不足报错，可缩小 `/window` 或另开较窄任务，不隐藏调用模型压缩。
- 模型响应最多接受 512 KiB；成果最多 8 个，每个 128 KiB，总共 256 KiB，允许 md/txt/html/css/js/py/json。

只有有界近期上下文进入模型，不声称每轮读过全部课程历史。精确请求、阶段指令和
原始输出均保留，可用课程目录中的记录追溯。无效 JSON、超限和执行错误不自动重试。

程序使用只读冻结请求，不携带使 Codex 走 workspace-write 的 observer，并在单独的
sandbox 目录调用模型。原有 CLI provider 的工具／网络隔离程度由后端决定；来源选择
限制是 host 的输入规则，不代表 CLI 模型不能读取其它文件。模型 API／CLI 调用仍需
使用者遵守目标仓库的数据和网络授权。不会自动打开生成 HTML 或启动原型服务。
