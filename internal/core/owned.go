package core

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Games you own but haven't installed, from the files the stores keep on
// this PC: nothing is signed in to and nothing goes online.
//
// Steam: the apps in the account's localconfig.vdf and its library cache
// (userdata\<id>\config\librarycache\<appid>.json), named and typed from
// Steam's appcache\appinfo.vdf, games only.
// Epic: the launcher's catalog cache (Data\Catalog\catcache.bin), base64 of
// the catalog items of the account's library, games only (no add-ons).
// ---------------------------------------------------------------------------

// OwnedGame is a game a store says you have, not installed on this PC.
type OwnedGame struct {
	Key     string `json:"key"` // "owned:steam:620", "owned:epic:<app name>", "owned:itch:<game id>"
	Name    string `json:"name"`
	Source  string `json:"source"`
	AppID   int    `json:"appId,omitempty"`
	StoreID string `json:"storeId,omitempty"` // Epic's, for StoreAction; itch.io's game id
	KeyID   int64  `json:"keyId,omitempty"`   // itch.io's download key
	Cover   string `json:"cover,omitempty"`   // itch.io's cover picture
}

// SteamOwned lists the games of the Steam account in userDir (Steam\
// userdata\<id>) that aren't in installed.
func SteamOwned(steamPath, userDir string, installed map[int]bool) []OwnedGame {
	if steamPath == "" || userDir == "" {
		return nil
	}
	want := map[int]bool{}
	for id := range steamConfigApps(userDir) {
		want[id] = true
	}
	cache, _ := filepath.Glob(filepath.Join(userDir, "config", "librarycache", "*.json"))
	for _, p := range cache {
		if id, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(p), ".json")); err == nil && id > 0 {
			want[id] = true
		}
	}
	for id := range installed {
		delete(want, id)
	}
	var out []OwnedGame
	for id, app := range steamAppInfo(filepath.Join(steamPath, "appcache", "appinfo.vdf"), want) {
		if app.name != "" && strings.EqualFold(app.kind, "game") {
			out = append(out, OwnedGame{Key: "owned:steam:" + strconv.Itoa(id), Name: app.name, Source: "steam", AppID: id})
		}
	}
	sortOwned(out)
	return out
}

// steamConfigApps is every app in localconfig.vdf's apps section.
func steamConfigApps(userDir string) map[int]bool {
	out := map[int]bool{}
	b, err := os.ReadFile(filepath.Join(userDir, "config", "localconfig.vdf"))
	if err != nil {
		return out
	}
	m, err := ParseVDF(string(b))
	if err != nil {
		return out
	}
	for k := range vdfMap(vdfMap(vdfMap(vdfMap(vdfMap(m, "UserLocalConfigStore"), "Software"), "Valve"), "Steam"), "apps") {
		if id, err := strconv.Atoi(k); err == nil && id > 0 {
			out[id] = true
		}
	}
	return out
}

type steamApp struct{ name, kind string }

// steamAppInfo reads the name and type of the apps in want from Steam's
// binary appinfo.vdf (versions 27 to 29); the others are skipped unread.
func steamAppInfo(path string, want map[int]bool) map[int]steamApp {
	out := map[int]steamApp{}
	f, err := os.Open(path)
	if err != nil || len(want) == 0 {
		return out
	}
	defer f.Close()
	le := binary.LittleEndian
	var hdr [8]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return out
	}
	skip, var29 := 60, false // each app's fixed fields before its data
	switch le.Uint32(hdr[:]) {
	case 0x07564427:
		skip = 40
	case 0x07564428:
	case 0x07564429:
		var29 = true
	default:
		return out
	}
	var strs []string
	if var29 {
		var off int64
		if binary.Read(f, le, &off) != nil {
			return out
		}
		if strs = appInfoStrings(f, off); strs == nil {
			return out
		}
		if _, err := f.Seek(16, io.SeekStart); err != nil {
			return out
		}
	}
	r := bufio.NewReader(f)
	for {
		var head [8]byte
		if _, err := io.ReadFull(r, head[:4]); err != nil {
			return out
		}
		id := le.Uint32(head[:4])
		if id == 0 {
			return out
		}
		if _, err := io.ReadFull(r, head[4:]); err != nil {
			return out
		}
		size := int(le.Uint32(head[4:]))
		if size < skip || size > 64<<20 {
			return out
		}
		if !want[int(id)] {
			if _, err := r.Discard(size); err != nil {
				return out
			}
			continue
		}
		b := make([]byte, size)
		if _, err := io.ReadFull(r, b); err != nil {
			return out
		}
		p := &binVDF{b: b[skip:], strs: strs, v29: var29}
		if m, ok := p.obj(0); ok {
			common := vdfMap(vdfMap(m, "appinfo"), "common")
			out[int(id)] = steamApp{vdfStr(common, "name"), vdfStr(common, "type")}
		}
	}
}

