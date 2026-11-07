package core

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Achievements, read from files already on this PC: Steam's own cache for
// the signed-in account, and the files other Steam-compatible tools write
// next to a game's saves. Nothing is sent anywhere.

// Achievement is one achievement of a game, unlocked or not.
type Achievement struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"` // https URL on Steam's image servers, or ""
	Hidden      bool      `json:"hidden,omitempty"`
	Unlocked    bool      `json:"unlocked"`
	At          time.Time `json:"at,omitempty"`
	Percent     float64   `json:"percent,omitempty"` // share of all players who have it, when known
}

// AchievementSet is what Lootnest knows of a game's achievements.
type AchievementSet struct {
	AppID    int           `json:"appId"`
	Source   string        `json:"source"` // "steam" (the account's) or "pc" (a file on this PC)
	File     string        `json:"file,omitempty"`
	Total    int           `json:"total"`
	Unlocked int           `json:"unlocked"`
	List     []Achievement `json:"list"`
	// Players are each player's own, for a game whose achievements go with
	// their profile (player id -> their set).
	Players map[string]*AchievementSet `json:"players,omitempty"`
}

// achFile is a place a Steam-compatible tool keeps a game's achievements.
type achFile struct {
	base   string // APPDATA, LOCALAPPDATA, PUBLIC, PROGRAMDATA or DOCUMENTS
	path   string // with {id} for the Steam app id
	format string
}

var achFiles = []achFile{
	{"APPDATA", `Goldberg SteamEmu Saves\{id}\achievements.json`, "json"},
	{"APPDATA", `GSE Saves\{id}\achievements.json`, "json"},
	{"APPDATA", `EMPRESS\remote\{id}\achievements.json`, "json"},
	{"PUBLIC", `Documents\EMPRESS\{id}\remote\{id}\achievements.json`, "json"},
	{"PUBLIC", `Documents\Steam\CODEX\{id}\achievements.ini`, "ini"},
	{"APPDATA", `Steam\CODEX\{id}\achievements.ini`, "ini"},
	{"PUBLIC", `Documents\Steam\RUNE\{id}\achievements.ini`, "ini"},
	{"PUBLIC", `Documents\OnlineFix\{id}\Stats\Achievements.ini`, "ini"},
	{"PUBLIC", `Documents\OnlineFix\{id}\Achievements.ini`, "ini"},
	{"PROGRAMDATA", `RLD!\{id}\achievements.ini`, "hex"},
	{"PROGRAMDATA", `Steam\Player\{id}\stats\achievements.ini`, "hex"},
	{"PROGRAMDATA", `Steam\RLD!\{id}\stats\achievements.ini`, "hex"},
	{"PROGRAMDATA", `Steam\dodi\{id}\stats\achievements.ini`, "hex"},
	{"DOCUMENTS", `SKIDROW\{id}\SteamEmu\UserStats\achiev.ini`, "at"},
	{"DOCUMENTS", `Player\{id}\SteamEmu\UserStats\achiev.ini`, "at"},
	{"LOCALAPPDATA", `SKIDROW\{id}\SteamEmu\UserStats\achiev.ini`, "at"},
	{"APPDATA", `CreamAPI\{id}\stats\CreamAPI.Achievements.cfg`, "ini"},
}

func (e *Env) achBase(name string) string {
	if name == "DOCUMENTS" {
		return e.Documents
	}
	if e.Getenv == nil {
		return ""
	}
	return e.Getenv(name)
}

// AchievementFolders are the folders that hold a game's achievements on this
// PC, whether or not they exist yet. Swapping one per player gives each
// player their own achievements.
func (e *Env) AchievementFolders(appID int) []string {
	var out []string
	for _, f := range achFiles {
		if b := e.achBase(f.base); b != "" {
			out = append(out, filepath.Dir(filepath.Join(b, achPath(f.path, appID))))
		}
	}
	return out
}

