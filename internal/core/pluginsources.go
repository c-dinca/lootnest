package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// A plugin's library sources (sources/*.json): where another app lists its
// games, read-only, as one more place the scan looks. Kinds:
//
//   - folder: every folder in path is a game, started by its main .exe;
//   - json, vdf: the list is at a dotted key path in the file (an array of
//     objects, or an object of objects), each entry's fields named;
//   - ini: every section is an entry (or those whose name starts with list);
//   - registry: every subkey of a key under HKCU or HKLM\SOFTWARE is an
//     entry, its fields named by value.
//
// A path starts with one of pluginVars and goes on in relative parts; the
// file's name may have * or ? in it.
// ---------------------------------------------------------------------------

type PluginSource struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	List   string `json:"list,omitempty"`
	Fields struct {
		Name    string `json:"name"`
		Exe     string `json:"exe"`
		Dir     string `json:"dir"`
		SteamID string `json:"steamId"`
	} `json:"fields"`
}

// RegRead reads one registry key, read-only: its subkeys' names and its
// values as text. win.ReadRegKey on Windows, a fake in tests.
type RegRead func(key string) (subkeys []string, values map[string]string, err error)

var pluginVars = []string{"APPDATA", "LOCALAPPDATA", "USERPROFILE", "PROGRAMDATA", "PROGRAMFILES", "PROGRAMFILES(X86)"}

const pluginSourceGames = 2000 // at most this many games from one source

// PluginSources reads and checks a plugin's sources.
func PluginSources(dir string) ([]PluginSource, error) {
	files := pluginFiles(dir, "sources", ".json")
	if len(files) == 0 {
		return nil, errors.New("it provides sources but has no sources/*.json")
	}
	if len(files) > 20 {
		return nil, errors.New("it has more than 20 sources")
	}
	var out []PluginSource
	for _, f := range files {
		name := "sources/" + filepath.Base(f)
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var s PluginSource
		if err := json.Unmarshal(b, &s); err != nil {
			return nil, fmt.Errorf("%s can't be read: %s", name, shortErr(err))
		}
		if err := s.check(); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out = append(out, s)
	}
	return out, nil
}

func (s *PluginSource) check() error {
	s.Name = strings.Join(strings.Fields(s.Name), " ")
	if s.Name == "" || len([]rune(s.Name)) > 60 {
		return errors.New("its name must be 1 to 60 characters")
	}
	if !slices.Contains([]string{"folder", "json", "ini", "vdf", "registry"}, s.Kind) {
		return errors.New("its kind must be folder, json, ini, vdf or registry")
	}
	if s.Kind == "registry" {
		if _, err := regKeyPath(s.Path); err != nil {
			return err
		}
	} else if _, _, err := splitPluginPath(s.Path); err != nil {
		return err
	}
	f := s.Fields
	switch {
	case s.Kind == "folder":
	case f.Exe == "" && f.Dir == "":
		return errors.New("its fields must name the exe or the folder")
	case f.Name == "" && s.Kind != "ini":
		return errors.New("its fields must name the game's name")
	}
	return nil
}

// splitPluginPath checks a source's path: %VAR% from pluginVars, then
// relative parts with no drive and no "..".
func splitPluginPath(p string) (v string, rest []string, err error) {
	bad := errors.New("its path must start with %APPDATA%, %LOCALAPPDATA%, %USERPROFILE%, %PROGRAMDATA%, %PROGRAMFILES% or %PROGRAMFILES(X86)%, then folders inside it")
	if !strings.HasPrefix(p, "%") {
		return "", nil, bad
	}
	end := strings.Index(p[1:], "%")
	if end < 0 {
		return "", nil, bad
	}
	v = strings.ToUpper(p[1 : end+1])
	if !slices.Contains(pluginVars, v) {
		return "", nil, bad
	}
	for _, part := range strings.FieldsFunc(p[end+2:], func(r rune) bool { return r == '\\' || r == '/' }) {
		if part == "." || part == ".." || strings.ContainsAny(part, ":\x00") {
			return "", nil, bad
		}
		rest = append(rest, part)
	}
	for _, part := range rest[:max(0, len(rest)-1)] {
		if strings.ContainsAny(part, "*?[") {
			return "", nil, errors.New("only the file's name in its path may have * or ?")
		}
	}
	return v, rest, nil
}

