# al — 跨 Shell 别名管理器

`al` 是一个跨 Shell 的别名管理器。它用同一份配置文件 `~/.config/al/alias.toml` 管理别名，并根据目标 Shell 生成对应的别名声明，支持 bash、zsh、fish 和 PowerShell。一份配置，在多台机器、多个 Shell 之间保持一致。

## 为什么用 al

换一台服务器，或者切换到不同的 Shell，已经用惯的别名往往会失效。`git status` 的缩写 `gs`、带一长串参数的 `git clone`、常用的 `grep | fzf` 管道，在每台新机器上都需要重新配置。如果别名分散在 `.zshrc`、`.bashrc`、`config.fish` 里各自维护一份，很快就会出现不一致。

`al` 把这些别名统一放到一份 `alias.toml` 中，再按 Shell 生成适用的别名声明：

```bash
go install github.com/jy-eggroll/al@latest
eval "$(al init zsh)"          # 初始化当前 zsh
al add gs "git status"         # 添加别名
```

添加后 `gs` 立即生效，不需要手动 `source` 配置文件，也不需要开启新终端。这是靠 `init` 输出的一段 `al()` 包装函数实现的：它拦截 `add`、`rm`、`edit`，在写盘成功后自动重新初始化当前 Shell（详见「自动重载」）。

同一份 `alias.toml` 可以在多台机器之间同步。把它放进 dotfiles 仓库，新机器装好 `al` 后执行一次初始化，全部别名即可用：

```bash
eval "$(al init bash)"
```

部分别名依赖特定的 Shell 语法。例如 fish 中常用的目录模糊搜索跳转写法 `cd "$(fd . ~ | fzf)"`，在 bash 或 zsh 中不兼容。这类别名可以在添加时指定作用域（`--fish`、`--zsh` 等），只对指定的 Shell 生效。

如果别名添加时忘了指定作用域，成了通用别名，不兼容的 Shell 下会出现执行报错。此时不需要删除重加，直接修改归属即可：

```bash
al edit cdf --fish
```

这条命令只调整别名 `cdf` 的作用域，命令内容保持不变。

## 安装

```bash
go install github.com/jy-eggroll/al@latest
```

## 初始化

每种 Shell 对应一条初始化命令，加到对应 Shell 的启动配置中（`.zshrc`、`.bashrc`、`config.fish`，或 PowerShell 的 `$PROFILE`）：

```bash
eval "$(al init zsh)"                                 # zsh
eval "$(al init bash)"                                # bash
al init fish | source                                 # fish
Invoke-Expression (&al init powershell | Out-String)  # PowerShell
```

初始化命令把配置文件中适用于当前 Shell 的别名声明输出到当前 Shell，同时定义一个 `al()` 包装函数（见「自动重载」）。

## 常用命令

```bash
# 添加别名（默认 --common，所有 Shell 共享）
al add gs "git status"
al add gtcl "git clone --depth 1 --no-single-branch --recurse-submodules --shallow-submodules"
al add f "fd . ~ --hidden --no-ignore --color always | fzf --preview=\"bat {}\""
al add c 'cd "$(fd . ~ --type d --hidden | fzf)"' --fish
al add ls "eza --git --icons=auto --all" --zsh --bash --fish

# 查询
al ls                    # 通用别名 + 当前 Shell 的别名
al ls --all              # 全部 Shell 的别名
al ls --fish             # 通用别名 + fish 的别名
al ls clone              # 按别名名或命令内容过滤
al show gtcl             # 查看单个别名的详情

# 修改
al edit cdf 'cd ~'       # 修改命令，保持当前作用域
al edit cdf --fish       # 修改作用域，保持命令内容
al edit cdf 'cd ~' --fish
al edit cdf --common     # 改为所有 Shell 共享

# 删除
al rm gs

# 生成补全
al completion zsh > /usr/local/share/zsh/site-functions/_al
```

### 作用域参数

`al add` 默认把别名标记为通用（所有 Shell 共享），也可以指定一个或多个目标 Shell：

| 参数 | 作用 |
|------|------|
| `--common` | 所有 Shell 共享（默认） |
| `--zsh` / `--bash` / `--fish` / `--powershell` | 只对指定 Shell 生效，可组合 |