func achPath(p string, appID int) string {
	return filepath.FromSlash(strings.ReplaceAll(strings.ReplaceAll(p, `\`, "/"), "{id}", strconv.Itoa(appID)))
}

// PCUnlocks reads the achievements a Steam-compatible tool saved on this PC
// for a game: achievement id -> when it was unlocked (zero if unknown). It
// takes the most recently written file when there are several.
func (e *Env) PCUnlocks(appID int) (map[string]time.Time, string) {
	if appID <= 0 {
		return nil, ""
	}
	var best string
	var bestFmt string
	var bestT time.Time
	for _, f := range achFiles {
		b := e.achBase(f.base)
		if b == "" {
			continue
		}
		p := filepath.Join(b, achPath(f.path, appID))
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.ModTime().After(bestT) {
			best, bestFmt, bestT = p, f.format, st.ModTime()
		}
	}
	if best == "" {
		return nil, ""
	}
	b, err := os.ReadFile(best)
	if err != nil {
		return nil, ""
	}
	return ParseUnlocks(b, bestFmt), best
}

// ParseUnlocks reads one achievements file: "json" (achievement -> earned,
// earned_time), "ini" (a section per achievement with Achieved=1 or
// achieved=true and its time), "hex" (State and Time as little-endian hex)
// or "at" (an [Achievements] section of id=1@…@time).
func ParseUnlocks(b []byte, format string) map[string]time.Time {
	out := map[string]time.Time{}
	if format == "json" {
		var obj map[string]struct {
			Earned     any   `json:"earned"`
			EarnedTime int64 `json:"earned_time"`
		}
		if json.Unmarshal(b, &obj) == nil {
			for id, a := range obj {
				if truthy(a.Earned) {
					out[id] = unixOrZero(a.EarnedTime)
				}
			}
			return out
		}
		var list []struct {
			Name       string `json:"name"`
			Earned     any    `json:"earned"`
			EarnedTime int64  `json:"earned_time"`
		}
		if json.Unmarshal(b, &list) == nil {
			for _, a := range list {
				if a.Name != "" && truthy(a.Earned) {
					out[a.Name] = unixOrZero(a.EarnedTime)
				}
			}
		}
		return out
	}
	for sec, kv := range parseINI(b) {
		switch format {
		case "at":
			if !strings.EqualFold(sec, "achievements") {
				continue
			}
			for id, v := range kv {
				parts := strings.Split(v, "@")
				if parts[0] == "1" {
					n, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)
					out[id] = unixOrZero(n)
				}
			}
		case "hex":
			if strings.EqualFold(sec, "steam") {
				continue
			}
			if hexLE(iget(kv, "state")) == 1 {
				out[sec] = unixOrZero(hexLE(iget(kv, "time")))
			}
		default:
			on := iget(kv, "achieved")
			if on != "1" && !strings.EqualFold(on, "true") {
				continue
			}
			t := iget(kv, "unlocktime")
			if t == "" {
				t = iget(kv, "timestamp")
			}
			if t == "" {
				t = iget(kv, "timeunlocked")
			}
			n, _ := strconv.ParseInt(t, 10, 64)
			out[sec] = unixOrZero(n)
		}
	}
	return out
}

// parseINI reads sections of key=value.
func parseINI(b []byte) map[string]map[string]string {
	out := map[string]map[string]string{}
	sec := ""
	sc := bufio.NewScanner(strings.NewReader(strings.TrimPrefix(string(b), "\uFEFF")))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || l[0] == ';' || l[0] == '#' {
			continue
		}
		if l[0] == '[' && strings.HasSuffix(l, "]") {
			sec = strings.TrimSpace(l[1 : len(l)-1])
			if out[sec] == nil {
				out[sec] = map[string]string{}
			}
			continue
		}
		k, v, ok := strings.Cut(l, "=")
		if !ok || sec == "" {
			continue
		}
		out[sec][strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"`)
	}
	return out
}

// iget is kv[k] with the key in any case.
func iget(kv map[string]string, k string) string {
	for kk, v := range kv {
		if strings.EqualFold(kk, k) {
			return v
		}
	}
	return ""
}

func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return x == "1" || strings.EqualFold(x, "true")
	}
	return false
}

