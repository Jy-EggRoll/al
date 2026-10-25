package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jy-eggroll/al/config"
)

// TestScanLangFlag_四种等价写法 覆盖语言参数的预扫描。
//
// 为什么要单独测它：语言必须在 cobra 解析之前确定，靠的是对原始参数的预扫描。
// 若预扫描只认 --lang=xx 这一种写法，用户常用的 `-l zh-CN` 与 `-lzh-CN` 就会静默失效，
// 表现是「参数没报错，界面语言却不变」，很难联想到预扫描。
func TestScanLangFlag_四种等价写法(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--lang", "zh-CN"}, "zh-CN"},
		{[]string{"--lang=zh-CN"}, "zh-CN"},
		{[]string{"-l", "zh-CN"}, "zh-CN"},
		{[]string{"-lzh-CN"}, "zh-CN"},
		{[]string{"ls", "--all"}, ""},
		{[]string{"--", "--lang", "zh-CN"}, ""},
	}
	for _, c := range cases {
		if got := scanLangFlag(c.args); got != c.want {
			t.Errorf("scanLangFlag(%v) = %q，期望 %q", c.args, got, c.want)
		}
	}
}

// TestChooseLanguage_命令行优先于配置文件 覆盖三级优先级的实际取舍。
//
// os.Args 是包级变量，这里临时改写并在结束时还原：chooseLanguage 读的正是它，
// 而测试进程自身的参数与 al 的命令行不是一回事。
func TestChooseLanguage_命令行优先于配置文件(t *testing.T) {
	home := isolatedHome(t)
	writePreferences(t, home, "language = \"zh-CN\"\n")

	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"al", "ls"}
	if got := chooseLanguage(); got != "zh-CN" {
		t.Fatalf("未给 --lang 时应取配置文件里的语言，实际 %q", got)
	}

	os.Args = []string{"al", "--lang", "en", "ls"}
	if got := chooseLanguage(); got != "en" {
		t.Fatalf("--lang 应高于配置文件，实际 %q", got)
	}
}

// TestChooseLanguage_偏好文件损坏时降级 覆盖「坏文件不影响命令可用」。
//
// 启动路径会读偏好文件，若它损坏就让进程报错，shell 每次启动都会跟着报错。
// 因此这里断言：坏文件下仍然返回空串（交给 l10n 用默认语言），且不返回错误路径。
func TestChooseLanguage_偏好文件损坏时降级(t *testing.T) {
	home := isolatedHome(t)
	writePreferences(t, home, "language = [unclosed\n")

	original := os.Args
	t.Cleanup(func() { os.Args = original })
	os.Args = []string{"al", "ls"}

	if got := chooseLanguage(); got != "" {
		t.Fatalf("偏好文件损坏时应降级为空串，实际 %q", got)
	}
}

// TestConfigSetLanguage_写入并拒绝非法取值 覆盖 config 子命令的读写与取值校验。
func TestConfigSetLanguage_写入并拒绝非法取值(t *testing.T) {
	isolatedHome(t)

	out, err := run(t, "config", "set", "language", "zh-CN")
	if err != nil {
		t.Fatalf("设置合法语言不应报错: %v\n输出:\n%s", err, out)
	}
	data, readErr := os.ReadFile(preferencesFileOf(t))
	if readErr != nil {
		t.Fatalf("读取偏好文件失败: %v", readErr)
	}
	if !strings.Contains(string(data), `language = "zh-CN"`) {
		t.Fatalf("偏好文件里没有写入语言，内容:\n%s", string(data))
	}

	out, err = run(t, "config", "get", "language")
	if err != nil {
		t.Fatalf("读取语言不应报错: %v\n输出:\n%s", err, out)
	}
	if !strings.Contains(out, "zh-CN") {
		t.Fatalf("config get language 没有输出当前语言，输出:\n%s", out)
	}

	// 非法取值必须报错并列出可选值：静默写入会让用户以为设置生效，
	// 而启动时只会回落到默认语言，界面语言为什么没变根本无从查起
	out, err = run(t, "config", "set", "language", "klingon")
	if err == nil {
		t.Fatalf("非法语言应当被拒绝，实际成功，输出:\n%s", out)
	}
	if !strings.Contains(out, "en") || !strings.Contains(out, "zh-CN") {
		t.Fatalf("拒绝信息里应列出可选语言，输出:\n%s", out)
	}

	out, err = run(t, "config", "set", "colour", "red")
	if err == nil {
		t.Fatalf("未知配置项应当被拒绝，实际成功，输出:\n%s", out)
	}
}

