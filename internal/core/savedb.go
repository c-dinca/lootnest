package core

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Where games keep their saves, from the Ludusavi manifest
// (github.com/mtkennerly/ludusavi-manifest), which collects them from
// PCGamingWiki for tens of thousands of games. Downloaded only when the user
// turns on "Find save locations online", then kept here as a small index
// (savedb.json.gz) of the Windows locations of each game.
// ---------------------------------------------------------------------------

// SaveDBURL is the manifest Lootnest downloads.
const SaveDBURL = "https://raw.githubusercontent.com/mtkennerly/ludusavi-manifest/master/data/manifest.yaml"

// DBGame is one game's Windows save and settings locations, with Ludusavi's
// placeholders (<winAppData>, <base>, ...).
type DBGame struct {
	Name  string   `json:"n"`
	Steam int      `json:"s,omitempty"`
	GOG   int      `json:"g,omitempty"`
	Dirs  []string `json:"d,omitempty"` // install folder names
	Files []string `json:"f"`
}

// SaveDB is the index, with lookups by name and store id.
type SaveDB struct {
	Games   []DBGame `json:"games"`
	byName  map[string]int
	bySteam map[int]int
	byGOG   map[int]int
}

// placeholders Lootnest can put a Windows path to.
var dbPlaceholders = []string{"<base>", "<game>", "<storeUserId>", "<osUserName>", "<winPublic>", "<winDir>", "<winAppData>",
	"<winLocalAppData>", "<winLocalAppDataLow>", "<winDocuments>", "<winProgramData>", "<home>"}

func windowsPath(p string) bool {
	if strings.Contains(p, "<root>") {
		return false // a store's own folder this PC may not have
	}
	rest := p
	for _, ph := range dbPlaceholders {
		rest = strings.ReplaceAll(rest, ph, "")
	}
	return !strings.Contains(rest, "<")
}

