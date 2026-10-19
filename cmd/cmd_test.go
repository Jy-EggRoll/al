package cmd

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jy-eggroll/al/config"

	"github.com/spf13/cobra"

	"github.com/pterm/pterm"
)

// 本文件补齐 cmd 包此前完全缺失的测试（原先 go test ./... 输出 cmd [no test files]）。
//
// 为什么必须在隔离 HOME 下运行：配置路径由 root.go 的 PersistentPreRunE 用
// os.UserHomeDir() 拼出，而 os.UserHomeDir() 在 Linux 上读 $HOME。若不在测试里改写 HOME，
// 命令会直接读写运行测试的人自己的 ~/.config/al/alias.toml——测试一旦写盘就会毁掉真实别名，
// 因此每个用例都先用 t.TempDir() 换掉 HOME，测试结束由 testing 框架自动清理。
//
// 为什么断言要同时看 stdout 与 stderr：cobra 把错误写到 ErrOrStderr()，而 Usage 走
// OutOrStderr()；真实终端里这两者都是同一个 tty，无法区分，测试里若不合并观察，
// 就可能因为「Usage 出现在另一个流」而漏判。

// resetCmdState 把包级全局状态复位。
//
// 为什么需要它：flag 变量与 rootCmd 都是包级单例，flag 值在多次 rootCmd.Execute() 之间
// 不会自动回到默认值（cobra 的 ParseFlags 只写入不重置），而 rootCmd.SilenceUsage 会被
// root.go 在配置失败时置为 true 并一直保留。真实进程每次只执行一条命令，不存在这类污染；
// 但同一个测试进程里连续跑多条命令时，上一个用例的 flag 与 SilenceUsage 会串到下一个用例，
// 让「参数用错是否打印 usage」这类断言得出与真实行为相反的错误结论。
func resetCmdState() {
	flagCommon, flagZsh, flagBash, flagFish, flagPowershell, flagForce = false, false, false, false, false, false
	flagEditCommon, flagEditZsh, flagEditBash, flagEditFish, flagEditPowershell = false, false, false, false, false
	lsAll, lsZsh, lsBash, lsFish, lsPwsh = false, false, false, false, false
	resetSilenceUsage(rootCmd)
	rootCmd.SetArgs(nil)
	cfg = config.Default()
	ConfigPath = ""
}

// resetSilenceUsage 递归复位整棵命令树的 usage/error 沉默开关。
//
// 为什么不能只复位 rootCmd：root.go 的 PersistentPreRunE 拿到的是「正在执行的子命令」，
// 它把 SilenceUsage 置在 addCmd 这类叶子命令上，此后整个进程内都会保持为 true。
// 真实使用中每条命令都是一个新进程，所以线上看不出问题；但同一个测试进程里连续跑多条命令时，
// 前一个用例留下的 true 会让后一个用例的 usage 断言凭空失败（第一轮实测就踩到了这一点）。
func resetSilenceUsage(c *cobra.Command) {
	c.SilenceUsage = false
	c.SilenceErrors = false
	for _, sub := range c.Commands() {
		resetSilenceUsage(sub)
	}
}

// isolatedHome 返回一个隔离的家目录，并把它设为当前进程的 HOME。
func isolatedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

// configFileOf 返回隔离 HOME 下配置文件的绝对路径。
func configFileOf(home string) string {
	return filepath.Join(home, ".config", "al", "alias.toml")
}

// writeConfig 在隔离 HOME 下写入配置内容，返回配置文件路径。
func writeConfig(t *testing.T, home, content string) string {
	t.Helper()
	dir := filepath.Dir(configFileOf(home))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("建立配置目录失败: %v", err)
	}
	path := configFileOf(home)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	return path
}

// digest 计算文件内容的 md5，用于「原文件一字不动」这类断言。
// 直接比对字节内容比只比对长度可靠：长度不变但内容被改的情形（例如把某条别名覆盖掉）也能发现。
func digest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", path, err)
	}
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

// run 执行一次 al 命令，返回 stdout+stderr 的合并文本与 cobra 返回的错误。
// 不调用 Execute()：Execute() 内部遇到错误会 os.Exit(1)，会把整个测试进程带走。
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetCmdState()

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(args)
	// pterm 自己持有输出目标，不经 cobra 的 writer，必须单独接管，
	// 否则 add/ls/show 的正常输出会直接打到测试进程的终端上，断言看不见。
	pterm.SetDefaultOutput(&out)
	pterm.DisableColor()

	err := rootCmd.Execute()
	return out.String(), err
}

