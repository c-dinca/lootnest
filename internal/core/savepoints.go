package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Save points: a named copy of a game's saves at one moment ("before the
// boss"), made from the game's page or with a hotkey while it runs. Each game
// has a timeline of them; going back to one keeps the saves as they are now
// as a point first, and a point can start a new run (a profile of its own).
//
// A point is a folder savepoints\<game>\<id> with point.json and the files
// of each location under files\<location id>. Copies made while the game
// runs are compared before and after, and made again if the game was
// writing.
// ---------------------------------------------------------------------------

// SavePoint is one point on a game's timeline.
type SavePoint struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Created  time.Time `json:"created"`
	Player   string    `json:"player,omitempty"`   // whose saves (the profile loaded then)
	Playtime int64     `json:"playtime,omitempty"` // seconds played by then
	Unlocked int       `json:"unlocked,omitempty"` // achievements unlocked by then
	Auto     bool      `json:"auto,omitempty"`     // made before going back to another point
	Size     int64     `json:"size"`
	Locs     []string  `json:"locs"` // location ids with files in the point
	// Paths are the locations' paths as Lootnest writes them (with
	// %APPDATA% and the like), to find them again if they're gone.
	Paths map[string]string `json:"paths,omitempty"`
}

// SaveLoc is one place a game keeps its saves.
type SaveLoc struct {
	ID   string // stable: a profile path's id, or a hash of the path
	Live string // where it is on this PC
	Path string // as written in Lootnest's settings
}

// LocID names a save location by its (tokenized) path.
func LocID(path string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(path)))
	return "loc-" + strconv.FormatUint(uint64(h.Sum32()), 16)
}

// SavePointsDir is where a game's points are kept.
func (s *Store) SavePointsDir(gameKey string) string {
	return filepath.Join(s.Dir, "savepoints", CoverKey(gameKey)+"-"+strings.TrimPrefix(LocID(gameKey), "loc-"))
}

// cleanPointName keeps a point's name to one line of at most 60 characters.
func cleanPointName(n string) string {
	n = strings.Join(strings.Fields(n), " ")
	if r := []rune(n); len(r) > 60 {
		n = strings.TrimSpace(string(r[:60]))
	}
	return n
}

func stampOf(p string) (map[string]FileStamp, error) {
	st, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]FileStamp{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return map[string]FileStamp{"": {Size: st.Size(), Mod: st.ModTime().UnixNano()}}, nil
	}
	return SnapshotDir(p)
}

// MakeSavePoint copies the locations into a new point in root. A copy that
// changed while it was made (the game was writing) is made again, up to
// tries times, waiting wait between them.
func MakeSavePoint(root string, locs []SaveLoc, p SavePoint, now time.Time, tries int, wait time.Duration) (SavePoint, error) {
	p.Name = cleanPointName(p.Name)
	if p.Name == "" {
		p.Name = "Save point, " + now.Format("2 Jan, 15:04")
	}
	p.Created = now.UTC()
	p.ID = now.Format("20060102-150405")
	dir := filepath.Join(root, p.ID)
	for n := 2; exists(dir); n++ {
		p.ID = fmt.Sprintf("%s-%d", now.Format("20060102-150405"), n)
		dir = filepath.Join(root, p.ID)
	}
	if tries < 1 {
		tries = 1
	}
	tmp := dir + tmpSuffix
	var lastErr error
	for t := 0; t < tries; t++ {
		if t > 0 {
			time.Sleep(wait)
		}
		_ = removeAll(tmp)
		p.Locs, p.Size, p.Paths = nil, 0, map[string]string{}
		steady := true
		lastErr = nil
		for _, l := range locs {
			before, err := stampOf(l.Live)
			if err != nil {
				lastErr = err
				break
			}
			if !HasFiles(l.Live) {
				continue
			}
			if err := Mirror(l.Live, filepath.Join(tmp, "files", l.ID)); err != nil {
				lastErr = err
				break
			}
			after, err := stampOf(l.Live)
			if err != nil || !sameStamps(before, after) {
				steady = false
				break
			}
			p.Locs = append(p.Locs, l.ID)
			if l.Path != "" {
				p.Paths[l.ID] = l.Path
			}
			p.Size += Size(filepath.Join(tmp, "files", l.ID))
		}
		if lastErr == nil && steady {
			if len(p.Locs) == 0 {
				_ = removeAll(tmp)
				return SavePoint{}, errors.New("there are no saves to keep yet")
			}
			b, _ := json.MarshalIndent(p, "", "  ")
			if err := os.WriteFile(filepath.Join(tmp, "point.json"), b, 0o644); err != nil {
				_ = removeAll(tmp)
				return SavePoint{}, err
			}
			if err := os.Rename(tmp, dir); err != nil {
				_ = removeAll(tmp)
				return SavePoint{}, err
			}
			return p, nil
		}
		if lastErr == nil {
			lastErr = errors.New("the game kept writing its saves while they were copied; try again in a moment")
		}
	}
	_ = removeAll(tmp)
	return SavePoint{}, lastErr
}

