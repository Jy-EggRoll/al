// locales 收集 al 自己的语言文件，以及它们对应的 l10n 配置。
//
// 文案分两层来源：本包 embed 的 *.json 只覆盖 al 自己的文案；随 eggokit 发布的库文案
// 通过 ExtraLayers 叠加进来。因此这里**不得**收录库的消息——收了就会随库升级而漂移。
//
// 本包同时是 Taskfile 里 l10n 任务的参数来源：--dir 指向本目录，--src 覆盖的命令目录
// 与 Supported 是一对，改动这里时要一起看。
package locales

import (
	"embed"

	"github.com/jy-eggroll/eggokit/l10n"
	eggolocales "github.com/jy-eggroll/eggokit/locales"
)

// Default 是 al 的默认语言，同时也是配置文件里 language 项的默认值。
const Default = "en"

//go:embed *.json
var files embed.FS

// Options 返回 al 的 l10n 配置。
//
// Supported 的顺序是语义的一部分：go-i18n 按注册顺序挑选最匹配的语言标签，
// 因此默认语言放在首位。
func Options() l10n.Options {
	return l10n.Options{
		Default:     Default,
		Supported:   Supported(),
		FS:          files,
		ExtraLayers: []l10n.Layer{eggolocales.Layer()},
	}
}

// Supported 返回 al 随二进制发布的语言列表（副本）。
// 供需要在 Init 之外做语言校验的场合使用，如 config set language 的取值校验。
func Supported() []string {
	return []string{"en", "zh-CN"}
}
