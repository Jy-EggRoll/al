# al — 跨 Shell 别名管理器

`al` 是 [alias-rs](https://github.com/nomyfan/alias-rs)（Rust 编写的跨 Shell 别名管理器）的 Go 移植版，并在此基础上做了大量增强。

## 与 alias-rs 的差异

| 变更                          | 说明                                                                                           |
| ----------------------------- | ---------------------------------------------------------------------------------------------- |
| **语言迁移**                  | 从 Rust 重写为 Go，消除 CGO 依赖，单二进制分发，交叉编译零配置                                 |
| **Cobra CLI 框架**            | 替代原始 `os.Args` 手动解析，自动生成 help/completion 子命令                                   |
| **`al ls [filter]`**          | 彩色表格列出所有别名，支持按别名名/命令值模糊过滤                                              |
| **`al show <name>`**          | 查看单个别名详情（类型、命令、适用 Shell）                                                     |
| **`al add <name> <command>`** | 添加别名到 `~/.config/al/alias.toml`，支持 `--zsh/--bash/--fish/--powershell/--all` 等 flag    |
| **`al completion <shell>`**   | Cobra 内置 Shell 补全生成，零代码维护                                                          |
| **`al rm <name>`**            | 删除别名，清理配置                                                                             |
| **自动重载**                  | `init` 输出一个 `al()` 包装函数，`add`/`rm` 成功后自动 `eval "$(al init shell)"`，别名立即可用 |
| **包结构**                    | 采用标准 Go `cmd/` 布局，模块化清晰，易于扩展                                                  |
| **彩色输出**                  | 集成 pterm，ls/show 输出带颜色和格式化表格                                                     |

## 用法

```bash
# 初始化（与原版兼容）
eval "$(al init zsh)"
eval "$(al init bash)"
al init fish | source
Invoke-Expression (&al init powershell | Out-String)

# 查询
al ls              # 列出所有别名
al ls git          # 过滤包含 "git" 的别名
al show gs         # 查看 gs 的详细信息

# 管理
al add gs "git status"                              # 全平台共享
al add rmrf "rm -rf" --zsh                          # 仅 zsh
al add rmrf "Remove-Item -Recurse -Force" --powershell # 仅 PowerShell
al add open "." --zsh --bash                        # zsh + bash

# 删除
al rm gs                                           # 删除别名 gs

# 补全
al completion zsh > /usr/local/share/zsh/site-functions/_al
```

## 自动重载

`al init` 在输出别名声明之前，会先定义一个同名的 `al()` 包装函数（Shell Function）。
当你通过这个包装函数执行 `al add` 或 `al rm` 时，它在写入配置后会**自动重新初始化**当前 Shell：

```
# 用户输入
al add gs "git status"

# 实际执行流程
al() 包装函数拦截
  └─ command al add gs "git status"   # 调用真实二进制写配置
  └─ 成功 → eval "$(command al init zsh)"  # 立即刷新当前 Shell 别名
  └─ 别名 gs 立即可用，无需手动 eval
```

这个机制参考了 `direnv`、`pyenv` 等工具的 Shell 集成模式——利用 `command` 关键字调用真实二进制避免递归，子进程退出后自动刷新父 Shell 状态。

支持的所有 Shell（zsh、bash、fish、powershell）均以对应语言的函数/脚本语法实现。

## 配置

别名存储在 `~/.config/al/alias.toml`：

```toml
[aliases]
gs = "git status"
p = "pnpm"

[aliases.rmrf]
zsh = "rm -rf"
powershell = "Remove-Item -Recurse -Force"
```

## 构建

```bash
git clone https://github.com/jy-eggroll/al
cd al
task build
```

构建所有平台（并行）：

```bash
task build-all
```

输出在 `build/` 目录下，命名格式为 `al-{GOOS}-{GOARCH}`。

支持的目标平台：

| 操作系统 | 架构                   |
| -------- | ---------------------- |
| Linux    | 386, amd64, arm, arm64 |
| macOS    | amd64, arm64           |
| Windows  | 386, amd64, arm64      |
| FreeBSD  | amd64, arm64           |

### 手动编译

```bash
go build -o al .
```

交叉编译：

```bash
GOOS=linux GOARCH=amd64   go build -o al-linux
GOOS=darwin GOARCH=amd64  go build -o al-macos-intel
GOOS=windows GOARCH=amd64 go build -o al.exe
```
