package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jy-eggroll/al/config"

	"github.com/spf13/cobra"
)

var (
	cfg        config.AliasConfig
	ConfigPath string
	Version    string
	BuildTime  string
)

var rootCmd = &cobra.Command{
	Use:   "al",
	Short: "Cross-shell alias manager",
	Long: `al reads aliases from ~/.config/al/alias.toml and exports them
for the target shell.

Supported shells: bash, zsh, fish, powershell`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// completion 命令树整体跳过配置加载。
		//
		// 为什么必须连 completion 的子命令（`al completion bash` 等）一起跳过：
		// cobra 的 completion 是一个父命令，真正干活的是 bash/zsh/fish/powershell
		// 这些子命令，此时 cmd.Name() 是 shell 名而不是 "completion"。若只判断
		// `cmd.Name() == "completion"`，实际使用的 `al completion bash` 仍然会走
		// 配置加载——原代码的判断范围本就覆盖不到它，只是原代码解析失败时只打印
		// 一行警告然后继续，所以这个漏洞一直被掩盖着；一旦解析失败改为返回 error
		// （本文件的修复目标），`al completion bash` 就会在用户配置损坏时直接报错，
		// 补全脚本完全不可用，用户连「靠补全敲出 al rm 去删坏配置」这条自救路径都断了。
		//
		// 为什么跳过是安全的：补全脚本的内容是 cobra 的命令树元数据，
		// 与用户的别名配置没有任何数据依赖，不读配置也能完整生成。
		// shell 每次启动都会执行这段脚本，它对配置损坏的鲁棒性必须优先于报错严格性。
		if isCompletionCommand(cmd) {
			return nil
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		ConfigPath = filepath.Join(home, ".config", "al", "alias.toml")

		// 刻意不在这里 MkdirAll：这个钩子对所有命令生效，包括 ls/show/init
		// 这些纯只读命令。无条件建目录会让用户只是看一眼配置，就在家目录里
		// 留下一个空目录，而目录本该由真正写盘的路径（Save 内部的
		// atomicfile.Write）按需创建。
		//
		// 读取失败必须与「文件不存在」分开处理，这是本函数最关键的一点：
		// 配置存在但读不出来（语法错误、权限不足、路径是目录），与配置压根不存在，
		// 是两种完全不同的情况。前者说明用户已经有配置且它坏了，此时若继续用
		// 空配置往下走，随后的任何 add/rm 都会把坏配置整体覆盖成只剩新别名的那一条，
		// 用户的全部别名静默消失，且退出码还是 0——用户得不到任何警示。
		// 后者是首次使用的正常场景，用空配置继续即可。
		data, err := os.ReadFile(ConfigPath)
		if err != nil {
			if os.IsNotExist(err) {
				cfg = config.Default()
				return nil
			}
			// 读不出来时一并关掉 usage 输出：这是运行期错误（配置文件的存在与权限问题），
			// 与「命令参数用错」是两回事，后者用户确实需要看到用法。cobra 默认在 RunE
			// 返回错误后追加整段 Usage 与 Flags，对一个「你的配置文件读不了」的错误
			// 附上十几行参数说明只是噪声，真正的信息是上面那个带绝对路径的错误
			cmd.SilenceUsage = true
			return fmt.Errorf("read config file %s: %w", ConfigPath, err)
		}

		// 解析失败同样必须中止：此时 cfg 仍是 nil，任何后续写盘都会把空配置
		// 当成现有配置保存，与上面是同一个「静默清空用户数据」的后果。
		c, err := config.Parse(data)
		if err != nil {
			// 与上面同一理由：关掉 usage，让用户只看到带路径与行号的解析错误。
			// 不放在 rootCmd 的定义处统一设置，那会让参数用错时也看不到用法
			cmd.SilenceUsage = true
			return fmt.Errorf("parse config file %s: %w", ConfigPath, err)
		}
		cfg = c
		return nil
	},
}

// isCompletionCommand 判断 cmd 是否属于 cobra 的 completion 命令树。
//
// 为什么用「向上找父命令」而不是逐个列举 shell 名：
// cobra 的 completion 子命令集合（bash/zsh/fish/powershell）是 cobra 自己注册的，
// 列举 shell 名等于把 cobra 的内部结构复制一份到本项目里，cobra 日后增删子命令
// 时这里会静默失效。沿 Parent() 链向上找到根，只看沿途是否有 completion，
// 与 cobra 的实际结构解耦，也不会误伤用户自己的子命令（本项目的命令树里
// 没有叫 completion 的其他命令）。
func isCompletionCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "completion" {
			return true
		}
	}
	return false
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