func sameStamps(a, b map[string]FileStamp) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// SavePoints lists a game's points, newest first.
func SavePoints(root string) []SavePoint {
	es, _ := os.ReadDir(root)
	var out []SavePoint
	for _, e := range es {
		if !e.IsDir() || strings.HasSuffix(e.Name(), tmpSuffix) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, e.Name(), "point.json"))
		if err != nil {
			continue
		}
		var p SavePoint
		if json.Unmarshal(b, &p) == nil && p.ID == e.Name() {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out
}

func loadPoint(root, id string) (SavePoint, string, error) {
	if id == "" || strings.HasPrefix(id, ".") || strings.ContainsAny(id, `/\:`) {
		return SavePoint{}, "", errors.New("unknown save point")
	}
	dir := filepath.Join(root, id)
	b, err := os.ReadFile(filepath.Join(dir, "point.json"))
	if err != nil {
		return SavePoint{}, "", errors.New("that save point is gone")
	}
	var p SavePoint
	if err := json.Unmarshal(b, &p); err != nil {
		return SavePoint{}, "", err
	}
	return p, dir, nil
}

// LoadSavePoint reads one point.
func LoadSavePoint(root, id string) (SavePoint, error) {
	p, _, err := loadPoint(root, id)
	return p, err
}

// RestoreSavePoint puts a point's saves back in place. Locations the point
// has no files for are left as they are. The caller keeps the saves as they
// are now first (an Auto point).
func RestoreSavePoint(root, id string, locs []SaveLoc) error {
	p, dir, err := loadPoint(root, id)
	if err != nil {
		return err
	}
	live := map[string]string{}
	for _, l := range locs {
		live[l.ID] = l.Live
	}
	for _, lid := range p.Locs {
		if live[lid] == "" {
			return fmt.Errorf("the game doesn't keep its saves where it did when %q was made", p.Name)
		}
	}
	for _, lid := range p.Locs {
		if err := Mirror(filepath.Join(dir, "files", lid), live[lid]); err != nil {
			return err
		}
	}
	return nil
}

// RenameSavePoint gives a point a new name.
func RenameSavePoint(root, id, name string) error {
	p, dir, err := loadPoint(root, id)
	if err != nil {
		return err
	}
	if p.Name = cleanPointName(name); p.Name == "" {
		return errors.New("give it a name")
	}
	b, _ := json.MarshalIndent(p, "", "  ")
	return os.WriteFile(filepath.Join(dir, "point.json"), b, 0o644)
}

// DeleteSavePoint removes a point.
func DeleteSavePoint(root, id string) error {
	_, dir, err := loadPoint(root, id)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// SaveLocs are the places a game with profiles keeps its saves (its profile
// paths, without its achievements, which a point doesn't take back).
func (e *Engine) SaveLocs(g *Game) ([]SaveLoc, error) {
	rs, err := e.resolveAll(g)
	if err != nil {
		return nil, err
	}
	var out []SaveLoc
	for _, r := range rs {
		if !strings.HasPrefix(r.entry.ID, AchEntryPrefix) {
			out = append(out, SaveLoc{ID: r.entry.ID, Live: r.live, Path: r.entry.Path})
		}
	}
	return out, nil
}

// ProfileFromSavePoint starts playerID's profile for a game from a point:
// a new run from that moment. Locations the point has no files for start
// empty.
func (e *Engine) ProfileFromSavePoint(root, id string, g *Game, playerID string) error {
	p, dir, err := loadPoint(root, id)
	if err != nil {
		return err
	}
	if e.HasProfile(playerID, g.ID) {
		return errors.New("that profile already has files for this game")
	}
	locs, err := e.SaveLocs(g)
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, l := range p.Locs {
		have[l] = true
	}
	dst := e.Store.ProfileDir(playerID, g.ID)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, l := range locs {
		if have[l.ID] {
			if err := Mirror(filepath.Join(dir, "files", l.ID), filepath.Join(dst, l.ID)); err != nil {
				return err
			}
		}
	}
	if err := os.WriteFile(filepath.Join(dst, initMarker), []byte(e.now().Format(time.RFC3339)), 0o644); err != nil {
		return err
	}
	e.logf("Started %s's files for %s from the save point %q", e.name(playerID), g.Name, p.Name)
	return nil
}
