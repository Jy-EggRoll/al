package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/jy-eggroll/al/config"
	"github.com/jy-eggroll/al/internal/locales"
	"github.com/jy-eggroll/eggokit/l10n"
	"github.com/jy-eggroll/eggokit/updater"

	"github.com/spf13/cobra"
)

var (
	cfg        config.AliasConfig
	ConfigPath string
	Version    string
	BuildTime  string
)

// newRootCmd 构造根命令并注册全局持久参数。
//
// 命令树在语言加载之后才构造（见 registry.go），因此这里出现的 T() 都是真实翻译调用。
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "al",
		Short: l10n.T("Cross-shell alias manager", nil),
		Long: l10n.T(`al reads aliases from ~/.config/al/alias.toml and exports them
for the target shell.

Supported shells: bash, zsh, fish, powershell`, nil),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// 进入运行阶段后一律不再打印 usage，把用法说明留给真正的参数错误。
			//
			// 为什么放在这里就足够了：cobra 执行一条命令的顺序是「解析标志 → 校验参数
			// （Args）→ PersistentPreRunE → PreRunE → RunE」，参数与标志用错时，
			// 报错发生在进入本钩子之前，此时 SilenceUsage 还是 false，cobra 照默认行为
			// 打印整段用法——这正是用户需要看到的。而配置读取失败、解析失败、Save 失败
			// 以及任何命令 RunE 返回的错误都发生在本钩子之后，一律只报错误本身。
			//
			// 由此得出一条必须遵守的约束：凡是「用户用法有错」的检查都要写在命令的
			// Args 字段里，不能写进 RunE。写进 RunE 就等于挂在本钩子之后，会被这里
			// 一并静默——edit 原先把「既没给新命令也没给 shell 标志」与「--common 与
			// shell 专属标志互斥」两条检查放在 RunE 中，就是这样丢掉 usage 的。
			// 判断标准是「用户是否需要看用法才能改对」：需要，就放 Args 这一层。
			//
			// 为什么集中在这里而不是在各个错误分支上分别设置：原先的写法是在每个已知的
			// 失败点各写一次 SilenceUsage，配置读取与解析都写了，add/rm 的 Save 失败却漏了，
			// 「同一类运行期错误两种待遇」的根源就在这——按失败点打补丁的做法必然随着
			// 新增失败分支而再次漏掉。这里只有一处，且对本包全部子命令自动生效。
			cmd.SilenceUsage = true

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

			path, err := config.AliasPath()
			if err != nil {
				return err
			}
			ConfigPath = path

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
				// 读不出来时只报这个带绝对路径的错误，不再追加 usage：
				// 这是运行期错误（配置文件的存在与权限问题），usage 由本函数开头统一关闭
				return fmt.Errorf("%s %s: %w", l10n.T("read config file", nil), ConfigPath, err)
			}

			// 解析失败同样必须中止：此时 cfg 仍是 nil，任何后续写盘都会把空配置
			// 当成现有配置保存，与上面是同一个「静默清空用户数据」的后果。
			c, err := config.Parse(data)
			if err != nil {
				// 与上面同一理由：让用户只看到带路径与行号的解析错误，usage 已在函数开头关闭
				return fmt.Errorf("%s %s: %w", l10n.T("parse config file", nil), ConfigPath, err)
			}
			cfg = c
			return nil
		},
	}

	// --lang 的声明唯一目的是让 cobra 认可这个参数，否则命令行里出现 --lang
	// 会被判为 unknown flag。真正生效的取值由 chooseLanguage 预扫描 os.Args 得到
	// （语言必须早于 cobra 解析才能确定），所以这里刻意不绑定变量，
	// 避免出现两个互相矛盾的取值来源。
	root.PersistentFlags().StringP("lang", "l", "",
		l10n.T("Output language (e.g. en, zh-CN); defaults to the language config value", nil))

	return root
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

// commonShellConflict 检查「--common 与 shell 专属标志同时出现」这一用法错误，add 与 edit 共用。
//
// 为什么两个命令共用一份判断：互斥规则本是一件同一的事，两处各写一遍，日后新增 shell
// 或放宽规则时必然只改一处，正是本项目此前 add 静默丢弃 --zsh、edit 却报错的成因。
//
// 为什么不用 cobra 的 MarkFlagsMutuallyExclusive：它的校验点在 cobra v1.10.2 的
// command.go 第 1010 行 ValidateFlagGroups，位于 PersistentPreRunE 与 RunE 之间，
// 而 root.go 的 PersistentPreRunE 一进入就打开了 SilenceUsage，互斥报错将不会附上用法说明。
// 换句话说它不满足本项目「用法错误必须打印 usage」的约束，只能在 Args 层手写这份判断。
func commonShellConflict(common bool, specific bool) error {
	if common && specific {
		return errors.New(l10n.T("--common cannot be combined with shell-specific flags", nil))
	}
	return nil
}

// Execute 是程序的入口，由 main.go 调用。
//
// 以下顺序是有意为之，不能调换：
//  1. 先清理上一次升级留下的文件。它与语言、命令树都无关，也不该被后面的提前返回跳过
//  2. 再确定并加载语言。cobra 的 --help 路径不会执行 PersistentPreRunE，而命令描述
//     在构造时就要用到，所以语言解析必须走在最前面
//  3. 接着构造命令树。各命令的构造函数会调用 T()，此时语言已就绪
//  4. 最后才交给 cobra 解析参数并执行
func Execute() {
	// 为什么清理只能等到「下一次运行」：升级成功的那一刻，旧版本正是当前进程自己的映像，
	// 正在运行的进程删不掉自己的映像，只能由下一次运行（旧进程已退出）来收拾。
	//
	// 失败一律忽略，也不打印任何东西：清理受阻（文件仍被占用、安装目录只读）不该影响这次
	// 正常使用，用户也不需要看到一条与自己无关的启动提示；清不掉的文件会在下次运行再试。
	if execPath, err := os.Executable(); err == nil {
		updater.CleanupLeftovers(execPath)
	}

	// 语言文件是 //go:embed 进 locales 包的，加载失败属于构建期错误，必须显式暴露。
	// 静默降级只会表现为「界面语言不对」，没有任何报错，极难排查
	if err := l10n.Init(chooseLanguage(), locales.Options()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	root := buildRoot()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
