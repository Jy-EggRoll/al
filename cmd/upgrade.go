// upgrade.go 把 `al upgrade` 接到 eggokit/upgradecli 的共享实现上。
//
// 命令行为（读 flag、选通道、组织提示语、决定确认时机、退出方式）全在共享实现里，
// 本文件只提供 al 的取值：程序名、发布目标（见 upgrade_target.go）、版本与构建时间来源、
// 输出界面，以及本命令自己的 --yes。
package cmd

import (
	"github.com/jy-eggroll/eggokit/l10n"
	"github.com/jy-eggroll/eggokit/ui"
	"github.com/jy-eggroll/eggokit/updater"
	"github.com/jy-eggroll/eggokit/upgradecli"
	"github.com/spf13/cobra"
)

// newUpgradeCmd 构造 `al upgrade` 命令。
//
// 命令树在语言加载之后才构造（见 registry.go），因此共享实现内部的 T() 都是真实翻译调用。
//
// --yes 是本命令的局部参数，al 没有全局的非交互总开关：在管道、CI 或 stdin 接 /dev/null 时
// 确认无法进行，必须靠它显式放行。共享实现刻意不注册同名 flag（会与宿主的同名定义冲突），
// 因此这里注册，再由 AssumeYes 闭包把取值转交过去。
func newUpgradeCmd() *cobra.Command {
	var assumeYes bool

	c := upgradecli.NewCommand(upgradecli.Options{
		Program:   "al",
		Target:    alTarget,
		Version:   func() string { return Version },
		BuildTime: func() string { return BuildTime },
		// 升级过程的状态、进度与确认一律写 stderr：stdout 留给业务数据，
		// 与 al 其它命令一致；用命令的流而不是 os.Stderr，测试才能替换输出目标。
		// 界面实现取 eggokit/ui，与 version 命令共用同一套平台标签，两边显示的名字不会分叉
		Reporter: func(cmd *cobra.Command) updater.Reporter {
			return ui.NewReporter(cmd.ErrOrStderr())
		},
		AssumeYes: func() bool { return assumeYes },
	})

	c.Flags().BoolVar(&assumeYes, "yes", false, l10n.T("Skip the confirmation prompt (required in non-interactive environments)", nil))

	return c
}

func init() {
	register(func(root *cobra.Command) { root.AddCommand(newUpgradeCmd()) })
}