// noTempFileLeft 断言配置目录里没有原子写留下的临时文件。
func noTempFileLeft(t *testing.T, home string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(configFileOf(home)))
	if err != nil {
		return // 目录不存在同样说明没有残留
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".eggokit-atomic-") {
			t.Fatalf("配置目录残留了临时文件 %s", e.Name())
		}
	}
}

const validConfig = `[aliases]
gs = "git status"
`

// TestRoot_坏配置不写盘且非零退出 覆盖「配置解析失败不再静默清空别名」。
// 类型不匹配（别名写成数字）属于「文件存在且坏掉」，是修复前会静默清空用户别名的典型场景。
func TestRoot_坏配置不写盘且非零退出(t *testing.T) {
	home := isolatedHome(t)
	path := writeConfig(t, home, "[aliases]\ngs = 123\n")
	before := digest(t, path)

	out, err := run(t, "add", "--common", "zz", "echo z")

	if err == nil {
		t.Fatal("坏配置下 add 应当返回错误，实际成功")
	}
	if got := digest(t, path); got != before {
		t.Fatal("坏配置下 add 改动了原配置文件，用户的别名可能已被清空")
	}
	if strings.Contains(out, "Usage:") {
		t.Fatalf("配置损坏属于运行期错误，不应打印整段 usage，实际输出:\n%s", out)
	}
	noTempFileLeft(t, home)
}

// TestRoot_配置路径是目录时不写盘 覆盖「读取失败」这一类与「文件不存在」必须区分的场景。
// 修复前 os.ReadFile 失败会被当成空配置继续走，随后的写盘会把路径上的目录顶掉或报出无关错误。
func TestRoot_配置路径是目录时不写盘(t *testing.T) {
	home := isolatedHome(t)
	if err := os.MkdirAll(configFileOf(home), 0o755); err != nil {
		t.Fatalf("建立同名目录失败: %v", err)
	}

	out, err := run(t, "add", "--common", "zz", "echo z")

	if err == nil {
		t.Fatal("配置路径是目录时 add 应当返回错误，实际成功")
	}
	if strings.Contains(out, "Usage:") {
		t.Fatalf("读取失败属于运行期错误，不应打印整段 usage，实际输出:\n%s", out)
	}
	if fi, statErr := os.Stat(configFileOf(home)); statErr != nil || !fi.IsDir() {
		t.Fatal("配置路径上的目录被写盘操作破坏了")
	}
}

// TestRoot_参数用错时打印Usage_配置损坏时也一样 覆盖「两类错误要区分对待」。
//
// 为什么专门测「配置损坏 + 参数用错」这个组合：参数校验由 cobra 在 PersistentPreRunE 之前
// 完成，此时 root.go 还没有机会把 SilenceUsage 置为 true，因此即使配置是坏的，参数错误
// 依然应当看到用法。如果实现者图省事在 rootCmd 上全局关闭 usage，这个用例就会失败——
// 它守住的正是「不要为了消掉噪音而把用法说明一起删掉」。
func TestRoot_参数用错时打印Usage_配置损坏时也一样(t *testing.T) {
	home := isolatedHome(t)
	writeConfig(t, home, "[aliases]\ngs = 123\n")

	out, err := run(t, "add", "onlyone")

	if err == nil {
		t.Fatal("参数个数不足时应当返回错误")
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("参数用错时应当打印 usage，实际输出:\n%s", out)
	}
}

// TestRoot_参数用错时打印Usage_配置正常时 用正常配置再测一遍，排除「损坏配置掩盖了 usage」的可能。
func TestRoot_参数用错时打印Usage_配置正常时(t *testing.T) {
	home := isolatedHome(t)
	writeConfig(t, home, validConfig)

	out, err := run(t, "add", "onlyone")

	if err == nil {
		t.Fatal("参数个数不足时应当返回错误")
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("参数用错时应当打印 usage，实际输出:\n%s", out)
	}
}

