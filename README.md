# al — Cross-Shell Alias Manager

A Go port of [alias-rs](https://github.com/nomyfan/alias-rs) by nomyfan, with significant enhancements.

## Reference

This project is a **rewrite and extension** of [alias-rs](https://github.com/nomyfan/alias-rs), a Rust-based cross-shell alias manager. The original design — TOML-based alias configuration, `init` subcommand, and shell-specific alias resolution — is preserved for compatibility.

## 独创性更改

| 变更 | 说明 |
|------|------|
| **语言迁移** | 从 Rust 重写为 Go，消除 CGO 依赖，单二进制分发，交叉编译零配置 |
| **Cobra CLI 框架** | 替代原始 `os.Args` 手动解析，自动生成 help/completion 子命令 |
| **`als ls [filter]`** | 彩色表格列出所有别名，支持按别名名/命令值模糊过滤 |
| **`als show <name>`** | 查看单个别名详情（类型、命令、适用 Shell） |
| **`als add <name> <command>`** | 交互式添加别名到 `~/.config/alias.toml`，支持 `--zsh/--bash/--fish/--powershell/--all` 等 flag |
| **`als completion <shell>`** | Cobra 内置 Shell 补全生成，零代码维护 |
| **包结构** | 采用标准 Go `cmd/` 布局，模块化清晰，易于扩展 |
| **彩色输出** | 集成 pterm，ls/show 输出带颜色和格式化表格 |

## Usage

```bash
# --- Init（与原版兼容）---
eval "$(al init zsh)"
eval "$(al init bash)"
al init fish | source
Invoke-Expression (&al init powershell | Out-String)

# --- 查询 ---
al ls              # 列出所有别名
al ls git          # 过滤包含 "git" 的别名
al show gs         # 查看 gs 的详细信息

# --- 管理 ---
al add gs "git status"                              # 全平台共享
al add rmrf "rm -rf" --zsh                          # 仅 zsh
al add rmrf "Remove-Item -Recurse -Force" --powershell # 仅 PowerShell
al add open "." --zsh --bash                        # zsh + bash

# --- 补全 ---
al completion zsh > /usr/local/share/zsh/site-functions/_al
```

## Config

Aliases are stored in `~/.config/alias.toml`:

```toml
[aliases]
gs = "git status"
p = "pnpm"

[aliases.rmrf]
zsh = "rm -rf"
powershell = "Remove-Item -Recurse -Force"
```

## Build

```bash
git clone https://github.com/jy-eggroll/al
cd al
go build -o al .
```

Cross-compile:

```bash
GOOS=linux GOARCH=amd64   go build -o al-linux
GOOS=darwin GOARCH=amd64  go build -o al-macos-intel
GOOS=windows GOARCH=amd64 go build -o al.exe
```
