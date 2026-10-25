package init

import (
	"fmt"
	"os"

	"github.com/jy-eggroll/al/config"
	"github.com/jy-eggroll/eggokit/l10n"
)

func Run(cfg config.AliasConfig, args ...string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, l10n.T("Usage: al init <shell>", nil))
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
		fmt.Fprintln(os.Stderr, l10n.T("Unsupported shell {{.Shell}}: choose one of bash, zsh, fish, powershell", map[string]any{"Shell": shell}))
		os.Exit(1)
	}
}
