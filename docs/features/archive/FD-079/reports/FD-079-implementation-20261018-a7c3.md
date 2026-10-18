# FD-079 实现报告

<!-- aiw-data: FD-079-implementation-20261018-a7c3.json -->

- FD：FD-079；Worker 会话：`codex-fd079-worker-20261018-a7c3`。
- 已领取来源事件：`FD-079-000005-design-ready`。
- 比较基线：父分支 develop 的计划提交 `28dd3bf`。
- 实现源码提交：`bd63c3d`；本报告之后的提交仅更新 FD/证据，不改变已编译源码。
- 状态：实现及 Worker 证据完成，待独立 Reviewer；不声明审查通过或已交付。

## 变更与原因

- `src/programs/aiw-packer` 为独立 Go 1.25.1 子模块，不改主模块依赖。ZIP/TAR 容器与 GZIP/BZIP2 流均有 pack/unpack 路径；默认 ZIP/内容检测、位置参数后备、重复 -i、-- 和 stdout 规则已实现。
- UTF-8 默认，GB18030/GBK/CP437/Shift_JIS/Big5 双向严格文件名转换。Big5-HKSCS 无完整承诺，按用户授权提供显式 Big5 回退；不静默换码或替换字符。ZIP Unicode 标志和 TAR PAX 字段优先；历史 TAR 用 GNU，GZ/BZ2 为无文件名传递的字节流。
- 打包先建立源快照并暂存；解包先暂存、校验所有内容及条目，才输出。ZIP 中央目录在标准库分配前预扫描；普通和 `./` 根目录条目都读取校验。TAR 读尽尾部压缩层，不在遇到 TAR EOF 后跳过 GZIP/BZIP2 trailer。
- 路径使用 os.Root 句柄边界，拒绝设备名、越界、重名/Unicode NFC 和大小写冲突、文件/目录冲突、输出别名与自包含；-O 只覆盖普通文件，目录合并，目标符号链接报错。
- -l 还原经过完整图验证的相对链接；-L 仅 pack，跟随目标并检测循环/越界；悬空链接均拒绝。stdout 解包只允许目录和一个普通文件，不能通过 -l 放行链接。
- -d 禁止与 stdin/stdout/-L 组合；整体操作成功、输出关闭后才开始删除。源身份和内容先重新检查，只删除已快照条目，目录非空时保留新增内容；删除失败保留结果、列出未删路径、非零退出。
- plugin manifest 和 Python launcher 透传参数、流与退出码，缺少二进制时报错。显式 package.py 可生成当前平台分发二进制；本流程未执行。README、稳定 spec、许可证和离线 compile.py 已提供。

## 静态证据

- 按 FD Work Items 逐项检查解析、编码、文件清单、压缩层收尾、解包预检、受约束发布、链接依赖顺序和删除错误传播。
- 检查最终 diff 范围：独立子模块、packer plugin、稳定 capability spec、FD/证据。大量新增行为外行数来自已固定的第三方编码/Unicode 表，非工具或测试代码；第三方包非测试源码保持原样，本地 go.mod 只保留模块名称。
- 最后一次静态核对读取了图校验、发布和删除关键路径。直接读 worktree/.ai/fd/workspace.json 曾因无该路径失败；随后从父工作区读取正确元数据，并确认 worktree 中 `aiw fd show` 读取同一已领取事件。未伪造或改写 receipt；本地 .ai/compile-cache 仅为编译缓存。
- 独立设计审阅历史 changes-needed 与 Planner 修订均保留。依赖授权阻塞已解决，不把设计修订自称为独立复查通过。

## 实际命令与结果

- 用户授权后仅一次 `go mod download -json github.com/dsnet/compress@v0.0.1` 成功；x/text 来源于本机已有缓存。源码复制只包含所需包闭包，没有拉取旧模块中其他工具/格式依赖。
- Issue approve/promote、Planner claim/design-ready、Worker claim、计划与 Work Item 聚焦提交、`aiw git wt add FD-079` 成功。记录父分支 develop，worktree `.wt/FD-079`，分支 feature/FD-079。
- `python src/programs/aiw-packer/scripts/compile.py` 在 `9f97746` 成功，之后静态发现 ZIP 目录根条目校验与预分配数量边界需要修正。
- 源码修正并提交 `bd63c3d` 后，同一窄范围 compile.py 再执行一次成功；不是对未变失败命令重跑。两次均设置 GOPROXY/GOSUMDB=off、GOTOOLCHAIN=local、CGO_ENABLED=0，`go build -o os.devnull .` 不保留最终程序；Python 仅内存 compile，不执行 launcher/package 脚本。
- 静态核对执行 git diff --stat、关键文件摘录、git status 及工作区元数据读取；失败的直接 .ai 路径读取仅作一次路径更正。未用验证脚本替代运行证据。

## 未执行与限制

未执行任何测试、packer 可执行运行、最终 package/build、formatter、linter、vet、仓库级验证、部署或发布。没有创建新测试。未下载第二个模块或执行其他网络访问。

没有运行证据证明 Windows/Linux/macOS 格式互操作、历史包或权限失败行为。Windows 链接创建可失败，旧 ZIP 工具可能忽略链接约定；帮助明确。源树需保持稳定，不承诺抵御同机恶意并发替换。输出发布/删除不具备多文件事务性回滚。

资源边界已写入帮助：4 GiB 字节流、100000 条目、4096 字节路径、128 层目录/链接链、64 MiB ZIP 中央目录元数据；不支持分卷或自解压偏移 ZIP。格式检测对“纯流内容正好是 TAR”存在固有歧义，可用显式 -f 指定流语义。

下一步由独立 Reviewer 领取新的真实事件，检查当前 diff 与批准 Issue。只有真实 verification-passed 及交付 Gate 满足后才本地合并、清理与 Complete 归档。
