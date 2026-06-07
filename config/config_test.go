package config

import (
	"os"
	"path/filepath"
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
