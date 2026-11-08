package core

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Import from Playnite, one way: Lootnest reads Playnite's library on this
// PC and never writes to it. Playnite 9 and later keep the library in
// <Playnite>\library as LiteDB 4 files (file format 7): games.db, and one
// file per kind of item it links to (completionstatuses.db, categories.db,
// tags.db). Playnite opens them exclusively, so they can't be read while
// it runs.
// ---------------------------------------------------------------------------

// PlayniteGame is what the import uses of one of Playnite's games.
type PlayniteGame struct {
	Name         string
	Store        string // steam, epic or gog, from Playnite's library plugin
	GameID       string // the game's id in that store: Steam's app id, Epic's app name, GOG's product id
	Playtime     int64  // seconds
	LastActivity time.Time
	Favorite     bool
	Status       string // the completion status's name: "Beaten", "Plan to Play"
	Categories   []string
	Tags         []string
	UserScore    int // 0 to 100, 0 for none
	Notes        string
	InstallDir   string
}

// The library plugins whose games Lootnest matches by store id.
var playnitePlugins = map[string]string{
	"cb91dfc9-b977-43bf-8e70-55f46e410fab": "steam",
	"00000002-dbd1-46c6-b5d0-b1ba559d10e4": "epic",
	"aebe8b7c-6dc3-4a66-af31-e7375c6b5e9e": "gog",
}

// PlayniteLibraryDir is the folder holding games.db: dir\library (an
// installed or portable Playnite's own folder) or dir itself. "" if neither.
func PlayniteLibraryDir(dir string) string {
	if dir == "" {
		return ""
	}
	for _, d := range []string{filepath.Join(dir, "library"), dir} {
		if st, err := os.Stat(filepath.Join(d, "games.db")); err == nil && st.Mode().IsRegular() {
			return d
		}
	}
	return ""
}

// ReadPlaynite reads the games in a Playnite library folder (see
// PlayniteLibraryDir), with the names of their statuses, categories and
// tags. It returns the library folder it read.
func ReadPlaynite(dir string) (string, []PlayniteGame, error) {
	lib := PlayniteLibraryDir(dir)
	if lib == "" {
		if st, err := os.Stat(filepath.Join(dir, "library", "games")); err == nil && st.IsDir() {
			return "", nil, errors.New("This Playnite library is from before Playnite 9. Update Playnite, open it once, then try again.")
		}
		return "", nil, errors.New("There's no Playnite library in this folder. Choose Playnite's own folder, the one with Playnite.DesktopApp.exe.")
	}
	docs, err := readLiteDB(filepath.Join(lib, "games.db"))
	if err != nil {
		return "", nil, err
	}
	names := func(file string) map[string]string {
		out := map[string]string{}
		list, _ := readLiteDB(filepath.Join(lib, file)) // missing or unreadable: the games come without these
		for _, d := range list {
			if id, name := bsonStr(d, "_id"), strings.TrimSpace(bsonStr(d, "Name")); id != "" && name != "" {
				out[id] = name
			}
		}
		return out
	}
	statuses, cats, tags := names("completionstatuses.db"), names("categories.db"), names("tags.db")
	each := func(d map[string]any, key string, m map[string]string) []string {
		var out []string
		list, _ := d[key].([]any)
		for _, v := range list {
			if id, _ := v.(string); m[id] != "" && !slices.Contains(out, m[id]) {
				out = append(out, m[id])
			}
		}
		return out
	}
	// {PlayniteDir} is Playnite's program folder: beside the library when
	// portable, else where the installer put it. Unknown, a path with it
	// isn't matched by folder.
	root := ""
	dirs := []string{filepath.Dir(lib)}
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		dirs = append(dirs, filepath.Join(la, "Playnite"))
	}
	for _, d := range dirs {
		if st, err := os.Stat(filepath.Join(d, "Playnite.DesktopApp.exe")); err == nil && st.Mode().IsRegular() {
			root = d
			break
		}
	}
	var games []PlayniteGame
	for _, d := range docs {
		g := PlayniteGame{
			Name: strings.TrimSpace(bsonStr(d, "Name")), Store: playnitePlugins[bsonStr(d, "PluginId")], GameID: strings.TrimSpace(bsonStr(d, "GameId")),
			Playtime: max(0, bsonInt(d, "Playtime")), Favorite: d["Favorite"] == true, Status: statuses[bsonStr(d, "CompletionStatusId")],
			Categories: each(d, "CategoryIds", cats), Tags: each(d, "TagIds", tags), UserScore: int(min(max(bsonInt(d, "UserScore"), 0), 100)),
			Notes: strings.TrimSpace(bsonStr(d, "Notes")), InstallDir: bsonStr(d, "InstallDirectory"),
		}
		if strings.Contains(g.InstallDir, "{PlayniteDir}") {
			g.InstallDir = strings.ReplaceAll(g.InstallDir, "{PlayniteDir}", root)
			if root == "" {
				g.InstallDir = ""
			}
		}
		if t, ok := d["LastActivity"].(time.Time); ok {
			g.LastActivity = t
		}
		if g.Name != "" {
			games = append(games, g)
		}
	}
	slices.SortFunc(games, func(a, b PlayniteGame) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return lib, games, nil
}

