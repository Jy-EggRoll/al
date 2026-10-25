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
	want := `Remove-Alias -Name fzf -Force -ErrorAction SilentlyContinue
function global:fzf {
  $__al_piped = @($input)
  if ($__al_piped.Count -gt 0) {
    $__al_piped | & (Get-Command fzf -Type Application | Select-Object -First 1) --height=50% --ansi @Args
  } else {
    & (Get-Command fzf -Type Application | Select-Object -First 1) --height=50% --ansi @Args
  }
}`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPwshFunc_normal(t *testing.T) {
	got := pwshFunc("f", "fd . --hidden | fzf --preview=\"bat {}\"")
	want := `Remove-Alias -Name f -Force -ErrorAction SilentlyContinue
function global:f {
  $__al_piped = @($input)
  if ($__al_piped.Count -gt 0) {
    $__al_piped | fd . --hidden | fzf --preview="bat {}" @Args
  } else {
    fd . --hidden | fzf --preview="bat {}" @Args
  }
}`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPwshFunc_dotdotAlias(t *testing.T) {
	got := pwshFunc("..", "cd ..")
	want := `Remove-Alias -Name .. -Force -ErrorAction SilentlyContinue
function global:.. {
  $__al_piped = @($input)
  if ($__al_piped.Count -gt 0) {
    $__al_piped | cd .. @Args
  } else {
    cd .. @Args
  }
}`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPwshFunc_simple(t *testing.T) {
	got := pwshFunc("cls", "clear")
	want := `Remove-Alias -Name cls -Force -ErrorAction SilentlyContinue
function global:cls {
  $__al_piped = @($input)
  if ($__al_piped.Count -gt 0) {
    $__al_piped | clear @Args
  } else {
    clear @Args
  }
}`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// TestFishFunc_emptyValue 固定空值与全空白值不再 panic。
// 背景：strings.Fields("") 返回空切片，取 [0] 会 index out of range，
// 把 al init fish 整个流程炸掉。空值理论上被 config 层拒绝，
// 但 TOML 可手写成 m = [] 绕过写入校验，所以生成器必须自己扛住。
func TestFishFunc_emptyValue(t *testing.T) {
	if got := fishFunc("m", ""); got != "" {
		t.Fatalf("expected empty output for empty value, got %q", got)
	}
	if got := fishFunc("m", "   "); got != "" {
		t.Fatalf("expected empty output for whitespace-only value, got %q", got)
	}
}

// TestPwshFunc_emptyValue 与 TestFishFunc_emptyValue 同理，覆盖 powershell 生成器。
func TestPwshFunc_emptyValue(t *testing.T) {
	if got := pwshFunc("m", ""); got != "" {
		t.Fatalf("expected empty output for empty value, got %q", got)
	}
	if got := pwshFunc("m", "   "); got != "" {
		t.Fatalf("expected empty output for whitespace-only value, got %q", got)
	}
}
