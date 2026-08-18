package init

import (
	"fmt"
	"strings"
)

func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func fishFunc(name, val string) string {
	words := strings.Fields(val)
	if words[0] == name {
		return fmt.Sprintf("function %s\n    command %s $argv\nend", name, val)
	}
	return fmt.Sprintf("function %s\n    %s $argv\nend", name, val)
}

// pwshFunc 生成单个 PowerShell 别名函数定义，返回内容形如：
//
//	Remove-Alias -Name fzf -Force -ErrorAction SilentlyContinue
//	function global:fzf {
//	  $__al_piped = @($input)
//	  if ($__al_piped.Count -gt 0) {
//	    $__al_piped | & (Get-Command fzf -Type Application | Select-Object -First 1) --height=50% @Args
//	  } else {
//	    & (Get-Command fzf -Type Application | Select-Object -First 1) --height=50% @Args
//	  }
//	}
//
// 设计背景（前因后果）：
//
//  1. 高级函数陷阱：函数一旦在 param 块里携带 [CmdletBinding()] 或 [Parameter()]
//     属性，就会被 PowerShell 升级为"高级函数"（advanced function），采用严格的
//     参数绑定——未声明的命名参数（如 fzf 的 --height=40%）会直接报错
//     "A parameter cannot be found..."，同时 $args 自动变量也会失效。
//     别名函数必须能原样转发任意参数（@Args），因此这里刻意保持普通函数身份。
//
//  2. 管道输入来源：普通函数天然拥有 $input 自动变量，函数体内（无 process 块时）
//     $input 枚举所有管道输入对象。但 $input 是枚举器，访问一次即被清空，
//     所以先用 @($input) 物化为数组，再通过 $__al_piped.Count 判断有无管道输入。
//     注意不能采用 [Parameter(ValueFromPipeline)] + process 块的写法——那会逐行
//     启动底层命令（每个候选启动一次 fzf 进程），交互式工具会立即失效。
//
// 3. 双分支的原因：交互式命令（如 fzf）在有/无管道时行为必须不同。
//
//   - 无管道时不能重定向 stdin，命令直接从终端（TTY/CONIN$）读取，保持交互模式；
//
//   - 有管道时把物化后的管道对象经 stdin 转交给命令链的第一个命令，
//     从而获得与 bash/zsh（alias 直接展开）和 fish（函数继承 stdin）一致的行为。
//     若统一写成 $input | <cmd>，无管道时 stdin 被管道占位并立即 EOF，
//     fzf 会因读不到候选而直接退出，交互模式被破坏。
//
//     4. 管道输入只作用于命令链的第一个命令（$__al_piped | <body>），例如别名
//     f = "fd . --hidden | fzf" 收到外部管道时，外部数据会先进入 fd，
//     这与 POSIX shell 中 "echo x | f" 的语义一致。
//
// 潜在影响点：
//   - 所有 PowerShell 别名函数均获得管道转发能力：读 stdin 的命令（fzf/cat/rg）
//     由此支持管道输入，不读 stdin 的命令不受影响（数据被其自身忽略）。
//   - cd 等 PowerShell cmdlet 收到外部管道输入时可能触发参数绑定行为，这属于
//     用户主动敲管道才会遇到的情况，与 POSIX shell 语义对齐，可接受。
//   - 函数体由单行变为多行，al init powershell 的输出随之变大，无性能影响。
func pwshFunc(name, val string) string {
	firstWord := strings.Fields(val)[0]
	body := val
	// 自引用别名（别名名 == 命令首词，如 fzf = "fzf --height=50%"）若直接内联
	// 会无限递归调用函数自身，这里把首词替换为调用同名的可执行文件来打破递归。
	// 注意必须用 `| Select-Object -First 1` 只取第一个匹配：当 PATH 中存在多个同名
	// 可执行文件（如 mise/scoop 的 shim 加真实二进制）时，Get-Command 会返回多个
	// CommandInfo，直接 & 会把整个数组 splat 成若干参数而报
	// "The term 'fzf fzf fzf' is not recognized as a name of a cmdlet"。
	if firstWord == name {
		body = strings.Replace(val, firstWord,
			fmt.Sprintf("& (Get-Command %s -Type Application | Select-Object -First 1)", firstWord), 1)
	}
	return fmt.Sprintf(`Remove-Alias -Name %s -Force -ErrorAction SilentlyContinue
function global:%s {
  $__al_piped = @($input)
  if ($__al_piped.Count -gt 0) {
    $__al_piped | %s @Args
  } else {
    %s @Args
  }
}`, name, name, body, body)
}
