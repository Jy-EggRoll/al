# al — 跨 Shell 别名管理器

深夜，你 SSH 上了一台服务器，手指肌肉记忆敲下 `gs`。

```
gs: command not found
```

你愣了一下，才想起这台机器没配过别名。你在 Mac 上已经形成肌肉记忆的 `gs`、`p`、`rmrf` 全都躺在本地的 `.zshrc` 里。服务器用 bash，Windows 本用 PowerShell——每换一个环境，别名就断一次。

`al` 就是为解决这个而来的。

---

## 一个故事

### 发现

你装好 `al`，跑了一行：

```bash
go install github.com/jy-eggroll/al@latest
```

然后按文档初始化：

```bash
eval "$(al init zsh)"
```

似乎什么也没发生。先不管了，你加加两个新的别名：

```bash
al add gs "git status"
al add gtcl "git clone --depth 1 --no-single-branch --recurse-submodules --shallow-submodules"
```

你敲 `gs`，它立即生效了。再敲 `gtcl`，那串怎么都记不全的参数从此变成一个单词。不是因为你 source 了文件，不是因为你开了新终端——就是直接、立刻、可以用了。

这才只是个开始。你接着加了最常用的那个：

```bash
al add f 'fd . ~ --hidden --no-ignore --color always | fzf --preview="bat {}"'
```

这种带管道、带参数的工具链才是你日常高频敲的东西——现在一个字母解决了。

你留意到 `al` 先写入了配置文件，在效果上似乎随即又调了一遍 `al init zsh`，重新声明了所有别名。整个过程发生在同一个终端会话里，你甚至没察觉到。这是 `al` 的一个小设计：`init` 会输出一个同名的 `al()` 包装函数，拦截 `add` 和 `rm`，操作成功后自动刷新当前 Shell。你什么都不用做。

### 尝试

你 SSH 上服务器，装好 `al`（`~/.config/al/alias.toml` 是从 dotfiles 仓库里 git pull 下来的），执行：

```bash
eval "$(al init bash)"
```

刚才加的 `gs`、`gtcl`、`f` 赫然可用。同一套配置，跨 Shell 直接同步——因为 `al` 读的是同一个 `alias.toml` 文件，输出什么语法的别名声明，由 `init` 参数的 shell 名决定。

有些别名是 shell 特有的。比如你的 fish 上有个 `c`，可以在全目录里模糊搜索跳转：

```bash
al add c 'cd "$(fd . ~ --type directory --hidden --no-ignore --color always | fzf)"' --fish
```

这个命令在 bash 或 zsh 里也用不上——语法不兼容。但没关系，打上 `--fish`，它只管 fish。

但你也有踩坑的时候。某天你加了这么一条：

```bash
al add cdf 'cd "$(fd . ~ --type directory --hidden --no-ignore --color always | fzf)"'
```

忘记加 `--fish`，它成了通用别名。在 fish 上好好的，切到 zsh 一敲 `cdf` 就报错——命令解析器不认识。要修正就得：

```bash
al rm cdf
al add cdf '...' --fish  # 又得把那串长参数打一遍
```

太麻烦。所以你用 `al edit`：

```bash
al edit cdf --fish
```

不改命令，只把归属从 common 切到 fish。一步到位。再也不用删了重加了。

`--common` 的别名（`gs`、`gtcl`、`f`、`fzf`……）全局通行，shell 特定的别名各归各位。`al ls` 不加 flag 时只看"通用 + 当前环境"，不啰嗦。想看全部就 `--all`。

### 打理

别名多了就需要管理。你想看 `gs` 到底绑了什么命令：

```bash
al show gs
```

想删除：

```bash
al rm gs
```

删除后自动刷新，`gs` 即刻消失。想找跟 git 相关的所有别名：

```bash
al ls git
```

### 放进仓库

`~/.config/al/alias.toml` 是一个普通文件：

```toml
[aliases]
f = "fd . ~ --hidden --no-ignore --color always | fzf --preview=\"bat {}\""
ff = "fastfetch"
fzf = "fzf --height=50% --border=rounded --layout=reverse --ansi"
gs = "git status"
gtcl = "git clone --depth 1 --no-single-branch --recurse-submodules --shallow-submodules"
hm = "cd ~"
tsss = "tailscale serve status"

[aliases.c]
fish = "cd \"$(fd . ~ --type directory --hidden --no-ignore --color always | fzf)\""

[aliases.ls]
zsh = "eza --git --icons=auto --all"
bash = "eza --git --icons=auto --all"
fish = "eza --git --icons=auto --all"

[aliases.ll]
zsh = "eza --git --long --icons=auto --all"
bash = "eza --git --long --icons=auto --all"
fish = "eza --git --long --icons=auto --all"
```

