package init

import (
	"testing"
)

func TestPosixQuote_normal(t *testing.T) {
	got := posixQuote("git status")
	want := "'git status'"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPosixQuote_withSingleQuote(t *testing.T) {
	got := posixQuote("it's fine")
	want := "'it'\\''s fine'"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPosixQuote_withDoubleQuotes(t *testing.T) {
	got := posixQuote(`echo "hello"`)
	want := `'echo "hello"'`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPosixQuote_withSpecialChars(t *testing.T) {
	got := posixQuote(`echo $HOME && echo ` + "`pwd`")
	want := `'echo $HOME && echo ` + "`pwd`'"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFishFunc_selfReferencing(t *testing.T) {
	got := fishFunc("fzf", "fzf --height=50% --ansi")
	want := "function fzf\n    command fzf --height=50% --ansi $argv\nend"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFishFunc_normal(t *testing.T) {
	got := fishFunc("f", "fd . --hidden | fzf --preview=\"bat {}\"")
	want := "function f\n    fd . --hidden | fzf --preview=\"bat {}\" $argv\nend"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFishFunc_simple(t *testing.T) {
	got := fishFunc("cls", "clear")
	want := "function cls\n    clear $argv\nend"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFishFunc_selfRefBinary(t *testing.T) {
	got := fishFunc("ls", "ls --color=auto")
	want := "function ls\n    command ls --color=auto $argv\nend"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPwshFunc_selfReferencing(t *testing.T) {
	got := pwshFunc("fzf", "fzf --height=50% --ansi")
	want := "function global:fzf { & (Get-Command fzf -Type Application) --height=50% --ansi @Args }"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPwshFunc_normal(t *testing.T) {
	got := pwshFunc("f", "fd . --hidden | fzf --preview=\"bat {}\"")
	want := "function global:f { fd . --hidden | fzf --preview=\"bat {}\" @Args }"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPwshFunc_simple(t *testing.T) {
	got := pwshFunc("cls", "clear")
	want := "function global:cls { clear @Args }"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
