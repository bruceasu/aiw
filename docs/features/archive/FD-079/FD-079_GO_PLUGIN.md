# FD-079: Go 独立压缩解压工具与 plugin

**Status:** Complete
**Revision:** 11
**Priority:** Medium
**Evidence policy:** Dual

## Problem

实现已批准的 ISSUE-00010：Go 独立压缩/解压命令，并作为 `aiw packer` plugin 使用。源代码位于 `src/programs/aiw-packer`；覆盖 ZIP、TAR、TAR.GZ/TGZ、TAR.BZ2、GZ、BZ2，支持历史文件名编码、标准流、安全覆盖及显式链接策略。

当前来源：`docs/issues/ISSUE-00010/issue-plan.md` 及其 CLI 审批记录。用户明确授权 approve、promote 和 fd-workflow auto 的完整本地生命周期；不包含网络、测试、最终构建或发布。

## Options and decision

1. 仅标准库：ZIP/TAR/GZIP 和 BZIP2 读取可用，但没有 BZIP2 写入和所需历史编码转换，不能满足已批准格式矩阵。
2. 调用系统 bzip2/编码工具：增加环境依赖，削弱独立 Go 工具的可移植性，不选。
3. 自行实现 BZIP2 编码器及完整字符映射：工作量和正确性风险不适合本 FD，不选。
4. 决定：独立 Go 子模块，复用标准库格式处理；新增固定版本 `golang.org/x/text v0.24.0` 负责文件名转换，`github.com/dsnet/compress v0.0.1` 提供 BZIP2 写入。已按明确授权下载一次并静态读取，`bzip2.NewWriter(io.Writer, *WriterConfig)`、Write 和 Close 满足流接口，BSD 三条款许可证保留在 third_party。只复制实际包闭包的非测试源码，并以本地 replace 固定；不拉取其旧 go.mod 中仅其他格式/工具使用的 golib、xz、klauspost 等依赖。

`x/text v0.24.0` 已在本机 Go 模块缓存中；静态读取确认存在 GB18030、GBK、CP437、Shift_JIS 和 Big5 编码入口。没有据此确认完整 Big5-HKSCS 支持；按用户授权优先核实，不能支持时明确采用 Big5，不把 HKSCS 参数静默当 Big5。GB18030 无需预先缩减到 GBK。

依赖授权已解决：用户明确选择“允许上述依赖和一次固定版本下载”。已执行一次 `go mod download -json github.com/dsnet/compress@v0.0.1`，取得校验和 `h1:PlZu0n3Tuv04TzpfPbrnI0HW/YwodEXDS+oPKahKF0Q=`。x/text 来自已有本机缓存，不新增下载。没有授权或执行其他依赖下载、测试或最终构建。

独立设计审阅已真实执行，原报告 changes-needed 保留。Planner 以 Revision 3/4 明确处理 D1 路径/重名、D2 格式编码矩阵、D3 暂存上限及清理、D4 自包含/删除身份；不是伪称独立复查通过。依据报告允许 Planner 定稿的结论，不新增重复审阅；无剩余设计阻塞后发 design-ready，再建立隔离 worktree。

## Solution

### 模块与职责

- `src/programs/aiw-packer/`：独立 `go.mod`，入口 `main.go`；选项解析、源清单、编解码适配、pack/unpack 执行和路径安全分文件组织，保持主模块依赖不变。
- `src/plugins/aiw-packer/`：按现有 plugin manifest/launcher 约定包装独立二进制，传递 stdin/stdout/stderr 和退出状态；不在调用时自动下载、编译或安装。
- 独立说明文档、窄范围 compile-only 脚本和 `openspec/specs/aiw-packer/spec.md` 记录本功能；不新建 OpenSpec change，不改通用插件分发接口。

### 已批准的 CLI 契约

