# AI Code Tools

两个互补的本地工具：`generate-ai-index` 为单个仓库生成索引；
`ai-code-index` 汇总多个仓库的索引并提供关键词检索。

## 快速开始

### 在 AIW 仓库中直接运行

无需安装，使用 Python 运行脚本：

```powershell
python plugins/aiw-ai-gen-index.py --root .
python plugins/aiw-ai-code-index.py repo add .
python plugins/aiw-ai-code-index.py update
python plugins/aiw-ai-code-index.py context "订单创建接口" --current-repo
```

直接运行仓库脚本时，先用 `aiw-ai-gen-index.py` 生成 `.ai/` 文件，再用
`aiw-ai-code-index.py update` 导入全局检索库。之后源码有变化时，重复这
两步。若已安装全局 `generate-ai-index` 命令，则可用
`ai-code-index update --run-generator` 一步刷新所有登记仓库。只想刷新单个
仓库时，直接运行生成器后再扫描：

```powershell
python plugins/aiw-ai-gen-index.py --root .
python plugins/aiw-ai-code-index.py scan .
```

### 安装为全局命令

在 Unix shell 中，从解压后的发布目录运行：

```bash
./install.sh
```

默认安装到 `~/.ai-tools`。将该目录加入 `PATH` 后，可以运行：

```text
generate-ai-index --help
ai-code-index --help
ai-code-index repo --help
```

也可以把安装目录作为 `install.sh` 的第一个参数。发布包不要求安装第三方
Python 依赖。

## 工具分别做什么

### `generate-ai-index`

静态扫描 Java、Go、Python、JavaScript 和 TypeScript 源文件，输出：

```text
<仓库>/.ai/PROJECT_MAP.md
<仓库>/.ai/API_INDEX.md
<仓库>/.ai/symbols.jsonl
<仓库>/.ai/apis.jsonl
<仓库>/.ai/files.jsonl
<仓库>/.ai/metadata.json
```

默认扫描当前目录并写入 `.ai/`。也可以指定仓库和输出目录：

```powershell
python plugins/aiw-ai-gen-index.py --root D:/work/my-repo --out .ai
```

`--out` 是相对于仓库根目录的路径。索引是生成文件，源码变化后需要重新
生成。生成器不执行仓库代码，也不访问网络。

### `ai-code-index`

在用户目录维护已登记的仓库清单和 SQLite 检索库：

```text
~/.ai-code-index/repos.json
~/.ai-code-index/db.sqlite
```

常用命令：

```powershell
# 登记、查看和移除仓库
python plugins/aiw-ai-code-index.py repo add D:/work/my-repo
python plugins/aiw-ai-code-index.py repo list
python plugins/aiw-ai-code-index.py repo remove D:/work/my-repo

# 更新索引；带 --run-generator 时先重新生成各仓库的 .ai 文件
python plugins/aiw-ai-code-index.py update --run-generator

# 搜索候选记录，或生成带文件位置的 Agent 上下文
python plugins/aiw-ai-code-index.py search "订单创建接口" --top 10 --current-repo
python plugins/aiw-ai-code-index.py context "订单创建接口" --top 8 --current-repo

# 查看收录数量
python plugins/aiw-ai-code-index.py stats
```

`search` 便于浏览候选结果；`context` 输出候选文件及行号，便于提供给
Agent 作为调查起点。`--current-repo` 会提高当前工作目录所属仓库的结果
排名，`--top` 控制返回数量。

`repo remove` 会删除全局登记和该仓库的检索记录，不会删除仓库源码或其
`.ai/` 生成文件。

## 检索能力与限制

检索使用符号、路径、API、关键词和少量中英文同义词进行本地评分。它是
代码导航的候选查找器，不是语义向量搜索，也不保证找到所有相关代码。索引
中的文件位置可能在源码更新后过期；先刷新索引，再打开源码核实结果。

API 路由识别依赖有限的静态规则；如果 `API_INDEX.md` 为空，可能表示仓库
没有受支持的路由，也可能是当前扫描规则无法识别项目使用的框架或写法。

## 仓库内源文件

发布命令分别对应仓库中的 `plugins/aiw-ai-gen-index.py` 和
`plugins/aiw-ai-code-index.py`。本目录内的可执行脚本是这两个源文件的分发
副本；更新工具时需要保持它们同步。