func unixOrZero(n int64) time.Time {
	if n < 946684800 || n > 4102444800 { // outside 2000–2100: not a real time
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}

// hexLE reads a little-endian hex number such as "01000000".
func hexLE(s string) int64 {
	b, err := hex.DecodeString(strings.TrimSpace(s))
	if err != nil || len(b) == 0 || len(b) > 8 {
		n, _ := strconv.ParseInt(s, 10, 64)
		return n
	}
	var n int64
	for i := len(b) - 1; i >= 0; i-- {
		n = n<<8 | int64(b[i])
	}
	return n
}

var skipForAppID = map[string]bool{"content": true, "paks": true, "movies": true, "localization": true, "shaders": true, "streamingassets": true, "_commonredist": true, "redist": true}

var appIDLine = regexp.MustCompile(`(?im)^\s*(?:real)?app_?id\s*=\s*(\d{2,8})\s*$`)

// FolderAppID finds the Steam app id of a game installed outside Steam
// from the files next to it: steam_appid.txt, or an AppId= line in the
// settings file of a Steam-compatible tool. 0 when there's none.
func FolderAppID(dir string) int {
	if dir == "" {
		return 0
	}
	found, seen := 0, 0
	root := filepath.Clean(dir)
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		seen++
		if found > 0 || seen > 20000 {
			return filepath.SkipAll
		}
		if err != nil {
			return filepath.SkipDir
		}
		if d.IsDir() {
			// Deep enough for Unreal games (<Game>\Binaries\Win64), without
			// walking their content.
			if p != root && skipForAppID[strings.ToLower(d.Name())] {
				return filepath.SkipDir
			}
			if rel, _ := filepath.Rel(root, p); strings.Count(rel, string(filepath.Separator)) >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		var b []byte
		switch {
		case name == "steam_appid.txt":
			b, _ = os.ReadFile(p)
			if n, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && n > 0 {
				found = n
			}
		case strings.HasSuffix(name, ".ini") && (strings.Contains(name, "steam") || strings.Contains(name, "emu") || strings.Contains(name, "cream") || strings.Contains(name, "onlinefix") || strings.Contains(name, "codex") || strings.Contains(name, "rune")):
			if st, err := d.Info(); err == nil && st.Size() < 256*1024 {
				b, _ = os.ReadFile(p)
				if m := appIDLine.FindSubmatch(b); m != nil {
					found, _ = strconv.Atoi(string(m[1]))
				}
			}
		}
		return nil
	})
	return found
}

// LocalSchema reads the names, descriptions and icons a Steam-compatible
// tool keeps next to a game (steam_settings\achievements.json), when it's
// there. Icons there are files, so they're left out.
func LocalSchema(dir string) []Achievement {
	if dir == "" {
		return nil
	}
	var file string
	root := filepath.Clean(dir)
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if file != "" {
			return filepath.SkipAll
		}
		if err != nil {
			return filepath.SkipDir
		}
		if d.IsDir() {
			if rel, _ := filepath.Rel(root, p); strings.Count(rel, string(filepath.Separator)) >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), "achievements.json") && strings.EqualFold(filepath.Base(filepath.Dir(p)), "steam_settings") {
			file = p
		}
		return nil
	})
	if file == "" {
		return nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var raw []struct {
		Name        string          `json:"name"`
		DisplayName json.RawMessage `json:"displayName"`
		Description json.RawMessage `json:"description"`
		Hidden      any             `json:"hidden"`
	}
	if json.Unmarshal(b, &raw) != nil {
		return nil
	}
	out := make([]Achievement, 0, len(raw))
	for _, r := range raw {
		if r.Name == "" {
			continue
		}
		out = append(out, Achievement{ID: r.Name, Name: langText(r.DisplayName), Description: langText(r.Description), Hidden: truthy(r.Hidden)})
	}
	return out
}

// langText reads a string, or {"english": "…", …}.
func langText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var m map[string]string
	if json.Unmarshal(raw, &m) == nil {
		if v := m["english"]; v != "" {
			return v
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if m[k] != "" {
				return m[k]
			}
		}
	}
	return ""
}

