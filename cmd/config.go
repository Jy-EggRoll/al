// config.go 实现 "al config" 及其子命令：读写界面偏好。
//
// 偏好与别名分处两个文件（config.toml 与 alias.toml），原因见 config/prefs.go 的说明。
// 本子树刻意遮蔽 root 的 PersistentPreRunE，因此这里**不得**使用全局 cfg。
//
// 为什么要遮蔽：root 的钩子会读 alias.toml，解析失败就返回 error。而「换界面语言」
// 与别名数据毫无关系，用户完全可能正因为在中文界面下修不好坏掉的别名文件才想改语言。
// 让它因为别名文件损坏而不可用，等于挡住了最该可用的那条路。
package cmd

import (
	"fmt"

	"github.com/jy-eggroll/al/config"
	"github.com/jy-eggroll/al/internal/locales"
	"github.com/jy-eggroll/eggokit/l10n"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// languageKey 是偏好里唯一支持按项读写的键。
const languageKey = "language"

func newConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: l10n.T("Show and edit interface preferences", nil),
		Long: l10n.T(`Show and edit al's interface preferences.

The preferences live in ~/.config/al/config.toml, separate from the aliases in
alias.toml so that adding or removing an alias never touches them.

Examples:
  al config get language          Print the language in effect
  al config set language zh-CN    Write the language preference`, nil),
		// 遮蔽 root 的 PersistentPreRunE，理由见文件头
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	c.AddCommand(newConfigGetCmd(), newConfigSetCmd())
	return c
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: l10n.T("Print one preference as it takes effect", nil),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 从这里起都是运行期，关闭 usage
			cmd.SilenceUsage = true

			if args[0] != languageKey {
				return fmt.Errorf("%s", l10n.T("unsupported setting {{.Key}}; only {{.Only}} can be read", map[string]any{
					"Key":  args[0],
					"Only": languageKey,
				}))
			}

			// 读不出来时按默认语言报告：启动路径本来就会静默降级到默认语言，
			// 因此「当前生效的语言」在最坏情况下确实是默认值，这里报默认值不算失真
			lang, err := config.LoadLanguage()
			if err != nil || lang == "" {
				lang = locales.Default
			}
			// 走 cobra 的输出目标而不是直接写 os.Stdout：脚本取用时仍落到 stdout，
			// 而测试与上层代码可以接管这个 writer
			fmt.Fprintln(cmd.OutOrStdout(), lang)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: l10n.T("Write one preference", nil),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			key, value := args[0], args[1]
			if key != languageKey {
				return fmt.Errorf("%s", l10n.T("unsupported setting {{.Key}}; only {{.Only}} can be set", map[string]any{
					"Key":  key,
					"Only": languageKey,
				}))
			}

			// 取值校验必须严格：写进去一个不受支持的语言标签，启动时只会静默回落到
			// 默认语言，用户会以为设置生效了，却怎么也看不出界面为什么没变
			if !l10n.IsSupported(value, locales.Supported()) {
				return fmt.Errorf("%s", l10n.T("unsupported language {{.Value}}; choose one of: {{.Options}}", map[string]any{
					"Value":   value,
					"Options": joinOptions(locales.Supported()),
				}))
			}

			path, err := config.PreferencesPath()
			if err != nil {
				return err
			}
			if err := config.SaveLanguage(path, value); err != nil {
				return err
			}

			// 语言在进程启动时由 l10n.Init 定死，本行必须告诉用户「现在还没生效」，
			// 否则用户会以为这一条命令就改好了当前这次输出
			pterm.Success.Println(l10n.T("{{.Key}} set to {{.Value}} in {{.Path}}; it takes effect the next time al runs", map[string]any{
				"Key":   key,
				"Value": value,
				"Path":  path,
			}))
			return nil
		},
	}
}

// joinOptions 用逗号连接可选值，供错误信息列出合法取值。
func joinOptions(values []string) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}

func init() {
	register(func(root *cobra.Command) { root.AddCommand(newConfigCmd()) })
}
