package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Captures from other apps: the screenshots and clips Game Bar, Steam,
// NVIDIA, AMD and OBS already made, and Lootnest's own (1.70). They're
// indexed in captures.json and matched to Library games, by Steam's app id
// folder, then by name, then by the time of a session; the files are only
// ever read, never moved, renamed or deleted.
// ---------------------------------------------------------------------------

const (
	CapGameBar  = "Game Bar"
	CapSteam    = "Steam"
	CapNVIDIA   = "NVIDIA"
	CapAMD      = "AMD"
	CapOBS      = "OBS"
	CapLootnest = "Lootnest"

	// CaptureNoGame is the key of a capture the user said is of no game.
	CaptureNoGame = "none"

	capturesMax  = 20000
	captureSlack = time.Minute // around a session, for a clip saved as it ends
)

// CaptureSource is one folder an app keeps captures in.
type CaptureSource struct {
	Name, Dir string
	Depth     int    // folder levels below Dir that are read
	AppID     int    // Steam's folder is one game's
	Game      string // a folder named after the game
	OBS       bool   // only files named the way OBS names them
}

// CaptureEnv is where the apps keep their captures on this PC.
type CaptureEnv struct {
	Videos, Pictures, SteamPath string
	GameBar                     string   // Game Bar's captures folder, when Windows names one
	Games                       []string // Library games' names, to tell NVIDIA's folders
}

// CaptureSources lists the capture folders that exist on this PC.
func CaptureSources(e CaptureEnv) []CaptureSource {
	var out []CaptureSource
	add := func(s CaptureSource) {
		if fi, err := os.Stat(s.Dir); err == nil && fi.IsDir() {
			out = append(out, s)
		}
	}
	subdirs := func(dir string) []string {
		var names []string
		es, _ := os.ReadDir(dir)
		for _, x := range es {
			if x.IsDir() && !strings.HasPrefix(x.Name(), ".") {
				names = append(names, x.Name())
			}
		}
		return names
	}
	gameBar := e.GameBar
	if gameBar == "" && e.Videos != "" {
		gameBar = filepath.Join(e.Videos, "Captures")
	}
	if gameBar != "" {
		add(CaptureSource{Name: CapGameBar, Dir: gameBar})
	}
	if e.SteamPath != "" {
		users := filepath.Join(e.SteamPath, "userdata")
		for _, u := range subdirs(users) {
			remote := filepath.Join(users, u, "760", "remote")
			for _, a := range subdirs(remote) {
				if id, err := strconv.Atoi(a); err == nil && id > 0 {
					add(CaptureSource{Name: CapSteam, Dir: filepath.Join(remote, a, "screenshots"), AppID: id})
				}
			}
		}
	}
	if e.Videos != "" {
		names := map[string]bool{}
		for _, g := range e.Games {
			if n := CopyName(g); n != "" {
				names[n] = true
			}
		}
		for _, d := range subdirs(e.Videos) {
			if names[CopyName(d)] {
				add(CaptureSource{Name: CapNVIDIA, Dir: filepath.Join(e.Videos, d), Depth: 1, Game: d})
			}
		}
		relive := filepath.Join(e.Videos, "Radeon ReLive")
		for _, d := range subdirs(relive) {
			add(CaptureSource{Name: CapAMD, Dir: filepath.Join(relive, d), Depth: 1, Game: d})
		}
		add(CaptureSource{Name: CapOBS, Dir: e.Videos, OBS: true})
	}
	if e.Pictures != "" {
		own := filepath.Join(e.Pictures, "Lootnest", "Screenshots")
		for _, d := range subdirs(own) {
			add(CaptureSource{Name: CapLootnest, Dir: filepath.Join(own, d), Game: d})
		}
	}
	return out
}

// Capture is one screenshot or clip in the index.
type Capture struct {
	Path   string    `json:"path"`
	Key    string    `json:"key,omitempty"` // the Library game it's matched to, "" for unsorted
	Kind   string    `json:"kind"`          // "picture" or "clip"
	At     time.Time `json:"at"`            // when it was last written
	Size   int64     `json:"size"`
	Source string    `json:"source"`
	AppID  int       `json:"appId,omitempty"`  // from Steam's folder
	Hint   string    `json:"hint,omitempty"`   // the game's name, from a folder or the file's name
	Manual bool      `json:"manual,omitempty"` // matched by the user
}

var (
	captureKinds = map[string]string{".png": "picture", ".jpg": "picture", ".jpeg": "picture", ".webp": "picture", ".jxr": "picture",
		".mp4": "clip", ".mkv": "clip", ".mov": "clip", ".webm": "clip"}
	// Game Bar: "Game Name 2026-11-08 21-04-13", or "Game Name 11_8_2026 9_04_13 PM" on Windows 10.
	gameBarName = regexp.MustCompile(`^(.+?)\s+(?:\d{4}[-_.]\d{1,2}[-_.]\d{1,2}|\d{1,2}[-_.]\d{1,2}[-_.]\d{4})\s+\d{1,2}[-_.]\d{2}[-_.]\d{2}(?:\s*[AaPp][Mm])?(?:\s*\(\d+\))?$`)
	// OBS: "2026-11-08 21-04-13", or "Replay 2026-11-08 21-04-13" from its replay buffer.
	obsName = regexp.MustCompile(`^(?:Replay )?\d{4}-\d\d-\d\d \d\d-\d\d-\d\d$`)
)

