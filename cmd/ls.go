package cmd

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls [filter]",
	Short: "List all aliases",
	Long: `List all aliases with their commands and applicable shells.

Optional filter argument filters by alias name or command value.
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entries := cfg.Flatten()

		filter := ""
		if len(args) > 0 {
			filter = strings.ToLower(args[0])
		}

		if len(entries) == 0 {
			pterm.Info.Println("No aliases defined")
			return nil
		}

		tableData := pterm.TableData{
			{"别名", "命令", "Shell"},
		}

		for _, e := range entries {
			if filter != "" {
				lowerName := strings.ToLower(e.Name)
				lowerCmd := strings.ToLower(e.Command)
				if !strings.Contains(lowerName, filter) && !strings.Contains(lowerCmd, filter) {
					continue
				}
			}
			tableData = append(tableData, []string{e.Name, e.Command, e.Shell})
		}

		if len(tableData) == 1 {
			pterm.Info.Println("No aliases matched")
			return nil
		}

		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

		total := len(entries)
		shown := len(tableData) - 1
		if filter != "" {
			fmt.Printf("\n%d / %d aliases\n", shown, total)
		} else {
			fmt.Printf("\n%d aliases\n", total)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
