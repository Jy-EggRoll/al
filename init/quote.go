package init

import (
	"fmt"
	"strings"
)

func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func fishFunc(name, val string) string {
	words := strings.Fields(val)
	if words[0] == name {
		return fmt.Sprintf("function %s\n    command %s $argv\nend", name, val)
	}
	return fmt.Sprintf("function %s\n    %s $argv\nend", name, val)
}

func pwshFunc(name, val string) string {
	firstWord := strings.Fields(val)[0]
	body := val
	if firstWord == name {
		body = strings.Replace(val, firstWord,
			fmt.Sprintf("& (Get-Command %s -Type Application)", firstWord), 1)
	}
	return fmt.Sprintf("Remove-Alias -Name %s -Force -ErrorAction SilentlyContinue\nfunction global:%s { %s @Args }", name, name, body)
}
