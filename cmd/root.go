package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jy-eggroll/al/config"

	"github.com/spf13/cobra"
)

var (
	cfg       config.AliasConfig
	ConfigPath string
	Version   string
	BuildTime string
)

var rootCmd = &cobra.Command{
	Use:   "al",
	Short: "Cross-shell alias manager",
	Long: `al reads aliases from ~/.config/al/alias.toml and exports them
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
		ConfigPath = filepath.Join(home, ".config", "al", "alias.toml")

		if err := os.MkdirAll(filepath.Dir(ConfigPath), 0755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}

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