// SteamCacheAchievements reads what Steam's library keeps on this PC about
// a game's achievements for the signed-in account
// (userdata\<account>\config\librarycache\<appid>.json).
func SteamCacheAchievements(userDir string, appID int) ([]Achievement, int, bool) {
	if userDir == "" || appID <= 0 {
		return nil, 0, false
	}
	b, err := os.ReadFile(filepath.Join(userDir, "config", "librarycache", strconv.Itoa(appID)+".json"))
	if err != nil {
		return nil, 0, false
	}
	return ParseSteamCache(b, appID)
}

// ParseSteamCache reads the "achievements" entry of a library cache file:
// its lists of achievements and its total.
func ParseSteamCache(b []byte, appID int) ([]Achievement, int, bool) {
	var top []json.RawMessage
	if json.Unmarshal(b, &top) != nil {
		return nil, 0, false
	}
	for _, e := range top {
		var pair []json.RawMessage
		if json.Unmarshal(e, &pair) != nil || len(pair) < 2 {
			continue
		}
		var name string
		if json.Unmarshal(pair[0], &name) != nil || name != "achievements" {
			continue
		}
		var v struct {
			Data map[string]json.RawMessage `json:"data"`
		}
		if json.Unmarshal(pair[1], &v) != nil || v.Data == nil {
			return nil, 0, false
		}
		total := 0
		_ = json.Unmarshal(v.Data["nTotal"], &total)
		seen := map[string]bool{}
		var out []Achievement
		keys := make([]string, 0, len(v.Data))
		for k := range v.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			var list []struct {
				ID          string  `json:"strID"`
				Name        string  `json:"strName"`
				Description string  `json:"strDescription"`
				Image       string  `json:"strImage"`
				Achieved    bool    `json:"bAchieved"`
				Unlocked    int64   `json:"rtUnlocked"`
				Hidden      bool    `json:"bHidden"`
				Percent     float64 `json:"flAchieved"`
			}
			if json.Unmarshal(v.Data[k], &list) != nil {
				continue
			}
			for _, a := range list {
				if a.ID == "" || seen[a.ID] {
					continue
				}
				seen[a.ID] = true
				out = append(out, Achievement{ID: a.ID, Name: a.Name, Description: a.Description, Icon: steamAchIcon(a.Image, appID),
					Hidden: a.Hidden, Unlocked: a.Achieved, At: unixOrZero(a.Unlocked), Percent: a.Percent})
			}
		}
		if total < len(out) {
			total = len(out)
		}
		return out, total, len(out) > 0 || total > 0
	}
	return nil, 0, false
}

func steamAchIcon(img string, appID int) string {
	img = strings.TrimSpace(img)
	switch {
	case img == "":
		return ""
	case strings.HasPrefix(img, "https://"):
	case strings.HasPrefix(img, "/"):
		img = "https://cdn.akamai.steamstatic.com" + img
	default:
		img = "https://cdn.akamai.steamstatic.com/steamcommunity/public/images/apps/" + strconv.Itoa(appID) + "/" + img
	}
	if !SafeNewsImage(img) {
		return ""
	}
	return img
}

// MergeAchievements puts together what's known: a list (with names) and
// the unlocks read from a file on this PC, which win for whether and when
// each one was unlocked. Unlocked come first, newest first.
func MergeAchievements(list []Achievement, total int, unlocks map[string]time.Time) ([]Achievement, int, int) {
	byID := map[string]int{}
	for i := range list {
		byID[list[i].ID] = i
	}
	if unlocks != nil {
		for i := range list {
			list[i].Unlocked, list[i].At = false, time.Time{}
		}
		for id, at := range unlocks {
			if i, ok := byID[id]; ok {
				list[i].Unlocked, list[i].At = true, at
			} else {
				list = append(list, Achievement{ID: id, Unlocked: true, At: at})
			}
		}
	}
	n := 0
	for i := range list {
		if list[i].Name == "" {
			list[i].Name = prettyAchID(list[i].ID)
		}
		if list[i].Unlocked {
			n++
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.Unlocked != b.Unlocked {
			return a.Unlocked
		}
		if a.Unlocked && !a.At.Equal(b.At) {
			return a.At.After(b.At)
		}
		return false
	})
	if total < len(list) {
		total = len(list)
	}
	return list, total, n
}