- 首个位置参数为 pack/unpack，没有位置参数时默认 pack。`-i` 优先于位置输入，不合并；pack 可重复 -i，unpack 重复 -i 报错；位置 unpack 仅用 ACTION 后第一个输入，忽略其余。
- 选项可位于 ACTION 前后，支持 `--`。支持 -h/--help、-f/--format、-e/--encoding、-i/--input、-o/--output、-O/--overwrite、-c、-d/--delete、-l、-L。--ouput 不作为别名；覆盖长参数规范拼写为 --overwrite，未要求兼容 --overwirte。
- pack 默认 zip；unpack 自动检测。GZ/BZ2 接收单普通文件或字节流，目录或多输入报错并提示 TAR 组合格式，不静默转换格式。
- 单输入 pack 输出为输入名追加后缀；多输入默认为当前目录 archive.<后缀>。unpack 默认目录剥离完整后缀。stdin 必须显式指定 -o 或 -c，归档内部名为 stdin。
- 默认 UTF-8；-e 同时作用于打包/解包文件名，不改变内容。不支持的编码或无法表示的文件名报错；无文件名元数据时不转换，并在帮助中说明。具体格式元数据能力在依赖 Gate 解除后核实，不能承诺所有格式能承载任意编码。
- 格式矩阵：ZIP UTF-8 名设置对应标志，历史编码名显式标为非 UTF-8；解包 UTF-8 标志优先，否则按所选编码严格解码。TAR UTF-8 使用 PAX，历史字节名用 GNU：已静态确认标准库 GNU 字段校验允许非 ASCII 字节，拒绝 NUL 并支持长名；PAX path/linkpath 属于 UTF-8 元数据，解包不再按历史编码二次转换。TAR.GZ/TAR.BZ2 继承 TAR 内部编码；GZ/BZ2 不传递本工具文件名，仅压缩字节流。Big5-HKSCS 无独立完整支持证据，首版明确提供 Big5，指定 big5-hkscs 报错提示 Big5；GB18030、GBK、CP437、Shift_JIS 均由本地 x/text 包提供，非法解码以严格往返校验防止静默替换。

### 数据与安全策略