// --------------------------------------------------------------- LiteDB --
// A read-only reader for LiteDB 4 files, after LiteDB's own reader of the
// format (FileReaderV7): page 0 lists the collections, each collection's
// first index (_id) is a skip list over index pages whose nodes point at
// data blocks, and a large document goes on in a chain of extend pages.

const (
	litePage    = 4096
	liteHead    = 25 // page header: id, type, prev, next, item count, free bytes, 8 reserved
	liteMaxFile = 512 << 20
	liteNone    = math.MaxUint32
)

var errLiteBad = errors.New("Playnite's library file looks damaged, so Lootnest left it alone. Open Playnite once to let it repair the file, then try again.")

func readLiteDB(path string) ([]map[string]any, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Size() > liteMaxFile {
		return nil, errors.New("Playnite's library file is too large to read.")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Lootnest couldn't read Playnite's library. If Playnite is open, close it, then try again. (%w)", err)
	}
	return liteDocs(b)
}

func litePageAt(b []byte, id uint32) []byte {
	if id == liteNone || int64(id+1)*litePage > int64(len(b)) {
		return nil
	}
	return b[int(id)*litePage : int(id+1)*litePage]
}

// liteString reads a length-prefixed UTF-8 string at p[i:].
func liteString(p []byte, i int) (string, int, bool) {
	if i+4 > len(p) {
		return "", 0, false
	}
	n := int(int32(binary.LittleEndian.Uint32(p[i:])))
	if n < 0 || i+4+n > len(p) {
		return "", 0, false
	}
	return string(p[i+4 : i+4+n]), i + 4 + n, true
}

func liteDocs(b []byte) ([]map[string]any, error) {
	h := litePageAt(b, 0)
	if h == nil || h[4] != 1 || string(h[liteHead:liteHead+27]) != "** This is a LiteDB file **" {
		return nil, errors.New("This isn't a Playnite library file.")
	}
	if h[52] != 7 {
		return nil, errors.New("This Playnite library is from a version Lootnest can't read yet.")
	}
	// The salt (bytes 85 to 100) is set only on encrypted files.
	if !bytes.Equal(h[85:101], make([]byte, 16)) {
		return nil, errors.New("This Playnite library is encrypted, so Lootnest can't read it.")
	}
	var out []map[string]any
	at := 102
	for range int(h[101]) {
		_, next, ok := liteString(h, at)
		if !ok || next+4 > len(h) {
			return nil, errLiteBad
		}
		docs, err := liteCollection(b, binary.LittleEndian.Uint32(h[next:]))
		if err != nil {
			return nil, err
		}
		out = append(out, docs...)
		at = next + 4
	}
	return out, nil
}

