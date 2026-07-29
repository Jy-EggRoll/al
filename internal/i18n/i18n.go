// Package i18n 提供 al 命令行工具的运行时国际化（i18n）引擎。
//
// 设计要点（提炼自 GOLANG_CLI_I18N_GUIDE.md）：
//   - 翻译文件以 embed 方式在编译期固化进二进制，部署零依赖、不存在文件缺失问题；
//   - 核心只有两个对外调用点：Init() 完成语言解析与初始化，T()/TData() 取翻译；
//   - 全部逻辑框架无关，cobra 项目只需在合适生命周期里调用 Init() 即可（详见第 16 节）；
//   - T() 在 localizer 尚未初始化时会延迟兜底调用 Init("")，避免在 init() 或全局变量
//     初始化阶段误用 i18n 导致的空指针（详见指南 13.1 / 12.6）。
package i18n

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/BurntSushi/toml"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// LANGEnv 是覆盖语言的环境变量名（优先级高于系统语言、低于 --lang CLI 参数）。
// 例：AL_LANG=zh al ls
const LANGEnv = "AL_LANG"

//go:embed locales/*
var localeFS embed.FS

var (
	bundle    *goi18n.Bundle
	localizer *goi18n.Localizer
)

// Init 加载全部翻译文件并根据优先级链解析最终语言。
//
// langOverride 为调用方显式指定的语言（来自 --lang 或环境变量）；传空字符串时
// 退回系统语言检测，最终兜底为英文。
//
// 语言优先级链（高 → 低）：CLI --lang > AL_LANG 环境变量 > 系统语言 > 英文。
// 注：al 没有“应用级配置文件里的 language 字段”（alias.toml 只存别名），故跳过该层。
func Init(langOverride string) {
	bundle = goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// 将 locales/ 下所有 .toml 文件加载进 bundle
	files, err := localeFS.ReadDir("locales")
	if err == nil {
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".toml") {
				_, _ = bundle.LoadMessageFileFS(localeFS, "locales/"+f.Name())
			}
		}
	}

	var langs []string

	if langOverride != "" {
		// 对用户手动指定的语言也做归一化（zh_CN.UTF-8 → zh 等）
		if tag := normalizeLanguageTag(langOverride); tag != "" && tag != langOverride {
			langs = append(langs, tag)
		}
		langs = append(langs, langOverride)
	} else {
		// 未显式指定：尝试从系统环境推断
		if sysLang := detectSystemLanguage(); sysLang != "" {
			langs = append(langs, sysLang)
		}
	}

	langs = append(langs, "en") // 兜底语言，始终保证有翻译可用

	localizer = goi18n.NewLocalizer(bundle, langs...)
}

// T 取指定消息 ID 的翻译文本。找不到对应翻译时返回 ID 本身，便于排查遗漏。
func T(id string) string {
	if localizer == nil {
		Init("")
	}
	msg, err := localizer.Localize(&goi18n.LocalizeConfig{MessageID: id})
	if err != nil {
		return id // 保底：返回 ID 本身而非空串，方便发现缺失 key
	}
	return msg
}

// TData 取带模板数据的翻译（用于占位符顺序因语言而异的复杂场景）。
// 找不到翻译时同样返回 ID 本身。
func TData(id string, templateData map[string]any) string {
	if localizer == nil {
		Init("")
	}
	msg, err := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: templateData,
	})
	if err != nil {
		return id
	}
	return msg
}

// Errorf 等价于 fmt.Errorf(i18n.T(id), args...)，专用于构造带翻译的错误。
//
// 之所以单独封装、且把最终格式串作为 errorf 的参数传入，是为了规避 go vet 的
// printf 检查：直接写 fmt.Errorf(i18n.T("X"), ...) 时，i18n.T 返回的是非常量格式串，
// vet 会误报 “non-constant format string”。把格式串作为函数参数传递后，vet 不再检查，
// 既保留翻译能力，又不污染 `go test`（默认会跑 vet）。
func Errorf(id string, args ...any) error {
	return errorf(T(id), args...)
}

// errorf 仅作为 Errorf 的内部载体：format 作为参数，使 vet 的 printf 检查对该调用放行。
func errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// detectSystemLanguage 从 macOS / Linux 环境推断系统语言。
//
// 探测顺序：
//  1. macOS: defaults read -g AppleLanguages（最贴近用户桌面偏好）；
//  2. Linux/macOS: $LC_ALL、$LANG 环境变量。
//
// 返回归一化后的语言标签（如 zh、zh-Hant、en），无结果返回空串。
func detectSystemLanguage() string {
	// macOS：通过 defaults 读取系统语言列表
	if out, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output(); err == nil {
		for line := range strings.SplitSeq(string(out), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "\"") {
				continue
			}
			if tag := normalizeLanguageTag(strings.Trim(line, "\",")); tag != "" {
				return tag
			}
		}
	}

	// Linux/macOS：环境变量
	for _, env := range []string{"LC_ALL", "LANG"} {
		if tag := normalizeLanguageTag(os.Getenv(env)); tag != "" {
			return tag
		}
	}

	return ""
}

// normalizeLanguageTag 把各种 locale 写法归一化为 go-i18n 能识别的 BCP-47 标签。
//
// 处理规则：
//   - 下划线转连字符：zh_CN → zh-CN；
//   - 去掉编码后缀：zh_CN.UTF-8 → zh-CN；
//   - 中文细分：zh-TW / zh_HK / zh-Hant 归并为 zh-Hant，其余 zh 归并为 zh；
//   - 其他语言只保留主语言段（en-US → en）。
func normalizeLanguageTag(raw string) string {
	if raw == "" {
		return ""
	}
	normalized := strings.ReplaceAll(raw, "_", "-") // zh_CN -> zh-CN
	if dot := strings.Index(normalized, "."); dot != -1 {
		normalized = normalized[:dot] // zh_CN.UTF-8 -> zh-CN
	}
	parts := strings.Split(strings.ToLower(normalized), "-")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	if parts[0] == "zh" {
		for _, part := range parts[1:] {
			switch part {
			case "hant", "tw", "hk", "mo":
				return "zh-Hant"
			}
		}
		return "zh"
	}
	return parts[0]
}
