// prefs.go 负责界面偏好配置文件（~/.config/al/config.toml）的读写。
//
// 为什么偏好不并进 alias.toml：alias.toml 的 Save 是整份重建（见 config.go 的 Save），
// 它只知道 aliases 一张表，任何 add/rm 都会用内存里的别名表覆盖整个文件。偏好一旦写进去，
// 用户下次增删别名就会把 language 静默抹掉，且退出码仍是 0，用户得不到任何提示。
//
// 由此得出一条必须遵守的约定：本文件与 alias.toml 同目录、同格式，但**互不知情**。
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/jy-eggroll/eggokit/atomicfile"
)

// 配置目录与两个文件名集中在这里。
//
// 为什么不让各处自己拼 "~/.config/al/..."：偏好文件与别名文件必须同目录，
// 两处各写一份字面量，日后改目录时必然只改一处，表现是「偏好读不到」或
// 「别名写到了新目录、旧目录还留着」这类没有报错的问题。
const (
	configSubdir        = ".config/al"
	aliasFileName       = "alias.toml"
	preferencesFileName = "config.toml"
)

// ConfigDir 返回 al 的配置目录（不创建）。
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configSubdir), nil
}

// AliasPath 返回别名配置文件的路径。
func AliasPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, aliasFileName), nil
}

// PreferencesPath 返回偏好配置文件的路径。
func PreferencesPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, preferencesFileName), nil
}

// LoadLanguage 只读配置文件里的 language 字段，供命令行解析之前确定输出语言。
//
// 单独提供这个薄封装，是为了让启动路径不必关心路径与文件名的细节：
// 它在 --help 之前就会被调用，任何一步出问题都必须能被调用方按「静默降级」处理。
func LoadLanguage() (string, error) {
	path, err := PreferencesPath()
	if err != nil {
		return "", err
	}
	prefs, err := LoadPreferences(path)
	if err != nil {
		return "", err
	}
	return prefs.Language, nil
}

// readFileIfExists 读文件，不存在时返回 (nil, nil)。
//
// 刻意把「不存在」与「读不出来」分开：前者是首次使用的正常场景，后者说明用户
// 已有文件但它坏了，两者的处理方式相反（继续 vs 报错），混在一起就没有办法区分。
func readFileIfExists(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read preferences file %s: %w", path, err)
	}
	return data, nil
}

// Preferences 是界面偏好。字段刻意保持极少：每加一项都要想清楚它是否属于
// 「写在这里会让 add/rm 抹掉」那一类问题。
type Preferences struct {
	// Language 是输出语言标签，空串表示交给 l10n 用默认语言
	Language string
}

// LoadPreferences 读取偏好配置文件。
//
// 文件名由调用方传入而不是内部拼 HOME：HOME 的取值在测试里会被改写，
// 拼路径的职责留给 cmd 层（它本来就负责 ConfigPath），这里只做读写。
//
// 文件不存在时返回零值与 nil：首次使用是正常场景，不是错误。
// 读不出来或解析失败时返回 error，由调用方决定是否降级——启动路径会静默忽略它，
// 因为语言只影响展示，不该让 shell 启动报错。
func LoadPreferences(path string) (Preferences, error) {
	data, err := readFileIfExists(path)
	if err != nil {
		return Preferences{}, err
	}
	if data == nil {
		return Preferences{}, nil
	}

	var raw struct {
		Language string `toml:"language"`
	}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Preferences{}, fmt.Errorf("parse preferences file %s: %w", path, err)
	}
	return Preferences{Language: raw.Language}, nil
}

// SaveLanguage 把语言写进偏好配置文件，其余键原样保留。
//
// 为什么按 map 读改写、而不是直接用结构体整体编码：本文件将来会随需求增加字段，
// 整份重建会让旧版本 al 丢掉新版本写进去的未知键——那正是 alias.toml 上已经出现过的问题，
// 这里从一开始就避开。
//
// 写盘走 atomicfile.Write：它会按需创建父目录，写入失败时目标文件保持原样，
// 因此「进程中途被杀」不会留下半份损坏的偏好。
func SaveLanguage(path, language string) error {
	data, err := readFileIfExists(path)
	if err != nil {
		return err
	}

	fields := map[string]interface{}{}
	if data != nil {
		if err := toml.Unmarshal(data, &fields); err != nil {
			// 坏掉的偏好文件不允许被静默覆盖：用户可能手工写了别的内容，
			// 覆盖等于替他丢数据。让命令报错，由用户决定是修还是删
			return fmt.Errorf("parse preferences file %s: %w", path, err)
		}
	}
	fields["language"] = language

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(fields); err != nil {
		return fmt.Errorf("encode preferences: %w", err)
	}
	if err := atomicfile.Write(path, buf.Bytes()); err != nil {
		return fmt.Errorf("write preferences file %s: %w", path, err)
	}
	return nil
}
