# aiw-packer

独立 Go 压缩/解压工具，也是 `aiw packer` plugin。Go 1.25.1+，运行二进制不需要
Go、Python 或系统压缩命令；通过 Python plugin 包装调用时需要 Python 3。

## 准备与使用

只编译检查、不留下二进制：

```text
python src/programs/aiw-packer/scripts/compile.py
```

显式生成当前平台的 plugin 二进制（这会产生最终构建产物）：

```text
python src/programs/aiw-packer/scripts/package.py
```

二进制放在 `src/plugins/aiw-packer/`，可独立运行，也可随该完整目录安装到 AIW
的 `plugins/aiw-packer/`。从源码安装仓库 plugins 前先显式执行上述打包命令；本程序
不修改或自动调用根构建流程。复制完整 plugin 目录时保留 LICENSES 和版权说明。
`AIW_PACKER_BINARY` 可指定已有的独立二进制。plugin 缺少二进制时明确报错，不自动
编译、下载或提升权限。两个依赖的必要源码已随子模块固定，构建默认离线。

```text
aiw packer -h
aiw packer -i report.txt
aiw packer pack -i a.txt -i b.txt -o bundle.zip
aiw packer unpack -i bundle.zip -o extracted
aiw packer pack -f tar.gz -i folder -o folder.tar.gz
aiw packer unpack -e gb18030 -i old.zip -o restored
aiw packer unpack -O -i bundle.zip -o extracted
aiw packer pack -f gz -i - -c
```

## CLI 与默认路径

- 首个位置参数为 `pack|unpack`。没有位置参数时默认 pack，但仍须提供 `-i`。
  `aiw packer file.txt` 不是隐式 pack；应写 `aiw packer pack file.txt`。
- `-i/--input` 优先于位置输入，两者不合并。pack 可重复 `-i`；unpack 只允许一个
  `-i`，位置输入则仅使用 ACTION 后第一个，忽略其余。
- 选项可在 ACTION 前后混排；`--` 后均为位置参数。`--ouput`、`--overwirte` 不提供别名。
- `-f/--format`：pack 默认 zip；unpack 默认按内容检测，允许显式指定。
- `-o/--output`：pack 单输入追加格式后缀，多输入在当前目录生成 `archive.<后缀>`。
  unpack 在源文件旁创建剥离完整后缀的目录，例如 backup.tar.gz 解到 backup/。
  GZ/BZ2 在该目录中生成一个剥离后缀的普通文件；stdin 的合成文件名为 stdin。
- `-i -` 从 stdin 读取，必须显式给出 `-o` 或 `-c`，不能与其他输入混合或重复。
  pack 的 stdin 归档内部名为 stdin。目录输入保留顶层 basename；重复 basename、
  重叠输入、大小写或 Unicode NFC 规范化后重名均拒绝。
- `-c` 强制 stdout，优先于普通 `-o`。unpack 只允许目录与恰好一个普通文件；多个
  文件、任何链接或特殊条目均在 stdout 写出前报错。错误和提示仅写 stderr。

## 格式与文件名编码

| 格式 | pack/unpack | 文件名策略 |
| --- | --- | --- |
| ZIP | 支持 | UTF-8 标志优先，否则使用所选历史编码 |
| TAR | 支持 | UTF-8 使用 PAX；历史字节名使用 GNU；PAX Unicode path/linkpath 优先 |
| TAR.GZ / TGZ | 支持 | 使用 TAR 内部文件名策略 |
| TAR.BZ2 | 支持 | 使用 TAR 内部文件名策略 |
| GZ | 单普通文件或字节流 | 不写可选 FNAME，不用输入 FNAME 选择路径 |
| BZ2 | 单普通文件或字节流 | 合成输出名，不传递内部文件名 |

GZ/BZ2 遇到目录或多个输入时报错，提示 TAR 组合格式，不自动转换。
内容检测不能消除所有歧义：纯 GZ/BZ2 内容本身是 TAR 或全零 TAR 头时可能按 TAR
组合格式识别；需要保留原始流含义时用 `-f gz` / `-f bz2`。

`-e/--encoding` 默认 UTF-8；支持 GB18030、GBK、CP437、Shift_JIS 和 Big5，打包和
解包双向生效。Big5-HKSCS 没有独立完整支持承诺；按需求回退为显式 Big5，指定
big5-hkscs 会报错提示 Big5，不静默替代。纯压缩流不使用 `-e` 改写其合成路径。
编码只处理文件名与链接目标文本，绝不转换文件内容。无法表示或无法无损解码的
文件名明确报错；严格往返检查也可能拒绝非规范的等价字节序列。

## 链接、覆盖与删除

默认拒绝符号链接。`-l` 保存/恢复链接本身，只接受指向归档清单内有效条目的相对
目标；悬空、循环、越界目标报错。解包先发布普通文件/目录，再按依赖顺序恢复链接。
Windows 创建链接可能因权限/文件系统失败；不会自动提权。旧 ZIP 解压工具可能不识别
Unix 模式位链接约定。硬链接条目和特殊文件不支持；源普通硬链接作为普通文件打包，
不保存硬链接身份。

`-L` 仅用于 pack，将目标内容以链接逻辑路径入包，与 `-l`、`-d` 互斥。显式顶层
链接可跟随；目标为目录时以解析后的目录作为内部链接不可越出的边界。

默认不覆盖文件；`-O/--overwrite` 只覆盖普通文件。已有目录合并，文件/目录类型
冲突或已有目标为链接时报错，不为覆盖递归删除目标目录。

`-d/--delete` 在整个操作完成并关闭输出后才开始删除源。目录源删除本次已打包的
条目后自底向上删除空目录；新增/变化条目导致保留源并报错，不盲目删除新内容。
stdin、stdout 和 `-L` 禁止与 `-d` 同用。删除失败保留结果、报告未删路径并返回非零。
目标发布与多源删除不是事务；中途失败可留下部分已发布结果或已完成的删除。

## 边界与资源

禁止输入输出同一文件或别名、pack 输出位于源目录内、解包路径越界，以及目标父路径
中的符号链接。临时目录不能位于待打包源目录内，可用 TMPDIR/TEMP 选择其他暂存位置。
源/目标目录通过 Go 目录句柄限制访问。删除前重新验证身份和摘要，但不承诺抵御本机
其他进程对相同源树的恶意并发修改；使用时保持源内容稳定。

先暂存和完整校验内容，再发布输出。默认限制为压缩输入/结果 4 GiB、展开数据
4 GiB、100000 条目、4096 字节路径及链接目标、64 MiB ZIP 中央目录元数据，目录遍历/链接链深度 128；只清理
本操作创建的临时目录。资源超限、磁盘满、校验失败或输出关闭失败均报错，不触发源删除。

ZIP 在分配条目表前预扫描中央目录，拒绝数量不一致、分卷和自解压偏移布局；ZIP64
在同一条目数与资源限制内支持。安全的 `./` 路径前缀和无内容的根目录条目可读取。

错误返回非零；不声明未经执行的跨平台测试结果。第三方版本与许可证见
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md)。
