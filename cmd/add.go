package cmd

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	flagAll       bool
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
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		command := args[1]

		shells := resolveShells()
		if len(shells) == 0 {
			return fmt.Errorf("at least one shell flag required (or use --all)")
		}

		newCfg, err := cfg.AddAlias(name, command, shells)
		if err != nil {
			return err
		}
		cfg = newCfg

		if err := cfg.Save(ConfigPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		pterm.Success.Printfln("Alias %s → %s (%s)", name, command, formatShells(shells))
		return nil
	},
}

func resolveShells() []string {
	if flagAll {
		return []string{"all"}
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
		return []string{"all"}
	}
	return shells
}

func formatShells(shells []string) string {
	if len(shells) == 1 && shells[0] == "all" {
		return "all shells"
	}
	label := "shells: "
	for i, s := range shells {
		if i > 0 {
			label += ", "
		}
		label += s
	}
	return label
}

func init() {
	addCmd.Flags().BoolVar(&flagAll, "all", false, "share across all shells (default)")
	addCmd.Flags().BoolVar(&flagZsh, "zsh", false, "zsh only")
	addCmd.Flags().BoolVar(&flagBash, "bash", false, "bash only")
	addCmd.Flags().BoolVar(&flagFish, "fish", false, "fish only")
	addCmd.Flags().BoolVar(&flagPowershell, "powershell", false, "powershell only")
	addCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "overwrite without confirmation")
	rootCmd.AddCommand(addCmd)
}
