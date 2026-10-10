# AI Code Tools

`generate-ai-index` 为仓库生成静态代码索引；`ai-code-index` 汇总并搜索多个仓库的索引。工具实现的唯一权威来源是本目录中的两个脚本。它们不需要第三方 Python 依赖。

## 在 AIW 中使用

从仓库根目录运行：

```powershell
aiw ai-gen-index --root .
aiw ai-code-index repo add .
aiw ai-code-index update
aiw ai-code-index context "订单创建接口" --current-repo
```

索引生成后，源码变化时再次运行 `aiw ai-gen-index --root .` 和 `aiw ai-code-index update`。也可以执行 `aiw ai-code-index update --run-generator`，先运行与工具一起部署的生成器，再刷新已登记仓库。

运行 `python build.py plugins` 或 `python build.py all` 会把本目录的两个脚本复制到 `dist/plugins/`，改名为 `aiw-ai-gen-index.py`、`aiw-ai-code-index.py` 后直接安装到 AIW 的 `plugins/` 根目录（默认 `C:\green\aiw\plugins\`）。源码树中的 `src/plugins/` 入口用于开发时转交给本目录实现；部署时会由权威脚本生成同名插件文件。不会安装独立的 `ai-code-tools` 程序目录，也不会创建全局命令。

## `generate-ai-index`

扫描 Java、Go、Python、JavaScript 和 TypeScript 源文件，默认在仓库 `.ai/` 下生成：

```text
PROJECT_MAP.md
API_INDEX.md
symbols.jsonl
apis.jsonl
files.jsonl
metadata.json
```

可以指定仓库和相对输出目录：

```powershell
aiw ai-gen-index --root D:/work/my-repo --out .ai
```

生成器执行静态扫描，不执行仓库代码，也不访问网络。

## `ai-code-index`

在用户目录维护已登记的仓库清单和 SQLite 检索库：

```text
~/.ai-code-index/repos.json
~/.ai-code-index/db.sqlite
```

常用命令：

```powershell
aiw ai-code-index repo add D:/work/my-repo
aiw ai-code-index repo list
aiw ai-code-index update --run-generator
aiw ai-code-index search "订单创建接口" --top 10 --current-repo
aiw ai-code-index context "订单创建接口" --top 8 --current-repo
aiw ai-code-index stats
```

`context` 输出候选文件及行号，便于作为 Agent 调查起点。`--current-repo` 会提高当前工作目录所属仓库的结果排名，`--top` 控制返回数量。检索基于符号、路径、API、关键词和少量中英文同义词评分，不是语义向量搜索；索引可能在源码变化后过期，应先刷新再核实结果。

`repo remove` 会删除全局登记和该仓库的检索记录，不会删除仓库源码或 `.ai/` 文件。API 路由识别依赖有限的静态规则，空的 API 索引不一定表示仓库没有路由。
