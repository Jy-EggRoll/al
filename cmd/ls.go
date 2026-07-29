package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jy-eggroll/al/config"
	"github.com/jy-eggroll/al/internal/i18n"
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

var lsCmd = &cobra.Command{
	Use:   "ls [filter]",
	Short: "List all aliases",
	Long: `List all aliases with their commands and applicable shells.

Optional filter argument filters by alias name or command value.
`,
	Args: maxArgsCmd(1),
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
			pterm.Info.Println(i18n.T("CLI_NoAliasesMatched"))
			return nil
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name < entries[j].Name
		})

		tableData := pterm.TableData{{
			i18n.T("LS_HeaderAlias"),
			i18n.T("LS_HeaderCommand"),
			i18n.T("LS_HeaderShell"),
		}}
		for _, e := range entries {
			tableData = append(tableData, []string{e.Name, e.Command, e.Shell})
		}

		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
		fmt.Printf("\n"+i18n.T("CLI_AliasCount")+"\n", len(entries))
		return nil
	},
}

func init() {
	lsCmd.Flags().BoolVar(&lsAll, "all", false, "show all shells")
	lsCmd.Flags().BoolVar(&lsZsh, "zsh", false, "show zsh aliases")
	lsCmd.Flags().BoolVar(&lsBash, "bash", false, "show bash aliases")
	lsCmd.Flags().BoolVar(&lsFish, "fish", false, "show fish aliases")
	lsCmd.Flags().BoolVar(&lsPwsh, "powershell", false, "show powershell aliases")
	rootCmd.AddCommand(lsCmd)
}
