package cmd

import (
	"fmt"
	"strings"

	"github.com/jy-eggroll/al/internal/i18n"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <alias-name>",
	Short: "Show details of a specific alias",
	Long:  `Display detailed information about a single alias definition.`,
	Args:  exactArgsCmd(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		av, ok := cfg.Aliases[name]
		if !ok {
			return i18n.Errorf("ERR_AliasNotFound", name)
		}

		pterm.Println(pterm.Bold.Sprint(i18n.T("Show_AliasLabel")) + name)

		switch {
		case av.Inline != "":
			pterm.Println(pterm.Bold.Sprint(i18n.T("Show_CommandLabel")) + av.Inline)
			pterm.Println(pterm.Bold.Sprint(i18n.T("Show_TypeLabel")) + i18n.T("Show_TypeInline"))
			pterm.Println(pterm.Bold.Sprint(i18n.T("Show_ApplicableLabel")) + i18n.T("Show_ScopeCommon"))
		case av.Multi != nil:
			pterm.Println(pterm.Bold.Sprint(i18n.T("Show_CommandLabel")) + strings.Join(av.Multi, " "))
			pterm.Println(pterm.Bold.Sprint(i18n.T("Show_TypeLabel")) + i18n.T("Show_TypeMulti"))
			pterm.Println(pterm.Bold.Sprint(i18n.T("Show_ApplicableLabel")) + i18n.T("Show_ScopeCommon"))
		case av.Object != nil:
			pterm.Println(pterm.Bold.Sprint(i18n.T("Show_TypeLabel")) + i18n.T("Show_TypeObject"))
			pterm.Println(pterm.Bold.Sprint(i18n.T("Show_ApplicableShellsLabel")))
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

		pterm.Println(pterm.Bold.Sprint(i18n.T("Show_SourceLabel")) + "~/.config/al/alias.toml")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
