package cmd

import (
	"github.com/jy-eggroll/al/internal/i18n"
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
	Args: rangeArgsCmd(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		var command *string
		if len(args) >= 2 {
			command = &args[1]
		}

		hasShellFlag := flagEditCommon || flagEditZsh || flagEditBash || flagEditFish || flagEditPowershell

		if command == nil && !hasShellFlag {
			return i18n.Errorf("ERR_EditNothing")
		}

		if flagEditCommon && (flagEditZsh || flagEditBash || flagEditFish || flagEditPowershell) {
			return i18n.Errorf("ERR_EditCommonConflict")
		}

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
			return i18n.Errorf("ERR_SaveConfig", err)
		}

		pterm.Success.Printfln(i18n.T("CLI_AliasUpdated"), name)
		return nil
	},
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