// GameBarGame is the game's name at the start of a Game Bar file name.
func GameBarGame(file string) string {
	if m := gameBarName.FindStringSubmatch(strings.TrimSuffix(file, filepath.Ext(file))); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// ScanCaptures lists the pictures and clips in the sources that aren't in
// the index as they are (new, or written again), up to 20,000. It only reads.
func ScanCaptures(sources []CaptureSource, indexed []Capture) []Capture {
	old := map[string]Capture{}
	for _, c := range indexed {
		old[c.Path] = c
	}
	var out []Capture
	var walk func(s CaptureSource, dir string, depth int)
	walk = func(s CaptureSource, dir string, depth int) {
		es, _ := os.ReadDir(dir)
		for _, e := range es {
			if len(out) >= capturesMax {
				return
			}
			n := e.Name()
			if strings.HasPrefix(n, ".") {
				continue
			}
			if e.IsDir() {
				if depth < s.Depth {
					walk(s, filepath.Join(dir, n), depth+1)
				}
				continue
			}
			kind := captureKinds[strings.ToLower(filepath.Ext(n))]
			if kind == "" || !e.Type().IsRegular() || (s.OBS && !obsName.MatchString(strings.TrimSuffix(n, filepath.Ext(n)))) {
				continue
			}
			p := filepath.Join(dir, n)
			fi, err := e.Info()
			if o, ok := old[p]; err != nil || ok && o.Size == fi.Size() && o.At.Equal(fi.ModTime()) {
				continue
			}
			c := Capture{Path: p, Kind: kind, At: fi.ModTime(), Size: fi.Size(), Source: s.Name, AppID: s.AppID, Hint: s.Game}
			if s.Name == CapGameBar {
				c.Hint = GameBarGame(n)
			}
			out = append(out, c)
		}
	}
	for _, s := range sources {
		walk(s, s.Dir, 0)
	}
	return out
}

// CaptureGame is a Library game as captures are matched to it.
type CaptureGame struct {
	Key   string
	Name  string
	AppID int // its Steam app id, 0 if none is known
}

// MatchCapture is the Library key of the game a capture is of: by Steam's
// app id, else by the game's name in its folder or file name, else by a
// session it was made in (a minute either side), else "".
func MatchCapture(c Capture, games []CaptureGame, sessions []SessionRecord) string {
	if c.AppID > 0 {
		for _, g := range games {
			if g.AppID == c.AppID {
				return g.Key
			}
		}
	}
	if n := CopyName(c.Hint); n != "" {
		for _, g := range games {
			if CopyName(g.Name) == n {
				return g.Key
			}
		}
	}
	for _, s := range sessions {
		if !c.At.Before(s.Start.Add(-captureSlack)) && !c.At.After(s.End.Add(captureSlack)) && knownGame(games, s.Game) {
			return s.Game
		}
	}
	return ""
}

func knownGame(games []CaptureGame, key string) bool {
	for _, g := range games {
		if g.Key == key {
			return true
		}
	}
	return false
}

func (s *Store) CapturesPath() string { return filepath.Join(s.Dir, "captures.json") }

type captureIndex struct {
	Captures []Capture `json:"captures"`
}

// LoadCaptures reads the index, newest first.
func (s *Store) LoadCaptures() []Capture {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readCaptures()
}

func (s *Store) readCaptures() []Capture {
	var ix captureIndex
	if b, err := os.ReadFile(s.CapturesPath()); err == nil {
		_ = json.Unmarshal(b, &ix)
	}
	return ix.Captures
}

// UpdateCaptures brings the index up to date with a scan: what was scanned
// is added, an entry the scan missed stays while its file is there as it
// was, and every entry is matched again, but for one the user matched by
// hand to a game still in the Library (or to none). Only captures.json is
// written.
func (s *Store) UpdateCaptures(scanned []Capture, games []CaptureGame, sessions []SessionRecord) ([]Capture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := map[string]Capture{}
	for _, c := range s.readCaptures() {
		old[c.Path] = c
	}
	seen := map[string]bool{}
	var out []Capture
	keep := func(c Capture) {
		if o, ok := old[c.Path]; ok && o.Manual && o.Size == c.Size && o.At.Equal(c.At) && (o.Key == CaptureNoGame || knownGame(games, o.Key)) {
			c.Key, c.Manual = o.Key, true
		} else {
			c.Key, c.Manual = MatchCapture(c, games, sessions), false
		}
		seen[c.Path] = true
		out = append(out, c)
	}
	for _, c := range scanned {
		if !seen[c.Path] {
			keep(c)
		}
	}
	for p, o := range old {
		if fi, err := os.Stat(p); !seen[p] && err == nil && fi.Mode().IsRegular() && fi.Size() == o.Size && fi.ModTime().Equal(o.At) {
			keep(o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out, writeJSONAtomic(s.CapturesPath(), captureIndex{Captures: out})
}

// AssignCapture matches a capture in the index to a game by hand, or to
// CaptureNoGame. The file itself isn't touched.
func (s *Store) AssignCapture(path, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.readCaptures()
	for i := range list {
		if list[i].Path == path {
			list[i].Key, list[i].Manual = key, true
			return writeJSONAtomic(s.CapturesPath(), captureIndex{Captures: list})
		}
	}
	return os.ErrNotExist
}
