package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
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

	// 1) 错误前缀（cobra 硬编码的 "Error:"）本地化
	rootCmd.SetErrPrefix(i18n.T("Cobra_ErrorPrefix"))

	// 2) 接管帮助与用法输出，分节标题全部本地化
	rootCmd.SetHelpFunc(localizedHelpFunc)
	rootCmd.SetUsageFunc(localizedUsageFunc)

	// 3) 用本地化版本替换 cobra 内置的 help / completion 命令（Short/Long 均为英文硬编码）
	localizeBuiltinHelpCommand(rootCmd)
	localizeBuiltinCompletionCommand(rootCmd)

	// 4) 预置本地化的 --help / --version 选项 usage，使 cobra 不再注入英文默认文案
	localizeFrameworkFlags(rootCmd)

	// 5) 接管 flag 解析错误：pflag 内置英文错误按类型翻译成目标语言
	rootCmd.SetFlagErrorFunc(localizedFlagErrorFunc)

	// 6) 递归填充所有命令的 Short/Long 与自定义 flag usage 的翻译
	localizeCommandDescriptions(rootCmd)

	// 7) 静默 cobra 自身的错误打印与用法打印，统一由 cmd.Execute 包装后本地化输出
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
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
	renderHelp(cmd, cmd.OutOrStdout())
}

// localizedUsageFunc 接管 cobra 的“用法”输出（出错时展示），与帮助输出共用同一套本地化渲染。
// 注意：SetUsageFunc 的签名为 func(*Command) error，与 SetHelpFunc 不同，故单独包一层。
func localizedUsageFunc(cmd *cobra.Command) error {
	renderHelp(cmd, cmd.OutOrStdout())
	return nil
}

// renderHelp 是帮助与用法共用的渲染逻辑：所有分节标题均取自 i18n 翻译表。
func renderHelp(cmd *cobra.Command, out io.Writer) {
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

// ============================================================================
// 以下为 cobra 框架内置文本（help/completion 命令、--help/--version 选项 label、
// 错误前缀、pflag 解析错误、未知命令/参数错误）的本地化。
// 这些文本 cobra/pflag 默认硬编码为英文，需通过公开 API 接管或拦截翻译。
// ============================================================================

// localizeBuiltinHelpCommand 用本地化版本替换 cobra 内置的 help 命令。
// cobra 仅在 helpCommand 为 nil 时才会创建默认英文 help 命令，故通过 SetHelpCommand
// 提前注入即可阻止默认命令生成。
func localizeBuiltinHelpCommand(root *cobra.Command) {
	help := &cobra.Command{
		Use:   "help [command]",
		Short: i18n.T("CMD_help_Short"),
		Long:  i18n.T("CMD_help_Long"),
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
			var completions []cobra.Completion
			cmd, _, e := c.Root().Find(args)
			if e != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if cmd == nil {
				cmd = c.Root()
			}
			for _, sub := range cmd.Commands() {
				if sub.IsAvailableCommand() {
					if strings.HasPrefix(sub.Name(), toComplete) {
						completions = append(completions, cobra.CompletionWithDesc(sub.Name(), sub.Short))
					}
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(c *cobra.Command, args []string) {
			cmd, _, e := c.Root().Find(args)
			if cmd == nil || e != nil {
				c.Println(fmt.Sprintf(i18n.T("Cobra_UnknownHelpTopic"), args))
				_ = c.Root().Usage()
				return
			}
			cmd.Help()
		},
	}
	// 为 help 命令自身也预置本地化 --help 选项，避免 cobra 后续注入英文默认文案
	localizeFrameworkFlags(help)
	root.SetHelpCommand(help)
}

// localizeBuiltinCompletionCommand 禁用 cobra 默认 completion 命令（其 Short/Long 为英文硬编码），
// 改为自实现的本地化版本。生成逻辑直接复用 cobra 公开的 Gen*Completion 方法，覆盖 bash/zsh/fish/powershell。
func localizeBuiltinCompletionCommand(root *cobra.Command) {
	root.CompletionOptions.DisableDefaultCmd = true

	// 统一的脚本生成分发器：根据 shell 名调用对应的 cobra 公开方法。
	gen := func(shell string, w io.Writer) error {
		switch shell {
		case "bash":
			return root.GenBashCompletion(w)
		case "zsh":
			return root.GenZshCompletion(w)
		case "fish":
			return root.GenFishCompletion(w, true)
		case "powershell":
			return root.GenPowerShellCompletionWithDesc(w)
		}
		return i18n.Errorf("Cobra_CompletionUnsupportedShell", shell)
	}

	comp := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: i18n.T("CMD_completion_Short"),
		Long:  i18n.T("CMD_completion_Long"),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return i18n.Errorf("ERR_ArgsExact", 1, len(args))
			}
			switch args[0] {
			case "bash", "zsh", "fish", "powershell":
				return nil
			}
			return i18n.Errorf("Cobra_CompletionUnsupportedShell", args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return gen(args[0], cmd.OutOrStdout())
		},
	}

	// 各 shell 子命令：仅用于 help 展示，真正生成由父命令 RunE 统一分发。
	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		sh := sh
		comp.AddCommand(&cobra.Command{
			Use:   sh,
			Short: i18n.T("CMD_completion_" + sh + "_Short"),
			Long:  i18n.T("CMD_completion_" + sh + "_Long"),
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return gen(sh, cmd.OutOrStdout())
			},
		})
	}

	root.AddCommand(comp)
}

