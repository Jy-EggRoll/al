package init

import (
	_ "embed"
	"fmt"

	"github.com/jy-eggroll/al/config"
)

//go:embed pwsh.ps1
var pwshPreamble string

func pwsh(cfg config.AliasConfig) {
	fmt.Println(pwshPreamble)
	for name, av := range cfg.Aliases {
		if val, ok := av.ForShell("powershell"); ok {
			fmt.Println(pwshFunc(name, val))
		}
	}
}
