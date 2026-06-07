package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jy-eggroll/al/config"

	"github.com/spf13/cobra"
)

var cfg config.AliasConfig
var ConfigPath string

var rootCmd = &cobra.Command{
	Use:   "als",
	Short: "Cross-shell alias manager",
	Long: `als reads aliases from ~/.config/alias.toml and exports them
for the target shell.

Supported shells: bash, zsh, fish, powershell`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "completion" {
			return nil
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		ConfigPath = filepath.Join(home, ".config", "alias.toml")

		data, err := os.ReadFile(ConfigPath)
		if err != nil {
			data = []byte("[aliases]")
		}

		c, err := config.Parse(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse config file %v\n", err)
			c = config.Default()
		}
		cfg = c
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
