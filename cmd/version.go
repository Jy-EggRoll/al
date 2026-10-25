package cmd

import (
	"github.com/jy-eggroll/eggokit/ui"
	"github.com/spf13/cobra"
)

// versionOptions 是 version 命令的全部宿主取值。
//
// 命令本体、三行输出格式与 ver 别名都来自 eggokit/ui：这份输出原先在 al、flk、ggt 各写一份，
// 而平台标签又与升级命令共用同一份信息，各自维护迟早会让「version 显示的平台」与
// 「升级时挑资产的平台」对不上。
//
// Version 与 BuildTime 用取值函数而不是此刻的字符串，是因为它们由发布构建通过 -ldflags
// 注入到本包（见 Taskfile 的 LDFLAGS 与 release.json 的 ldflagsPackage），而命令树在
// Execute 阶段才构造，取值时机不能提前。因此注入路径必须留在包级变量上，这里只做转发。
var versionOptions = ui.VersionOptions{
	Program:   "al",
	Version:   func() string { return Version },
	BuildTime: func() string { return BuildTime },
}

// newVersionCmd 只输出版本信息，刻意不做位置参数校验。
//
// 为什么有意不加 Args：兄弟项目 ggt 与 flk 的 version 命令同样不校验，多余参数一律静默忽略，
// 三者保持一致，用户在不同工具之间不会遇到两套口径。这是 al 命令里唯一没有位置参数校验的一个
// （add/edit/rm/show/init/ls 都有 Args），看到「只有它没校验」不要当成漏网补回来——
// 那是一次会破坏三项目一致性的行为改动，应当先与 ggt、flk 对齐再决定。
//
// 输出格式的变更需要留意：原实现输出单行 `al version x (built at y)`，并刻意不翻译，
// 因为那一行常被脚本与问题单当作机器可读文本引用。改为共享实现后固定为三行，
// 且标签随界面语言变化（中文界面下是「版本 / 构建时间 / 平台」）。
// 脚本若要取出稳定可解析的版本号，做法是显式指定语言并取第一行
// （`al -l en version | head -n 1`），不要依赖用户配置里的界面语言；这一变化已记入 CHANGELOG。
func newVersionCmd() *cobra.Command {
	return ui.NewVersionCommand(versionOptions)
}

func init() {
	register(func(root *cobra.Command) { root.AddCommand(newVersionCmd()) })
}