// appInfoStrings reads appinfo.vdf 29's table of key names at off.
func appInfoStrings(f *os.File, off int64) []string {
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return nil
	}
	r := bufio.NewReader(f)
	var n uint32
	if binary.Read(r, binary.LittleEndian, &n) != nil || n > 1<<20 {
		return nil
	}
	out := make([]string, 0, n)
	for i := uint32(0); i < n; i++ {
		s, err := r.ReadString(0)
		if err != nil {
			return nil
		}
		out = append(out, strings.TrimSuffix(s, "\x00"))
	}
	return out
}

// binVDF reads Steam's binary key-values: a type byte, the key (a string,
// or in appinfo 29 an index into its table), then the value.
type binVDF struct {
	b    []byte
	i    int
	strs []string
	v29  bool
}

func (p *binVDF) cstr() (string, bool) {
	j := bytes.IndexByte(p.b[p.i:], 0)
	if j < 0 {
		return "", false
	}
	s := string(p.b[p.i : p.i+j])
	p.i += j + 1
	return s, true
}

func (p *binVDF) key() (string, bool) {
	if !p.v29 {
		k, ok := p.cstr()
		return strings.ToLower(k), ok
	}
	if p.i+4 > len(p.b) {
		return "", false
	}
	n := binary.LittleEndian.Uint32(p.b[p.i:])
	p.i += 4
	if int(n) >= len(p.strs) {
		return "", false
	}
	return strings.ToLower(p.strs[n]), true
}

func (p *binVDF) obj(depth int) (map[string]any, bool) {
	m := map[string]any{}
	for p.i < len(p.b) {
		t := p.b[p.i]
		p.i++
		if t == 8 {
			return m, true
		}
		k, ok := p.key()
		if !ok {
			return m, false
		}
		switch t {
		case 0:
			if depth > 32 {
				return m, false
			}
			sub, ok := p.obj(depth + 1)
			if !ok {
				return m, false
			}
			m[k] = sub
		case 1:
			s, ok := p.cstr()
			if !ok {
				return m, false
			}
			m[k] = s
		case 2, 3, 4, 6: // int32, float, pointer, colour
			p.i += 4
		case 7, 10: // uint64, int64
			p.i += 8
		default:
			return m, false
		}
	}
	return m, false
}

// EpicOwned lists the games in Epic's catalog cache whose id isn't in
// installed (EpicID of the installed ones).
func EpicOwned(catcache string, installed map[string]bool) []OwnedGame {
	b, err := os.ReadFile(catcache)
	if err != nil || len(b) > 64<<20 {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil {
		return nil
	}
	var items []struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		Namespace  string `json:"namespace"`
		Categories []struct {
			Path string `json:"path"`
		} `json:"categories"`
		ReleaseInfo []struct {
			AppID string `json:"appId"`
		} `json:"releaseInfo"`
		MainGameItem json.RawMessage `json:"mainGameItem"`
	}
	if json.Unmarshal(raw, &items) != nil {
		return nil
	}
	var out []OwnedGame
	seen := map[string]bool{}
	for _, it := range items {
		game := false
		for _, c := range it.Categories {
			game = game || c.Path == "games"
		}
		if !game || len(it.MainGameItem) > 0 && string(it.MainGameItem) != "null" || len(it.ReleaseInfo) == 0 || it.ReleaseInfo[0].AppID == "" || strings.TrimSpace(it.Title) == "" {
			continue
		}
		app := it.ReleaseInfo[0].AppID
		id := EpicID(it.Namespace, it.ID, app)
		if installed[id] || seen[id] || StoreAction("epic", id, "install") == "" {
			continue
		}
		seen[id] = true
		out = append(out, OwnedGame{Key: "owned:epic:" + app, Name: strings.TrimSpace(it.Title), Source: "epic", StoreID: id})
	}
	sortOwned(out)
	return out
}

func sortOwned(l []OwnedGame) {
	sort.Slice(l, func(i, j int) bool { return strings.ToLower(l[i].Name) < strings.ToLower(l[j].Name) })
}
