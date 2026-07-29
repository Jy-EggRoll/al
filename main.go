package main

import "github.com/jy-eggroll/al/cmd"

func main() {
	// 在 Execute 之前完成 i18n 的早期初始化与帮助系统本地化装配
	cmd.SetupI18n()
	cmd.Execute()
}