- unpack 输入先暂存为可寻址私有临时文件；解码归档到受控临时区域和清单，完成全包条目/路径/编码/冲突验证及内容校验和后才开始目标发布或 stdout 输出。ZIP 由可寻址输入读取；TAR 系列统一逐条目解析并读尽压缩层以核实 trailer/checksum。退出清理仅限本操作实际创建的临时目录，资源上限见下。
- 归档路径采用每个显式输入的 basename 为顶层（目录保留该根目录），子项相对该顶层组织；stdin 为 stdin。多输入 basename 重复、编码/分隔符/平台规范化后的重名、重复条目、文件/目录前缀冲突一律在输出前拒绝，不以 -O 掩盖归档内部冲突。源根为磁盘根或不能形成安全 basename 时，明确报错而不猜测条目名。
- pack 输出不得位于任一输入目录内，即使未使用 -d 也拒绝，避免自包含和预先存在输出被再次归档；还需比较实际解析路径/文件身份，拒绝别名指向同一输入。stdin 与其他输入混合、重复 stdin 均拒绝以保持单流语义。
- 私有暂存目录统一用 os.MkdirTemp 创建并只清理这个已记录的目录；不删除用户输出目录。上限选择为压缩输入 4 GiB、总展开数据 4 GiB、100000 个条目、单路径 4096 字节；以限制读取和条目累计计数强制实施，超限报错保留源，stdout 尚未写出。按实际流量计数，不相信归档自报大小；这些是首版资源保护边界，文档明确列出。临时文件关闭/清理错误必须报告，不能被主错误吞掉。
- 普通文件通过临时文件写完关闭后发布；不覆盖使用排他创建，显式 -O 仅替换普通文件。已有目录合并；文件/目录冲突、已有目标符号链接报错，不为覆盖递归删除目录。不承诺多目标发布的事务性回滚；发布失败保留源且明确报告。
- 使用 Go `os.Root` 或等效受约束访问避免目标路径经链接逃逸；不能只检查字符串前缀。拒绝绝对条目、上级越界、Windows 设备名/驱动器路径和输入输出同一文件。恢复链接前单独验证其相对目标；`Root.Symlink` 本身不校验目标，不能代替此检查。
- 默认拒绝链接；-l 保存/恢复范围内相对符号链接，不跟随内容；拒绝悬空链接。解包先验证归档中的完整目标关系，不因目标条目排在后面误判悬空；普通条目完成后才创建安全链接。硬链接及特殊文件不支持。
- -L 仅 pack，读取目标内容并以链接逻辑名入包；与 -l/-d 互斥。目录循环、悬空及越界目标报错。显式顶层链接可跟随，以解析后的目录目标作为边界；内部链接不能越界。路径读取过程中仍必须使用受约束访问，不能仅靠事先 EvalSymlinks。
- 纯 GZ/BZ2 按字节流处理，不承载本工具的归档条目命名。GZIP 写入省略可选 FNAME；读取不依据 FNAME 选择路径，以避免标准库 GZIP FNAME Latin-1 限制与默认 UTF-8 冲突。合成唯一普通文件名取输入 basename 剥离完整格式后缀；stdin 合成名为 stdin；-e 不改变纯流内容或合成路径，帮助明确 GZIP 可选 FNAME 未用于文件名传递，而不是错误宣称格式没有该字段。
- stdout 解包仅接受目录及恰好一个普通文件；发现多个文件、任何链接或特殊条目时保证 stdout 零输出后报错，-l 不豁免。错误和提示仅写 stderr。
- -d 禁止与 stdin/stdout/-L 同用。全部打包或解包及输出关闭成功后才开始源删除；目录源递归删除，删除链接本身而非目标。输出不能处于待删除源目录内。删除失败保留输出、报告未删路径、非零退出；不承诺多源删除回滚。
- 删除安全：显式输入重复或父子重叠时在 pack 输出前拒绝。源清单记录文件身份/类型/内容摘要和链接文本；读取前后变化时报错。删除前重新验证整份源清单，任何变化则不启动删除。目录源仅删除本次已归档且身份仍一致的条目，再自底向上删除空目录；不对原始源调用无边界 RemoveAll，以免删除打包后新增的内容。并发新增项导致目录非空时保留并报告删除失败；剩余竞态需设计审阅，不把元数据比较宣称为原子保证。

### 尚待工程核实（不是批准产品规则变更）

上述依赖、API、许可证及 TAR 字节名已有本地静态证据；Go 1.25.1 的 os.Root 具备所需 Lstat/OpenFile/MkdirAll/Readlink/Symlink/Remove/Rename。ZIP 采用 Unix 模式位与链接目标作为条目内容的互操作约定，旧解压器可能不支持该约定，帮助明确这一限制。Windows 创建目录链接要求目标已存在，按已验证依赖顺序恢复。路径竞态通过句柄约束访问和删除前后身份复查降低风险，不承诺抵御本机其他进程对同一源树的恶意并发写入。全部平台运行兼容性未测试，记录为证据限制而非已通过。

## Scope

完整保留 ISSUE-00010 的单一用户可验收结果，不拆分格式交付或遗漏 plugin。主项目 API、依赖与通用分发不变。无 GUI、加密、多卷、任意外部链接、硬链接保留、权限提升、网络自动安装、发布或部署；本 FD 不写测试，不执行测试或最终构建。

## Work items

