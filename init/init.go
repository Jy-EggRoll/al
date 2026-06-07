package init

import (
	"fmt"
	"os"

	"github.com/jy-eggroll/al/config"
)

func Run(cfg config.AliasConfig, args ...string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: al init <shell>")
		os.Exit(1)
	}

	shell := args[0]
	switch shell {
	case "bash":
		bash(cfg)
	case "zsh":
		zsh(cfg)
	case "fish":
		fish(cfg)
	case "powershell":
		pwsh(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s\n", shell)
		os.Exit(1)
	}
}
