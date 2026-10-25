// registry.go 负责命令树的两段式装配：先登记构造函数，等语言就绪后再真正构造。
//
// 为什么不能在包级变量里直接写命令、再在 init() 里 AddCommand：命令的 Short/Long
// 与 flag 说明要经 l10n.T 翻译，而 T 依赖语言已加载，语言又只能等到 Execute 才能确定
// （cobra 的 --help 不执行 PersistentPreRunE，那些文案进入 Execute 时就必须是最终语言）。
// 包级变量在 init 阶段就求值，那时语言还不存在，只能事后遍历命令树覆盖字段——
// 那套覆盖机制会就地改写字段、破坏幂等，还要为框架自身的代码开例外。
//
// 改为延迟构造后，T 在构造期就是一次真调用，全仓语义一致。
package cmd

import "github.com/spf13/cobra"

// constructors 收集各命令文件 init() 里登记的装配函数。
// Go 保证 init() 按文件名字典序执行，因此装配顺序固定且可复现。
var constructors []func(root *cobra.Command)

// register 由各命令文件的 init() 调用，登记「把本命令挂到根命令上」的动作。
// 登记动作而不是命令实例，正是为了把构造推迟到语言加载之后。
func register(add func(root *cobra.Command)) {
	constructors = append(constructors, add)
}

// buildRoot 构造完整的命令树。
// 必须在 l10n.Init 之后调用：各构造函数会调用 T 取当前语言的文案。
func buildRoot() *cobra.Command {
	root := newRootCmd()
	for _, add := range constructors {
		add(root)
	}
	return root
}