- [x] 1.1 固定依赖及格式/编码决策。小，中等；依赖：人工依赖授权；完成：取得并静态审阅候选源码、许可证与完整格式编码矩阵，清除设计 Gate；不以依赖获取冒充运行验证。
- [x] 1.2 独立入口和 CLI 解析。小，中等；依赖 1.1；完成：选项混排、位置输入后备、重复输入、默认路径和参数冲突符合契约；静态审阅调用路径。
- [x] 1.3 编码适配。小，中等；依赖 1.1；完成：UTF-8 与已确定历史编码双向转换、严格失败、格式适用帮助及内容不变；审阅编码映射与错误传播。
- [x] 1.4 源清单与链接策略。小，中等；依赖 1.2；完成：普通文件/目录遍历、-l/-L、循环/悬空/越界和输入输出重叠检查；审阅身份和范围判断。
- [x] 1.5 ZIP/TAR 打包。小，中等；依赖 1.3/1.4；完成：清单按名称编码写入对应容器并传播读写/关闭错误，ZIP 默认与 TAR 显式格式均满足范围。
- [x] 1.6 压缩流适配。小，中等；依赖 1.1/1.5；完成：TAR.GZ/TGZ、TAR.BZ2、纯 GZ/BZ2 编解码与输入限制一致；审阅 writer finalize 与错误路径。
- [x] 1.7 解包解析与暂存。小，中等；依赖 1.3/1.6；完成：内容检测、输入暂存、全条目清单、损坏输入及不支持类型报错；不提前发布 stdout。
- [x] 1.8 目标发布与覆盖。小，中等；依赖 1.7；完成：受约束写入、排他创建/-O 普通文件覆盖、目录合并、安全链接恢复、stdout 完整预检；审阅路径和关闭错误。
- [x] 1.9 源删除与错误报告。小，中等；依赖 1.4/1.8；完成：整体成功后才删、stdin/stdout/-L 限制、残留路径与非零退出；禁止误删输出。
- [x] 1.10 plugin、用户文档与稳定 spec。小，中等；依赖 1.2–1.9；完成：可独立运行与现有 plugin 发现兼容、帮助完整、不隐式下载/编译，编译脚本无最终产物。
- [x] 1.11 编译与独立审查证据。小，中等；依赖全部实现项；完成：一次窄范围离线 compile-only、一次静态核对、真实 Worker 双报告及独立 Reviewer 交接材料；未运行检查明确记录。

各项不超过半日是拆分目标而非工期承诺。设计 Gate 已解除，可按依赖顺序实现。实现开始后保持编号不变。

- [x] 1.12 修复 R1：符号链接目标按文件系统逐组件展开后处理 ..，拒绝真实越界与非目录穿越；小，中等，依赖首轮 R1；源码 c3d2c27，静态追踪原反例。
- [x] 1.13 修复 R2：ZIP 目录精确衔接 EOCD/ZIP64，匹配尾部选择与兼容分支，防止未受限记录继续分配；小，中等，依赖首轮 R2；源码 c3d2c27，静态对照标准库。
- [x] 1.14 修复后编译及补充 Worker 双报告；源码 c3d2c27 的同一离线 compile.py 成功，独立复查尚未授权。

## TODO

- [x] 取得依赖变更及一次固定版本下载授权，静态核实候选 API/许可证后定稿。
- [x] 完成必要的独立设计静态审阅，Planner 据 D1–D4 修订定稿；原始 changes-needed 报告保留，不伪称第二轮通过。
- [x] 提交范围内 Issue/FD 计划，确保 develop 干净，再用 `aiw git wt add FD-079` 建立 `feature/FD-079` / `.wt/FD-079`。
- [x] 实现并按可审阅 Work Item 提交，完成当前源码离线 compile-only 和真实 Worker 双报告。
- [x] 独立审查通过，CLI squash 交付至 develop 并验证清理，FD 已 Complete 归档；本次归档提交保存最终记录。

## Acceptance

验收基线是批准 Issue 的主规则与示例，全部映射到上述 Work Items：

