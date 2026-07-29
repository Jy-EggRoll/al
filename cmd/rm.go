package cmd

import (
	"github.com/jy-eggroll/al/internal/i18n"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove an alias",
	Long:  `Remove an alias from the configuration file.`,
	Args:  exactArgsCmd(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		newCfg, err := cfg.RemoveAlias(name)
		if err != nil {
			return err
		}
		cfg = newCfg

		if err := cfg.Save(ConfigPath); err != nil {
			return i18n.Errorf("ERR_SaveConfig", err)
		}

		pterm.Success.Printfln(i18n.T("CLI_AliasRemoved"), name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
