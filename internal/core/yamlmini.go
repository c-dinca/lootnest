package core

import (
	"strconv"
	"strings"
)

// readYAMLMap reads the block-style YAML that Python's PyYAML writes for
// Lutris's game files and Bottles' bottle.yml: maps nested by indentation,
// with plain, 'single' or "double" quoted scalars, a plain scalar folded
// onto more indented lines, and lists (read as []any of scalars). Anchors,
// flow style ({...}, [...] beyond empty ones), multi-document files and
// block scalars (| and >) aren't read; their keys come back empty.
func readYAMLMap(text string) map[string]any {
	type frame struct {
		indent int
		m      map[string]any
		key    string // the last key read at this level
	}
	root := map[string]any{}
	stack := []frame{{indent: -1, m: root}}
	var lastKey string
	var lastMap map[string]any
	lastIndent := -1
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed == "---" {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		// A plain scalar PyYAML folded onto the next lines.
		if lastMap != nil && indent > lastIndent && !strings.HasPrefix(trimmed, "- ") {
			if s, ok := lastMap[lastKey].(string); ok && s != "" && !yamlIsKey(trimmed) {
				lastMap[lastKey] = s + " " + trimmed
				continue
			}
		}
		for len(stack) > 1 && indent <= stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
		}
		top := &stack[len(stack)-1]
		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			// A list item under the last key read: PyYAML writes them at the
			// key's own indentation, other writers further in (then the
			// empty map made for the key is where they land).
			owner := top
			if top.key == "" && len(top.m) == 0 && len(stack) > 1 {
				owner = &stack[len(stack)-2]
			}
			if owner.key != "" {
				list, _ := owner.m[owner.key].([]any)
				owner.m[owner.key] = append(list, yamlValue(strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))))
			}
			lastMap = nil
			continue
		}
		key, val, ok := yamlSplit(trimmed)
		if !ok {
			lastMap = nil
			continue
		}
		if val == "" {
			child := map[string]any{}
			top.m[key] = child
			top.key = key
			stack = append(stack, frame{indent: indent, m: child})
			lastMap = nil
			continue
		}
		top.m[key] = yamlValue(val)
		top.key = key
		lastKey, lastMap, lastIndent = key, top.m, indent
	}
	return root
}

func yamlIsKey(s string) bool {
	_, _, ok := yamlSplit(s)
	return ok
}

// yamlSplit cuts "key: value" (the key possibly quoted).
func yamlSplit(s string) (key, val string, ok bool) {
	if s == "" {
		return "", "", false
	}
	if s[0] == '"' || s[0] == '\'' {
		end := strings.IndexByte(s[1:], s[0])
		if end < 0 {
			return "", "", false
		}
		rest := s[end+2:]
		if !strings.HasPrefix(rest, ":") {
			return "", "", false
		}
		return s[1 : end+1], strings.TrimSpace(rest[1:]), true
	}
	i := strings.Index(s, ": ")
	if i < 0 {
		if strings.HasSuffix(s, ":") {
			return strings.TrimSpace(s[:len(s)-1]), "", true
		}
		return "", "", false
	}
	return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+2:]), true
}

// yamlValue reads a scalar: quoted strings unquoted, true/false, null, and
// plain text as a string (numbers too: callers read what they expect).
func yamlValue(s string) any {
	switch {
	case s == "" || s == "~" || s == "null" || s == "Null" || s == "NULL":
		return nil
	case s == "true" || s == "True" || s == "TRUE":
		return true
	case s == "false" || s == "False" || s == "FALSE":
		return false
	case s == "{}":
		return map[string]any{}
	case s == "[]":
		return []any{}
	case len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'':
		return strings.ReplaceAll(s[1:len(s)-1], "''", "'")
	case len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"':
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
		return s[1 : len(s)-1]
	}
	if i := strings.Index(s, " #"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// yamlStr reads a string at a path of keys ("game", "exe"), or "".
func yamlStr(m map[string]any, keys ...string) string {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = mm[k]
	}
	switch v := cur.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	}
	return ""
}
