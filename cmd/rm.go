package cmd

import (
	"fmt"

	"github.com/jy-eggroll/eggokit/l10n"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: l10n.T("Remove an alias", nil),
		Long:  l10n.T("Remove an alias from the configuration file.", nil),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			newCfg, err := cfg.RemoveAlias(name)
			if err != nil {
				return err
			}
			cfg = newCfg

			if err := cfg.Save(ConfigPath); err != nil {
				return fmt.Errorf("%s: %w", l10n.T("save config", nil), err)
			}

			pterm.Success.Println(l10n.T("Alias {{.Name}} removed", map[string]any{"Name": name}))
			return nil
		},
	}
}

func init() {
	register(func(root *cobra.Command) { root.AddCommand(newRmCmd()) })
}
