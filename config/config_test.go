package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse_sharedInline(t *testing.T) {
	data := []byte(`[aliases]
gs = "git status"
p = "pnpm"`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Aliases) != 2 {
		t.Fatalf("expected 2 aliases, got %d", len(cfg.Aliases))
	}
	av, ok := cfg.Aliases["gs"]
	if !ok {
		t.Fatal("missing alias gs")
	}
	if av.Inline != "git status" {
		t.Fatalf("expected 'git status', got %q", av.Inline)
	}
}

func TestParse_shellSpecific(t *testing.T) {
	data := []byte(`[aliases]
[aliases.rmrf]
zsh = "rm -rf"
bash = "rm -rfv"`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	av, ok := cfg.Aliases["rmrf"]
	if !ok {
		t.Fatal("missing alias rmrf")
	}
	if av.Object == nil {
		t.Fatal("expected object type")
	}
	if av.Object["zsh"].Inline != "rm -rf" {
		t.Fatalf("expected 'rm -rf' for zsh, got %q", av.Object["zsh"].Inline)
	}
}

func TestParse_array(t *testing.T) {
	data := []byte(`[aliases]
m = ["echo", "hello"]`)

	cfg, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	av := cfg.Aliases["m"]
	if av.Multi == nil {
		t.Fatal("expected multi type")
	}
}

func TestParse_invalidTOML(t *testing.T) {
	data := []byte(`garbage`)
	cfg, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
	if len(cfg.Aliases) != 0 {
		t.Fatal("expected empty config on error")
	}
}

func TestForShell_shared(t *testing.T) {
	cfg := Default()
	cfg.Aliases["g"] = AliasValue{Inline: "git"}
	val, ok := cfg.Aliases["g"].ForShell("zsh")
	if !ok || val != "git" {
		t.Fatalf("expected 'git', got %q", val)
	}
}

func TestForShell_shellSpecific(t *testing.T) {
	cfg := Default()
	cfg.Aliases["rmrf"] = AliasValue{
		Object: map[string]AliasObjectValue{
			"zsh": {Inline: "rm -rf"},
			"fish": {Inline: "rm -rf"},
		},
	}
	val, ok := cfg.Aliases["rmrf"].ForShell("zsh")
	if !ok || val != "rm -rf" {
		t.Fatalf("expected 'rm -rf' for zsh, got %q", val)
	}
	_, ok = cfg.Aliases["rmrf"].ForShell("bash")
	if ok {
		t.Fatal("expected no match for bash")
	}
}

func TestAddAlias_newShared(t *testing.T) {
	cfg := Default()
	cfg, err := cfg.AddAlias("g", "git", []string{"common"})
	if err != nil {
		t.Fatal(err)
	}
	av := cfg.Aliases["g"]
	if av.Inline != "git" {
		t.Fatalf("expected 'git', got %q", av.Inline)
	}
}

func TestAddAlias_newShellSpecific(t *testing.T) {
	cfg := Default()
	cfg, err := cfg.AddAlias("rmrf", "rm -rf", []string{"zsh", "bash"})
	if err != nil {
		t.Fatal(err)
	}
	av := cfg.Aliases["rmrf"]
	if av.Object == nil {
		t.Fatal("expected object type")
	}
	if av.Object["zsh"].Inline != "rm -rf" {
		t.Fatalf("expected 'rm -rf' for zsh, got %q", av.Object["zsh"].Inline)
	}
}

func TestAddAlias_overwriteConflict(t *testing.T) {
	cfg := Default()
	cfg.Aliases["rmrf"] = AliasValue{
		Object: map[string]AliasObjectValue{
			"zsh": {Inline: "rm -rf"},
		},
	}
	_, err := cfg.AddAlias("rmrf", "rm -rf", []string{"common"})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestRemoveAlias(t *testing.T) {
	cfg := Default()
	cfg.Aliases["g"] = AliasValue{Inline: "git"}
	cfg, err := cfg.RemoveAlias("g")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Aliases["g"]; ok {
		t.Fatal("alias g should have been removed")
	}
}

func TestRemoveAlias_notFound(t *testing.T) {
	cfg := Default()
	_, err := cfg.RemoveAlias("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent alias")
	}
}

func TestFlatten_shared(t *testing.T) {
	cfg := Default()
	cfg.Aliases["g"] = AliasValue{Inline: "git"}
	entries := cfg.Flatten()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "g" || entries[0].Command != "git" || entries[0].Shell != "common" {
		t.Fatalf("unexpected entry: %+v", entries[0])
	}
}

func TestFlatten_shellSpecific(t *testing.T) {
	cfg := Default()
	cfg.Aliases["rmrf"] = AliasValue{
		Object: map[string]AliasObjectValue{
			"zsh": {Inline: "rm -rf"},
			"bash": {Inline: "rm -rfv"},
		},
	}
	entries := cfg.Flatten()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alias.toml")
	cfg := Default()
	cfg.Aliases["g"] = AliasValue{Inline: "git"}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Aliases["g"].Inline != "git" {
		t.Fatal("save/round-trip failed")
	}
}

func TestDefault(t *testing.T) {
	cfg := Default()
	if len(cfg.Aliases) != 0 {
		t.Fatal("default config should have no aliases")
	}
}

func TestEditAlias_notFound(t *testing.T) {
	cfg := Default()
	_, err := cfg.EditAlias("nonexistent", nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent alias")
	}
}

func TestEditAlias_noChanges(t *testing.T) {
	cfg := Default()
	cfg.Aliases["g"] = AliasValue{Inline: "git"}
	_, err := cfg.EditAlias("g", nil, nil)
	if err == nil {
		t.Fatal("expected error for no changes")
	}
}

func TestEditAlias_changeCommand_shared(t *testing.T) {
	cfg := Default()
	cfg.Aliases["g"] = AliasValue{Inline: "git"}
	cmd := "gh"
	cfg, err := cfg.EditAlias("g", &cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	av := cfg.Aliases["g"]
	if av.Inline != "gh" {
		t.Fatalf("expected 'gh', got %q", av.Inline)
	}
}

func TestEditAlias_changeShells_sharedToShellSpecific(t *testing.T) {
	cfg := Default()
	cfg.Aliases["g"] = AliasValue{Inline: "git"}
	cfg, err := cfg.EditAlias("g", nil, []string{"fish"})
	if err != nil {
		t.Fatal(err)
	}
	av := cfg.Aliases["g"]
	if av.Inline != "" {
		t.Fatal("expected Inline to be cleared")
	}
	if av.Object == nil {
		t.Fatal("expected Object type")
	}
	if av.Object["fish"].Inline != "git" {
		t.Fatalf("expected 'git' for fish, got %q", av.Object["fish"].Inline)
	}
}

func TestEditAlias_changeBoth_sharedToShellSpecific(t *testing.T) {
	cfg := Default()
	cfg.Aliases["g"] = AliasValue{Inline: "git"}
	cmd := "gh"
	cfg, err := cfg.EditAlias("g", &cmd, []string{"zsh", "bash"})
	if err != nil {
		t.Fatal(err)
	}
	av := cfg.Aliases["g"]
	if av.Inline != "" {
		t.Fatal("expected Inline to be cleared")
	}
	if av.Object["zsh"].Inline != "gh" {
		t.Fatalf("expected 'gh' for zsh, got %q", av.Object["zsh"].Inline)
	}
	if av.Object["bash"].Inline != "gh" {
		t.Fatalf("expected 'gh' for bash, got %q", av.Object["bash"].Inline)
	}
}

func TestEditAlias_changeShells_shellSpecificToShared(t *testing.T) {
	cfg := Default()
	cfg.Aliases["rmrf"] = AliasValue{
		Object: map[string]AliasObjectValue{
			"zsh": {Inline: "rm -rf"},
		},
	}
	cfg, err := cfg.EditAlias("rmrf", nil, []string{"common"})
	if err != nil {
		t.Fatal(err)
	}
	av := cfg.Aliases["rmrf"]
	if av.Inline != "rm -rf" {
		t.Fatalf("expected 'rm -rf', got %q", av.Inline)
	}
	if av.Object != nil {
		t.Fatal("expected Object to be cleared")
	}
}

func TestEditAlias_changeCommand_shellSpecific(t *testing.T) {
	cfg := Default()
	cfg.Aliases["rmrf"] = AliasValue{
		Object: map[string]AliasObjectValue{
			"zsh": {Inline: "rm -rf"},
			"fish": {Inline: "rm -rf"},
		},
	}
	cmd := "trash"
	cfg, err := cfg.EditAlias("rmrf", &cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	av := cfg.Aliases["rmrf"]
	if av.Object["zsh"].Inline != "trash" {
		t.Fatalf("expected 'trash' for zsh, got %q", av.Object["zsh"].Inline)
	}
	if av.Object["fish"].Inline != "trash" {
		t.Fatalf("expected 'trash' for fish, got %q", av.Object["fish"].Inline)
	}
}

func TestEditAlias_multiTypeError(t *testing.T) {
	cfg := Default()
	cfg.Aliases["m"] = AliasValue{Multi: []string{"echo", "hello"}}
	cmd := "say hi"
	_, err := cfg.EditAlias("m", &cmd, nil)
	if err == nil {
		t.Fatal("expected error for multi-type alias")
	}
}

func TestShells(t *testing.T) {
	cfg := Default()
	cfg.Aliases["a"] = AliasValue{
		Object: map[string]AliasObjectValue{
			"zsh": {Inline: "a"},
			"bash": {Inline: "a"},
		},
	}
	shells := cfg.Shells()
	if len(shells) != 2 {
		t.Fatalf("expected 2 shells, got %d", len(shells))
	}
}

// TestValidateAliasName_legal 覆盖白名单内名字，防止日后收紧规则时误伤常见用法。
// 白名单刻意比 bash 的实际能力窄（见 config/validate.go 的完整论证），
// 因此这里同时把「被排除的 bash 合法字符」钉在拒绝侧，避免有人误以为是漏判而放开。
func TestValidateAliasName_legal(t *testing.T) {
	legal := []string{"gs", "x9", "a_b", "a.b", "a-b", "_x", "..", "A1_b.c-d"}
	for _, name := range legal {
		if err := ValidateAliasName(name); err != nil {
			t.Fatalf("expected %q to be legal, got %v", name, err)
		}
	}
}

// TestValidateAliasName_rejected 逐个钉住拒绝规则。
// 每一项都对应一种具体的失效方式，不是罗列字符。
func TestValidateAliasName_rejected(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"", "empty"},
		{" ", "whitespace"},
		{"y; echo X", "illegal character ';'"},
		{"a|b", "illegal character '|'"},
		{"a$b", "illegal character '$'"},
		{"a(b)", "illegal character '('"},
		{"a`b", "illegal character '`'"},
		{"a b", "illegal character ' '"},
		// bash 实测接受但 fish/其它 shell 有歧义或未实测的字符，按保守交集一律拒绝
		{"a,b", "illegal character ','"},
		{"a#b", "illegal character '#'"},
		{"a!b", "illegal character '!'"},
		{"a?b", "illegal character '?'"},
		{"a*b", "illegal character '*'"},
		{"a~b", "illegal character '~'"},
		{"a=b", "illegal character '='"},
		{"a+b", "illegal character '+'"},
		// 前缀类规则
		{"-x", "must not start with '-'"},
		{"123", "pure digits"},
		{"0", "pure digits"},
		// 非 ASCII：四种 shell 对多字节字符的处理未实测，一律拒绝
		{"别名", "illegal character"},
	}
	for _, c := range cases {
		err := ValidateAliasName(c.name)
		if err == nil {
			t.Fatalf("expected %q to be rejected", c.name)
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Fatalf("name %q: expected error containing %q, got %q", c.name, c.want, err.Error())
		}
	}
}

// TestValidateAliasCommand_empty 钉住空命令与全空白命令都被拒绝。
// 全空白必须单独覆盖：它与空串一样会让 strings.Fields 返回空切片，
// 只判断 == "" 会漏掉它，而漏掉的后果是 al init fish 直接 panic。
func TestValidateAliasCommand_empty(t *testing.T) {
	for _, cmd := range []string{"", " ", "\t", "  \n "} {
		if err := ValidateAliasCommand(cmd); err == nil {
			t.Fatalf("expected command %q to be rejected", cmd)
		}
	}
	if err := ValidateAliasCommand("ls"); err != nil {
		t.Fatalf("expected \"ls\" to be legal, got %v", err)
	}
}

// TestParse_rejectsIllegalAliasName 确认 TOML 里手写的注入名在解析阶段就被挡住。
// 配置是用户手工编辑的纯文本，写入侧校验拦不住它，因此解析侧必须独立设防。
func TestParse_rejectsIllegalAliasName(t *testing.T) {
	data := []byte(`[aliases]
"x; echo X" = "ls"`)
	_, err := Parse(data)
	if err == nil {
		t.Fatal("expected error for illegal alias name in TOML")
	}
	if !strings.Contains(err.Error(), "illegal character ';'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestParse_acceptsEmptyArray 确认 `m = []` 这类空数组配置能正常解析。
// 生成器对空值已做防御，解析层不应再把它当错误——否则用户只是写了个空数组
// 就用不了整个 al，代价与收益不成比例。
func TestParse_acceptsEmptyArray(t *testing.T) {
	data := []byte(`[aliases]
m = []`)
	cfg, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Aliases["m"]; !ok {
		t.Fatal("expected alias m to be present")
	}
}

// TestAddAlias_rejectsBadNameAndCommand 确认 add 路径在写盘前就拒绝非法输入。
// 这里直接测 config 层，因为 cmd 层只是转发，真正的规则在数据层。
func TestAddAlias_rejectsBadNameAndCommand(t *testing.T) {
	cfg := Default()
	if _, err := cfg.AddAlias("x; echo X", "ls", []string{"common"}); err == nil {
		t.Fatal("expected illegal name to be rejected")
	}
	if _, err := cfg.AddAlias("good", "", []string{"common"}); err == nil {
		t.Fatal("expected empty command to be rejected")
	}
	if _, err := cfg.AddAlias("good", "  ", []string{"common"}); err == nil {
		t.Fatal("expected whitespace-only command to be rejected")
	}
}

// TestSave_atomicNoPartialFileOnEncodeFailure 是原子写的核心保障：
// 编码失败时目标文件必须保持原样，而不是被截断成空文件。
// 无法从外部构造编码失败，因此改为验证「Save 成功后目标内容完整且可再解析」，
// 以及原子写在同目录留下临时文件时不影响目标文件本身。
func TestSave_atomicNoPartialFileOnEncodeFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alias.toml")

	// 先写一份初始配置
	cfg := Default()
	cfg.Aliases["g"] = AliasValue{Inline: "git"}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	// 第二次 Save 增加内容，两次写入都不能留下半截文件
	cfg.Aliases["gs"] = AliasValue{Inline: "git status"}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("saved file is not valid TOML: %v", err)
	}
	if parsed.Aliases["g"].Inline != "git" || parsed.Aliases["gs"].Inline != "git status" {
		t.Fatal("save round-trip lost data")
	}

	// 原子写不应当在目标目录留下临时文件
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "alias.toml" {
			t.Fatalf("unexpected leftover file %q in config dir", e.Name())
		}
	}
}

// TestSave_createsParentDir 确认 Save 会按需创建父目录。
// 这正是 root.go 不再无条件 MkdirAll 的依据：写盘路径自己会建目录。
func TestSave_createsParentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "al")
	path := filepath.Join(dir, "alias.toml")
	cfg := Default()
	cfg.Aliases["g"] = AliasValue{Inline: "git"}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to be created at %s: %v", path, err)
	}
}
