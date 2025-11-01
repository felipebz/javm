package cfg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var schemaTypes = map[string]string{
	"java.default_distribution": "string",
}

var defaults = map[string]any{
	"java": map[string]any{
		"default_distribution": "temurin",
	},
}

func ConfigFile() string {
	return filepath.Join(Dir(), "config.json")
}

var ErrInvalidConfigFile = errors.New("invalid config file")

func LoadUserOverrides() (map[string]any, error) {
	path := ConfigFile()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, ErrInvalidConfigFile
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func EffectiveValue(key string) (string, error) {
	if !IsKnownKey(key) {
		return "", fmt.Errorf("unknown key")
	}
	overrides, err := LoadUserOverrides()
	if err != nil {
		return "", err
	}
	// Try overrides first
	if v, ok := getByPath(overrides, strings.Split(key, ".")); ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
		// Non-string present: coerce to string if possible
		return fmt.Sprintf("%v", v), nil
	}
	// Fallback to defaults (guaranteed to exist)
	if v, ok := getByPath(defaults, strings.Split(key, ".")); ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
		return fmt.Sprintf("%v", v), nil
	}
	return "", nil
}

func ListEffective() ([]string, error) {
	keys := make([]string, 0, len(schemaTypes))
	for k := range schemaTypes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res := make([]string, 0, len(keys))
	for _, k := range keys {
		v, err := EffectiveValue(k)
		if err != nil {
			return nil, err
		}
		res = append(res, fmt.Sprintf("%s = %s", k, v))
	}
	return res, nil
}

func SetValue(key string, value string) error {
	if !IsKnownKey(key) {
		return fmt.Errorf("unknown key")
	}
	overrides, err := LoadUserOverrides()
	if err != nil {
		return err
	}

	setByPath(overrides, strings.Split(key, "."), value)

	defVal, _ := getByPath(defaults, strings.Split(key, "."))
	if s, ok := defVal.(string); ok && s == value {
		deleteByPath(overrides, strings.Split(key, "."))
	}

	pruneEmptyMaps(overrides)

	if len(overrides) == 0 {
		_ = os.Remove(ConfigFile())
		return nil
	}
	return writeAtomicJSON(ConfigFile(), overrides)
}

func UnsetValue(key string) error {
	if !IsKnownKey(key) {
		return fmt.Errorf("unknown key")
	}
	overrides, err := LoadUserOverrides()
	if err != nil {
		return err
	}

	deleteByPath(overrides, strings.Split(key, "."))

	pruneEmptyMaps(overrides)

	if len(overrides) == 0 {
		_ = os.Remove(ConfigFile())
		return nil
	}
	return writeAtomicJSON(ConfigFile(), overrides)
}

func writeAtomicJSON(dst string, data any) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	tmp := dst + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o666)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// On Windows, rename fails if target exists
	_ = os.Remove(dst)
	if err := os.Rename(tmp, dst); err != nil {
		return err
	}
	return nil
}

func IsKnownKey(key string) bool {
	_, ok := schemaTypes[key]
	return ok
}

func getByPath(m map[string]any, path []string) (any, bool) {
	cur := any(m)
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := mm[p]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

func setByPath(m map[string]any, path []string, v any) {
	cur := m
	for i, p := range path {
		if i == len(path)-1 {
			cur[p] = v
			return
		}
		nxt, ok := cur[p]
		if !ok {
			nm := map[string]any{}
			cur[p] = nm
			cur = nm
			continue
		}
		mm, ok := nxt.(map[string]any)
		if !ok {
			// Replace non-object with object
			nm := map[string]any{}
			cur[p] = nm
			cur = nm
			continue
		}
		cur = mm
	}
}

func deleteByPath(m map[string]any, path []string) {
	if len(path) == 0 {
		return
	}
	cur := m
	for i, p := range path {
		if i == len(path)-1 {
			delete(cur, p)
			return
		}
		nxt, ok := cur[p]
		if !ok {
			return
		}
		mm, ok := nxt.(map[string]any)
		if !ok {
			return
		}
		cur = mm
	}
}

func pruneEmptyMaps(m map[string]any) bool {
	for k, v := range m {
		mm, ok := v.(map[string]any)
		if ok {
			if pruneEmptyMaps(mm) {
				delete(m, k)
			}
		}
	}
	return len(m) == 0
}