// TestConfigGetLanguage_报告生效语言而非文件原文 覆盖手工写入非法取值后的展示一致性。
//
// 回归背景：偏好文件可以被用户手改，非法值在启动时会被静默降级为默认语言，
// 而 config get 若原样回显那个值，用户看到的就与界面实际表现相反
// （实测：文件写 "xx"，帮助文本是英文，命令却打印 xx）。
func TestConfigGetLanguage_报告生效语言而非文件原文(t *testing.T) {
	home := isolatedHome(t)
	writePreferences(t, home, "language = \"xx\"\n")

	out, err := run(t, "config", "get", "language")
	if err != nil {
		t.Fatalf("config get language 不应因为文件非法而报错: %v\n输出:\n%s", err, out)
	}
	if got := strings.TrimSpace(out); got != "en" {
		t.Fatalf("非法取值应报告生效的默认语言 en，实际 %q", got)
	}
}

// TestConfig_偏好与别名互不覆盖 覆盖两个配置文件分开存放这条设计约束。
//
// 这条用例守住的是最容易被「顺手合并」破坏的一点：alias.toml 的 Save 是整份重建，
// 偏好一旦并进去，用户下次 add/rm 就会把 language 静默抹掉，且退出码为 0。
func TestConfig_偏好与别名互不覆盖(t *testing.T) {
	home := isolatedHome(t)
	writeConfig(t, home, validConfig)

	if _, err := run(t, "config", "set", "language", "zh-CN"); err != nil {
		t.Fatalf("设置语言失败: %v", err)
	}
	prefsPath := preferencesFileOf(t)
	before := digest(t, prefsPath)

	if _, err := run(t, "add", "--common", "zz", "echo z"); err != nil {
		t.Fatalf("新增别名失败: %v", err)
	}
	if got := digest(t, prefsPath); got != before {
		t.Fatal("add 改动了偏好文件，语言设置会被静默抹掉")
	}

	aliasPath := configFileOf(home)
	aliasBefore := digest(t, aliasPath)
	if _, err := run(t, "config", "set", "language", "en"); err != nil {
		t.Fatalf("再次设置语言失败: %v", err)
	}
	if got := digest(t, aliasPath); got != aliasBefore {
		t.Fatal("config set 改动了别名文件")
	}
}

// TestInit_stdout只有shell代码 覆盖「init 的输出会被 shell eval」这条约束。
//
// 为什么必须逐字节比对而不是「包含某段文字」：init 的输出会被 `eval "$(al init zsh)"`
// 直接执行，多出一行说明文字就会被 shell 当成命令执行并报错，而 eval 的退出码仍是 0，
// 用户看到的是一句莫名其妙的 command not found。因此断言的是「除了预置脚本与别名声明，
// 一个字节都不许多」。
func TestInit_stdout只有shell代码(t *testing.T) {
	isolatedHome(t)

	preamble, readErr := os.ReadFile(filepath.Join("..", "init", "zsh.zsh"))
	if readErr != nil {
		t.Fatalf("读取 zsh 预置脚本失败: %v", readErr)
	}

	original := os.Stdout
	tmp, tmpErr := os.CreateTemp(t.TempDir(), "init-stdout-*")
	if tmpErr != nil {
		t.Fatalf("创建临时文件失败: %v", tmpErr)
	}
	t.Cleanup(func() { os.Stdout = original })
	os.Stdout = tmp

	_, runErr := run(t, "init", "zsh")

	os.Stdout = original
	if runErr != nil {
		t.Fatalf("init zsh 不应报错: %v", runErr)
	}
	if closeErr := tmp.Close(); closeErr != nil {
		t.Fatalf("关闭临时文件失败: %v", closeErr)
	}
	got, readOutErr := os.ReadFile(tmp.Name())
	if readOutErr != nil {
		t.Fatalf("读取 init 输出失败: %v", readOutErr)
	}

	// 空配置下唯一允许输出的就是预置脚本本身，fmt.Println 会再补一个换行
	expected := string(preamble) + "\n"
	if string(got) != expected {
		t.Fatalf("init zsh 的 stdout 混入了多余文本（会破坏 shell 的 eval），实际输出:\n%s", string(got))
	}

}

// writePreferences 在隔离 HOME 下写入偏好文件。
func writePreferences(t *testing.T, home, content string) {
	t.Helper()
	path := preferencesFileOf(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("建立偏好目录失败: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入偏好文件失败: %v", err)
	}
}

// preferencesFileOf 返回隔离 HOME 下偏好文件的绝对路径。
// 路径交给 config 包拼装，测试里不另写一份字面量。
func preferencesFileOf(t *testing.T) string {
	t.Helper()
	path, err := config.PreferencesPath()
	if err != nil {
		t.Fatalf("解析偏好文件路径失败: %v", err)
	}
	return path
}
