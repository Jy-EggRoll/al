package config

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/jy-eggroll/eggokit/atomicfile"
)

type AliasConfig struct {
	Aliases map[string]AliasValue
}

type AliasValue struct {
	Inline string
	Multi  []string
	Object map[string]AliasObjectValue
}

type AliasObjectValue struct {
	Inline string
	Multi  []string
}

func Default() AliasConfig {
	return AliasConfig{Aliases: map[string]AliasValue{}}
}

// Parse 解析 TOML 配置字节流。
//
// 除了 TOML 语法错误，这里还会拒绝别名名不合法的配置。理由：别名名最终会被四种 Shell
// 解析，非法名（如 `x; echo X`）在 `al init bash` 的输出里等同于一段可执行命令，
// 属于真正的代码注入，不是「显示难看」这种量级的问题。因此配置里出现这种名字时，
// 宁可让命令整体失败并指出是哪个名字，也不要继续把它渲染成 Shell 代码。
//
// 潜在影响点：解析失败会让所有命令（含只读的 ls/show/init）以非零退出码中止，
// 用户必须手工编辑配置文件删掉那个名字才能继续使用 al——这是刻意选择的代价，
// 因为带着注入名的配置本身就是坏的。
func Parse(data []byte) (AliasConfig, error) {
	var raw struct {
		Aliases map[string]interface{} `toml:"aliases"`
	}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Default(), fmt.Errorf("toml parse error: %w", err)
	}

	cfg := Default()
	for key, val := range raw.Aliases {
		// 先校验名字再解析值：名字不合法时，值的具体形态无关紧要，
		// 而且先报名字能让错误信息直接指向「用户该改哪里」。
		if err := ValidateAliasName(key); err != nil {
			return Default(), err
		}
		av, err := toAliasValue(val)
		if err != nil {
			return Default(), fmt.Errorf("alias %q: %w", key, err)
		}
		cfg.Aliases[key] = av
	}
	return cfg, nil
}

func toAliasValue(val interface{}) (AliasValue, error) {
	switch v := val.(type) {
	case string:
		return AliasValue{Inline: v}, nil
	case []interface{}:
		multi := make([]string, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok {
				return AliasValue{}, fmt.Errorf("array element %d is not a string", i)
			}
			multi[i] = s
		}
		return AliasValue{Multi: multi}, nil
	case map[string]interface{}:
		obj := make(map[string]AliasObjectValue, len(v))
		for k, item := range v {
			switch item := item.(type) {
			case string:
				obj[k] = AliasObjectValue{Inline: item}
			case []interface{}:
				multi := make([]string, len(item))
				for i, e := range item {
					s, ok := e.(string)
					if !ok {
						return AliasValue{}, fmt.Errorf("array element %d for %q is not a string", i, k)
					}
					multi[i] = s
				}
				obj[k] = AliasObjectValue{Multi: multi}
			default:
				return AliasValue{}, fmt.Errorf("unsupported value type for key %q", k)
			}
		}
		return AliasValue{Object: obj}, nil
	default:
		return AliasValue{}, fmt.Errorf("unsupported value type")
	}
}

func (av AliasValue) ForShell(shell string) (string, bool) {
	switch {
	case av.Inline != "":
		return av.Inline, true
	case av.Multi != nil:
		return strings.Join(av.Multi, " "), true
	case av.Object != nil:
		ov, ok := av.Object[shell]
		if !ok {
			return "", false
		}
		switch {
		case ov.Inline != "":
			return ov.Inline, true
		case ov.Multi != nil:
			return strings.Join(ov.Multi, " "), true
		}
	}
	return "", false
}

type AliasEntry struct {
	Name    string
	Command string
	Shell   string
}

// Shells returns all distinct shell names referenced in Object aliases.
func (c AliasConfig) Shells() []string {
	seen := map[string]bool{}
	for _, av := range c.Aliases {
		for shell := range av.Object {
			if !seen[shell] {
				seen[shell] = true
			}
		}
	}
	shells := make([]string, 0, len(seen))
	for s := range seen {
		shells = append(shells, s)
	}
	return shells
}