- 1.2/1.10：独立和 plugin 帮助、命名/位置优先级、pack 多输入和 unpack 单输入、ACTION 和长短选项一致。
- 1.3/1.5/1.6：六类格式 pack/unpack；默认 zip/内容检测；UTF-8 默认及历史编码双向、文件内容不转码；未支持能力明确失败。
- 1.4/1.8：-l 保留相对安全链接，-L 展开内容；循环/悬空/越界和硬链接/特殊文件报错；Windows 创建链接失败报错，不提升权限。
- 1.7/1.8：stdout 在条目数/类型验证完成后只输出一个普通文件内容；拒绝时零输出。stdin 输出显式、内部名 stdin。
- 1.8/1.9：-O 仅普通文件覆盖，目录合并，类型冲突及目标链接报错；同文件、越界及删除源包含输出均拒绝。-d 在全操作成功之后执行；删除失败保留输出、残留路径及非零退出。
- 1.10/1.11：安装依赖由显式准备负责，plugin 不在调用时构建下载；提供离线 compile-only 路径和真实静态审查证据，不宣称未运行的兼容性测试通过。

## Verification

- 用户授权的一次聚焦复查已由独立 Reviewer `codex-fd079-reviewer-20261018-r2` 完成，真实来源事件 `FD-079-000010-implementation-ready`，审阅源码 `c3d2c27`（相对 `bd63c3d`）：**verification-passed**。R1/R2 已解决，直接影响路径无新增阻塞；未改变范围沿用首轮静态证据。报告 `docs/features/reviews/FD-079-implementation-review-20261018-r2.md` 及同名 JSON。未重跑整个实现审阅，未运行测试、编译、packer、恶意包、构建或网络；跨平台运行证据限制保留。
- 首轮独立 Reviewer `codex-fd079-reviewer-20261018-r1` 已领取 `FD-079-000006-implementation-ready` 并完成静态审阅：**changes-requested**。报告 `docs/features/reviews/FD-079-implementation-review-20261018-r1.md` 及同名 JSON。R1 链接逐组件解析与 `..` 清理顺序可产生越界/悬空目标；R2 ZIP 预扫描声明范围与标准库实际目录读取范围不一致，可绕过预分配资源保护。两项随后由 Worker 在 c3d2c27 修复并完成离线编译；本条保留首轮失败事实，当前结论见上一条。Reviewer 未运行测试、编译或程序。
- Worker 已完成实现；原始源码提交 bd63c3d 的窄范围离线 compile-only 成功，未运行测试或 packer 程序。原始 Worker 报告：docs/features/reports/FD-079-implementation-20261018-a7c3.md 及同名 JSON。修复源码及聚焦复查结论见上述记录。
- 已执行 `python src/programs/aiw-packer/scripts/compile.py`：已检查脚本源码并运行，离线 `go build -o os.devnull .`，禁用下载，不保留可分发二进制；如 wrapper 为 Python，可在同命令中内存 compile，不写 pyc。
- 默认测试、最终构建、格式化、lint、vet、验证脚本及网络预算均为零。固定版本依赖源码下载是待决的单独授权，不扩展验证权限。
- post-edit 静态核对最多一条命令；不重复未变失败命令。compile-only 若源错误修复后可按仓库规则重跑一次。
- 必要设计审阅与最终独立实现审阅分开。独立设计 Reviewer 对 Revision 2 的 changes-needed 报告保留；Planner 在 Revision 3/4 据源码证据处理 D1–D4，没有伪称重复独立审阅。依赖 Gate 已解决；首轮实现审阅要求修改，修复后获授权的一次聚焦复查已通过。完整本地交付仍须满足其他实际 Gate。
- 原始独立设计报告：`docs/features/reviews/FD-079-design-review-20261018-a7c3.md` 及同名 JSON；Planner 修订决定以本 FD 为准。阻塞报告 `docs/features/reports/FD-079-blocker-dependencies-20261018-a7c3.md` 已记录授权和实际下载，不擦除未授权时的历史事实。

## Sources

- Issue: ISSUE-00010

