package init

import (
	_ "embed"
	"fmt"

	"github.com/jy-eggroll/al/config"
)

//go:embed fish.fish
var fishPreamble string

func fish(cfg config.AliasConfig) {
	fmt.Println(fishPreamble)
	for name, av := range cfg.Aliases {
		if val, ok := av.ForShell("fish"); ok {
			fmt.Println(fishFunc(name, val))
		}
	}
}
