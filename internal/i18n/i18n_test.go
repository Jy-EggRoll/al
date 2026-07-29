package i18n

import (
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// loadCatalog 直接读取并解析单个翻译文件，返回扁平的 key→文本 映射。
// 所有翻译文件均为顶层 key（无 [section]），故可直接解到 map[string]string。
func loadCatalog(t *testing.T, name string) map[string]string {
	t.Helper()
	data, err := localeFS.ReadFile("locales/" + name)
	if err != nil {
		t.Fatalf("读取翻译文件 %s 失败: %v", name, err)
	}
	var catalog map[string]string
	if err := toml.Unmarshal(data, &catalog); err != nil {
		t.Fatalf("解析翻译文件 %s 失败: %v", name, err)
	}
	return catalog
}

// sortedKeys 返回排序后的 key 列表，便于稳定比较与 diff。
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// diffStrings 返回在 a 中但不在 b 中的字符串。
func diffStrings(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, s := range b {
		set[s] = true
	}
	var missing []string
	for _, s := range a {
		if !set[s] {
			missing = append(missing, s)
		}
	}
	return missing
}

// TestLocalesMatchEnglishCatalog 确保所有非英文翻译文件的 key 集合与英文基准完全一致。
// 这是 i18n 最关键的测试：缺 key 会导致运行时回退成 ID 文本，多 key 则永远不会被使用。
func TestLocalesMatchEnglishCatalog(t *testing.T) {
	t.Parallel()

	english := loadCatalog(t, "active.en.toml")
	englishKeys := sortedKeys(english)

	files, err := localeFS.ReadDir("locales")
	if err != nil {
		t.Fatalf("读取 locales 目录失败: %v", err)
	}

	for _, f := range files {
		name := f.Name()
		if name == "active.en.toml" || !strings.HasSuffix(name, ".toml") {
			continue
		}
		catalog := loadCatalog(t, name)
		keys := sortedKeys(catalog)

		if missing := diffStrings(englishKeys, keys); len(missing) > 0 {
			t.Errorf("%s 缺少 key: %s", name, strings.Join(missing, ", "))
		}
		if extra := diffStrings(keys, englishKeys); len(extra) > 0 {
			t.Errorf("%s 有多余 key: %s", name, strings.Join(extra, ", "))
		}
	}
}

// TestPlaceholderConsistency 确保同一 key 在各语言中的 fmt 占位符集合完全一致。
// 占位符顺序可因语言而异，但数量与类型必须相同，否则格式化会错位或报错。
func TestPlaceholderConsistency(t *testing.T) {
	t.Parallel()

	english := loadCatalog(t, "active.en.toml")
	englishKeys := sortedKeys(english)

	files, _ := localeFS.ReadDir("locales")
	for _, f := range files {
		name := f.Name()
		if name == "active.en.toml" || !strings.HasSuffix(name, ".toml") {
			continue
		}
		other := loadCatalog(t, name)
		for _, key := range englishKeys {
			want := placeholders(english[key])
			got := placeholders(other[key])
			if !slices.Equal(want, got) {
				t.Errorf("%s: key %s 占位符不一致: 英文 %v, %s %v", name, key, want, name, got)
			}
		}
	}
}

// placeholders 提取字符串中的 fmt 风格占位符（%s %d %q %w 等，忽略 %%）。
func placeholders(s string) []string {
	var out []string
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		if i+1 < len(s) && s[i+1] == '%' { // 转义的 %%
			i++
			continue
		}
		j := i + 1
		for j < len(s) && !isPlaceholderEnd(s[j]) {
			j++
		}
		if j < len(s) {
			out = append(out, s[i:j+1])
		}
	}
	return out
}

func isPlaceholderEnd(c byte) bool {
	switch c {
	case 's', 'd', 'q', 'w', 'v', 'c', 'b', 'o', 'x', 'X', 'e', 'E', 'f', 'F', 'g', 'G', 't', 'T':
		return true
	}
	return false
}

// TestNormalizeLanguageTag 覆盖各种 locale 写法的归一化结果。
func TestNormalizeLanguageTag(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"en", "en"},
		{"en-US", "en"},
		{"en_US.UTF-8", "en"},
		{"zh-CN", "zh"},
		{"zh-Hans", "zh"},
		{"zh_TW", "zh-Hant"},
		{"zh_TW.UTF-8", "zh-Hant"},
		{"zh-Hant", "zh-Hant"},
		{"zh-HK", "zh-Hant"},
		{"ja", "ja"},
	}
	for _, tc := range cases {
		if got := normalizeLanguageTag(tc.in); got != tc.want {
			t.Errorf("normalizeLanguageTag(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
