package cmd

import (
	initpkg "github.com/jy-eggroll/al/init"

	"github.com/jy-eggroll/eggokit/l10n"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <shell>",
		Short: l10n.T("Generate alias declarations for a shell", nil),
		Long: l10n.T(`Output alias declarations for the given shell.

Supported shells: bash, zsh, fish, powershell

Usage:
  eval "$(al init zsh)"     # zsh
  eval "$(al init bash)"    # bash
  al init fish | source     # fish
  Invoke-Expression (&al init powershell | Out-String)  # PowerShell`, nil),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// init 的输出会被 shell 直接 eval，因此这条路径上**不得**向 stdout 输出任何
			// 说明性文字：shell 会把多余的行当成命令执行并报错，而 eval 的退出码仍是 0，
			// 用户看到的是一句莫名其妙的「command not found」。诊断信息一律走 stderr
			initpkg.Run(cfg, args[0])
			return nil
		},
	}
}

func init() {
	register(func(root *cobra.Command) { root.AddCommand(newInitCmd()) })
}