func liteCollection(b []byte, colPage uint32) ([]map[string]any, error) {
	p := litePageAt(b, colPage)
	if p == nil || p[4] != 2 {
		return nil, errLiteBad
	}
	_, at, ok := liteString(p, liteHead) // the collection's name
	if !ok {
		return nil, errLiteBad
	}
	at += 12
	head := uint32(liteNone)
	for range 16 {
		field, next, ok := liteString(p, at)
		if !ok || next+17 > len(p) {
			return nil, errLiteBad
		}
		if field != "" {
			head = binary.LittleEndian.Uint32(p[next+1:])
			break
		}
		at = next + 17 // unique, head page, head node index, tail node, free index page
	}
	if head == liteNone {
		return nil, nil
	}
	type addr struct {
		page uint32
		idx  uint16
	}
	seen := map[addr]bool{}
	var out []map[string]any
	queue, visited := []uint32{head}, map[uint32]bool{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ip := litePageAt(b, id)
		if ip == nil && id != liteNone {
			return nil, errLiteBad // past the end: the file was cut short
		}
		if visited[id] || ip == nil || ip[4] != 3 {
			continue
		}
		visited[id] = true
		at := liteHead
		for range int(binary.LittleEndian.Uint16(ip[13:])) {
			if at+19 > len(ip) {
				return nil, errLiteBad
			}
			levels := int(ip[at+2])
			key := int(binary.LittleEndian.Uint16(ip[at+16:]))
			at += 18 + 1 + key // index, levels, slot, prev and next nodes, key length, type, key
			if levels < 1 || at+18+(levels-1)*12 > len(ip) {
				return nil, errLiteBad
			}
			data := addr{binary.LittleEndian.Uint32(ip[at:]), binary.LittleEndian.Uint16(ip[at+4:])}
			for _, link := range []uint32{binary.LittleEndian.Uint32(ip[at+6:]), binary.LittleEndian.Uint32(ip[at+12:])} {
				if !visited[link] {
					queue = append(queue, link)
				}
			}
			at += 18 + (levels-1)*12
			if data.page == liteNone || seen[data] {
				continue
			}
			seen[data] = true
			raw, err := liteBlock(b, data.page, data.idx)
			if err != nil {
				return nil, err
			}
			if raw == nil {
				continue
			}
			doc, err := bsonDoc(raw, 0)
			if err != nil {
				return nil, errLiteBad
			}
			out = append(out, doc)
		}
	}
	return out, nil
}

// liteBlock is one document's bytes; nil if its block is gone.
func liteBlock(b []byte, page uint32, idx uint16) ([]byte, error) {
	p := litePageAt(b, page)
	if p == nil {
		return nil, errLiteBad
	}
	if p[4] != 4 {
		return nil, nil
	}
	at := liteHead
	for range int(binary.LittleEndian.Uint16(p[13:])) {
		if at+8 > len(p) {
			return nil, errLiteBad
		}
		n := int(binary.LittleEndian.Uint16(p[at+6:]))
		if at+8+n > len(p) {
			return nil, errLiteBad
		}
		if binary.LittleEndian.Uint16(p[at:]) != idx {
			at += 8 + n
			continue
		}
		ext := binary.LittleEndian.Uint32(p[at+2:])
		if ext == liteNone {
			return p[at+8 : at+8+n], nil
		}
		var data []byte
		for seen := map[uint32]bool{}; ext != liteNone; {
			ep := litePageAt(b, ext)
			if seen[ext] || ep == nil || ep[4] != 5 {
				return nil, errLiteBad
			}
			seen[ext] = true
			n := int(binary.LittleEndian.Uint16(ep[13:]))
			if liteHead+n > len(ep) {
				return nil, errLiteBad
			}
			data = append(data, ep[liteHead:liteHead+n]...)
			ext = binary.LittleEndian.Uint32(ep[9:])
		}
		return data, nil
	}
	return nil, nil
}

// bsonDoc decodes a BSON document: strings, numbers (as int64 or float64),
// booleans, dates (time.Time, UTC), GUIDs (as their usual text), documents
// (map[string]any) and arrays ([]any); other values are nil.
func bsonDoc(b []byte, depth int) (map[string]any, error) {
	out := map[string]any{}
	err := bsonEach(b, depth, func(name string, v any) { out[name] = v })
	return out, err
}

// bsonFixed are the sizes of the BSON values with one.
var bsonFixed = map[byte]int{0x01: 8, 0x07: 12, 0x08: 1, 0x09: 8, 0x0A: 0, 0x10: 4, 0x12: 8, 0x13: 16, 0x7F: 0, 0xFF: 0}

