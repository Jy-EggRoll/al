package cmd

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove an alias",
	Long:  `Remove an alias from the configuration file.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		newCfg, err := cfg.RemoveAlias(name)
		if err != nil {
			return err
		}
		cfg = newCfg

		if err := cfg.Save(ConfigPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		pterm.Success.Printfln("Alias %s removed", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
