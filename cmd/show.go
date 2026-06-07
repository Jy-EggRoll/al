package cmd

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <alias-name>",
	Short: "Show details of a specific alias",
	Long:  `Display detailed information about a single alias definition.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		av, ok := cfg.Aliases[name]
		if !ok {
			return fmt.Errorf("alias %q not found", name)
		}

		pterm.Println(pterm.Bold.Sprint("别名: ") + name)

		switch {
		case av.Inline != "":
			pterm.Println(pterm.Bold.Sprint("命令: ") + av.Inline)
			pterm.Println(pterm.Bold.Sprint("类型: ") + "Inline (所有 Shell 共享)")
			pterm.Println(pterm.Bold.Sprint("适用: ") + "通用")
		case av.Multi != nil:
			pterm.Println(pterm.Bold.Sprint("命令: ") + strings.Join(av.Multi, " "))
			pterm.Println(pterm.Bold.Sprint("类型: ") + "Multi (数组，所有 Shell 共享)")
			pterm.Println(pterm.Bold.Sprint("适用: ") + "通用")
		case av.Object != nil:
			pterm.Println(pterm.Bold.Sprint("类型: ") + "Object (按 Shell 区分)")
			pterm.Println(pterm.Bold.Sprint("适用 Shell:"))
			for shell, ov := range av.Object {
				cmd := ""
				switch {
				case ov.Inline != "":
					cmd = ov.Inline
				case ov.Multi != nil:
					cmd = strings.Join(ov.Multi, " ")
				}
				pterm.Println(fmt.Sprintf("  %s: %s", pterm.Green(shell), cmd))
			}
		}

		pterm.Println(pterm.Bold.Sprint("来源: ") + "~/.config/al/alias.toml")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
