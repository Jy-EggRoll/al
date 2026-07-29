package cmd

import (
	"github.com/jy-eggroll/al/internal/i18n"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	flagCommon    bool
	flagZsh       bool
	flagBash      bool
	flagFish      bool
	flagPowershell bool
	flagForce     bool
)

var addCmd = &cobra.Command{
	Use:   "add <name> <command>",
	Short: "Add or update an alias",
	Long: `Add a new alias or update an existing one.

The command is stored in ~/.config/al/alias.toml.

Examples:
  al add gs "git status"              # shared across all shells
  al add rmrf "rm -rf" --zsh          # zsh only
  al add rmrf "Remove-Item -Recurse -Force" --powershell  # PowerShell only
  al add o "open ." --zsh --bash      # zsh and bash only
`,
	Args: exactArgsCmd(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		command := args[1]

		shells := resolveShells()
		if len(shells) == 0 {
			return i18n.Errorf("ERR_AddAtLeastOneShell")
		}

		newCfg, err := cfg.AddAlias(name, command, shells)
		if err != nil {
			return err
		}
		cfg = newCfg

		if err := cfg.Save(ConfigPath); err != nil {
			return i18n.Errorf("ERR_SaveConfig", err)
		}

		pterm.Success.Printfln(i18n.T("CLI_AliasAdded"), name, command, formatShells(shells))
		return nil
	},
}

func resolveShells() []string {
	if flagCommon {
		return []string{"common"}
	}
	var shells []string
	if flagZsh {
		shells = append(shells, "zsh")
	}
	if flagBash {
		shells = append(shells, "bash")
	}
	if flagFish {
		shells = append(shells, "fish")
	}
	if flagPowershell {
		shells = append(shells, "powershell")
	}
	if len(shells) == 0 {
		return []string{"common"}
	}
	return shells
}

func formatShells(shells []string) string {
	if len(shells) == 1 && shells[0] == "common" {
		return i18n.T("Show_ScopeCommon")
	}
	label := i18n.T("Show_ScopeShellsPrefix")
	for i, s := range shells {
		if i > 0 {
			label += ", "
		}
		label += s
	}
	return label
}

func init() {
	addCmd.Flags().BoolVar(&flagCommon, "common", false, "share across all shells")
	addCmd.Flags().BoolVar(&flagZsh, "zsh", false, "zsh only")
	addCmd.Flags().BoolVar(&flagBash, "bash", false, "bash only")
	addCmd.Flags().BoolVar(&flagFish, "fish", false, "fish only")
	addCmd.Flags().BoolVar(&flagPowershell, "powershell", false, "powershell only")
	addCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "overwrite without confirmation")
	rootCmd.AddCommand(addCmd)
}
