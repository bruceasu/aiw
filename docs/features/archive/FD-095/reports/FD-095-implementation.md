# FD-095 包列表可读输出实现报告

<!-- aiw-data: FD-095-implementation.json -->

按用户要求，将 aiw pkg list 的默认输出由完整 JSON 改为对齐表格，列为
NAME / VERSION / KIND。--installed 额外显示 REASON，旧记录缺少 reason 时仅在
显示层使用 manual，不改写记录。空列表有简短提示。

新增 list --json，同时覆盖仓库和已安装列表，保留原始完整数组、字段和顺序。
表格在副本上按 name/version 排序，不修改 JSON 数据。--all / --kind 与版本、
仓库、安装目录选择逻辑保持原样；已安装列表仍在读取仓库之前直接返回。
show/info/search/entries/skill 的输出没有改变。文档明确脚本需要加 --json。

独立设计反馈 ready，用户明确要求改变默认展示方式，JSON 兼容入口已保留。
实际执行一次 Python 内存 compile，exit 0；一次最终实现 git diff 静态检查。

```powershell
python -c "from pathlib import Path; p = Path('src/plugins/aiw-pkg/aiw-pkg.py'); compile(p.read_text(encoding='utf-8-sig'), str(p), 'exec'); print('Python compile-only: OK')"
```

未运行测试、实际 list、网络、构建或安装。已安装插件和 ZIP 均未修改。
用户沿用当前主工作区，无 Worker handoff，source_event=null，走受管状态/review 路由。
独立实现审阅待进行。