func bsonEach(b []byte, depth int, fn func(string, any)) error {
	bad := errors.New("bad BSON")
	if depth > 16 || len(b) < 5 {
		return bad
	}
	n := int(int32(binary.LittleEndian.Uint32(b)))
	if n < 5 || n > len(b) || b[n-1] != 0 {
		return bad
	}
	b = b[:n-1]
	for at := 4; at < len(b); {
		t := b[at]
		end := bytes.IndexByte(b[at+1:], 0)
		if end < 0 {
			return bad
		}
		name := string(b[at+1 : at+1+end])
		at += 2 + end
		size, ok := bsonFixed[t]
		if !ok {
			if at+4 > len(b) {
				return bad
			}
			size = int(int32(binary.LittleEndian.Uint32(b[at:])))
			switch t {
			case 0x02:
				size += 4
			case 0x05:
				size += 5
			case 0x03, 0x04:
			default:
				return bad
			}
			if size < 5 {
				return bad
			}
		}
		if at+size > len(b) {
			return bad
		}
		v, val := any(nil), b[at:at+size]
		switch t {
		case 0x01:
			v = math.Float64frombits(binary.LittleEndian.Uint64(val))
		case 0x02:
			v = string(val[4 : size-1])
		case 0x03:
			d, err := bsonDoc(val, depth+1)
			if err != nil {
				return err
			}
			v = d
		case 0x04:
			list := []any{}
			if err := bsonEach(val, depth+1, func(_ string, x any) { list = append(list, x) }); err != nil {
				return err
			}
			v = list
		case 0x05:
			if val[4] == 4 && size == 21 {
				v = dotnetGUID(val[5:])
			}
		case 0x08:
			v = val[0] != 0
		case 0x09:
			v = time.UnixMilli(int64(binary.LittleEndian.Uint64(val))).UTC()
		case 0x10:
			v = int64(int32(binary.LittleEndian.Uint32(val)))
		case 0x12:
			v = int64(binary.LittleEndian.Uint64(val))
		}
		fn(name, v)
		at += size
	}
	return nil
}

// dotnetGUID writes a GUID in .NET's byte order (the first three parts
// little-endian) as text, lowercase.
func dotnetGUID(g []byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%x-%x", binary.LittleEndian.Uint32(g), binary.LittleEndian.Uint16(g[4:]), binary.LittleEndian.Uint16(g[6:]), g[8:10], g[10:16])
}

func bsonStr(d map[string]any, k string) string { s, _ := d[k].(string); return s }

func bsonInt(d map[string]any, k string) int64 {
	switch v := d[k].(type) {
	case int64:
		return v
	case float64:
		if v > -1e15 && v < 1e15 {
			return int64(v)
		}
	}
	return 0
}

// ----------------------------------------------------------------- plan --

// PlayniteOptions are what the user chose to bring in.
type PlayniteOptions struct {
	Playtime   bool `json:"playtime"` // with when it was last played
	Statuses   bool `json:"statuses"`
	Favourites bool `json:"favourites"`
	Ratings    bool `json:"ratings"`
	Notes      bool `json:"notes"`
	Categories bool `json:"categories"`
	Tags       bool `json:"tags"`
}

// PlayniteHave is what Lootnest has for the profile the import is for.
type PlayniteHave struct {
	Shelf    map[string]ShelfEntry // the profile's, by Library key
	Imported map[string]int64      // Playnite time already brought in, by key
	Played   map[string]time.Time  // when the profile last played, by key
	Tags     map[string][]string   // Config.Tags
}