// AddAlias adds or updates an alias. shells = ["common"] or empty for shared.
//
// 名字与命令在改动内存中的配置之前先校验：一旦校验通过，后续 Save 才会把内容写盘，
// 因此这里挡住的东西绝不会以「半个合法状态」留在配置文件里。
func (c AliasConfig) AddAlias(name, command string, shells []string) (AliasConfig, error) {
	if err := ValidateAliasName(name); err != nil {
		return c, err
	}
	if err := ValidateAliasCommand(command); err != nil {
		return c, err
	}

	existing, exists := c.Aliases[name]
	isShared := len(shells) == 0 || (len(shells) == 1 && shells[0] == "common")

	// Aliases that don't exist yet — create fresh
	if !exists {
		if isShared {
			c.Aliases[name] = AliasValue{Inline: command}
		} else {
			obj := make(map[string]AliasObjectValue, len(shells))
			for _, s := range shells {
				obj[s] = AliasObjectValue{Inline: command}
			}
			c.Aliases[name] = AliasValue{Object: obj}
		}
		return c, nil
	}

	// Type conflict checks
	if existing.Multi != nil {
		return c, fmt.Errorf("alias %q is a multi-type, cannot overwrite", name)
	}
	if isShared && existing.Object != nil {
		return c, fmt.Errorf("alias %q already has shell-specific values — run 'al rm %s' first", name, name)
	}
	if !isShared && existing.Inline != "" {
		return c, fmt.Errorf("alias %q is shared across all shells — run 'al rm %s' first", name, name)
	}

	// Safe overwrite
	if isShared {
		c.Aliases[name] = AliasValue{Inline: command}
	} else {
		for _, s := range shells {
			existing.Object[s] = AliasObjectValue{Inline: command}
		}
		c.Aliases[name] = existing
	}
	return c, nil
}

// EditAlias modifies an existing alias's command and/or shell scope.
// command = nil → keep existing; shells = nil → keep existing scope.
//
// 名字与命令同样在写盘前校验。注意命令可能是「沿用旧值」（command == nil）：
// 这种情况下沿用值已经通过过校验，不会在这里被误报；只有用户显式传入新命令时才校验。
func (c AliasConfig) EditAlias(name string, command *string, shells []string) (AliasConfig, error) {
	if err := ValidateAliasName(name); err != nil {
		return c, err
	}
	if command != nil {
		if err := ValidateAliasCommand(*command); err != nil {
			return c, err
		}
	}

	existing, ok := c.Aliases[name]
	if !ok {
		return c, fmt.Errorf("alias %q not found", name)
	}

	if command == nil && shells == nil {
		return c, fmt.Errorf("nothing to change for alias %q", name)
	}

	// Resolve effective command
	cmd := ""
	if command != nil {
		cmd = *command
	} else {
		switch {
		case existing.Inline != "":
			cmd = existing.Inline
		case existing.Multi != nil:
			cmd = strings.Join(existing.Multi, " ")
		case existing.Object != nil:
			for _, ov := range existing.Object {
				switch {
				case ov.Inline != "":
					cmd = ov.Inline
				case ov.Multi != nil:
					cmd = strings.Join(ov.Multi, " ")
				}
				if cmd != "" {
					break
				}
			}
		}
	}

	// Apply changes
	if shells != nil {
		isShared := len(shells) == 1 && shells[0] == "common"
		if isShared {
			c.Aliases[name] = AliasValue{Inline: cmd}
		} else {
			obj := make(map[string]AliasObjectValue, len(shells))
			for _, s := range shells {
				obj[s] = AliasObjectValue{Inline: cmd}
			}
			c.Aliases[name] = AliasValue{Object: obj}
		}
	} else {
		switch {
		case existing.Multi != nil:
			return c, fmt.Errorf("alias %q is a multi-type, edit the TOML file directly", name)
		case existing.Inline != "":
			c.Aliases[name] = AliasValue{Inline: cmd}
		case existing.Object != nil:
			obj := make(map[string]AliasObjectValue, len(existing.Object))
			for shell := range existing.Object {
				obj[shell] = AliasObjectValue{Inline: cmd}
			}
			c.Aliases[name] = AliasValue{Object: obj}
		}
	}

	return c, nil
}