// localizeFrameworkFlags 在 cobra 自动注入 --help/--version 之前，预置本地化版选项，
// 使 InitDefaultHelpFlag/InitDefaultVersionFlag 发现已存在而跳过（其默认 usage 为英文硬编码）。
// 递归处理命令树上的每一个命令。
func localizeFrameworkFlags(cmd *cobra.Command) {
	if cmd.Flags().Lookup("help") == nil {
		usage := i18n.T("Cobra_HelpFlagUsageThis")
		if name := cmd.DisplayName(); name != "" {
			usage = fmt.Sprintf(i18n.T("Cobra_HelpFlagUsageFmt"), name)
		}
		cmd.Flags().BoolP("help", "h", false, usage)
		_ = cmd.Flags().SetAnnotation("help", cobra.FlagSetByCobraAnnotation, []string{"true"})
	}
	if cmd == rootCmd && cmd.PersistentFlags().Lookup("version") == nil {
		// --version 选项由我们自行接管输出（不走 cobra 的版本模板机制，
		// 因其模板执行时传入的数据为 nil，无法正确渲染 .Version）。
		usage := i18n.T("Cobra_VersionFlagUsageThis")
		if name := cmd.DisplayName(); name != "" {
			usage = fmt.Sprintf(i18n.T("Cobra_VersionFlagUsageFmt"), name)
		}
		cmd.PersistentFlags().BoolP("version", "v", false, usage)
		_ = cmd.PersistentFlags().SetAnnotation("version", cobra.FlagSetByCobraAnnotation, []string{"true"})
	}
	for _, sub := range cmd.Commands() {
		localizeFrameworkFlags(sub)
	}
}

// localizedFlagErrorFunc 是 SetFlagErrorFunc 的实现：pflag 解析失败时抛出的英文错误在此被翻译。
func localizedFlagErrorFunc(cmd *cobra.Command, err error) error {
	if translated := flagErrorTranslated(err); translated != nil {
		return translated
	}
	return err
}

// flagErrorTranslated 针对 pflag 的四种导出错误类型进行本地化翻译；非 flag 错误返回 nil。
func flagErrorTranslated(err error) error {
	var ne *pflag.NotExistError
	if errors.As(err, &ne) {
		s := ne.Error()
		switch {
		case strings.HasPrefix(s, "unknown shorthand flag"):
			r := ""
			if n := ne.GetSpecifiedName(); n != "" {
				r = string([]rune(n)[0])
			}
			return i18n.Errorf("Cobra_UnknownShorthandFlagFmt", r, ne.GetSpecifiedShortnames())
		case strings.HasPrefix(s, "unknown flag"):
			return i18n.Errorf("Cobra_UnknownFlagFmt", ne.GetSpecifiedName())
		default:
			return i18n.Errorf("Cobra_FlagNotExistFmt", ne.GetSpecifiedName())
		}
	}

	var vr *pflag.ValueRequiredError
	if errors.As(err, &vr) {
		return i18n.Errorf("Cobra_FlagNeedsArgFmt", vr.GetSpecifiedName())
	}

	var iv *pflag.InvalidValueError
	if errors.As(err, &iv) {
		flag := iv.GetFlag()
		flagName := "--" + flag.Name
		if flag.Shorthand != "" {
			flagName = "-" + flag.Shorthand + ", --" + flag.Name
		}
		return i18n.Errorf("Cobra_InvalidArgFmt", iv.GetValue(), flagName, errors.Unwrap(iv))
	}

	var se *pflag.InvalidSyntaxError
	if errors.As(err, &se) {
		return i18n.Errorf("Cobra_BadFlagSyntaxFmt", se.GetSpecifiedFlag())
	}

	return nil
}

