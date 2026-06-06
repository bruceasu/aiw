# `cz` 独立程序说明

`cz` 已不再作为 `aiw` 主程序内建子命令实现，而是改为独立 Go 程序，通过 plugin 方式接入。

## 运行方式

```bash
aiw cz [options]
```

主程序会通过 plugin fallback 调用：

- `plugins/aiw-cz/aiw-cz.py`
- Windows 下执行同目录 `cz.exe`
- Linux/macOS 下执行同目录 `cz`

## 独立源码位置

```text
program/cz/
```

其中包含：

- 独立 `go.mod`
- `main.go`
- 复制后的 `internal/cz`
- `internal/envx`
- `internal/fsx`

## 构建

Windows:

```powershell
./program/cz/build.ps1
```

Linux/macOS:

```bash
sh ./program/cz/build.sh
```

构建产物输出到：

```text
plugins/aiw-cz/
```

## 当前支持的帮助入口

```bash
aiw cz -h
python plugins/aiw-cz/aiw-cz.py -h
```