// prettyAchID turns "ACH_WIN_ONE_GAME" into "Win one game" when nothing
// better is known.
func prettyAchID(id string) string {
	s := strings.TrimSpace(id)
	for _, p := range []string{"ACH_", "ACHIEVEMENT_", "ACHIEVEMENTS_", "NEW_ACHIEVEMENT_"} {
		if len(s) > len(p) && strings.EqualFold(s[:len(p)], p) {
			s = s[len(p):]
			break
		}
	}
	s = strings.ToLower(strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(s))
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return id
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// AchEntryPrefix starts the id of a swapped achievements folder in a
// player's profile, so it's never taken for one of the game's own paths.
const AchEntryPrefix = "ach-"

// achEntryID is the profile folder name for one achievements location: the
// same on every PC and in every version, whatever the user's folders are.
func achEntryID(f achFile) string {
	h := uint32(2166136261)
	for _, c := range []byte(f.base + "|" + f.path) {
		h = (h ^ uint32(c)) * 16777619
	}
	return AchEntryPrefix + strconv.FormatUint(uint64(h), 16)
}

// GameAppID is a game's Steam app id: Steam's own, or the one found next to
// a game started from its exe. 0 when there's none.
func GameAppID(g *Game) int {
	if g.Launch.AppID > 0 {
		return g.Launch.AppID
	}
	if g.Launch.Exe != "" {
		return FolderAppID(filepath.Dir(g.Launch.Exe))
	}
	return 0
}

// achievementEntries are the achievements folders swapped with a game's
// profiles, so each player has their own: the ones that exist on this PC
// or in someone's profile, and that don't overlap a path the game already
// swaps. None when the game keeps achievements shared.
func (e *Engine) achievementEntries(g *Game, have []resolved) []resolved {
	if g.SharedAchievements || e.Env == nil {
		return nil
	}
	appID := GameAppID(g)
	if appID <= 0 {
		return nil
	}
	var out []resolved
	clashes := func(p string) bool {
		for _, r := range append(append([]resolved{}, have...), out...) {
			if isWithin(p, r.live) || isWithin(r.live, p) {
				return true
			}
		}
		return false
	}
	for _, f := range achFiles {
		b := e.Env.achBase(f.base)
		if b == "" {
			continue
		}
		file := filepath.Join(b, achPath(f.path, appID))
		// The whole folder (achievements and the stats they count from), or
		// only the file when the game already swaps something inside the
		// folder, such as its saves.
		for _, c := range []struct{ id, live string }{{achEntryID(f), filepath.Dir(file)}, {achEntryID(f) + "-f", file}} {
			if !exists(c.live) && !e.anyProfileHas(g.ID, c.id) {
				continue
			}
			if e.Env.CheckSafe(c.live) != nil || clashes(c.live) {
				continue
			}
			out = append(out, resolved{PathEntry{ID: c.id, Path: c.live}, c.live})
			break
		}
	}
	return out
}

func (e *Engine) anyProfileHas(gameID, entryID string) bool {
	if e.Store == nil {
		return false
	}
	m, _ := filepath.Glob(filepath.Join(e.Store.ProfilesDir(), "*", gameID, entryID))
	return len(m) > 0
}

// ProfileUnlocks reads a player's own achievements for a game from their
// profile (the copy swapped out with their files). ok is false when their
// profile holds none.
func (e *Engine) ProfileUnlocks(playerID string, g *Game) (map[string]time.Time, bool) {
	appID := GameAppID(g)
	if appID <= 0 || e.Store == nil {
		return nil, false
	}
	dir := e.Store.ProfileDir(playerID, g.ID)
	var best, bestFmt string
	var bestT time.Time
	for _, f := range achFiles {
		for _, p := range []string{filepath.Join(dir, achEntryID(f), filepath.Base(achPath(f.path, appID))), filepath.Join(dir, achEntryID(f)+"-f")} {
			if st, err := os.Stat(p); err == nil && !st.IsDir() && st.ModTime().After(bestT) {
				best, bestFmt, bestT = p, f.format, st.ModTime()
			}
		}
	}
	if best == "" {
		return nil, false
	}
	b, err := os.ReadFile(best)
	if err != nil {
		return nil, false
	}
	return ParseUnlocks(b, bestFmt), true
}

// ParseSteamSchema reads Steam's GetSchemaForGame answer: every
// achievement's id, name, description, hidden flag and icon.
func ParseSteamSchema(b []byte, appID int) []Achievement {
	var v struct {
		Game struct {
			Stats struct {
				Achievements []struct {
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
					Description string `json:"description"`
					Hidden      any    `json:"hidden"`
					Icon        string `json:"icon"`
				} `json:"achievements"`
			} `json:"availableGameStats"`
		} `json:"game"`
	}
	if json.Unmarshal(b, &v) != nil {
		return nil
	}
	var out []Achievement
	for _, a := range v.Game.Stats.Achievements {
		if a.Name == "" {
			continue
		}
		out = append(out, Achievement{ID: a.Name, Name: a.DisplayName, Description: a.Description, Hidden: truthy(a.Hidden), Icon: steamAchIcon(a.Icon, appID)})
	}
	return out
}

// CompleteList fills a partial list (Steam's library cache keeps only some
// achievements) from a full schema: names and icons come from either, and
// whether each is unlocked from the list.
func CompleteList(list, schema []Achievement) []Achievement {
	if len(schema) == 0 {
		return list
	}
	have := map[string]Achievement{}
	for _, a := range list {
		have[a.ID] = a
	}
	out := make([]Achievement, 0, len(schema))
	for _, s := range schema {
		if a, ok := have[s.ID]; ok {
			if a.Name == "" {
				a.Name = s.Name
			}
			if a.Description == "" {
				a.Description = s.Description
			}
			if a.Icon == "" {
				a.Icon = s.Icon
			}
			out = append(out, a)
			delete(have, s.ID)
			continue
		}
		out = append(out, s)
	}
	for _, a := range list { // in the list but not the schema: keep it
		if _, ok := have[a.ID]; ok {
			out = append(out, a)
		}
	}
	return out
}

// AchToken starts the path of an achievements folder in cloud saves: the
// same on every PC (the folder itself is under each PC's own user folders).
const AchToken = "{ACHIEVEMENTS}/"

// SyncPaths are a game's locations for cloud saves: its own paths, and
// the achievements folders that go with each profile (by AchToken+id). With
// all, every place this game's achievements could be on this PC is listed,
// so a download can bring them to a PC that doesn't have them yet.
func (e *Engine) SyncPaths(g *Game, all bool) []PathEntry {
	out := append([]PathEntry{}, g.Paths...)
	if g.SharedAchievements {
		return out
	}
	if all {
		appID := GameAppID(g)
		if appID <= 0 || e.Env == nil {
			return out
		}
		for _, f := range achFiles {
			if e.Env.achBase(f.base) == "" {
				continue
			}
			id := achEntryID(f)
			out = append(out, PathEntry{ID: id, Path: AchToken + id}, PathEntry{ID: id + "-f", Path: AchToken + id + "-f"})
		}
		return out
	}
	rs, err := e.resolveAll(g)
	if err != nil {
		return out
	}
	for _, r := range rs {
		if strings.HasPrefix(r.entry.ID, AchEntryPrefix) {
			out = append(out, PathEntry{ID: r.entry.ID, Path: AchToken + r.entry.ID})
		}
	}
	return out
}

// AchLive is where an achievements entry (by its id) is on this PC: the
// tool's folder, or its file for an id ending in -f. "" if unknown.
func (e *Engine) AchLive(g *Game, id string) string {
	appID := GameAppID(g)
	if appID <= 0 || e.Env == nil {
		return ""
	}
	for _, f := range achFiles {
		b := e.Env.achBase(f.base)
		if b == "" {
			continue
		}
		file := filepath.Join(b, achPath(f.path, appID))
		switch id {
		case achEntryID(f):
			return filepath.Dir(file)
		case achEntryID(f) + "-f":
			return file
		}
	}
	return ""
}