// translateCobraError 把 cobra/pflag 内置的英文错误统一翻译为目标语言。
// 对我方自身的 i18n.Errorf 错误（已经过本地化）则原样返回。
func translateCobraError(err error) string {
	if err == nil {
		return ""
	}
	if translated := flagErrorTranslated(err); translated != nil {
		return translated.Error()
	}
	return translateCobraArgError(err)
}

// translateCobraArgError 处理 cobra 的参数/未知命令类英文错误（legacyArgs、OnlyValidArgs、Min/Max/Exact/RangeArgs）。
func translateCobraArgError(err error) string {
	s := err.Error()

	// 未知命令："unknown command "bogus" for "al""（可能附带 "\n\nDid you mean this?\n\t..." 建议）
	if strings.HasPrefix(s, "unknown command ") {
		return translateUnknownCommand(s)
	}

	// 无效参数（OnlyValidArgs）："invalid argument "x" for "al""（可能附带建议）
	if strings.HasPrefix(s, "invalid argument ") {
		re := regexp.MustCompile(`^invalid argument "(.+)" for "(.+)"(\n\nDid you mean this\?(\n\t.+)*)?$`)
		if m := re.FindStringSubmatch(s); m != nil {
			out := fmt.Sprintf(i18n.T("Cobra_InvalidArgForCmdFmt"), m[1], m[2], "")
			if m[4] != "" {
				out += "\n\n" + i18n.T("Cobra_DidYouMean") + m[4]
			}
			return out
		}
	}

	// 参数数量类错误（cobra 内置 Min/Max/Exact/Range）
	if m := regexp.MustCompile(`requires at least (\d+) arg\(s\), only received (\d+)`).FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		got, _ := strconv.Atoi(m[2])
		return fmt.Sprintf(i18n.T("Cobra_RequiresMinArgsFmt"), n, got)
	}
	if m := regexp.MustCompile(`accepts at most (\d+) arg\(s\), received (\d+)`).FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		got, _ := strconv.Atoi(m[2])
		return fmt.Sprintf(i18n.T("Cobra_AcceptsMaxArgsFmt"), n, got)
	}
	if m := regexp.MustCompile(`accepts between (\d+) and (\d+) arg\(s\), received (\d+)`).FindStringSubmatch(s); m != nil {
		mn, _ := strconv.Atoi(m[1])
		mx, _ := strconv.Atoi(m[2])
		got, _ := strconv.Atoi(m[3])
		return fmt.Sprintf(i18n.T("Cobra_AcceptsRangeArgsFmt"), mn, mx, got)
	}
	if m := regexp.MustCompile(`accepts (\d+) arg\(s\), received (\d+)`).FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		got, _ := strconv.Atoi(m[2])
		return fmt.Sprintf(i18n.T("Cobra_AcceptsExactArgsFmt"), n, got)
	}

	// 其余（含我方已本地化的错误）原样返回
	return s
}

// translateUnknownCommand 把 cobra 的 "unknown command" 错误本地化，并保留拼写建议（替换英文标题）。
func translateUnknownCommand(s string) string {
	re := regexp.MustCompile(`^unknown command "(.+)" for "(.+)"(\n\nDid you mean this\?(\n\t.+)*)?$`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return s
	}
	arg, cmdPath, sug := m[1], m[2], m[4]
	out := fmt.Sprintf(i18n.T("Cobra_UnknownCommandFmt"), arg, cmdPath, "")
	if sug != "" {
		out += "\n\n" + i18n.T("Cobra_DidYouMean") + sug
	}
	return out
}

// isUsageError 判断该错误是否属于“用法类”错误（flag/参数/未知命令），
// 这类错误应在输出后附带用法提示与用法详情，与 cobra 默认行为保持一致。
func isUsageError(err error) bool {
	var ne *pflag.NotExistError
	if errors.As(err, &ne) {
		return true
	}
	var vr *pflag.ValueRequiredError
	if errors.As(err, &vr) {
		return true
	}
	var iv *pflag.InvalidValueError
	if errors.As(err, &iv) {
		return true
	}
	var se *pflag.InvalidSyntaxError
	if errors.As(err, &se) {
		return true
	}
	s := err.Error()
	switch {
	case strings.HasPrefix(s, "unknown command "),
		strings.HasPrefix(s, "invalid argument "),
		strings.HasPrefix(s, "requires at least"),
		strings.HasPrefix(s, "accepts at most"),
		strings.HasPrefix(s, "accepts between"),
		strings.HasPrefix(s, "accepts "):
		return true
	}
	return false
}
