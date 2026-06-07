package init

import (
	_ "embed"
	"fmt"

	"github.com/jy-eggroll/al/config"
)

//go:embed bash.bash
var bashPreamble string

func bash(cfg config.AliasConfig) {
	fmt.Println(bashPreamble)
	for name, av := range cfg.Aliases {
		if val, ok := av.ForShell("bash"); ok {
			fmt.Printf("alias %s=\"%s\"\n", name, val)
		}
	}
}