// ParseSaveDB reads the Ludusavi manifest into the index. The manifest is
// machine-made YAML with a fixed layout (2-space indents, block style), so it
// is read line by line: a few MB of memory and a second or two instead of
// the whole document in memory.
func ParseSaveDB(r io.Reader) (*SaveDB, error) {
	type file struct {
		tags   []string
		whens  []string // each "when" entry's os ("" for any)
		inWhen bool
	}
	db := &SaveDB{}
	var cur *DBGame
	var files map[string]*file
	var fileOrder []string
	var curFile *file
	section := ""
	flush := func() {
		if cur == nil {
			return
		}
		for _, p := range fileOrder {
			f := files[p]
			win := len(f.whens) == 0
			for _, w := range f.whens {
				win = win || w == "" || w == "windows"
			}
			keep := len(f.tags) == 0
			for _, t := range f.tags {
				keep = keep || t == "save" || t == "config"
			}
			if win && keep && windowsPath(p) && len(p) < 400 {
				cur.Files = append(cur.Files, p)
			}
		}
		if len(cur.Files) > 0 {
			sort.Strings(cur.Files)
			sort.Strings(cur.Dirs)
			db.Games = append(db.Games, *cur)
		}
		cur = nil
	}
	sc := bufio.NewScanner(io.LimitReader(r, 200<<20))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	lines := 0
	for sc.Scan() {
		line := sc.Text()
		lines++
		if strings.TrimSpace(line) == "" || line == "---" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		text := line[indent:]
		switch {
		case indent == 0:
			flush()
			name, rest, ok := yamlKey(text)
			if !ok || rest != "" {
				return nil, fmt.Errorf("unexpected line %d in the save locations list", lines)
			}
			cur = &DBGame{Name: name}
			files, fileOrder, curFile, section = map[string]*file{}, nil, nil, ""
		case cur == nil:
			continue
		case indent == 2:
			key, _, _ := yamlKey(text)
			section = key
		case section == "files" && indent == 4:
			key, _, ok := yamlKey(text)
			if !ok {
				continue
			}
			curFile = &file{}
			if _, dup := files[key]; !dup {
				fileOrder = append(fileOrder, key)
			}
			files[key] = curFile
		case section == "files" && indent == 6 && curFile != nil:
			key, _, _ := yamlKey(text)
			curFile.inWhen = key == "when"
		case section == "files" && indent == 8 && curFile != nil && strings.HasPrefix(text, "- "):
			item := strings.TrimSpace(text[2:])
			if !curFile.inWhen {
				curFile.tags = append(curFile.tags, yamlScalar(item))
				continue
			}
			curFile.whens = append(curFile.whens, "")
			if k, v, ok := yamlKey(item); ok && k == "os" {
				curFile.whens[len(curFile.whens)-1] = yamlScalar(v)
			}
		case section == "files" && indent == 10 && curFile != nil && curFile.inWhen && len(curFile.whens) > 0:
			if k, v, ok := yamlKey(text); ok && k == "os" {
				curFile.whens[len(curFile.whens)-1] = yamlScalar(v)
			}
		case section == "installDir" && indent == 4:
			if k, _, ok := yamlKey(text); ok {
				cur.Dirs = append(cur.Dirs, k)
			}
		case (section == "steam" || section == "gog") && indent == 4:
			if k, v, ok := yamlKey(text); ok && k == "id" {
				n, _ := strconv.Atoi(yamlScalar(v))
				if section == "steam" {
					cur.Steam = n
				} else {
					cur.GOG = n
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, errors.New("The save locations list couldn't be read.")
	}
	flush()
	if len(db.Games) == 0 {
		return nil, errors.New("The save locations list is empty.")
	}
	sort.Slice(db.Games, func(i, j int) bool { return db.Games[i].Name < db.Games[j].Name })
	db.index()
	return db, nil
}

// yamlKey splits "key: value" (the key bare or double-quoted).
func yamlKey(s string) (key, rest string, ok bool) {
	if strings.HasPrefix(s, `"`) {
		end := 1
		for end < len(s) {
			if s[end] == '\\' {
				end += 2
				continue
			}
			if s[end] == '"' {
				break
			}
			end++
		}
		if end >= len(s) {
			return "", "", false
		}
		k, err := strconv.Unquote(s[:end+1])
		if err != nil {
			return "", "", false
		}
		after := s[end+1:]
		if !strings.HasPrefix(after, ":") {
			return "", "", false
		}
		return k, strings.TrimSpace(after[1:]), true
	}
	i := strings.Index(s, ": ")
	if i < 0 {
		if strings.HasSuffix(s, ":") {
			return s[:len(s)-1], "", true
		}
		return "", "", false
	}
	return s[:i], strings.TrimSpace(s[i+2:]), true
}

// yamlScalar unquotes a scalar value if it's quoted.
func yamlScalar(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, `"`) {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
	}
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") && len(s) >= 2 {
		return strings.ReplaceAll(s[1:len(s)-1], "''", "'")
	}
	return s
}

func (db *SaveDB) index() {
	db.byName, db.bySteam, db.byGOG = map[string]int{}, map[int]int{}, map[int]int{}
	for i, g := range db.Games {
		if k := norm(g.Name); k != "" {
			if _, ok := db.byName[k]; !ok {
				db.byName[k] = i
			}
		}
		for _, d := range g.Dirs {
			if k := norm(d); k != "" {
				if _, ok := db.byName[k]; !ok {
					db.byName[k] = i
				}
			}
		}
		if g.Steam > 0 {
			db.bySteam[g.Steam] = i
		}
		if g.GOG > 0 {
			db.byGOG[g.GOG] = i
		}
	}
}

// Find looks a game up by Steam id, then by its name or install folder.
func (db *SaveDB) Find(steamID int, names ...string) *DBGame {
	if db == nil {
		return nil
	}
	if i, ok := db.bySteam[steamID]; ok && steamID > 0 {
		return &db.Games[i]
	}
	for _, n := range names {
		if i, ok := db.byName[norm(n)]; ok && norm(n) != "" {
			return &db.Games[i]
		}
	}
	return nil
}

// SaveDBFile writes the index (gzip JSON) and reads it back.
func SaveDBFile(path string, db *SaveDB) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path + ".tmp")
	if err != nil {
		return err
	}
	zw := gzip.NewWriter(f)
	err = json.NewEncoder(zw).Encode(db)
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path + ".tmp")
		return err
	}
	return os.Rename(path+".tmp", path)
}

