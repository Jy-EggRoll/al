package cmd

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <name> [command]",
	Short: "Modify an existing alias",
	Long: `Modify an existing alias's command and/or shell scope.

Shell flags set the new scope (not additive). Omit them to keep the current scope.

Examples:
  al edit cdf 'cd ~' --fish                    # fish only, new command
  al edit cdf --fish                            # fish only, keep command
  al edit cdf --common                          # shared across all shells
  al edit cdf 'cd ~'                            # change command, keep current shells
`,
	// 位置参数与标志组合的校验都必须留在 Args 层，不能挪进 RunE。
	//
	// 为什么：cobra 的顺序是「解析标志 → ValidateArgs → PersistentPreRunE → RunE」，
	// 而 root.go 的 PersistentPreRunE 一进入就打开 SilenceUsage。校验放在 Args 里，
	// 报错发生在 SilenceUsage 打开之前，cobra 会照默认行为打印整段用法；放进 RunE
	// 就会和配置读取、Save 失败等运行期错误一起被静默，用户看不到该怎么改。
	// 标志变量在这一层已经可用：cobra 在调用 ValidateArgs 之前就完成了标志解析。
	//
	// 这里的两个条件都是「用法用错」而非「运行失败」：只给别名等于没说要改什么，
	// --common 与 shell 专属标志则是互斥组合，两者都属于用户需要看用法说明才能改对的情况。
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(1, 2)(cmd, args); err != nil {
			return err
		}
		if len(args) < 2 && !anyEditShellFlag() {
			return fmt.Errorf("specify a new command or shell flags (nothing to change)")
		}
		return commonShellConflict(flagEditCommon, anyEditShellSpecificFlag())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		var command *string
		if len(args) >= 2 {
			command = &args[1]
		}

		// 走到这里说明标志组合已经由 Args 校验通过，只需判断「是否要重建作用域」
		hasShellFlag := anyEditShellFlag()

		var shells []string
		if hasShellFlag {
			if flagEditCommon {
				shells = []string{"common"}
			} else {
				if flagEditZsh {
					shells = append(shells, "zsh")
				}
				if flagEditBash {
					shells = append(shells, "bash")
				}
				if flagEditFish {
					shells = append(shells, "fish")
				}
				if flagEditPowershell {
					shells = append(shells, "powershell")
				}
			}
		}

		newCfg, err := cfg.EditAlias(name, command, shells)
		if err != nil {
			return err
		}
		cfg = newCfg

		if err := cfg.Save(ConfigPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		pterm.Success.Printfln("Alias %s updated", name)
		return nil
	},
}

// anyEditShellFlag 报告 edit 是否收到了任意一个 shell 作用域标志（含 --common）。
// Args 与 RunE 都要用：前者判断标志组合是否成立，后者决定是否重建作用域，
// 抽成一个函数是为了让「哪些标志算作用域标志」只有一处定义，新增 shell 时不会漏改。
func anyEditShellFlag() bool {
	return flagEditCommon || anyEditShellSpecificFlag()
}

// anyEditShellSpecificFlag 报告 edit 是否收到了「某个具体 shell」的标志，不含 --common。
// 互斥校验只关心这一侧：--common 与具体 shell 同时出现才是用法错误。
func anyEditShellSpecificFlag() bool {
	return flagEditZsh || flagEditBash || flagEditFish || flagEditPowershell
}

var (
	flagEditCommon     bool
	flagEditZsh        bool
	flagEditBash       bool
	flagEditFish       bool
	flagEditPowershell bool
)

func init() {
	editCmd.Flags().BoolVar(&flagEditCommon, "common", false, "share across all shells")
	editCmd.Flags().BoolVar(&flagEditZsh, "zsh", false, "zsh only")
	editCmd.Flags().BoolVar(&flagEditBash, "bash", false, "bash only")
	editCmd.Flags().BoolVar(&flagEditFish, "fish", false, "fish only")
	editCmd.Flags().BoolVar(&flagEditPowershell, "powershell", false, "powershell only")
	rootCmd.AddCommand(editCmd)
}