`al edit` 同样接受这些参数来调整别名作用域。

### 别名作用域

- 通用别名：所有 Shell 都能使用，如 `gs`、`gtcl`、`fzf`。
- Shell 特定别名：只对指定 Shell 生效。`al ls` 不传参数时只展示"通用 + 当前 Shell"的别名，需要查看全部时加 `--all`。

## 自动重载

`al init` 在输出别名声明之前，会先在当前 Shell 中定义一个同名的 `al()` 包装函数。之后通过这个包装函数执行 `al add` 或 `al rm` 时，会先调用真实的 `al` 二进制写入配置，写盘成功后再自动重新初始化当前 Shell：

```
al add gs "git status"
  └─ command al add gs "git status"                   # 调用真实二进制写入配置
  └─ 写入成功 → eval "$(command al init zsh)"         # 立即刷新当前 Shell
  └─ 别名 gs 立即可用，无需手动 source
```

实现上参考了 `direnv`、`pyenv` 等工具的 Shell 集成模式：用 `command` 关键字调用真实二进制，避免包装函数陷入递归；子进程退出后由父 Shell 完成刷新。四种 Shell 分别用各自的函数语法实现，PowerShell 中使用 `function global:` 确保别名函数落入全局作用域。

### PowerShell 的管道输入

PowerShell 的别名函数会读取管道输入并转发给命令链的第一个命令。配置了 `fzf = "fzf --height=50%"` 之后，`Get-ChildItem | fzf`、`rg pattern | fzf` 等用法与 bash、zsh 一致；没有管道输入时，命令照常以交互模式运行。这弥补了 PowerShell 函数默认丢弃管道数据的差异——bash、zsh 的 alias 直接展开、fish 函数继承 stdin，都天然支持管道，唯独 PowerShell 需要包装函数显式处理 `$input`。

## 配置文件

文件位置：`~/.config/al/alias.toml`，支持三种写法：

```toml
[aliases]

# 字符串 — 所有 Shell 共享
gs = "git status"
hm = "cd ~"
f = "fd . ~ --hidden --no-ignore --color always | fzf --preview=\"bat {}\""

# 数组 — 所有 Shell 共享，拼接为一条命令
m = ["echo", "hello"]

[aliases.c]
# 对象 — 按 Shell 区分
fish = "cd \"$(fd . ~ --type directory --hidden --no-ignore --color always | fzf)\""

[aliases.ls]
zsh = "eza --git --icons=auto --all"
bash = "eza --git --icons=auto --all"
fish = "eza --git --icons=auto --all"
```

`al add`、`al edit`、`al rm` 会改写这个文件，手动编辑别名本身同样有效。但需要注意：`al` 写回时会用内存中的配置重新生成整个文件，**你在文件里手写的注释与空行不会保留**。因此建议把注释写在别处（例如 dotfiles 仓库的 README），文件本身只放别名。

把它放进 dotfiles 仓库后，新机器 clone 下来即可同步全部别名。

## 构建

构建与发版通过 Taskfile 驱动，平台清单集中在一份 `release.json` 里：

```bash
git clone https://github.com/jy-eggroll/al
cd al
task                # 先校验再构建全部平台
task build          # 当前平台
task build-all      # 所有平台（由 eggokit 的 buildall 并行构建）
```

产物输出在 `build/` 目录，命名规则为 `al-<系统>-<架构>`，Windows 产物带 `.exe`。支持哪些平台由 `release.json` 的 `platforms` 决定，增删平台只改这份清单，无需改动 Taskfile。

```bash
task verify           # 依赖整洁、格式、静态检查、单测、竞态检测
task changelog VERSION=1.0.0   # 按 CHANGELOG.md 生成发布说明并输出到 stdout
```

发布说明的平台下载表格与小节内容分别来自 `release.json` 与 `CHANGELOG.md`，因此发版前要在 `CHANGELOG.md` 里补上同名版本小节，否则该版本的更新内容会为空。

### 手动编译 / 交叉编译

```bash
go build -o al .
GOOS=linux GOARCH=amd64   go build -o al-linux
GOOS=darwin GOARCH=amd64  go build -o al-macos-intel
GOOS=windows GOARCH=amd64 go build -o al.exe
```
