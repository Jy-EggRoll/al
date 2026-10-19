package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd 只输出版本信息，刻意不做位置参数校验。
//
// 为什么有意不加 Args：兄弟项目 ggt 与 flk 的 version 命令同样不校验，多余参数一律静默忽略，
// 三者保持一致，用户在不同工具之间不会遇到两套口径。这是 al 命令里唯一没有位置参数校验的一个
// （add/edit/rm/show/init/ls 都有 Args），看到「只有它没校验」不要当成漏网补回来——
// 那是一次会破坏三项目一致性的行为改动，应当先与 ggt、flk 对齐再决定。
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("al version %s (built at %s)\n", Version, BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
