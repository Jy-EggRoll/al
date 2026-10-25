package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jy-eggroll/al/config"
	"github.com/jy-eggroll/eggokit/l10n"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	lsAll  bool
	lsZsh  bool
	lsBash bool
	lsFish bool
	lsPwsh bool
)

func targetShells() []string {
	if lsAll {
		return []string{"common", "zsh", "bash", "fish", "powershell"}
	}

	var explicit []string
	if lsZsh {
		explicit = append(explicit, "zsh")
	}
	if lsBash {
		explicit = append(explicit, "bash")
	}
	if lsFish {
		explicit = append(explicit, "fish")
	}
	if lsPwsh {
		explicit = append(explicit, "powershell")
	}

	if len(explicit) > 0 {
		return append([]string{"common"}, explicit...)
	}

	shells := []string{"common"}
	if cur := os.Getenv("ALIAS_SHELL"); cur != "" {
		shells = append(shells, cur)
	}
	return shells
}

func newLsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "ls [filter]",
		Short: l10n.T("List all aliases", nil),
		Long: l10n.T(`List all aliases with their commands and applicable shells.

Optional filter argument filters by alias name or command value.`, nil),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entries := cfg.Flatten()

			filter := ""
			if len(args) > 0 {
				filter = strings.ToLower(args[0])
			}
			if filter != "" {
				var filtered []config.AliasEntry
				for _, e := range entries {
					lowerName := strings.ToLower(e.Name)
					lowerCmd := strings.ToLower(e.Command)
					if strings.Contains(lowerName, filter) || strings.Contains(lowerCmd, filter) {
						filtered = append(filtered, e)
					}
				}
				entries = filtered
			}

			targets := targetShells()
			targetSet := make(map[string]bool, len(targets))
			for _, s := range targets {
				targetSet[s] = true
			}

			var filtered []config.AliasEntry
			for _, e := range entries {
				if targetSet[e.Shell] {
					filtered = append(filtered, e)
				}
			}
			entries = filtered

			if len(entries) == 0 {
				pterm.Info.Println(l10n.T("No aliases matched", nil))
				return nil
			}

			sort.Slice(entries, func(i, j int) bool {
				return entries[i].Name < entries[j].Name
			})

			tableData := pterm.TableData{{
				l10n.T("Alias", nil),
				l10n.T("Command", nil),
				l10n.T("Shell", nil),
			}}
			for _, e := range entries {
				tableData = append(tableData, []string{e.Name, e.Command, e.Shell})
			}

			pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
			fmt.Printf("\n%s\n", l10n.T("{{.Count}} aliases", map[string]any{"Count": len(entries)}))
			return nil
		},
	}

	// flag 说明同样经 T() 翻译，因此注册必须发生在构造期（语言已就绪）而不是包级 init
	c.Flags().BoolVar(&lsAll, "all", false, l10n.T("show all shells", nil))
	c.Flags().BoolVar(&lsZsh, "zsh", false, l10n.T("show zsh aliases", nil))
	c.Flags().BoolVar(&lsBash, "bash", false, l10n.T("show bash aliases", nil))
	c.Flags().BoolVar(&lsFish, "fish", false, l10n.T("show fish aliases", nil))
	c.Flags().BoolVar(&lsPwsh, "powershell", false, l10n.T("show powershell aliases", nil))
	return c
}

func init() {
	register(func(root *cobra.Command) { root.AddCommand(newLsCmd()) })
}
