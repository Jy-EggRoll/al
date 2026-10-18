package init

import (
	"fmt"
	"strings"
)

func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// fishFunc 生成单个 fish 函数定义。
//
// 空值防御：strings.Fields("") 返回长度 0 的切片，直接取 words[0] 会 panic
// （index out of range [0] with length 0），把 al init fish 整个流程炸掉。
// 空值理论上已被 config 层的 ValidateAliasCommand 挡住，但配置文件是用户手工编辑的
// 纯文本，TOML 里写 `m = []` 或 `m = ""` 都能绕过写入侧校验直接进入这里，
// 因此生成器必须自己也能扛住空值——这是双保险，不是重复劳动。
// 返回值不含任何 fish 语句，调用方打印出来就是空行，无害。
func fishFunc(name, val string) string {
	words := strings.Fields(val)
	if len(words) == 0 {
		return ""
	}
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
//
// 空值防御：与 fishFunc 同理，strings.Fields 对空串返回空切片，取 [0] 会 panic。
// config 层已拒绝空命令，但 TOML 可被手工写成空数组/空串而绕过写入校验，
// 所以这里也做一层防御，返回空串让该别名在输出中静默跳过。
func pwshFunc(name, val string) string {
	firstWord := ""
	if words := strings.Fields(val); len(words) > 0 {
		firstWord = words[0]
	}
	body := val
	// 自引用别名（别名名 == 命令首词，如 fzf = "fzf --height=50%"）若直接内联
	// 会无限递归调用函数自身，这里把首词替换为调用同名的可执行文件来打破递归。
	// 注意必须用 `| Select-Object -First 1` 只取第一个匹配：当 PATH 中存在多个同名
	// 可执行文件（如 mise/scoop 的 shim 加真实二进制）时，Get-Command 会返回多个
	// CommandInfo，直接 & 会把整个数组 splat 成若干参数而报
	// "The term 'fzf fzf fzf' is not recognized as a name of a cmdlet"。
	//
	// firstWord 为空串时不可能等于 name（name 已被 ValidateAliasName 拒绝为空），
	// 因此空值不会走到这个替换分支，直接生成空函数体也无害——调用方打印空行。
	if firstWord == name && firstWord != "" {
		body = strings.Replace(val, firstWord,
			fmt.Sprintf("& (Get-Command %s -Type Application | Select-Object -First 1)", firstWord), 1)
	}
	if strings.TrimSpace(body) == "" {
		return ""
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
