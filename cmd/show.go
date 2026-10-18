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

		// 路径来自 ConfigPath 而非字面量：真实路径由 root.go 的 PersistentPreRunE
		// 按 os.UserHomeDir() 拼出，写死成 "~/.config/al/alias.toml" 会在
		// HOME 被改动（容器、CI、测试隔离）时显示一个用户机器上并不存在的路径，
		// 排查问题时把用户引向错误的方向。
		pterm.Println(pterm.Bold.Sprint("来源: ") + ConfigPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
