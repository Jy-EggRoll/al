// lang.go 负责在命令行解析之前确定输出语言。
//
// 为什么必须提前于 cobra 解析：语言的生效范围包括各命令的 Short/Long 与 flag 说明，
// 而 cobra 的 --help 路径不会执行 PersistentPreRunE（那些文本要立即翻译好才能输出），
// 因此语言取值不能依赖 cobra 的解析结果，只能自行预扫描原始参数。
//
// 已知范围限制：只支持命令行与配置文件两种来源，不读取 LANG/LC_ALL 等环境变量。
package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/jy-eggroll/al/config"
	"github.com/spf13/pflag"
)

// chooseLanguage 按优先级挑出要使用的语言串（未经归一化）：
//
//	命令行 --lang/-l  >  配置文件 language 字段  >  空串（交由 l10n 用默认语言）
//
// 刻意不在这里归一化：语言白名单属于 l10n.Options，归一化在 Init 内部完成即可，
// 否则这里就得再维护一份语言列表，两份迟早对不上。
//
// 任何一步失败都静默降级、绝不返回错误：语言只影响展示，不该让命令整体失败。
// 尤其是配置文件损坏时也必须能正常输出帮助——这是 --help 路径会走到这里的前提。
func chooseLanguage() string {
	if lang := scanLangFlag(os.Args[1:]); lang != "" {
		return lang
	}
	// LoadLanguage 只读文件，不会建目录，也不会因主目录不可用而退出进程
	if lang, err := config.LoadLanguage(); err == nil && lang != "" {
		return lang
	}
	return ""
}

// scanLangFlag 从原始命令行参数里预扫描 --lang/-l 的取值。
//
// 用 pflag 而不是手写循环，原因是手写容易在两点上出错，而 pflag 已经处理妥当：
//   - --lang=en / --lang en / -l en / -len 四种等价写法都要能识别
//   - 独立的 -- 之后应当停止 flag 解析，其后的内容不能当成 flag
//
// pflag 还会对未知 flag 做剥离取值的处理，因此 `al ls --all` 里的值不会被误读成语言。
//
// 刻意忽略 Parse 返回的错误：pflag 是边解析边赋值，即使后面遇到无法识别的参数而报错，
// 之前已解析出的 --lang 值依然有效。args 不含程序名，单独作为参数是为了可测试。
func scanLangFlag(args []string) string {
	fs := pflag.NewFlagSet("lang-scan", pflag.ContinueOnError)
	// 允许出现未知 flag，否则子命令自己的 flag 会让预扫描直接失败
	fs.ParseErrorsWhitelist.UnknownFlags = true
	// 屏蔽 pflag 自身的报错输出，避免污染用户终端
	fs.SetOutput(io.Discard)

	var lang string
	fs.StringVarP(&lang, "lang", "l", "", "")
	_ = fs.Parse(args)

	return strings.TrimSpace(lang)
}
