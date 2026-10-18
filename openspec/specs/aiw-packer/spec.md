# 独立打包与解包

## Purpose

提供与 `aiw packer` plugin 行为一致的独立 Go 命令。来源：ISSUE-00010、FD-079。

## Requirements

### Requirement: 格式、输入与命名

命令 MUST 支持 ZIP、TAR、TAR.GZ/TGZ、TAR.BZ2、GZ、BZ2 的打包与解包。pack
默认 ZIP，unpack 默认内容检测；显式格式 MUST 可用于消除嵌套 TAR 与纯流歧义。
GZ/BZ2 MUST 拒绝目录和多输入，并提示对应 TAR 组合格式。

首个位置参数 MUST 是 pack 或 unpack；无位置参数时默认 pack。命名输入 MUST
优先于位置输入且不合并。pack MUST 允许重复 -i；unpack MUST 拒绝重复 -i，且
位置输入仅使用 ACTION 后第一个。选项 MUST 支持 ACTION 前后混排和 --。

stdin MUST 要求显式输出，不能重复或混入其他输入。单输入 pack MUST 追加格式后缀，
多输入 MUST 默认当前目录 archive.<后缀>；unpack MUST 剥离完整格式后缀命名输出目录。

#### Scenario: 多输入与默认 ZIP

- WHEN 用户执行 pack a.txt b.txt 且未给输出
- THEN 在当前目录输出 archive.zip，内部保留每个输入 basename
- AND 未显式允许覆盖时，同名已有文件导致报错

### Requirement: 文件名编码

-e MUST 默认 UTF-8，支持 GB18030、GBK、CP437、Shift_JIS、Big5 的双向文件名转换，
MUST NOT 转换内容。不支持编码或无法无损转换时 MUST 报错。Big5-HKSCS 能力回退
MUST 明确披露为 Big5，MUST NOT 静默把显式 HKSCS 参数当成 Big5。
ZIP UTF-8 标志与 TAR PAX Unicode 元数据 MUST 优先，历史 TAR 字节名 MUST 使用
兼容的 GNU 表示。GZ/BZ2 MUST 按字节流生成输出名，不使用 GZIP 可选 FNAME 决定
输出路径；帮助 MUST 说明这些流中 -e 不参与文件名传递。

### Requirement: stdout、链接与完整预检

unpack 到 stdout MUST 仅接受目录与恰好一个普通文件。多个普通文件、符号链接、
硬链接或特殊条目 MUST 在写入任何 stdout 内容前报错，即使存在 -l。内容解码和
校验 MUST 在输出之前完成，诊断 MUST 写 stderr。

默认 MUST 拒绝符号链接；-l MUST 保存/恢复范围内有效相对链接本身，拒绝悬空、
循环或越界目标。-L MUST 仅用于 pack，以链接逻辑名存储目标内容；MUST 与 -l/-d
互斥。显式顶层目录链接可跟随，内部目标 MUST 受解析后目录范围限制。

### Requirement: 覆盖、发布及删除

默认 MUST 不覆盖文件；-O/--overwrite MUST 仅覆盖普通文件。目录 MUST 可合并；
文件/目录类型冲突和已有目标符号链接 MUST 报错，不得为覆盖递归删除目录。
输入输出身份相同、源目录包含 pack 输出、解包路径越界 MUST 被拒绝。

-d MUST 只在整体操作和输出关闭成功之后开始删除源。stdin/stdout/-L MUST 禁止
与 -d 同用；目录中未归档的新内容 MUST 保留。删除失败 MUST 保留结果、报告未删除
源并返回非零。多文件发布/删除不要求事务性回滚，失败边界 MUST 在帮助中披露。

#### Scenario: 覆盖不能删除目录

- WHEN 包内普通文件 report 与目标目录 report/ 冲突，即使用户指定 -O
- THEN 报错并保留已有目录及内容
- AND 此限制不改变 -d 在打包成功后删除源目录的含义

### Requirement: 可移植插件与资源

plugin MUST 透传参数、标准流和退出码，缺少二进制时明确报错，不自动下载、构建或
提升权限。独立子模块 MUST 保持主模块依赖不变，随附固定依赖的必要源码与许可证。
暂存 MUST 仅清理本操作拥有的资源；4 GiB 字节数、100000 条目、4096 字节路径及
128 层目录/链接链的首版边界 MUST 明确披露，超限失败不得触发删除源。
ZIP MUST 在分配条目表前预检实际条目数和中央目录范围；目录元数据限 64 MiB，
不支持分卷或自解压偏移布局。根目录空条目亦 MUST 完成内容/校验验证。