// TestAdd_拒绝可注入别名名 覆盖「别名名不得可注入」在 add 路径上生效。
// 反引号与 $() 会在 bash 里被命令替换，分号会开启新命令——三者任一进入生成的别名声明，
// 用户敲下别名就等于执行了配置里写的任意命令。
func TestAdd_拒绝可注入别名名(t *testing.T) {
	cases := []struct {
		name string
		alt  string
	}{
		{"分号开启新命令", "a;b"},
		{"反引号命令替换", "a`b`"},
		{"美元括号命令替换", "a$(id)"},
		{"空格拆分词", "a b"},
		{"换行拆行", "a\nb"},
		{"管道", "a|b"},
		{"后台执行", "a&b"},
		{"重定向", "a>b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			home := isolatedHome(t)
			path := writeConfig(t, home, validConfig)
			before := digest(t, path)

			out, err := run(t, "add", "--common", c.alt, "echo hi")

			if err == nil {
				t.Fatalf("别名名 %q 应当被拒绝，实际成功，输出:\n%s", c.alt, out)
			}
			if !strings.Contains(out, "illegal character") {
				t.Fatalf("拒绝理由应当是字符非法，实际输出:\n%s", out)
			}
			if got := digest(t, path); got != before {
				t.Fatal("被拒绝的 add 不应改动配置文件")
			}
		})
	}
}

// TestEdit_拒绝可注入别名名 覆盖「别名名不得可注入」在 edit 路径上生效。
// edit 与 add 是两条独立的写入入口，只测其中一条会放过另一条上的注入通道。
func TestEdit_拒绝可注入别名名(t *testing.T) {
	home := isolatedHome(t)
	path := writeConfig(t, home, validConfig)
	before := digest(t, path)

	out, err := run(t, "edit", "--common", "a;b", "echo hi")

	if err == nil {
		t.Fatalf("edit 应当拒绝可注入的别名名，实际成功，输出:\n%s", out)
	}
	if !strings.Contains(out, "illegal character") {
		t.Fatalf("拒绝理由应当是字符非法，实际输出:\n%s", out)
	}
	if got := digest(t, path); got != before {
		t.Fatal("被拒绝的 edit 不应改动配置文件")
	}
}

// TestAdd_拒绝空命令 覆盖「空数组别名不再 panic」的写入侧防线。
// 空命令写进配置后，fish 与 powershell 生成器会对 strings.Fields("") 取下标而 panic，
// 因此必须在写盘前就拒绝，而不是留到 al init 时炸。
func TestAdd_拒绝空命令(t *testing.T) {
	for _, cmd := range []string{"", "   "} {
		t.Run("命令为"+strings.ReplaceAll(cmd, " ", "空格"), func(t *testing.T) {
			home := isolatedHome(t)
			path := writeConfig(t, home, validConfig)
			before := digest(t, path)

			out, err := run(t, "add", "--common", "m", cmd)

			if err == nil {
				t.Fatalf("空命令应当被拒绝，实际成功，输出:\n%s", out)
			}
			if !strings.Contains(out, "empty or whitespace-only") {
				t.Fatalf("拒绝理由应当是命令为空，实际输出:\n%s", out)
			}
			if got := digest(t, path); got != before {
				t.Fatal("被拒绝的 add 不应改动配置文件")
			}
		})
	}
}

// TestRoot_只读命令不创建配置目录 覆盖「只读命令不再创建配置目录」。
// 修复前 PersistentPreRunE 无条件 MkdirAll，用户只是查看一眼配置就会在家目录留下空目录；
// 这里对四条只读路径逐一断言目录不存在。
func TestRoot_只读命令不创建配置目录(t *testing.T) {
	cases := [][]string{
		{"ls"},
		{"show", "nope"},
		{"init", "bash"},
		{"version"},
		{"completion", "bash"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			home := isolatedHome(t)

			run(t, args...)

			if _, err := os.Stat(filepath.Dir(configFileOf(home))); !os.IsNotExist(err) {
				t.Fatalf("只读命令 %v 创建了配置目录", args)
			}
		})
	}
}

// TestShow_来源路径与真实配置路径一致 覆盖「show 的配置路径改用 ConfigPath 变量」。
// 写死成 "~/.config/al/alias.toml" 在 HOME 被改动的环境（容器、CI、测试隔离）里会显示
// 一个并不存在的路径，把排查问题的人引向错误方向；这里断言输出的路径必须落在隔离 HOME 下。
func TestShow_来源路径与真实配置路径一致(t *testing.T) {
	home := isolatedHome(t)
	writeConfig(t, home, validConfig)

	out, err := run(t, "show", "gs")
	if err != nil {
		t.Fatalf("show 正常别名不应报错: %v", err)
	}
	if !strings.Contains(out, configFileOf(home)) {
		t.Fatalf("show 输出的来源路径与实际配置路径不一致，期望包含 %s，实际输出:\n%s", configFileOf(home), out)
	}
}

