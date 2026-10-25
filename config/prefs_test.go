package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// prefsPath 返回临时目录下的偏好文件路径（该目录尚未创建）。
func prefsPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "al", "config.toml")
}

// TestLoadPreferences_文件不存在时返回零值 覆盖首次使用这一正常场景。
func TestLoadPreferences_文件不存在时返回零值(t *testing.T) {
	path := prefsPath(t)

	prefs, err := LoadPreferences(path)

	if err != nil {
		t.Fatalf("偏好文件不存在不应算错误: %v", err)
	}
	if prefs.Language != "" {
		t.Fatalf("期望零值，实际 %q", prefs.Language)
	}
	// 读取路径不得创建目录：只读命令不该在用户家目录里留下空目录
	if _, statErr := os.Stat(filepath.Dir(path)); !os.IsNotExist(statErr) {
		t.Fatal("读取偏好文件时创建了目录")
	}
}

// TestSaveLanguage_往返读写 覆盖写入后能读回同一条取值。
func TestSaveLanguage_往返读写(t *testing.T) {
	path := prefsPath(t)

	if err := SaveLanguage(path, "zh-CN"); err != nil {
		t.Fatalf("写入语言失败: %v", err)
	}
	prefs, err := LoadPreferences(path)
	if err != nil {
		t.Fatalf("读取偏好失败: %v", err)
	}
	if prefs.Language != "zh-CN" {
		t.Fatalf("期望读回 zh-CN，实际 %q", prefs.Language)
	}
}

// TestSaveLanguage_保留未知键 覆盖「写入 language 不丢别的键」。
//
// 为什么专门守这一点：本文件会随需求增加字段，若写入时整份重建，旧版本 al 写一次
// 就会把新版本写进去的键抹掉——alias.toml 上已经有这类问题的先例。
func TestSaveLanguage_保留未知键(t *testing.T) {
	path := prefsPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("建立目录失败: %v", err)
	}
	if err := os.WriteFile(path, []byte("future_key = \"keep me\"\n"), 0o644); err != nil {
		t.Fatalf("准备文件失败: %v", err)
	}

	if err := SaveLanguage(path, "en"); err != nil {
		t.Fatalf("写入语言失败: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取结果失败: %v", err)
	}
	if !strings.Contains(string(data), "future_key") {
		t.Fatalf("写入语言时丢掉了未知键，内容:\n%s", string(data))
	}
	if !strings.Contains(string(data), `language = "en"`) {
		t.Fatalf("语言没有写入，内容:\n%s", string(data))
	}
}

// TestSaveLanguage_坏文件不被静默覆盖 覆盖「用户手写的内容不能被悄悄丢掉」。
//
// 为什么读取侧允许降级、写入侧必须报错：读取只影响展示，降级到默认语言可以接受；
// 写入会覆盖磁盘上用户的数据，静默重建等于替他丢内容。
func TestSaveLanguage_坏文件不被静默覆盖(t *testing.T) {
	path := prefsPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("建立目录失败: %v", err)
	}
	broken := "language = [unclosed\n"
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatalf("准备文件失败: %v", err)
	}

	if err := SaveLanguage(path, "en"); err == nil {
		t.Fatal("坏掉的偏好文件应当让写入失败")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取结果失败: %v", err)
	}
	if string(data) != broken {
		t.Fatalf("坏文件被改写了，内容:\n%s", string(data))
	}
}

// TestLoadPreferences_坏文件报错 覆盖解析失败与文件不存在必须区分。
func TestLoadPreferences_坏文件报错(t *testing.T) {
	path := prefsPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("建立目录失败: %v", err)
	}
	if err := os.WriteFile(path, []byte("= broken\n"), 0o644); err != nil {
		t.Fatalf("准备文件失败: %v", err)
	}

	if _, err := LoadPreferences(path); err == nil {
		t.Fatal("格式错误的偏好文件应当返回错误")
	}
}