// RemoveAlias deletes an alias by name.
func (c AliasConfig) RemoveAlias(name string) (AliasConfig, error) {
	if _, ok := c.Aliases[name]; !ok {
		return c, fmt.Errorf("alias %q not found", name)
	}
	delete(c.Aliases, name)
	return c, nil
}

// Save 把配置以 TOML 原子写回磁盘。
//
// 为什么不再用 os.Create 直接往目标文件里流式编码：
// os.Create 会先把目标文件截断成 0 字节，此后才开始写入。截断与写入之间存在一个窗口，
// 进程在此期间崩溃、磁盘写满、或编码中途返回错误，用户留下的就是一个空文件或半截 TOML。
// 这份文件是用户别名的唯一存放处，没有第二份副本，损坏后无法自动恢复。
//
// 现在的做法：先在内存里把 TOML 完整编码进 bytes.Buffer，再由 atomicfile.Write
// 以「同目录临时文件 + rename」整体落位。编码失败时目标文件根本没被碰过；
// 写入失败时目标文件保持原样，只有临时文件被清理掉。用户在任何时刻看到的
// 配置文件，要么是完整的旧版本，要么是完整的新版本。
//
// 为什么必须先编码到 buffer：toml.NewEncoder 需要 io.Writer，而 atomicfile.Write
// 收的是 []byte，两者的接口形状不同。buffer 在这里同时承担两个职责：
// 作为编码目标和「编码成功与否」的判据——只要 Encode 返回 nil，就知道内容完整。
// 潜在影响点：整个配置现在会被完整载入内存。当前量级（几百条别名）下开销可忽略；
// 若未来配置规模长到数 MB，需要重新评估，但到那时更该做的是分文件而非改回流式写。
func (c AliasConfig) Save(path string) error {
	flat := map[string]interface{}{}
	for name, av := range c.Aliases {
		switch {
		case av.Inline != "":
			flat[name] = av.Inline
		case av.Multi != nil:
			flat[name] = av.Multi
		case av.Object != nil:
			obj := map[string]interface{}{}
			for shell, ov := range av.Object {
				switch {
				case ov.Inline != "":
					obj[shell] = ov.Inline
				case ov.Multi != nil:
					obj[shell] = ov.Multi
				}
			}
			flat[name] = obj
		}
	}

	// 先把 TOML 完整编码进内存：编码失败时磁盘上的目标文件完全不受影响，
	// 这是「编码失败不破坏原文件」的关键前提。
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(map[string]interface{}{"aliases": flat}); err != nil {
		return fmt.Errorf("encode toml: %w", err)
	}

	// atomicfile.Write 内部会按需创建父目录，因此这里不需要（也不应该）提前 MkdirAll：
	// 提前建目录会让只读命令也在用户机器上留下空目录。
	if err := atomicfile.Write(path, buf.Bytes()); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Flatten produces one AliasEntry per alias per applicable shell.
func (c AliasConfig) Flatten() []AliasEntry {
	var entries []AliasEntry
	for name, av := range c.Aliases {
		switch {
		case av.Inline != "":
			entries = append(entries, AliasEntry{Name: name, Command: av.Inline, Shell: "common"})
		case av.Multi != nil:
			entries = append(entries, AliasEntry{Name: name, Command: strings.Join(av.Multi, " "), Shell: "common"})
		case av.Object != nil:
			for shell, ov := range av.Object {
				cmd := ""
				switch {
				case ov.Inline != "":
					cmd = ov.Inline
				case ov.Multi != nil:
					cmd = strings.Join(ov.Multi, " ")
				}
				entries = append(entries, AliasEntry{Name: name, Command: cmd, Shell: shell})
			}
		}
	}
	return entries
}
