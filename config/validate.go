package config

import (
	"fmt"
	"strings"
)

// aliasNameAllowedRunes 是别名名的字符白名单。
//
// 为什么用白名单而不是「只列黑名单」：黑名单的思路是「先记住哪些字符会坏事，其余放行」，
// 这套思路在本项目行不通——别名名最终会被四种 Shell 各自解析一遍，每种 Shell 的非法字符集
// 并不重合，甚至互相矛盾（fish 允许 `;`、`&` 出现在函数名里，bash 直接拒绝）。
// 只要漏掉一个字符，就等于给用户开了一条注入通道，而漏掉的成本由用户的数据承担。
// 白名单相反：默认拒绝，只有被逐个论证过安全的字符才放行，日后新增字符也必须先经过论证。
//
// 白名单的收敛依据（为什么取的是四种 Shell 的保守交集，而不是照搬 bash 的集合）：
//
//   - bash 5.1.16 实测接受 `. - _ + ~ @ : , = ^ % # ? ! * [ ] { }`，但这是 bash 一家的结论。
//     `al` 的同一个别名要交给 bash/zsh/fish/powershell 四家解析，任何一家解析不了，
//     用户敲下别名时看到的就是「命令不存在」或更糟的「执行了不该执行的东西」。
//     因此凡是只在部分 Shell 里成立的字符，一律不进白名单——宁可让用户少一个花哨的别名名，
//     也不要让同一份配置在 A 机器能用、B 机器炸掉。
//   - `,` 被明确排除：fish 官方文档中 `,` 与 `;` 同为命令分隔符（两者语义等价），
//     与 bash 的 `;` 地位相同，因此 `,` 在 fish 里属于「有歧义」而非「安全」。
//   - `#` 被排除：四种 Shell 都把它当注释起始，别名名里出现它会把这行截断。
//   - `!` 被排除：交互式 bash 会做历史扩展，即使 `alias '!x'=ls` 能建成功，
//     用户敲 `!x` 时也可能被历史扩展抢先解释。
//   - `?` `*` `[` `]` `{` `}` 被排除：glob 与 brace 扩展字符。bash 允许它们出现在别名名中，
//     但它们在 zsh 里的行为更强（zsh 有大量 glob 限定符与扩展语法），
//     属于「能建别名但语义不确定」的灰区。
//   - `~` 被排除：tilde 扩展字符，`~x` 在多数 Shell 里会被当成家目录展开。
//   - `=` `+` `^` `%` `@` `:` 被排除：这些字符单独看没有强歧义，但收益极低
//     （没有用户会去取 `a+b` 这样的别名名），收紧白名单比逐个论证更稳妥。
//
// 不确定项（必须如实记录，不能伪装成已核实的事实）：
//
//   - zsh 本机未安装，无法实测。上面关于 zsh 的判断是依据其公开文档语法的推断，不是实测结论。
//     正因如此，这里选择了比 bash 更窄的集合：宁可错杀，不可放过。
//     后续若在装有 zsh 的机器上逐字符实测，可以按实测结果放宽白名单，
//     届时应当同步补充一条测试，把「放宽的那个字符」钉住，防止回归。
//   - 非 ASCII 字符（含中文）一律拒绝：bash 与 fish 对 UTF-8 别名名是宽容的，
//     但四种 Shell 的交互补全、历史展开、终端渲染对多字节字符的处理并不一致，
//     同样属于未实测区间。若用户确实需要中文别名名，需要先在四种 Shell 里实测后再放开。
//
// 潜在影响点：
//   - 白名单比 bash 的实际能力窄，历史上用 `a+b` 之类名字建过别名的用户，
//     在 `al add`/`al edit` 时会收到报错；但 `al ls`/`al show`/`al init` 只读路径不受影响，
//     已存在的旧别名仍可被 `al rm` 清理，不会出现「配置坏了却删不掉」的死锁。
const aliasNameAllowedRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-"

// ValidateAliasName 校验别名是否可以作为 Shell 别名/函数名安全使用。
//
// 为什么这道校验必须放在 config 包（数据层）而不是 cmd 层：
// 别名名是数据的属性，不是某个命令的属性。放在数据层，`al add`、`al edit` 以及日后任何
// 新增的写入入口都会自动获得同一套规则；放在 cmd 层则每加一个入口就要重抄一遍，
// 漏抄的那个入口就是注入通道。
//
// 拒绝规则及其理由：
//   - 空名：空字符串无法生成任何有意义的 Shell 声明。
//   - 首尾空白：空白是 Shell 的词法边界，出现在名字里会让生成的代码被拆成多段。
//   - `-` 开头：命令行解析器会把 `-x` 当成选项而不是位置参数，用户根本传不进来，
//     生成的 Shell 代码里也会被解释成选项。
//   - 纯数字：纯数字名在所有 Shell 里都会被优先解释成数字字面量/退出码，语义歧义极大。
//   - 白名单之外的字符：见 aliasNameAllowedRunes 上方关于「为什么取交集」的完整论证。
func ValidateAliasName(name string) error {
	if name == "" {
		return fmt.Errorf("alias name is empty: a name must contain at least one character")
	}
	if strings.TrimSpace(name) != name {
		return fmt.Errorf("alias name %q must not start or end with whitespace: whitespace is a shell word boundary and would split the generated alias into several commands", name)
	}
	if name[0] == '-' {
		return fmt.Errorf("alias name %q must not start with '-': a leading dash is parsed as a command-line option, not as a positional argument", name)
	}
	if isAllDigits(name) {
		return fmt.Errorf("alias name %q must not be pure digits: every shell reads a bare number as a numeric literal or an exit code, never as a name", name)
	}
	for _, r := range name {
		if !strings.ContainsRune(aliasNameAllowedRunes, r) {
			return fmt.Errorf("alias name %q contains illegal character %q: allowed characters are ASCII letters, digits, '_', '.' and '-' (not as the first character); the allowed set is the conservative intersection of bash, zsh, fish and powershell, because each shell rejects or reinterprets a different set of characters", name, r)
		}
	}
	return nil
}

// ValidateAliasCommand 拒绝空命令与全空白命令。
//
// 为什么空命令必须在这里就被挡住，而不是留到生成 Shell 代码时再处理：
// `al add m ""` 会写出一条 `m = ""`，随后 `al init fish` 的 fishFunc 会执行
// `strings.Fields("")[0]` 而直接 panic（index out of range [0] with length 0），
// 把整个 init 流程炸掉。配置是用户数据的唯一存放处，一条空命令换来一次 panic
// 是不成比例的代价，因此在写入侧就应当拒绝。
//
// 为什么 TrimSpace 而非只判断 `== ""`：全空白（如 `" "`）同样会被 strings.Fields 切成
// 空切片，与空串一样触发 panic，必须一并拒绝。
func ValidateAliasCommand(command string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("alias command is empty or whitespace-only: an empty command produces a fish/powershell function with no body and crashes the generators, so it is rejected at write time")
	}
	return nil
}

// isAllDigits 判断字符串是否全部由 ASCII 数字组成。
// 空字符串返回 false——空名已由 ValidateAliasName 单独拒绝，不在此重复判断。
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