// PlayniteChange is what an import would change for one Library game.
type PlayniteChange struct {
	Key        string    `json:"key"`
	Name       string    `json:"name"`            // in Lootnest
	From       string    `json:"from"`            // in Playnite
	By         string    `json:"by"`              // store, folder or name
	Add        int64     `json:"add,omitempty"`   // seconds added to the time played
	Total      int64     `json:"total,omitempty"` // Playnite's time, remembered so it's added once
	LastPlayed time.Time `json:"lastPlayed,omitzero"`
	Status     string    `json:"status,omitempty"`
	Stars      int       `json:"stars,omitempty"`
	Note       string    `json:"note,omitempty"`
	Favourite  bool      `json:"favourite,omitempty"`
	Categories []string  `json:"categories,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
}

// PlaynitePlan is an import before it's made: the matched games with
// something to bring in, and Playnite's games that aren't on this PC.
type PlaynitePlan struct {
	Changes   []PlayniteChange `json:"changes"`
	Matched   int              `json:"matched"`
	Unmatched []string         `json:"unmatched"`
}

var playniteStatuses = map[string]string{
	"playing": "playing", "beaten": "finished", "completed": "complete", "abandoned": "stopped", "plantoplay": "next", "onhold": "next",
}

// PlayniteStars turns Playnite's score (1 to 100) into 1 to 5 stars.
func PlayniteStars(score int) int {
	if score <= 0 {
		return 0
	}
	return min(5, (score+19)/20)
}

func importPath(p string) string {
	return strings.TrimRight(strings.ToLower(strings.ReplaceAll(p, "/", `\`)), `\`)
}

func epicApp(storeID string) string {
	i := strings.LastIndex(storeID, "%3A")
	if i < 0 {
		return ""
	}
	return storeID[i+3:]
}

// PlayniteImport matches Playnite's games to the Library's (lib, by Key):
// by store and id first, then by install folder, then by name (CopyName).
// Each game on either side is matched once. Only what the profile doesn't
// have is changed: time played grows by what Playnite has beyond what an
// earlier import brought in, a status, rating or note is set where there's
// none, last played only moves later, and collections are only added.
func PlayniteImport(games []PlayniteGame, lib []FoundGame, have PlayniteHave) PlaynitePlan {
	byStore, byDir, byName := map[string]int{}, map[string]int{}, map[string]int{}
	for i, f := range lib {
		if id := steamIDOf(f); id > 0 {
			byStore["steam:"+strconv.Itoa(id)] = i
		}
		switch f.Source {
		case "epic":
			if app := epicApp(f.StoreID); app != "" {
				byStore["epic:"+strings.ToLower(app)] = i
			}
		case "gog":
			if f.StoreID != "" {
				byStore["gog:"+f.StoreID] = i
			}
		}
		if f.Dir != "" {
			byDir[importPath(f.Dir)] = i
		}
		if n := CopyName(f.Name); n != "" {
			if _, ok := byName[n]; !ok {
				byName[n] = i
			}
		}
	}
	match := make([]int, len(games))
	by := make([]string, len(games))
	taken := map[int]bool{}
	for i := range match {
		match[i] = -1
	}
	pass := func(kind string, key func(PlayniteGame) string, idx map[string]int) {
		for i, g := range games {
			if match[i] >= 0 {
				continue
			}
			k := key(g)
			if j, ok := idx[k]; ok && k != "" && !taken[j] {
				match[i], by[i], taken[j] = j, kind, true
			}
		}
	}
	pass("store", func(g PlayniteGame) string {
		if g.Store == "" || g.GameID == "" {
			return ""
		}
		return g.Store + ":" + strings.ToLower(g.GameID)
	}, byStore)
	pass("folder", func(g PlayniteGame) string { return importPath(g.InstallDir) }, byDir)
	pass("name", func(g PlayniteGame) string { return CopyName(g.Name) }, byName)

	plan := PlaynitePlan{Changes: []PlayniteChange{}, Unmatched: []string{}}
	for i, g := range games {
		if match[i] < 0 {
			plan.Unmatched = append(plan.Unmatched, g.Name)
			continue
		}
		plan.Matched++
		f := lib[match[i]]
		e := have.Shelf[f.Key]
		c := PlayniteChange{Key: f.Key, Name: f.Name, From: g.Name, By: by[i]}
		if g.Playtime > have.Imported[f.Key] {
			c.Add, c.Total = g.Playtime-have.Imported[f.Key], g.Playtime
		}
		if g.LastActivity.After(have.Played[f.Key]) {
			c.LastPlayed = g.LastActivity
		}
		if e.Status == "" {
			c.Status = playniteStatuses[strings.ToLower(strings.ReplaceAll(g.Status, " ", ""))]
		}
		if e.Rating == 0 {
			c.Stars = PlayniteStars(g.UserScore)
		}
		if e.Note == "" {
			note := []rune(strings.ReplaceAll(g.Notes, "\r\n", "\n"))
			c.Note = strings.TrimSpace(string(note[:min(len(note), 500)]))
		}
		in := func(list []string, t string) bool {
			return slices.ContainsFunc(list, func(x string) bool { return strings.EqualFold(x, t) })
		}
		c.Favourite = g.Favorite && !in(have.Tags[f.Key], FavouritesTag)
		for _, t := range g.Categories {
			if !in(have.Tags[f.Key], t) {
				c.Categories = append(c.Categories, t)
			}
		}
		for _, t := range g.Tags {
			if !in(have.Tags[f.Key], t) && !in(c.Categories, t) {
				c.Tags = append(c.Tags, t)
			}
		}
		if !c.empty() {
			plan.Changes = append(plan.Changes, c)
		}
	}
	return plan
}

// Only keeps what o chose.
func (c PlayniteChange) Only(o PlayniteOptions) PlayniteChange {
	if !o.Playtime {
		c.Add, c.Total, c.LastPlayed = 0, 0, time.Time{}
	}
	if !o.Statuses {
		c.Status = ""
	}
	if !o.Ratings {
		c.Stars = 0
	}
	if !o.Notes {
		c.Note = ""
	}
	if !o.Favourites {
		c.Favourite = false
	}
	if !o.Categories {
		c.Categories = nil
	}
	if !o.Tags {
		c.Tags = nil
	}
	return c
}

// Only is the plan's changes with only what o chose, leaving out games
// with nothing left.
func (p PlaynitePlan) Only(o PlayniteOptions) []PlayniteChange {
	var out []PlayniteChange
	for _, c := range p.Changes {
		if c = c.Only(o); !c.empty() {
			out = append(out, c)
		}
	}
	return out
}

func (c PlayniteChange) empty() bool {
	return c.Add == 0 && c.LastPlayed.IsZero() && c.Status == "" && c.Stars == 0 && c.Note == "" && !c.Favourite && len(c.Categories) == 0 && len(c.Tags) == 0
}

// ---------------------------------------------------------------- apply --

func (s *Store) playnitePath() string { return filepath.Join(s.Dir, "playnite.json") }

// PlayniteImported is the Playnite time already brought in for a profile,
// by Library key (playnite.json).
func (s *Store) PlayniteImported(playerID string) map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all map[string]map[string]int64
	if b, err := os.ReadFile(s.playnitePath()); err == nil {
		_ = json.Unmarshal(b, &all)
	}
	if all[playerID] == nil {
		return map[string]int64{}
	}
	return all[playerID]
}

// ApplyPlaynite writes an import's changes for one profile to the shelf,
// recent.json and playnite.json; favourites and collections are in the
// config (Config.ApplyPlaynite). Each is checked again against what's
// there now, so nothing the profile set is replaced.
func (s *Store) ApplyPlaynite(playerID string, changes []PlayniteChange, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh, err := s.readShelfToChange()
	if err != nil {
		return err
	}
	if sh[playerID] == nil {
		sh[playerID] = map[string]ShelfEntry{}
	}
	rec := map[string]Played{}
	if b, err := os.ReadFile(s.RecentPath()); err == nil {
		_ = json.Unmarshal(b, &rec)
	}
	all := map[string]map[string]int64{}
	if b, err := os.ReadFile(s.playnitePath()); err == nil {
		_ = json.Unmarshal(b, &all)
	}
	if all[playerID] == nil {
		all[playerID] = map[string]int64{}
	}
	for _, c := range changes {
		e := sh[playerID][c.Key]
		was := e
		if c.Add > 0 && e.Offset+c.Add <= maxOffset {
			e.Offset += c.Add
		}
		if e.Status == "" {
			e.Status = c.Status
		}
		if e.Rating == 0 {
			e.Rating = c.Stars
		}
		if e.Note == "" {
			e.Note = c.Note
		}
		if e != was && e.Validate() == nil {
			if e.Offset != was.Offset {
				all[playerID][c.Key] = max(all[playerID][c.Key], c.Total)
			}
			e.Updated = now.UTC()
			sh[playerID][c.Key] = e
		}
		if p := rec[c.Key].withLast(); !c.LastPlayed.IsZero() && c.LastPlayed.After(p.Players[playerID]) {
			m := map[string]time.Time{playerID: c.LastPlayed}
			for id, t := range p.Players {
				if id != playerID {
					m[id] = t
				}
			}
			p.Players = m
			if c.LastPlayed.After(p.At) {
				p.At, p.Player = c.LastPlayed, playerID
			}
			rec[c.Key] = p
		}
	}
	trimRecent(rec)
	if err := writeJSONAtomic(s.ShelfPath(), sh); err != nil {
		return err
	}
	if err := writeJSONAtomic(s.RecentPath(), rec); err != nil {
		return err
	}
	return writeJSONAtomic(s.playnitePath(), all)
}

// ApplyPlaynite puts the games in Favourites and in collections named
// after Playnite's categories and tags. It returns how many it couldn't
// add (too many collections, or a name too long).
func (c *Config) ApplyPlaynite(changes []PlayniteChange) (skipped int) {
	for _, ch := range changes {
		var tags []string
		if ch.Favourite {
			tags = append(tags, FavouritesTag)
		}
		for _, t := range append(append([]string{}, ch.Categories...), ch.Tags...) {
			if !slices.Contains(tags, t) {
				tags = append(tags, t)
			}
		}
		for _, t := range tags {
			if c.AddToTag([]string{ch.Key}, t) != nil {
				skipped++
			}
		}
	}
	return skipped
}

// The files an import changes, copied to backups\import\<id> first.
var importFiles = []string{"config.json", "shelf.json", "recent.json", "playnite.json"}

var importID = regexp.MustCompile(`^\d{8}-\d{6}$`)

func (s *Store) importDir(id string) string { return filepath.Join(s.BackupsDir(), "import", id) }

// BackupBeforeImport copies the files an import changes to
// backups\import\<id>, noting in had.json which of them existed.
func (s *Store) BackupBeforeImport(now time.Time) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := now.Format("20060102-150405")
	for n := 1; ; n++ {
		if _, err := os.Stat(s.importDir(id)); errors.Is(err, os.ErrNotExist) {
			break
		}
		id = now.Add(time.Duration(n) * time.Second).Format("20060102-150405")
	}
	had, err := copyDataFiles(s.Dir, s.importDir(id))
	if err != nil {
		return "", err
	}
	return id, writeJSONAtomic(filepath.Join(s.importDir(id), "had.json"), had)
}

// UndoImport puts back the files from before an import. What's there now
// is copied to backups\import\<id>\undone first; a file the import made is
// removed (its copy stays there).
func (s *Store) UndoImport(id string) error {
	if !importID.MatchString(id) {
		return errors.New("unknown import")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := s.importDir(id)
	var had []string
	b, err := os.ReadFile(filepath.Join(dir, "had.json"))
	if err != nil || json.Unmarshal(b, &had) != nil {
		return errors.New("The copy from before that import is gone, so nothing was changed.")
	}
	if _, err := copyDataFiles(s.Dir, filepath.Join(dir, "undone")); err != nil {
		return fmt.Errorf("Nothing was changed: Lootnest couldn't copy what's here now first (%w).", err)
	}
	for _, n := range importFiles {
		if !slices.Contains(had, n) {
			if err := os.Remove(filepath.Join(s.Dir, n)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			continue
		}
		from := filepath.Join(dir, n)
		st, err := os.Stat(from)
		if err != nil {
			return err
		}
		if err := copyFile(from, filepath.Join(s.Dir, n+".tmp"), st); err != nil {
			return err
		}
		if err := os.Rename(filepath.Join(s.Dir, n+".tmp"), filepath.Join(s.Dir, n)); err != nil {
			return err
		}
	}
	return nil
}

// copyDataFiles copies the import's files that exist in from to to, and
// says which they were.
func copyDataFiles(from, to string) ([]string, error) {
	had := []string{}
	for _, n := range importFiles {
		st, err := os.Stat(filepath.Join(from, n))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if err := copyFile(filepath.Join(from, n), filepath.Join(to, n), st); err != nil {
			return nil, err
		}
		had = append(had, n)
	}
	return had, nil
}
