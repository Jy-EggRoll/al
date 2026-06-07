package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
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

func Parse(data []byte) (AliasConfig, error) {
	var raw struct {
		Aliases map[string]interface{} `toml:"aliases"`
	}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Default(), fmt.Errorf("toml parse error: %w", err)
	}

	cfg := Default()
	for key, val := range raw.Aliases {
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
func (c AliasConfig) AddAlias(name, command string, shells []string) (AliasConfig, error) {
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
func (c AliasConfig) EditAlias(name string, command *string, shells []string) (AliasConfig, error) {
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

// Save writes the config back to disk as TOML.
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

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(map[string]interface{}{"aliases": flat}); err != nil {
		return fmt.Errorf("encode toml: %w", err)
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