// regKeyPath checks a registry key: under HKCU, or HKLM\SOFTWARE.
func regKeyPath(p string) (string, error) {
	p = strings.Trim(strings.ReplaceAll(p, "/", `\`), `\`)
	parts := strings.Split(p, `\`)
	u := strings.ToUpper(p)
	if !(strings.HasPrefix(u, `HKCU\`) || strings.HasPrefix(u, `HKLM\SOFTWARE\`)) || len(parts) < 2 {
		return "", errors.New(`its registry key must be under HKCU\ or HKLM\SOFTWARE\`)
	}
	for _, part := range parts {
		if part == "" || part == ".." {
			return "", errors.New("its registry key has an empty part")
		}
	}
	return p, nil
}

// PluginGames reads a plugin's sources on this PC. Nothing is written.
func PluginGames(p Plugin, getenv func(string) string, reg RegRead) []FoundGame {
	srcs, err := PluginSources(p.Dir)
	if err != nil {
		return nil
	}
	var out []FoundGame
	for _, s := range srcs {
		for i, e := range s.entries(getenv, reg) {
			if i >= pluginSourceGames {
				break
			}
			if f, ok := s.game(e); ok {
				f.Source, f.SourceName = "plugin:"+p.ID, s.Name
				out = append(out, f)
			}
		}
	}
	return out
}

// entry is one game's fields as the source lists them.
type entry struct{ name, exe, dir, steam string }

func (s PluginSource) entries(getenv func(string) string, reg RegRead) []entry {
	if s.Kind == "registry" {
		return s.regEntries(reg)
	}
	v, rest, err := splitPluginPath(s.Path)
	base := getenv(v)
	if err != nil || base == "" || !filepath.IsAbs(base) {
		return nil
	}
	path := filepath.Join(append([]string{base}, rest...)...)
	if s.Kind == "folder" {
		return folderEntries(path)
	}
	files, _ := filepath.Glob(path)
	var out []entry
	for _, f := range files {
		b, err := readCapped(f, 16<<20)
		if err != nil {
			continue
		}
		switch s.Kind {
		case "json":
			var v any
			if json.Unmarshal(b, &v) == nil {
				out = append(out, s.fromTree(listAt(v, s.List, false))...)
			}
		case "vdf":
			if v, err := ParseVDF(string(b)); err == nil {
				out = append(out, s.fromTree(listAt(v, s.List, true))...)
			}
		case "ini":
			for sec, kv := range parseINI(b) {
				if !strings.HasPrefix(strings.ToLower(sec), strings.ToLower(s.List)) {
					continue
				}
				e := entry{name: iget(kv, s.Fields.Name), exe: iget(kv, s.Fields.Exe), dir: iget(kv, s.Fields.Dir), steam: iget(kv, s.Fields.SteamID)}
				if s.Fields.Name == "" {
					e.name = sec
				}
				out = append(out, e)
			}
		}
	}
	return out
}

func readCapped(p string, n int64) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, n))
}

func folderEntries(dir string) []entry {
	ents, _ := os.ReadDir(dir)
	var out []entry
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, entry{name: prettyName(e.Name()), dir: filepath.Join(dir, e.Name())})
		}
	}
	return out
}

func (s PluginSource) regEntries(reg RegRead) []entry {
	key, err := regKeyPath(s.Path)
	if err != nil || reg == nil {
		return nil
	}
	subs, _, err := reg(key)
	if err != nil {
		return nil
	}
	var out []entry
	for i, sub := range subs {
		if i >= pluginSourceGames {
			break
		}
		_, kv, err := reg(key + `\` + sub)
		if err != nil {
			continue
		}
		out = append(out, entry{name: iget(kv, s.Fields.Name), exe: iget(kv, s.Fields.Exe), dir: iget(kv, s.Fields.Dir), steam: iget(kv, s.Fields.SteamID)})
	}
	return out
}

// listAt walks a dotted key path down a parsed JSON or VDF file to the
// list of entries: an array of objects, or an object's object values.
func listAt(v any, path string, lower bool) []map[string]any {
	for _, k := range strings.Split(path, ".") {
		if k == "" {
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		if lower {
			k = strings.ToLower(k)
		}
		v = m[k]
	}
	var out []map[string]any
	switch x := v.(type) {
	case []any:
		for _, it := range x {
			if m, ok := it.(map[string]any); ok {
				out = append(out, m)
			}
		}
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if m, ok := x[k].(map[string]any); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func (s PluginSource) fromTree(list []map[string]any) []entry {
	field := func(m map[string]any, k string) string {
		if k == "" {
			return ""
		}
		for kk, v := range m {
			if strings.EqualFold(kk, k) {
				switch x := v.(type) {
				case string:
					return x
				case float64:
					return strconv.FormatFloat(x, 'f', -1, 64)
				}
			}
		}
		return ""
	}
	out := make([]entry, 0, len(list))
	for _, m := range list {
		out = append(out, entry{name: field(m, s.Fields.Name), exe: field(m, s.Fields.Exe), dir: field(m, s.Fields.Dir), steam: field(m, s.Fields.SteamID)})
	}
	return out
}

// game turns an entry into a Library game, when its folder and .exe are
// on this PC.
func (s PluginSource) game(e entry) (FoundGame, bool) {
	name := strings.Join(strings.Fields(e.name), " ")
	if r := []rune(name); len(r) > 200 {
		name = string(r[:200])
	}
	dir, exe := cleanAbs(e.dir), strings.TrimSpace(e.exe)
	if exe != "" && !filepath.IsAbs(filepath.FromSlash(exe)) {
		if dir == "" {
			return FoundGame{}, false
		}
		rel, err := updateRel(exe)
		if err != nil {
			return FoundGame{}, false
		}
		exe = filepath.Join(dir, filepath.FromSlash(rel))
	} else {
		exe = cleanAbs(exe)
	}
	if exe == "" && dir != "" {
		exe, _ = MainExe(dir)
	}
	if dir == "" && exe != "" {
		dir = filepath.Dir(exe)
	}
	if name == "" || exe == "" || !strings.EqualFold(filepath.Ext(exe), ".exe") || !isFile(exe) || !isDir(dir) || tooBroadDir(dir) {
		return FoundGame{}, false
	}
	f := FoundGame{Key: "dir:" + strings.ToLower(dir), Name: name, Exe: exe, Dir: dir,
		ProcessName: strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))}
	if n, err := strconv.Atoi(strings.TrimSpace(e.steam)); err == nil && n > 0 {
		f.SteamID = n
	}
	return f, true
}

func cleanAbs(p string) string {
	p = filepath.FromSlash(strings.Trim(strings.TrimSpace(p), `"`))
	if p == "" || !filepath.IsAbs(p) {
		return ""
	}
	return filepath.Clean(p)
}

// DropPluginGames leaves out the games of plugins that aren't on (by id),
// so turning a plugin off takes its games out of the scan's results.
func DropPluginGames(list []FoundGame, on map[string]bool) []FoundGame {
	out := list[:0:0]
	for _, f := range list {
		if id, ok := strings.CutPrefix(f.Source, "plugin:"); !ok || on[id] {
			out = append(out, f)
		}
	}
	return out
}