你可以把它放进 dotfiles 仓库。新机器上只要装好 `al`，把文件往这一放，`eval "$(al init zsh)"` 完事。

最后生成一下补全：

```bash
al completion zsh > /usr/local/share/zsh/site-functions/_al
```

从此你再也没有 `command not found` 过。

---

## 常用命令

```bash
# 初始化
eval "$(al init zsh)"
eval "$(al init bash)"
al init fish | source
Invoke-Expression (&al init powershell | Out-String)

# 查询
al ls                    # 通用 + 当前 Shell
al ls --all              # 全部 Shell
al ls --fish             # 通用 + fish
al ls clone              # 按别名名或命令过滤
al show gtcl             # 查看单个别名详情

# 添加（默认 --common，所有 Shell 共享）
al add gs "git status"
al add gtcl "git clone --depth 1 --no-single-branch --recurse-submodules --shallow-submodules"
al add f "fd . ~ --hidden --no-ignore --color always | fzf --preview=\"bat {}\""
al add c 'cd "$(fd . ~ --type d --hidden | fzf)"' --fish
al add ls "eza --git --icons=auto --all" --zsh --bash --fish

# 删除
al rm gs

# 补全
al completion zsh > /usr/local/share/zsh/site-functions/_al
```

## 自动重载

`al init` 在输出别名声明之前，会先定义一个同名的 `al()` 包装函数。
当你通过这个包装函数执行 `al add` 或 `al rm` 时，它在写入配置后自动重新初始化当前 Shell：

```
al add gs "git status"
  └─ command al add gs "git status"    # 调用真实二进制写入配置
  └─ 写入成功 → eval "$(command al init zsh)"  # 立即刷新当前 Shell
  └─ 别名 gs 立即可用，无需手动 source
```

这个机制参考了 `direnv`、`pyenv` 等工具的 Shell 集成模式——利用 `command` 关键字调用真实二进制避免递归，子进程退出后自动刷新父 Shell 状态。支持所有四种 Shell（zsh、bash、fish、powershell），各自以对应语言的函数语法实现。PowerShell 中使用 `function global:` 确保别名函数落入全局作用域。

PowerShell 别名函数会读取管道输入并转发给命令链的第一个命令：例如配置了 `fzf = "fzf --height=50%"` 之后，`Get-ChildItem | fzf`、`rg pattern | fzf` 都能像 bash/zsh 一样工作；没有管道输入时命令照常以交互模式运行。这弥补了 PowerShell 函数默认丢弃管道数据的差异（bash/zsh 的 alias 直接展开、fish 函数继承 stdin，天然支持管道，唯独 PowerShell 需要包装函数显式处理 `$input`）。

## 配置

文件位置：`~/.config/al/alias.toml`

支持三种格式：

```toml
[aliases]

# 字符串 — 所有 Shell 共享
gs = "git status"
hm = "cd ~"
f = "fd . ~ --hidden --no-ignore --color always | fzf --preview=\"bat {}\""

# 数组 — 所有 Shell 共享，多条命令
m = ["echo", "hello"]

[aliases.c]
# 对象 — 按 Shell 区分
fish = "cd \"$(fd . ~ --type directory --hidden --no-ignore --color always | fzf)\""

[aliases.ls]
zsh = "eza --git --icons=auto --all"
bash = "eza --git --icons=auto --all"
fish = "eza --git --icons=auto --all"
```

## 构建

```bash
git clone https://github.com/jy-eggroll/al
cd al
task build          # 当前平台
task build-all      # 所有平台（并行）
```

输出在 `build/` 目录：

| 操作系统 | 架构 |
|----------|------|
| Linux | 386, amd64, arm, arm64 |
| macOS | amd64, arm64 |
| Windows | 386, amd64, arm64 |
| FreeBSD | amd64, arm64 |

### 手动编译 / 交叉编译

```bash
go build -o al .
GOOS=linux GOARCH=amd64   go build -o al-linux
GOOS=darwin GOARCH=amd64  go build -o al-macos-intel
GOOS=windows GOARCH=amd64 go build -o al.exe
```