func LoadSaveDB(path string) (*SaveDB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	var db SaveDB
	if err := json.NewDecoder(io.LimitReader(zr, 200<<20)).Decode(&db); err != nil {
		return nil, err
	}
	db.index()
	return &db, nil
}

// --------------------------------------------------------- on this PC ---

// SaveLocation is a game's save folder or file that exists on this PC.
type SaveLocation struct {
	Path     string `json:"path"`     // tokenized, the same on every PC
	Resolved string `json:"resolved"` // here
	Size     int64  `json:"size"`
}

// FindSaves turns a game's locations into the folders and files that exist
// here: base is the game's install folder.
func (e *Env) FindSaves(g *DBGame, base string) []SaveLocation {
	get := e.Getenv
	if get == nil {
		get = os.Getenv
	}
	home := e.Home
	repl := []string{
		"<winAppData>", get("APPDATA"),
		"<winLocalAppDataLow>", e.LocalLow,
		"<winLocalAppData>", get("LOCALAPPDATA"),
		"<winDocuments>", e.Documents,
		"<winPublic>", get("PUBLIC"),
		"<winProgramData>", get("ProgramData"),
		"<winDir>", get("WINDIR"),
		"<home>", home,
		"<osUserName>", get("USERNAME"),
		"<storeUserId>", "*",
	}
	seen := map[string]bool{}
	var out []SaveLocation
	for _, f := range g.Files {
		if strings.Contains(f, "<base>") || strings.Contains(f, "<game>") {
			if base == "" {
				continue
			}
			f = strings.ReplaceAll(f, "<base>", base)
			f = strings.ReplaceAll(f, "<game>", filepath.Base(base))
		}
		p := strings.NewReplacer(repl...).Replace(f)
		if strings.Contains(p, "<") || !filepath.IsAbs(filepath.FromSlash(p)) {
			continue
		}
		for _, m := range globAll(filepath.FromSlash(p)) {
			if e.CheckSafe(m) != nil {
				continue // never a whole profile folder, AppData itself, ...
			}
			key := strings.ToLower(m)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, SaveLocation{Path: e.Tokenize(m), Resolved: m, Size: Size(m)})
		}
	}
	// A folder and a file inside it: keep the folder.
	var kept []SaveLocation
	for _, a := range out {
		inside := false
		for _, b := range out {
			if a.Resolved != b.Resolved && isWithin(a.Resolved, b.Resolved) {
				inside = true
			}
		}
		if !inside {
			kept = append(kept, a)
		}
	}
	return kept
}

// globAll expands * and ** (any folders) in a Windows path into what
// exists, at most 200 matches.
func globAll(pattern string) []string {
	if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") {
		if exists(pattern) {
			return []string{pattern}
		}
		return nil
	}
	sep := string(filepath.Separator)
	vol := filepath.VolumeName(pattern)
	parts := strings.Split(strings.TrimPrefix(pattern[len(vol):], sep), sep)
	var out []string
	var walk func(dir string, i int)
	walk = func(dir string, i int) {
		if len(out) >= 200 {
			return
		}
		if i == len(parts) {
			if exists(dir) {
				out = append(out, dir)
			}
			return
		}
		part := parts[i]
		if part == "**" {
			walk(dir, i+1)
			_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
				if err != nil || len(out) >= 200 {
					return filepath.SkipDir
				}
				if d.IsDir() && p != dir {
					walk(p, i+1)
				}
				return nil
			})
			return
		}
		if !strings.ContainsAny(part, "*?[") {
			walk(filepath.Join(dir, part), i+1)
			return
		}
		ents, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range ents {
			if ok, _ := filepath.Match(strings.ToLower(part), strings.ToLower(e.Name())); ok {
				walk(filepath.Join(dir, e.Name()), i+1)
			}
		}
	}
	walk(vol+sep, 0)
	return out
}
