package init

import (
	_ "embed"
	"fmt"

	"github.com/jy-eggroll/al/config"
)

//go:embed zsh.zsh
var zshPreamble string

func zsh(cfg config.AliasConfig) {
	fmt.Println(zshPreamble)
	for name, av := range cfg.Aliases {
		if val, ok := av.ForShell("zsh"); ok {
			fmt.Printf("alias %s=%s\n", name, posixQuote(val))
		}
	}
}
