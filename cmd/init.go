package cmd

import (
	initpkg "github.com/jy-eggroll/al/init"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <shell>",
	Short: "Generate alias declarations for a shell",
	Long: `Output alias declarations for the given shell.

Supported shells: bash, zsh, fish, powershell

Usage:
  eval "$(al init zsh)"     # zsh
  eval "$(al init bash)"    # bash
  al init fish | source     # fish
  Invoke-Expression (&al init powershell | Out-String)  # PowerShell
`,
	Args: exactArgsCmd(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		initpkg.Run(cfg, args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
