package cmd

import (
	"fmt"
	"strings"

	"github.com/jy-eggroll/eggokit/l10n"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <alias-name>",
		Short: l10n.T("Show details of a specific alias", nil),
		Long:  l10n.T("Display detailed information about a single alias definition.", nil),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			av, ok := cfg.Aliases[name]
			if !ok {
				return fmt.Errorf("%s", l10n.T("alias {{.Name}} not found", map[string]any{"Name": name}))
			}

			pterm.Println(pterm.Bold.Sprint(l10n.T("Alias: {{.Name}}", map[string]any{"Name": name})))

			switch {
			case av.Inline != "":
				pterm.Println(pterm.Bold.Sprint(l10n.T("Command: {{.Command}}", map[string]any{"Command": av.Inline})))
				pterm.Println(pterm.Bold.Sprint(l10n.T("Type: {{.Type}}", map[string]any{"Type": l10n.T("Inline (shared by all shells)", nil)})))
				pterm.Println(pterm.Bold.Sprint(l10n.T("Scope: {{.Scope}}", map[string]any{"Scope": l10n.T("common (all shells)", nil)})))
			case av.Multi != nil:
				pterm.Println(pterm.Bold.Sprint(l10n.T("Command: {{.Command}}", map[string]any{"Command": strings.Join(av.Multi, " ")})))
				pterm.Println(pterm.Bold.Sprint(l10n.T("Type: {{.Type}}", map[string]any{"Type": l10n.T("Multi (array, shared by all shells)", nil)})))
				pterm.Println(pterm.Bold.Sprint(l10n.T("Scope: {{.Scope}}", map[string]any{"Scope": l10n.T("common (all shells)", nil)})))
			case av.Object != nil:
				pterm.Println(pterm.Bold.Sprint(l10n.T("Type: {{.Type}}", map[string]any{"Type": l10n.T("Object (per shell)", nil)})))
				pterm.Println(pterm.Bold.Sprint(l10n.T("Shell scope:", nil)))
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
			pterm.Println(pterm.Bold.Sprint(l10n.T("Source: {{.Path}}", map[string]any{"Path": ConfigPath})))
			return nil
		},
	}
}

func init() {
	register(func(root *cobra.Command) { root.AddCommand(newShowCmd()) })
}
