package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jy-eggroll/al/internal/i18n"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// SetupI18n 在 rootCmd.Execute() 之前由 main() 调用，完成 i18n 的“Phase 1 早期初始化”
// 以及帮助系统/命令描述的本地化装配。
//
// 之所以要在 Execute 之前做：cobra 会在 PreRun/Run 之前就处理 --help 与 --version，
// 并且在用户尚未提供 --lang 之前，系统语言也应尽早生效，因此必须先于一切调用 i18n.Init。
func SetupI18n() {
	// Phase 1：仅依据 CLI --lang 与 AL_LANG 环境变量确定语言（此时配置文件尚未读取，
	// 而 al 的配置 alias.toml 不含语言设置，故系统语言作为兜底即可）。
	earlyLang := earlyResolveLanguage()
	i18n.Init(earlyLang)

	// 接管帮助输出，使用本地化的分节标题
	rootCmd.SetHelpFunc(localizedHelpFunc)

	// 递归填充所有命令的 Short/Long 与 flag usage 的翻译
	localizeCommandDescriptions(rootCmd)
}

// earlyResolveLanguage 手动扫描 os.Args（不依赖 flag.Parse），在解析前尽早拿到
// 用户显式指定的语言。优先级：--lang 参数 > AL_LANG 环境变量。
func earlyResolveLanguage() string {
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		if after, ok := strings.CutPrefix(a, "--lang="); ok {
			return after
		}
		if after, ok := strings.CutPrefix(a, "-lang="); ok {
			return after
		}
		if (a == "--lang" || a == "-lang") && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	if envLang := os.Getenv(i18n.LANGEnv); envLang != "" {
		return envLang
	}
	return ""
}

// localizeCommandDescriptions 递归地为命令树填充翻译后的 Short/Long，并本地化 flag 说明。
//
// 关键点（见指南 16 节）：命令描述通常在 init() 里以 struct 字面量定义，此时 i18n 尚未初始化，
// 不能在 struct 中直接调用 i18n.T()。因此统一在 Execute 前的 SetupI18n 里递归填充一次。
// help 展示依赖 cmd.Short/Long，故填充一次即满足运行时需求。
func localizeCommandDescriptions(cmd *cobra.Command) {
	if s := i18n.T("CMD_" + cmd.Name() + "_Short"); s != ("CMD_" + cmd.Name() + "_Short") {
		cmd.Short = s
	}
	if s := i18n.T("CMD_" + cmd.Name() + "_Long"); s != ("CMD_" + cmd.Name() + "_Long") {
		cmd.Long = s
	}
	localizeFlagUsages(cmd)
	for _, sub := range cmd.Commands() {
		localizeCommandDescriptions(sub)
	}
}

// localizeFlagUsages 为本命令及其持久 flag 设置翻译后的 usage 文案。
// 键名采用 Flag_<命令名>_<flag名>，以区分同名 flag 在不同命令下含义不同的情况
// （如 --zsh 在 add 中是“仅 zsh”，在 ls 中是“显示 zsh 别名”）。
func localizeFlagUsages(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := "Flag_" + cmd.Name() + "_" + f.Name
		if localized := i18n.T(key); localized != key {
			f.Usage = localized
		}
	})
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		key := "Flag_" + cmd.Name() + "_" + f.Name
		if localized := i18n.T(key); localized != key {
			f.Usage = localized
		}
	})
}

// localizedHelpFunc 完全接管 cobra 的帮助输出，把各分节标题（Usage/Examples/Flags 等）
// 用翻译文本替换，其余仍借助 cobra 自身的能力（UseLine、FlagUsages 等）生成。
func localizedHelpFunc(cmd *cobra.Command, args []string) {
	out := cmd.OutOrStdout()

	// 命令描述：Long 优先，否则 Short
	if cmd.Long != "" {
		fmt.Fprintln(out, cmd.Long)
	} else if cmd.Short != "" {
		fmt.Fprintln(out, cmd.Short)
	}
	fmt.Fprintln(out)

	// Usage 行
	fmt.Fprintf(out, "%s\n", i18n.T("Help_Usage"))
	if cmd.Runnable() {
		fmt.Fprintf(out, "  %s\n", cmd.UseLine())
	}
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(out, "  %s [command]\n", cmd.CommandPath())
	}

	// 示例
	if cmd.Example != "" {
		fmt.Fprintf(out, "\n%s\n%s\n", i18n.T("Help_Examples"), cmd.Example)
	}

	// 子命令列表
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(out, "\n%s\n", i18n.T("Help_AvailableCommands"))
		for _, c := range cmd.Commands() {
			if c.IsAvailableCommand() || c.Name() == "help" {
				fmt.Fprintf(out, "  %s  %s\n", rpad(c.Name(), cmd.NamePadding()), c.Short)
			}
		}
	}

	// 本命令自身的 flag
	if cmd.HasAvailableLocalFlags() {
		fmt.Fprintf(out, "\n%s\n%s\n", i18n.T("Help_Flags"), cmd.LocalFlags().FlagUsages())
	}

	// 从父命令继承来的全局 flag
	if cmd.HasAvailableInheritedFlags() {
		fmt.Fprintf(out, "\n%s\n%s\n", i18n.T("Help_GlobalFlags"), cmd.InheritedFlags().FlagUsages())
	}

	// 进一步帮助的提示
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(out, "\n%s\n", fmt.Sprintf(i18n.T("Help_MoreInfo"), cmd.CommandPath()))
	}
}

// rpad 向右补齐空格到指定显示宽度（命令名对齐用）。命令名均为 ASCII，按字节长度即可。
func rpad(s string, padding int) string {
	if len(s) >= padding {
		return s
	}
	return s + strings.Repeat(" ", padding-len(s))
}

// 以下参数数量校验器替代 cobra 内置的 ExactArgs/RangeArgs/MaxArgs/MinArgs，
// 目的在于让“参数数量不符”这类高频报错也能被翻译成目标语言（cobra 内置版本是英文硬编码）。
// 它们仅在 Execute 阶段、Phase 2 i18n 初始化完成之后才执行，因此可安全调用 i18n.Errorf。

func exactArgsCmd(n int) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return i18n.Errorf("ERR_ArgsExact", n, len(args))
		}
		return nil
	}
}

func minArgsCmd(n int) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < n {
			return i18n.Errorf("ERR_ArgsMin", n, len(args))
		}
		return nil
	}
}

func maxArgsCmd(n int) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			return i18n.Errorf("ERR_ArgsMax", n, len(args))
		}
		return nil
	}
}

func rangeArgsCmd(min, max int) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < min || len(args) > max {
			return i18n.Errorf("ERR_ArgsRange", min, max, len(args))
		}
		return nil
	}
}