// TestAdd_写入失败时原文件完好 覆盖「Save 改为原子写」。
//
// 用「把配置目录设为只读」来制造写入失败：原子写先在目录里建临时文件，
// 建不出来就直接返回，目标文件根本不会被打开，因此内容必然完好。
// 修复前 os.Create 会先截断目标文件再写入，目录权限对已存在的文件不起保护作用，
// 这个用例就会看到配置被清空。
func TestAdd_写入失败时原文件完好(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("以 root 运行时目录权限位不生效，无法制造写入失败")
	}
	home := isolatedHome(t)
	path := writeConfig(t, home, validConfig)
	before := digest(t, path)

	dir := filepath.Dir(path)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("设置目录只读失败: %v", err)
	}
	// 退出时务必恢复权限，否则 t.TempDir() 清理时会因为目录不可写而残留垃圾
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	_, err := run(t, "add", "--common", "zz", "echo z")

	if err == nil {
		t.Fatal("目录不可写时 add 应当返回错误")
	}
	if got := digest(t, path); got != before {
		t.Fatal("写入失败时原配置被改动了，原子写没有生效")
	}
	noTempFileLeft(t, home)
}

// TestSave_写盘失败时不打印Usage 覆盖「运行期错误一律不打印 usage」这条统一规则。
//
// 为什么专门测 Save 失败：参数用错与配置读取/解析失败此前都已有用例，
// 唯独 Save 失败这条路径上没有管 usage，`al add` 在配置目录不可写时会先报
// 「save config: ... permission denied」再跟一整段 Usage，与配置损坏时的干净报错不一致。
// 修复后 usage 只在进入 PersistentPreRunE 之前（参数与标志校验）才打印，
// 因此这里断言两件事同时成立：错误里带得上 save 失败的上下文，且输出里没有 Usage。
//
// 为什么同时覆盖 add 与 rm：两者是各自独立的写入入口，各写一次 Save 失败的处理，
// 只测其中一个会放过另一个，正是原先「按失败点打补丁」漏掉杀伤力的原因。
func TestSave_写盘失败时不打印Usage(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("以 root 运行时目录权限位不生效，无法制造写入失败")
	}
	cases := []struct {
		name string
		args []string
	}{
		{"add", []string{"add", "--common", "zz", "echo z"}},
		{"rm", []string{"rm", "gs"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			home := isolatedHome(t)
			path := writeConfig(t, home, validConfig)

			dir := filepath.Dir(path)
			if err := os.Chmod(dir, 0o500); err != nil {
				t.Fatalf("设置目录只读失败: %v", err)
			}
			t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

			out, err := run(t, c.args...)

			if err == nil {
				t.Fatalf("目录不可写时 %s 应当返回错误，实际成功", c.name)
			}
			if !strings.Contains(err.Error(), "save config") {
				t.Fatalf("错误应当带上保存失败的上下文，实际: %v", err)
			}
			if strings.Contains(out, "Usage:") {
				t.Fatalf("保存失败属于运行期错误，不应打印整段 usage，实际输出:\n%s", out)
			}
		})
	}
}

// TestAdd_省略shell标志时写入common 覆盖问题 2 修正后的实际行为。
//
// 不带任何 shell 标志执行 add，等价于 --common，别名写进 [aliases] 顶层，
// 而不是报「必须指定 shell」。这条用例同时守住两件事：resolveShells 的兜底不能删，
// 以及 add 的文档不能再说成 shell 标志必填。
func TestAdd_省略shell标志时写入common(t *testing.T) {
	home := isolatedHome(t)
	path := writeConfig(t, home, validConfig)

	out, err := run(t, "add", "zz", "echo z")

	if err != nil {
		t.Fatalf("省略 shell 标志时 add 不应报错: %v\n输出:\n%s", err, out)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("读取写回后的配置失败: %v", readErr)
	}
	content := string(data)
	if !strings.Contains(content, `zz = "echo z"`) {
		t.Fatalf("省略 shell 标志的别名未写入 common 分组，配置内容:\n%s", content)
	}
	if strings.Contains(content, "[aliases.zz]") {
		t.Fatalf("省略 shell 标志的别名不应写成分 shell 的对象，配置内容:\n%s", content)
	}
}