- Issue 需求与批准：`docs/issues/ISSUE-00010/`。
- 标准 plugin 约定：`openspec/specs/cli-and-plugins/spec.md`。
- 独立 Go 子模块先例：`src/programs/md-converter/go.mod`、`src/programs/md-converter/scripts/compile.py`。
- 插件包装先例：`src/plugins/aiw-file-convert/plugin.toml`。
- 主模块边界：`src/go.mod`；本机 Go `compress/bzip2` 只有 NewReader；本机缓存 `golang.org/x/text@v0.24.0/encoding` 的相关编码入口。
- Planner 会话：`codex-fd079-planner-20261018-a7c3`；design-ready 已发。Worker 会话 `codex-fd079-worker-20261018-a7c3` 已 claim `FD-079-000005-design-ready`。

## Worker 完成记录

源码 bd63c3d 已编译通过。首次 9f97746 编译也成功，之后静态核对发现 ZIP 根条目校验/预分配资源边界并修复，当前源码重新编译成功；两次均不保留最终产物。未运行测试、运行验证、最终打包或其他网络。ZIP 中央目录另限 64 MiB，预扫描实际条目；该边界与帮助/spec 一致。实现报告和独立审查报告分离，尚未宣称审查通过或已交付。

## 首轮实现审查与修复 Gate

独立 Reviewer 会话 codex-fd079-reviewer-20261018-r1 已真实审查并发出 FD-079-000007-changes-requested；原报告 docs/features/reviews/FD-079-implementation-review-20261018-r1.md 保留，不能描述为通过。Worker 已领取该真实事件，修复 R1/R2 并提交 c3d2c27；源码修复后一次离线 compile-only 退出0，未测试或运行程序。

当前修复证据：docs/features/reports/FD-079-repair-20261018-r1.md 及同名 JSON。预算 Gate：docs/features/reports/FD-079-blocker-review-budget-20261018-r1.md 及同名 JSON。按仓库 repeated review passes 默认0，未再 dispatch Reviewer，未本地合并、清理或归档；等待一次聚焦独立静态复查的明确授权。原自动交付授权仍保留，真实复查通过后继续，无需另一次 auto/merge 请求。
## 一次聚焦复查授权

用户明确回复“允许一次聚焦静态复查并继续交付”。授权对象为源码 c3d2c27 的 R1/R2 修复及直接影响调用路径，由独立 Reviewer 执行一次静态复查；不新增测试、程序运行、编译、最终构建或网络权限。本授权解除重复审阅预算 Gate，不表示复查已通过。真实通过后继续原已授权本地交付、清理、Complete 归档。

**Completed:** 2026-10-18

## PM 本地完成记录

本节只补记实际交付/归档结果，不改变已审阅源码或能力决策。获用户授权的第二轮聚焦独立复查已真实通过，事件 FD-079-000011-verification-passed，R1/R2 已解决；报告位于本归档 reviews/FD-079-implementation-review-20261018-r2.md 及同名 JSON。首次失败和原始报告保留，复查次数合计2（第二次有明确预算例外授权）。

- 已交付源码编译证据：c3d2c27 的离线 compile-only 成功，没有本轮额外运行检查。
- 审查及证据最终 FD HEAD：18787856ae5f691918eca3162b9194ff1f00bed3，审查后源码无变化。
- develop 上单父提交：d5698e8458d0013c84e7347b174c5ba5f4a01fe5；FD-Source 与上述 FD HEAD 完全一致。
- aiw git wt local-merge FD-079 成功，并确认已移除 .wt/FD-079 和 feature/FD-079，保留 FD receipts。
- aiw fd close FD-079 Complete 成功；本文件及证据已位于 docs/features/archive/FD-079/。
- 两个授权 Gate 均已解决；复查后的当前验证状态为通过，不以此前等待授权记录代替当前状态。
- PM 通知不是可由 agent claim 的角色：一次 PM claim 被 CLI 拒绝后未重试、未改 receipt，按正常 host PM 交付入口继续成功；未使用 force、跳过审查或重写历史事件。
- 没有测试、运行程序、最终构建、打包分发、推送、发布或部署；授权网络仍仅为早先一次固定版本 BZIP2 源码下载。跨平台/旧工具互操作未运行验证，详见原报告和 README。