package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jy-eggroll/al/config"
	"github.com/jy-eggroll/al/internal/i18n"

	"github.com/spf13/cobra"
)

var (
	cfg        config.AliasConfig
	ConfigPath string
	Version    string
	BuildTime  string
)

var rootCmd = &cobra.Command{
	Use:   "al",
	Short: "Cross-shell alias manager",
	Long: `al reads aliases from ~/.config/al/alias.toml and exports them
for the target shell.

Supported shells: bash, zsh, fish, powershell`,
	// Run 为空时 cobra 默认展示帮助；这里显式处理根命令无子命令参数时的行为，
	// 并在其中优先拦截 --version（cobra 的 help 检查早于 PersistentPreRunE，
	// 而版本模板机制又因数据为 nil 无法使用，故在此手动接管）。
	RunE: func(cmd *cobra.Command, args []string) error {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Println(fmt.Sprintf(i18n.T("CLI_Version"), Version, BuildTime))
			return nil
		}
		return cmd.Help()
	},
	// PersistentPreRunE 在每个命令的 RunE 之前统一执行：
	// 1) Phase 2 完整初始化 i18n（CLI --lang > AL_LANG > 系统语言）；
	// 2) 读取别名配置文件，供后续命令使用。
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// ===== Phase 2：完整语言初始化 =====
		// 优先级：CLI --lang > AL_LANG 环境变量 > 系统语言（Init 内部兜底）。
		// i18n.Init 是幂等的（重新加载翻译文件而已），与 Phase 1 相同也无害。
		lang := os.Getenv(i18n.LANGEnv)
		if cliLang := cmd.Flags().Lookup("lang").Value.String(); cliLang != "" {
			lang = cliLang
		}
		i18n.Init(lang)

		// completion 命令不需要读取配置文件，直接放行
		if cmd.Name() == "completion" {
			return nil
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		ConfigPath = filepath.Join(home, ".config", "al", "alias.toml")

		if err := os.MkdirAll(filepath.Dir(ConfigPath), 0755); err != nil {
			return i18n.Errorf("ERR_CreateConfigDir", err)
		}

		data, err := os.ReadFile(ConfigPath)
		if err != nil {
			data = []byte("[aliases]")
		}

		c, err := config.Parse(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, i18n.T("ERR_ParseConfigFile")+"\n", err)
			c = config.Default()
		}
		cfg = c
		return nil
	},
}

// Execute 是程序入口：在 SetupI18n 完成框架文本本地化后，执行命令树。
// 由于已通过 SetSilenceErrors 静默了 cobra 自身的错误打印，这里统一把（可能来自
// cobra/pflag 的）英文内置错误翻译为目标语言后再输出，从而实现框架文本的彻底 i18n。
func Execute() {
	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		// 框架内置的英文错误（pflag 解析、未知命令、参数数量等）在此翻译；
		// 我方自身的 i18n.Errorf 错误已经过本地化，原样输出。
		msg := translateCobraError(err)
		fmt.Fprintln(os.Stderr, rootCmd.ErrPrefix(), msg)
		// 用法类错误：附带“运行 --help 查看用法”提示并展示本地化用法
		if isUsageError(err) {
			fmt.Fprintf(os.Stderr, i18n.T("Cobra_UsageHintFmt")+"\n", cmd.CommandPath())
			_ = cmd.Usage()
		}
		os.Exit(1)
	}
}

func init() {
	// --lang 作为全局持久参数，供所有子命令与 Phase 1/Phase 2 初始化使用。
	// 字面量用英文，SetupI18n 中的 localizeFlagUsages 会按当前语言覆盖 usage 文案。
	rootCmd.PersistentFlags().String("lang", "", "Language override (e.g., en, zh)")
}
