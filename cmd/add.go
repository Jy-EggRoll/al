package cmd

import (
	"fmt"
	"strings"

	"github.com/jy-eggroll/eggokit/l10n"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	flagCommon     bool
	flagZsh        bool
	flagBash       bool
	flagFish       bool
	flagPowershell bool
)

func newAddCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add <name> <command>",
		Short: l10n.T("Add or update an alias", nil),
		Long: l10n.T(`Add a new alias or update an existing one.

The command is stored in ~/.config/al/alias.toml.

Shell flags are optional: omit all of them and the alias goes to the
"common" group, which every shell shares. Examples:
  al add gs "git status"              # no shell flag → common, all shells
  al add rmrf "rm -rf" --zsh          # zsh only
  al add rmrf "Remove-Item -Recurse -Force" --powershell  # PowerShell only
  al add o "open ." --zsh --bash      # zsh and bash only`, nil),
		// 位置参数与标志组合都在 Args 层校验：互斥检查放在这里，`al add x y --common --zsh`
		// 才会明确报错，而不是像此前那样静默把 --zsh 丢掉改写成通用别名。
		// 为什么必须在 Args 而不是 RunE，见 root.go 的 commonShellConflict。
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(2)(cmd, args); err != nil {
				return err
			}
			return commonShellConflict(flagCommon, flagZsh || flagBash || flagFish || flagPowershell)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			command := args[1]

			// 不在这里检查 shells 是否为空：resolveShells 保证至少返回 ["common"]，
			// 空切片分支永远不会执行，留着只会让读者以为「必须指定 shell」
			shells := resolveShells()

			newCfg, err := cfg.AddAlias(name, command, shells)
			if err != nil {
				return err
			}
			cfg = newCfg

			if err := cfg.Save(ConfigPath); err != nil {
				return fmt.Errorf("%s: %w", l10n.T("save config", nil), err)
			}

			pterm.Success.Println(l10n.T("Alias {{.Name}} → {{.Command}} ({{.Scope}})", map[string]any{
				"Name":    name,
				"Command": command,
				"Scope":   shellsLabel(shells),
			}))
			return nil
		},
	}

	c.Flags().BoolVar(&flagCommon, "common", false, l10n.T("share across all shells", nil))
	c.Flags().BoolVar(&flagZsh, "zsh", false, l10n.T("zsh only", nil))
	c.Flags().BoolVar(&flagBash, "bash", false, l10n.T("bash only", nil))
	c.Flags().BoolVar(&flagFish, "fish", false, l10n.T("fish only", nil))
	c.Flags().BoolVar(&flagPowershell, "powershell", false, l10n.T("powershell only", nil))
	return c
}

// resolveShells 把 add 的 shell 标志解析成别名所属的 shell 列表。
//
// 一个 alias 都不指定（且未给 --common）时返回 ["common"]，也就是「省略即通用」，
// 这是 add 的默认语义。正因为这里已经处理了空列表，调用方不必也不应再检查空列表：
// 那个检查既不可达，又会让读代码的人误以为 shell 标志是必填的。
func resolveShells() []string {
	if flagCommon {
		return []string{"common"}
	}
	var shells []string
	if flagZsh {
		shells = append(shells, "zsh")
	}
	if flagBash {
		shells = append(shells, "bash")
	}
	if flagFish {
		shells = append(shells, "fish")
	}
	if flagPowershell {
		shells = append(shells, "powershell")
	}
	if len(shells) == 0 {
		return []string{"common"}
	}
	return shells
}

// shellsLabel 把 shell 列表渲染成给用户看的作用域说明。
//
// common 单独处理：它表示「所有 shell 共享」，直接列名字会让人以为只对某个叫 common 的
// shell 生效。其余情况只是名字列表，其中的 shell 名不翻译（它们是 shell 自己的名字）。
func shellsLabel(shells []string) string {
	if len(shells) == 1 && shells[0] == "common" {
		return l10n.T("common (all shells)", nil)
	}
	return l10n.T("shells: {{.Shells}}", map[string]any{"Shells": strings.Join(shells, ", ")})
}

func init() {
	register(func(root *cobra.Command) { root.AddCommand(newAddCmd()) })
}
